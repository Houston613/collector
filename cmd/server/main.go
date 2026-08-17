package main

import (
	"collector/internal/audit"
	"collector/internal/config"
	"collector/internal/grpcserver"
	"collector/internal/handler"
	"collector/internal/middleware"
	pb "collector/internal/proto/gen"
	"collector/internal/repository"
	"collector/internal/version"
	"collector/pkg/crypto"
	"context"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	version.PrintBuildInfo()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
	}
}

func run() error {
	cfgVal, err := config.ParseServerConfig(os.Args[1:])
	if err != nil {
		return err
	}

	// Initialize the logger
	cfg := zap.NewProductionEncoderConfig()
	cfg.TimeKey = "timestamp"
	cfg.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder

	log := zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(cfg),
		zapcore.AddSync(os.Stdout),
		zapcore.InfoLevel,
	))
	defer log.Sync()

	// Select storage back-end: DB, File, or In-Memory
	var storage repository.MemRepository
	ctx := context.Background()

	if cfgVal.DbDSN != "" {
		dbStorage, err := repository.NewDBStorage(ctx, cfgVal.DbDSN)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		if err := dbStorage.Bootstrap("migrations"); err != nil {
			return fmt.Errorf("failed to run database migrations: %w", err)
		}
		storage = dbStorage
		log.Info("using database storage backend")
	} else if cfgVal.FileStoragePath != "" {
		fileStorage := repository.NewFileBackedStorage(cfgVal.FileStoragePath, cfgVal.StoreInterval == 0, log)

		if cfgVal.Restore {
			if err := fileStorage.Load(); err != nil {
				log.Warn("failed to load metrics from file", zap.String("path", cfgVal.FileStoragePath), zap.Error(err))
			} else {
				log.Info("metrics loaded from file", zap.String("path", cfgVal.FileStoragePath))
			}
		}

		// Periodic disk synchronization loop
		if cfgVal.StoreInterval > 0 {
			go func() {
				ticker := time.NewTicker(time.Duration(cfgVal.StoreInterval) * time.Second)
				defer ticker.Stop()
				for range ticker.C {
					if err := fileStorage.Save(); err != nil {
						log.Error("failed to save metrics to file", zap.String("path", cfgVal.FileStoragePath), zap.Error(err))
					} else {
						log.Info("metrics saved to file", zap.String("path", cfgVal.FileStoragePath))
					}
				}
			}()
		}

		storage = fileStorage
		log.Info("using file-backed storage", zap.String("path", cfgVal.FileStoragePath))
	} else {
		storage = repository.NewStructMem()
		log.Info("using in-memory storage backend")
	}

	// Setup auditor: register observers based on configured parameters
	var observers []audit.Observer
	if cfgVal.AuditFilePath != "" {
		fo, err := audit.NewFileObserver(cfgVal.AuditFilePath)
		if err != nil {
			return fmt.Errorf("failed to open audit file: %w", err)
		}
		observers = append(observers, fo)
		log.Info("file auditing enabled", zap.String("path", cfgVal.AuditFilePath))
	}
	if cfgVal.AuditURL != "" {
		ho := audit.NewHTTPObserver(cfgVal.AuditURL)
		observers = append(observers, ho)
		log.Info("HTTP auditing enabled", zap.String("url", cfgVal.AuditURL))
	}

	var auditor *audit.Notifier
	if len(observers) > 0 {
		auditor = audit.NewNotifier(log, observers...)
		defer func() {
			if err := auditor.Close(); err != nil {
				log.Error("failed to close auditor", zap.Error(err))
			}
		}()
	}

	var privKey *rsa.PrivateKey
	if cfgVal.CryptoKeyPath != "" {
		pk, err := crypto.LoadPrivateKey(cfgVal.CryptoKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load private key from %s: %w", cfgVal.CryptoKeyPath, err)
		}
		privKey = pk
		log.Info("asymmetric decryption enabled", zap.String("key_path", cfgVal.CryptoKeyPath))
	}

	e := echo.New()
	// Logger is implemented via middleware
	e.Use(middleware.RequestLogger(log))
	e.Use(middleware.TrustedSubnetMiddleware(cfgVal.TrustedSubnet, log))
	// Order of middleware: decrypt first, then decompress, then sign
	if privKey != nil {
		e.Use(middleware.CryptoMiddleware(privKey, log))
	}
	e.Use(middleware.GzipMiddleware(log))
	e.Use(middleware.SignatureMiddleware(cfgVal.Key, log))

	// TODO: Pass configuration object instead of connection DSN string directly
	metricsHandler := handler.NewMetricsHandler(storage, cfgVal.DbDSN, auditor, log)
	metricsHandler.RegisterRoutes(e)

	var grpcServer *grpc.Server
	if cfgVal.GrpcAddress != "" {
		lis, err := net.Listen("tcp", cfgVal.GrpcAddress)
		if err != nil {
			return fmt.Errorf("failed to listen on gRPC address %s: %w", cfgVal.GrpcAddress, err)
		}

		grpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(grpcserver.TrustedSubnetInterceptor(cfgVal.TrustedSubnet, log)),
		)
		pb.RegisterMetricsServer(grpcServer, grpcserver.NewMetricsServer(storage, log))

		go func() {
			log.Info("starting gRPC server", zap.String("addr", cfgVal.GrpcAddress))
			if err := grpcServer.Serve(lis); err != nil {
				log.Error("gRPC server stopped with error", zap.Error(err))
			}
		}()
	}

	// Listen for SIGINT, SIGTERM, SIGQUIT signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Start Echo HTTP server in a background goroutine
	go func() {
		if err := e.Start(cfgVal.Addr); err != nil && err != http.ErrServerClosed {
			log.Error("server stopped with error", zap.Error(err))
		}
	}()

	// Block until a signal is received
	sig := <-sigChan
	log.Info("received shutdown signal", zap.String("signal", sig.String()))

	// Create a context for graceful shutdown of the HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop Echo HTTP server
	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error("failed to gracefully shutdown HTTP server", zap.Error(err))
	} else {
		log.Info("HTTP server stopped gracefully")
	}

	if grpcServer != nil {
		grpcServer.GracefulStop()
		log.Info("gRPC server stopped gracefully")
	}

	// Save storage state if supported on shutdown
	if saver, ok := storage.(repository.Saver); ok {
		if err := saver.Save(); err != nil {
			log.Error("failed to save metrics on shutdown", zap.Error(err))
		} else {
			log.Info("metrics saved on shutdown")
		}
	}

	// Close storage resources if supported on shutdown
	if closer, ok := storage.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			log.Error("failed to close storage on shutdown", zap.Error(err))
		} else {
			log.Info("storage closed gracefully")
		}
	}

	return nil
}
