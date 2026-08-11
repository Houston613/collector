package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
)

// ServerConfig represents the JSON configuration file layout for the server.
type ServerConfig struct {
	Address       *string        `json:"address"`
	Restore       *bool          `json:"restore"`
	StoreInterval *DurationOrInt `json:"store_interval"`
	StoreFile     *string        `json:"store_file"`
	DatabaseDSN   *string        `json:"database_dsn"`
	CryptoKey     *string        `json:"crypto_key"`
	Key           *string        `json:"key"`
	AuditFile     *string        `json:"audit_file"`
	AuditURL      *string        `json:"audit_url"`
}

// ServerOptions contains the fully parsed and prioritized configuration options for the server.
type ServerOptions struct {
	Addr            string
	StoreInterval   int
	FileStoragePath string
	Restore         bool
	DbDSN           string
	Key             string
	AuditFilePath   string
	AuditURL        string
	CryptoKeyPath   string
}


func ParseServerConfig(args []string) (*ServerOptions, error) {
	// Using a local FlagSet instead of global flag package variables (flag.CommandLine)
	// provides isolation in unit tests (avoiding global state contamination and panics
	// on redefine/reparse) and allows returning errors instead of calling os.Exit.
	fs := flag.NewFlagSet("server", flag.ContinueOnError)

	addr := fs.String("a", "localhost:8080", "HTTP server address")
	storeInterval := fs.Int("i", 300, "metrics save interval (seconds, 0 for sync)")
	fileStoragePath := fs.String("f", "/tmp/metrics-storage.json", "path to metrics storage file")
	restore := fs.Bool("r", true, "restore previously saved metrics on startup")
	dbDSN := fs.String("d", "", "database connection DSN string")
	key := fs.String("k", "", "key for data signing")
	auditFilePath := fs.String("audit-file", "", "path to audit file (empty to disable file audit)")
	auditURL := fs.String("audit-url", "", "audit server URL (empty to disable HTTP audit)")
	cryptoKeyPath := fs.String("crypto-key", "", "path to file with RSA private key")

	var configPath string
	fs.StringVar(&configPath, "c", "", "path to JSON configuration file")
	fs.StringVar(&configPath, "config", "", "path to JSON configuration file")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if fs.NArg() > 0 {
		return nil, fmt.Errorf("unknown arguments: %v", fs.Args())
	}

	// Environment variables override configPath
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
	restoreVal := *restore
	storeIntervalVal := *storeInterval
	fileStoragePathVal := *fileStoragePath
	dbDSNVal := *dbDSN
	cryptoKeyPathVal := *cryptoKeyPath
	keyVal := *key
	auditFilePathVal := *auditFilePath
	auditURLVal := *auditURL

	// Apply configuration file if specified and exists
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		var fileCfg ServerConfig
		if err := json.Unmarshal(data, &fileCfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
		}

		if fileCfg.Address != nil && !isFlagPassedLocal("a") {
			addrVal = *fileCfg.Address
		}
		if fileCfg.Restore != nil && !isFlagPassedLocal("r") {
			restoreVal = *fileCfg.Restore
		}
		if fileCfg.StoreInterval != nil && !isFlagPassedLocal("i") {
			if val, err := parseDurationOrInt(string(*fileCfg.StoreInterval)); err == nil {
				storeIntervalVal = val
			}
		}
		if fileCfg.StoreFile != nil && !isFlagPassedLocal("f") {
			fileStoragePathVal = *fileCfg.StoreFile
		}
		if fileCfg.DatabaseDSN != nil && !isFlagPassedLocal("d") {
			dbDSNVal = *fileCfg.DatabaseDSN
		}
		if fileCfg.CryptoKey != nil && !isFlagPassedLocal("crypto-key") {
			cryptoKeyPathVal = *fileCfg.CryptoKey
		}
		if fileCfg.Key != nil && !isFlagPassedLocal("k") {
			keyVal = *fileCfg.Key
		}
		if fileCfg.AuditFile != nil && !isFlagPassedLocal("audit-file") {
			auditFilePathVal = *fileCfg.AuditFile
		}
		if fileCfg.AuditURL != nil && !isFlagPassedLocal("audit-url") {
			auditURLVal = *fileCfg.AuditURL
		}
	}

	// Environment variables take precedence over command-line flags and config files
	if envAddr := os.Getenv("ADDRESS"); envAddr != "" {
		addrVal = envAddr
	}
	if v := os.Getenv("STORE_INTERVAL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			storeIntervalVal = n
		}
	}
	if v := os.Getenv("FILE_STORAGE_PATH"); v != "" {
		fileStoragePathVal = v
	}
	if v := os.Getenv("STORE_FILE"); v != "" {
		fileStoragePathVal = v
	}
	if v := os.Getenv("RESTORE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			restoreVal = b
		}
	}
	if v := os.Getenv("DATABASE_DSN"); v != "" {
		dbDSNVal = v
	}
	if v := os.Getenv("KEY"); v != "" {
		keyVal = v
	}
	if v := os.Getenv("AUDIT_FILE"); v != "" {
		auditFilePathVal = v
	}
	if v := os.Getenv("AUDIT_URL"); v != "" {
		auditURLVal = v
	}
	if v := os.Getenv("CRYPTO_KEY"); v != "" {
		cryptoKeyPathVal = v
	}

	return &ServerOptions{
		Addr:            addrVal,
		StoreInterval:   storeIntervalVal,
		FileStoragePath: fileStoragePathVal,
		Restore:         restoreVal,
		DbDSN:           dbDSNVal,
		Key:             keyVal,
		AuditFilePath:   auditFilePathVal,
		AuditURL:        auditURLVal,
		CryptoKeyPath:   cryptoKeyPathVal,
	}, nil
}
