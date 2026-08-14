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

func TestParseAgentConfig(t *testing.T) {
	// Helper to clear environment variables that might interfere with tests
	cleanEnv := func() {
		os.Unsetenv("ADDRESS")
		os.Unsetenv("REPORT_INTERVAL")
		os.Unsetenv("POLL_INTERVAL")
		os.Unsetenv("KEY")
		os.Unsetenv("RATE_LIMIT")
		os.Unsetenv("CRYPTO_KEY")
		os.Unsetenv("CONFIG")
	}

	t.Run("default values", func(t *testing.T) {
		cleanEnv()
		cfg, err := config.ParseAgentConfig([]string{})
		require.NoError(t, err)
		assert.Equal(t, "localhost:8080", cfg.Addr)
		assert.Equal(t, 10, cfg.ReportInterval)
		assert.Equal(t, 2, cfg.PollInterval)
		assert.Equal(t, "", cfg.Key)
		assert.Equal(t, 3, cfg.RateLimit)
		assert.Equal(t, "", cfg.CryptoKeyPath)
	})

	t.Run("flags override defaults", func(t *testing.T) {
		cleanEnv()
		cfg, err := config.ParseAgentConfig([]string{
			"-a", "localhost:9090",
			"-r", "20",
			"-p", "5",
			"-k", "mykey",
			"-l", "10",
			"-crypto-key", "/tmp/pub.key",
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost:9090", cfg.Addr)
		assert.Equal(t, 20, cfg.ReportInterval)
		assert.Equal(t, 5, cfg.PollInterval)
		assert.Equal(t, "mykey", cfg.Key)
		assert.Equal(t, 10, cfg.RateLimit)
		assert.Equal(t, "/tmp/pub.key", cfg.CryptoKeyPath)
	})

	t.Run("config file", func(t *testing.T) {
		cleanEnv()
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		cfgData := map[string]interface{}{
			"address":         "localhost:7070",
			"report_interval": "15s",
			"poll_interval":   "3s",
			"crypto_key":      "/tmp/key.pem",
			"key":             "jsonkey",
			"rate_limit":      5,
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		cfg, err := config.ParseAgentConfig([]string{"-c", configPath})
		require.NoError(t, err)
		assert.Equal(t, "localhost:7070", cfg.Addr)
		assert.Equal(t, 15, cfg.ReportInterval)
		assert.Equal(t, 3, cfg.PollInterval)
		assert.Equal(t, "/tmp/key.pem", cfg.CryptoKeyPath)
		assert.Equal(t, "jsonkey", cfg.Key)
		assert.Equal(t, 5, cfg.RateLimit)
	})

	t.Run("flags override config file", func(t *testing.T) {
		cleanEnv()
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		cfgData := map[string]interface{}{
			"address":         "localhost:7070",
			"report_interval": "15s",
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		cfg, err := config.ParseAgentConfig([]string{
			"-c", configPath,
			"-a", "localhost:9090",
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost:9090", cfg.Addr) // flag overrides config
		assert.Equal(t, 15, cfg.ReportInterval)      // config value applied
	})

	t.Run("env vars override flags and config", func(t *testing.T) {
		cleanEnv()
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.json")

		cfgData := map[string]interface{}{
			"address":         "localhost:7070",
			"report_interval": "15s",
		}
		data, err := json.Marshal(cfgData)
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		os.Setenv("ADDRESS", "localhost:9999")
		os.Setenv("REPORT_INTERVAL", "45")
		defer cleanEnv()

		cfg, err := config.ParseAgentConfig([]string{
			"-c", configPath,
			"-a", "localhost:9090",
			"-r", "20",
		})
		require.NoError(t, err)
		assert.Equal(t, "localhost:9999", cfg.Addr) // env overrides flag and config
		assert.Equal(t, 45, cfg.ReportInterval)      // env overrides flag and config
	})
}
