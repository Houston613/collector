package main

import (
	"collector/internal/agent"
	"collector/internal/config"
	"collector/internal/version"
	"collector/pkg/crypto"
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	version.PrintBuildInfo()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
	}
}

func run() error {
	cfgVal, err := config.ParseAgentConfig(os.Args[1:])
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

	var pubKey *rsa.PublicKey
	if cfgVal.CryptoKeyPath != "" {
		pk, err := crypto.LoadPublicKey(cfgVal.CryptoKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load public key from %s: %w", cfgVal.CryptoKeyPath, err)
		}
		pubKey = pk
		log.Info("asymmetric encryption enabled", zap.String("key_path", cfgVal.CryptoKeyPath))
	}

	a := agent.NewAgent(
		// Prepend the http:// protocol scheme to the address
		"http://"+cfgVal.Addr,
		time.Duration(cfgVal.PollInterval)*time.Second,
		time.Duration(cfgVal.ReportInterval)*time.Second,
		cfgVal.Key,
		pubKey,
		cfgVal.RateLimit,
		log,
	)

	// Listen for SIGINT, SIGTERM, SIGQUIT signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := <-sigChan
		log.Info("received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	a.Run(ctx)
	return nil
}
