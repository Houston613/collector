package main

import (
	"collector/internal/config"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServerConfig(t *testing.T) {
	// Helper to clear environment variables that might interfere with tests
	cleanEnv := func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("STORE_INTERVAL")
		os.Unsetenv("FILE_STORAGE_PATH")
		os.Unsetenv("STORE_FILE")
		os.Unsetenv("RESTORE")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("KEY")
		os.Unsetenv("AUDIT_FILE")
		os.Unsetenv("AUDIT_URL")
		os.Unsetenv("CRYPTO_KEY")
		os.Unsetenv("CONFIG")
	}

	t.Run("default values", func(t *testing.T) {
		cleanEnv()
		cfg, err := config.ParseServerConfig([]string{})
		require.NoError(t, err)
		assert.Equal(t, "localhost:8080", cfg.Addr)
		assert.Equal(t, 300, cfg.StoreInterval)
		assert.Equal(t, "/tmp/metrics-storage.json", cfg.FileStoragePath)
		assert.True(t, cfg.Restore)
		assert.Equal(t, "", cfg.DbDSN)
		assert.Equal(t, "", cfg.Key)
		assert.Equal(t, "", cfg.AuditFilePath)
		assert.Equal(t, "", cfg.AuditURL)
		assert.Equal(t, "", cfg.CryptoKeyPath)
	})

	t.Run("flags override defaults", func(t *testing.T) {
		cleanEnv()
		cfg, err := config.ParseServerConfig([]string{
			"-a", "localhost:9090",
			"-i", "10",
			"-f", "/tmp/other.json",
			"-r=false",
			"-d", "postgres://user@localhost/db",
			"-k", "mykey",
			"-audit-file", "/tmp/audit.log",
			"-audit-url", "http://auditor",
			"-crypto-key", "/tmp/priv.key",
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost:9090", cfg.Addr)
		assert.Equal(t, 10, cfg.StoreInterval)
		assert.Equal(t, "/tmp/other.json", cfg.FileStoragePath)
		assert.False(t, cfg.Restore)
		assert.Equal(t, "postgres://user@localhost/db", cfg.DbDSN)
		assert.Equal(t, "mykey", cfg.Key)
		assert.Equal(t, "/tmp/audit.log", cfg.AuditFilePath)
		assert.Equal(t, "http://auditor", cfg.AuditURL)
		assert.Equal(t, "/tmp/priv.key", cfg.CryptoKeyPath)
	})

	t.Run("config file", func(t *testing.T) {
		cleanEnv()
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		cfgData := map[string]interface{}{
			"address":        "localhost:7070",
			"restore":        false,
			"store_interval": "15s",
			"store_file":     "/tmp/store.db",
			"database_dsn":   "postgres://db",
			"crypto_key":     "/tmp/key.pem",
			"key":            "jsonkey",
			"audit_file":     "/tmp/audit.json",
			"audit_url":      "http://jsonaudit",
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		cfg, err := config.ParseServerConfig([]string{"-c", configPath})
		require.NoError(t, err)
		assert.Equal(t, "localhost:7070", cfg.Addr)
		assert.False(t, cfg.Restore)
		assert.Equal(t, 15, cfg.StoreInterval)
		assert.Equal(t, "/tmp/store.db", cfg.FileStoragePath)
		assert.Equal(t, "postgres://db", cfg.DbDSN)
		assert.Equal(t, "/tmp/key.pem", cfg.CryptoKeyPath)
		assert.Equal(t, "jsonkey", cfg.Key)
		assert.Equal(t, "/tmp/audit.json", cfg.AuditFilePath)
		assert.Equal(t, "http://jsonaudit", cfg.AuditURL)
	})

	t.Run("flags override config file", func(t *testing.T) {
		cleanEnv()
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		cfgData := map[string]interface{}{
			"address":        "localhost:7070",
			"store_interval": "15s",
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		cfg, err := config.ParseServerConfig([]string{
			"-c", configPath,
			"-a", "localhost:9090",
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost:9090", cfg.Addr) // flag overrides config
		assert.Equal(t, 15, cfg.StoreInterval)      // config value applied
	})

	t.Run("env vars override flags and config", func(t *testing.T) {
		cleanEnv()
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		cfgData := map[string]interface{}{
			"address":        "localhost:7070",
			"store_interval": "15s",
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		os.Setenv("ADDRESS", "localhost:9999")
		os.Setenv("STORE_INTERVAL", "45")
		os.Setenv("STORE_FILE", "/tmp/env.db")
		defer cleanEnv()

		cfg, err := config.ParseServerConfig([]string{
			"-c", configPath,
			"-a", "localhost:9090",
			"-i", "20",
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost:9999", cfg.Addr) // env overrides flag and config
		assert.Equal(t, 45, cfg.StoreInterval)      // env overrides flag and config
		assert.Equal(t, "/tmp/env.db", cfg.FileStoragePath)
	})
}
