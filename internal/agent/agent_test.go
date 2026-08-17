package agent

import (
	"bytes"
	"collector/pkg/crypto"
	"compress/gzip"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	models "collector/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMetricsCollect_PopulatesGauges(t *testing.T) {
	expectedGauges := []string{
		"Alloc", "BuckHashSys", "Frees", "GCCPUFraction", "GCSys",
		"HeapAlloc", "HeapIdle", "HeapInuse", "HeapObjects", "HeapReleased",
		"HeapSys", "LastGC", "Lookups", "MCacheInuse", "MCacheSys",
		"MSpanInuse", "MSpanSys", "Mallocs", "NextGC", "NumForcedGC",
		"NumGC", "OtherSys", "PauseTotalNs", "StackInuse", "StackSys",
		"Sys", "TotalAlloc", "RandomValue",
	}

	a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", nil, 1, zap.NewNop())
	a.MetricsCollect()

	// Lock is set to simulate a production-like concurrent access scenario
	a.mu.RLock()
	// Using defer here as the test method may exit before the loop completes
	defer a.mu.RUnlock()

	for _, name := range expectedGauges {
		assert.Contains(t, a.gaugesMetrics, name, "gauge %q not found after MetricsCollect", name)
	}
}

// TestMetricsCollect_IncrementsPollCount verifies that PollCount increments correctly.
func TestMetricsCollect_IncrementsPollCount(t *testing.T) {
	tests := []struct {
		name  string
		calls int
		want  int64
	}{
		{"once", 1, 1},
		{"five calls", 5, 5},
		{"hundred calls", 100, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", nil, 1, zap.NewNop())
			for i := 0; i < tt.calls; i++ {
				a.MetricsCollect()
			}

			count := a.countersMetrics["PollCount"]
			assert.Equal(t, tt.want, count)
		})
	}
}

func TestMetricsCollect_UpdatesGaugeValues(t *testing.T) {
	tests := []struct {
		name  string
		calls int
	}{
		{"once", 1},
		{"five calls", 5},
		{"hundred calls", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", nil, 1, zap.NewNop())
			for i := 0; i < tt.calls; i++ {
				a.MetricsCollect()
			}

			a.mu.RLock()
			defer a.mu.RUnlock()

			assert.Greater(t, a.gaugesMetrics["Alloc"], 0.0)
		})
	}
}

func TestMetricsCollectGopsutil(t *testing.T) {
	a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", nil, 1, zap.NewNop())
	a.MetricsCollectGopsutil()

	a.mu.RLock()
	defer a.mu.RUnlock()

	assert.Contains(t, a.gaugesMetrics, "TotalMemory")
	assert.Contains(t, a.gaugesMetrics, "FreeMemory")
	assert.Contains(t, a.gaugesMetrics, "CPUutilization1")

	assert.Greater(t, a.gaugesMetrics["TotalMemory"], 0.0)
	assert.GreaterOrEqual(t, a.gaugesMetrics["FreeMemory"], 0.0)
}

func TestMetricsSendBatch_JSONFormat(t *testing.T) {
	tests := []struct {
		name     string
		gauges   map[string]float64
		counters map[string]int64
		want     models.Metrics
	}{
		{
			name:     "gauge",
			gauges:   map[string]float64{"TestGauge": 27.54},
			counters: map[string]int64{},
			want: models.Metrics{
				ID:    "TestGauge",
				MType: models.Gauge,
				Value: func() *float64 { v := 27.54; return &v }(),
			},
		},
		{
			name:     "counter",
			gauges:   map[string]float64{},
			counters: map[string]int64{"TestCounter": 17},
			want: models.Metrics{
				ID:    "TestCounter",
				MType: models.Counter,
				Delta: func() *int64 { v := int64(17); return &v }(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received []models.Metrics

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/updates/", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				assert.Equal(t, "gzip", r.Header.Get("Content-Encoding"))
				assert.NotEmpty(t, r.Header.Get("X-Real-IP"))
				// Check if request is gzip compressed
				var reader io.Reader = r.Body
				if r.Header.Get("Content-Encoding") == "gzip" {
					gr, err := gzip.NewReader(r.Body)
					require.NoError(t, err)
					defer gr.Close()
					reader = gr
				}

				body, err := io.ReadAll(reader)
				require.NoError(t, err)

				var m []models.Metrics
				require.NoError(t, json.Unmarshal(body, &m))
				received = append(received, m...)

				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			// Use zap.NewNop() to suppress logger outputs in tests and keep output clean
			a := NewAgent(srv.URL, DefaultPollInterval, DefaultReportInterval, "", nil, 1, zap.NewNop())
			a.mu.Lock()
			maps.Copy(a.gaugesMetrics, tt.gauges)
			maps.Copy(a.countersMetrics, tt.counters)
			a.mu.Unlock()

			jobs := make(chan []models.Metrics, 1)
			a.MetricsSend(jobs)
			m := <-jobs
			require.NoError(t, a.sender.Send(context.Background(), m))

			assert.Contains(t, received, tt.want)
		})
	}
}

func TestAgentAsymmetricEncryption(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	publicKey := &privateKey.PublicKey

	var received []models.Metrics

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		decrypted, err := crypto.Decrypt(privateKey, body)
		require.NoError(t, err)

		gr, err := gzip.NewReader(bytes.NewReader(decrypted))
		require.NoError(t, err)
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		require.NoError(t, err)

		var m []models.Metrics
		require.NoError(t, json.Unmarshal(decompressed, &m))
		received = m
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := NewAgent(srv.URL, DefaultPollInterval, DefaultReportInterval, "", publicKey, 1, zap.NewNop())
	a.mu.Lock()
	val := 42.0
	a.gaugesMetrics["EncryptedMetric"] = val
	a.mu.Unlock()

	jobs := make(chan []models.Metrics, 1)
	a.MetricsSend(jobs)
	m := <-jobs
	require.NoError(t, a.sender.Send(context.Background(), m))

	require.Len(t, received, 1)
	assert.Equal(t, "EncryptedMetric", received[0].ID)
	assert.Equal(t, 42.0, *received[0].Value)
}

func TestAgentGracefulShutdown(t *testing.T) {
	var received []models.Metrics
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		gr, err := gzip.NewReader(bytes.NewReader(body))
		require.NoError(t, err)
		defer gr.Close()

		decompressed, err := io.ReadAll(gr)
		require.NoError(t, err)

		var m []models.Metrics
		require.NoError(t, json.Unmarshal(decompressed, &m))
		
		mu.Lock()
		received = append(received, m...)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// High poll and report intervals to ensure standard tickers don't trigger
	a := NewAgent(srv.URL, 10*time.Hour, 10*time.Hour, "", nil, 1, zap.NewNop())
	
	a.mu.Lock()
	val := 123.45
	a.gaugesMetrics["FinalMetric"] = val
	a.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	
	runDone := make(chan struct{})
	go func() {
		a.Run(ctx)
		close(runDone)
	}()

	// Give it a tiny bit of time to start up, then cancel
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Wait for Run to finish
	select {
	case <-runDone:
	case <-time.After(3 * time.Second):
		t.Fatal("agent failed to shutdown gracefully in time")
	}

	mu.Lock()
	defer mu.Unlock()
	require.NotEmpty(t, received)
	
	found := false
	for _, m := range received {
		if m.ID == "FinalMetric" {
			found = true
			assert.Equal(t, 123.45, *m.Value)
		}
	}
	assert.True(t, found, "expected final metric to be sent during shutdown")
}

func TestGRPCSender_ConcurrentSend(t *testing.T) {
	sender := NewGRPCSender("localhost:50051", "127.0.0.1", zap.NewNop())
	defer sender.Close()

	var wg sync.WaitGroup
	metrics := []models.Metrics{
		{
			ID:    "TestGauge",
			MType: models.Gauge,
			Value: func() *float64 { v := 1.23; return &v }(),
		},
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			_ = sender.Send(ctx, metrics)
		}()
	}

	wg.Wait()
}
