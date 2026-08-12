package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

// AgentConfig represents the JSON configuration file layout for the agent.
type AgentConfig struct {
	Address        *string        `json:"address"`
	ReportInterval *DurationOrInt `json:"report_interval"`
	PollInterval   *DurationOrInt `json:"poll_interval"`
	CryptoKey      *string        `json:"crypto_key"`
	Key            *string        `json:"key"`
	RateLimit      *int           `json:"rate_limit"`
}

type AgentOptions struct {
	Addr           string
	ReportInterval int
	PollInterval   int
	Key            string
	RateLimit      int
	CryptoKeyPath  string
}


func ParseAgentConfig(args []string) (*AgentOptions, error) {
	// Using a local FlagSet instead of global flag package variables (flag.CommandLine)
	// provides isolation in unit tests (avoiding global state contamination and panics
	// on redefine/reparse) and allows returning errors instead of calling os.Exit.
	fs := flag.NewFlagSet("agent", flag.ContinueOnError)

	addr := fs.String("a", "localhost:8080", "HTTP server address")
	reportInterval := fs.Int("r", 10, "frequency of metric reports (seconds)")
	pollInterval := fs.Int("p", 2, "frequency of metric polling (seconds)")
	key := fs.String("k", "", "key for data signing")
	rateLimit := fs.Int("l", 3, "rate limit for outgoing concurrent requests")
	cryptoKeyPath := fs.String("crypto-key", "", "path to file with RSA public key")

	var configPath string
	fs.StringVar(&configPath, "c", "", "path to JSON configuration file")
	fs.StringVar(&configPath, "config", "", "path to JSON configuration file")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unknown arguments: %v", fs.Args())
	}

	if envConfig := os.Getenv("CONFIG"); envConfig != "" {
		configPath = envConfig
	}

	isFlagPassedLocal := func(name string) bool {
		found := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == name {
				found = true
			}
		})
		return found
	}

	addrVal := *addr
	reportIntervalVal := *reportInterval
	pollIntervalVal := *pollInterval
	cryptoKeyPathVal := *cryptoKeyPath
	keyVal := *key
	rateLimitVal := *rateLimit

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		var fileCfg AgentConfig
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}

		if fileCfg.Address != nil && !isFlagPassedLocal("a") {
			addrVal = *fileCfg.Address
		}
		if fileCfg.ReportInterval != nil && !isFlagPassedLocal("r") {
			if val, err := parseDurationOrInt(string(*fileCfg.ReportInterval)); err == nil {
				reportIntervalVal = val
			}
		}
		if fileCfg.PollInterval != nil && !isFlagPassedLocal("p") {
			if val, err := parseDurationOrInt(string(*fileCfg.PollInterval)); err == nil {
				pollIntervalVal = val
			}
		}
		if fileCfg.CryptoKey != nil && !isFlagPassedLocal("crypto-key") {
			cryptoKeyPathVal = *fileCfg.CryptoKey
		}
		if fileCfg.Key != nil && !isFlagPassedLocal("k") {
			keyVal = *fileCfg.Key
		}
		if fileCfg.RateLimit != nil && !isFlagPassedLocal("l") {
			rateLimitVal = *fileCfg.RateLimit
		}
	}

	// Environment variables take precedence over command-line flags and config files
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		addrVal = envAddr
	}
	if envReport := os.Getenv("REPORT_INTERVAL"); envReport != "" {
		if v, err := strconv.Atoi(envReport); err == nil {
			reportIntervalVal = v
		}
	}
	if envPoll := os.Getenv("POLL_INTERVAL"); envPoll != "" {
		if v, err := strconv.Atoi(envPoll); err == nil {
			pollIntervalVal = v
		}
	}
	if envKey := os.Getenv("KEY"); envKey != "" {
		keyVal = envKey
	}
	if envRateLimit := os.Getenv("RATE_LIMIT"); envRateLimit != "" {
		if v, err := strconv.Atoi(envRateLimit); err == nil {
			rateLimitVal = v
		}
	}
	if envCryptoKey := os.Getenv("CRYPTO_KEY"); envCryptoKey != "" {
		cryptoKeyPathVal = envCryptoKey
	}

	return &AgentOptions{
		Addr:           addrVal,
		ReportInterval: reportIntervalVal,
		PollInterval:   pollIntervalVal,
		Key:            keyVal,
		RateLimit:      rateLimitVal,
		CryptoKeyPath:  cryptoKeyPathVal,
	}, nil
}
