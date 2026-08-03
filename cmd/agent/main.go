package main

import (
	"collector/internal/agent"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "agent error: %v\n", err)
	}
}

func run() error {
	// Parse command-line flags
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	reportInterval := flag.Int("r", 10, "frequency of metric reports (seconds)")
	pollInterval := flag.Int("p", 2, "frequency of metric polling (seconds)")
	key := flag.String("k", "", "key for data signing")
	rateLimit := flag.Int("l", 3, "rate limit for outgoing concurrent requests")
	flag.Parse()

	// Check for unexpected positional arguments
	if flag.NArg() > 0 {
		return fmt.Errorf("unknown arguments: %v", flag.Args())
	}

	// Environment variables take precedence over command-line flags
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		*addr = envAddr
	}
	if envReport := os.Getenv("REPORT_INTERVAL"); envReport != "" {
		if v, err := strconv.Atoi(envReport); err == nil {
			*reportInterval = v
		}
	}
	if envPoll := os.Getenv("POLL_INTERVAL"); envPoll != "" {
		if v, err := strconv.Atoi(envPoll); err == nil {
			*pollInterval = v
		}
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		*key = envKey
	}
	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		if v, err := strconv.Atoi(envRateLimit); err == nil {
			*rateLimit = v
		}
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

	a := agent.NewAgent(
		// Prepend the http:// protocol scheme to the address
		"http://"+*addr,
		time.Duration(*pollInterval)*time.Second,
		time.Duration(*reportInterval)*time.Second,
		*key,
		*rateLimit,
		log,
	)
	a.Run()
	return nil
}
