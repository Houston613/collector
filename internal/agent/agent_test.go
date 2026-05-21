package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

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

	a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", 1, zap.NewNop())
	a.MetricsCollect()

	//все равно ставим блокировку, т.к больше похоже на настоящи кейс
	//
	a.mu.RLock()
	//здесь уже можно defer, потому-что тест может завершиться раньше цикла
	defer a.mu.RUnlock()

	for _, name := range expectedGauges {
		assert.Contains(t, a.gaugesMetrics, name, "gauge %q not found after MetricsCollect", name)
	}
}

// напиши такой же тест для gauge
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
			a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", 1, zap.NewNop())
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
			a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", 1, zap.NewNop())
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
	a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval, "", 1, zap.NewNop())
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
				// Проверяем, что клиент поддерживает gzip-ответы
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

			//NewNop - это заглушка для логгера, которая не будет ничего выводить. Это полезно в тестах, чтобы не засорять вывод.
			a := NewAgent(srv.URL, DefaultPollInterval, DefaultReportInterval, "", 1, zap.NewNop())
			a.mu.Lock()
			maps.Copy(a.gaugesMetrics, tt.gauges)
			maps.Copy(a.countersMetrics, tt.counters)
			a.mu.Unlock()

			jobs := make(chan []models.Metrics, 1)
			a.MetricsSend(jobs)
			m := <-jobs
			require.NoError(t, a.sendBatchJSON(m))

			assert.Contains(t, received, tt.want)
		})
	}
}
