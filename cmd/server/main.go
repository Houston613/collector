package main

import (
	"collector/internal/audit"
	"collector/internal/handler"
	"collector/internal/middleware"
	"collector/internal/repository"
	"collector/internal/version"
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

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
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	storeInterval := flag.Int("i", 300, "metrics save interval (seconds, 0 for sync)")
	fileStoragePath := flag.String("f", "/tmp/metrics-storage.json", "path to metrics storage file")
	restore := flag.Bool("r", true, "restore previously saved metrics on startup")
	dbDSN := flag.String("d", "", "database connection DSN string")
	key := flag.String("k", "", "key for data signing")
	auditFilePath := flag.String("audit-file", "", "path to audit file (empty to disable file audit)")
	auditURL := flag.String("audit-url", "", "audit server URL (empty to disable HTTP audit)")
	flag.Parse()

	if flag.NArg() > 0 {
		return fmt.Errorf("unknown arguments: %v", flag.Args())
	}

	// Environment variables take precedence over command-line flags
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*storeInterval = n
		}
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		*fileStoragePath = v
	}
	if v := os.Getenv("RESTORE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			*restore = b
		}
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		*dbDSN = v
	}
	if v := os.Getenv("KEY"); v != "" {
		*key = v
	}
	if v := os.Getenv("AUDIT_FILE"); v != "" {
		*auditFilePath = v
	}
	if v := os.Getenv("AUDIT_URL"); v != "" {
		*auditURL = v
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

	if *dbDSN != "" {
		dbStorage, err := repository.NewDBStorage(ctx, *dbDSN)
		if err != nil {
			return fmt.Errorf("failed to initialize database: %w", err)
		}
		if err := dbStorage.Bootstrap("migrations"); err != nil {
			return fmt.Errorf("failed to run database migrations: %w", err)
		}
		storage = dbStorage
		log.Info("using database storage backend")
	} else if *fileStoragePath != "" {
		fileStorage := repository.NewFileBackedStorage(*fileStoragePath, *storeInterval == 0, log)

		if *restore {
			if err := fileStorage.Load(); err != nil {
				log.Warn("failed to load metrics from file", zap.String("path", *fileStoragePath), zap.Error(err))
			} else {
				log.Info("metrics loaded from file", zap.String("path", *fileStoragePath))
			}
		}

		// Periodic disk synchronization loop
		if *storeInterval > 0 {
			go func() {
				ticker := time.NewTicker(time.Duration(*storeInterval) * time.Second)
				defer ticker.Stop()
				for range ticker.C {
					if err := fileStorage.Save(); err != nil {
						log.Error("failed to save metrics to file", zap.String("path", *fileStoragePath), zap.Error(err))
					} else {
						log.Info("metrics saved to file", zap.String("path", *fileStoragePath))
					}
				}
			}()
		}

		storage = fileStorage
		log.Info("using file-backed storage", zap.String("path", *fileStoragePath))
	} else {
		storage = repository.NewStructMem()
		log.Info("using in-memory storage backend")
	}

	// Setup auditor: register observers based on configured parameters
	var observers []audit.Observer
	if *auditFilePath != "" {
		fo, err := audit.NewFileObserver(*auditFilePath)
		if err != nil {
			return fmt.Errorf("failed to open audit file: %w", err)
		}
		observers = append(observers, fo)
		log.Info("file auditing enabled", zap.String("path", *auditFilePath))
	}
	if *auditURL != "" {
		ho := audit.NewHTTPObserver(*auditURL)
		observers = append(observers, ho)
		log.Info("HTTP auditing enabled", zap.String("url", *auditURL))
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

	e := echo.New()
	// Logger is implemented via middleware
	e.Use(middleware.RequestLogger(log))
	// Order of middleware: compress first, then sign
	e.Use(middleware.GzipMiddleware(log))
	e.Use(middleware.SignatureMiddleware(*key, log))

	// TODO: Pass configuration object instead of connection DSN string directly
	metricsHandler := handler.NewMetricsHandler(storage, *dbDSN, auditor, log)
	metricsHandler.RegisterRoutes(e)
	if err := e.Start(*addr); err != nil {
		log.Error("server stopped with error", zap.Error(err))
		return err
	}
	return nil
}
