package agent

import (
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	models "collector/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval)
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
			a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval)
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
			a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval)
			for i := 0; i < tt.calls; i++ {
				a.MetricsCollect()
			}

			a.mu.RLock()
			defer a.mu.RUnlock()

			assert.Greater(t, a.gaugesMetrics["Alloc"], 0.0)
		})
	}
}

func TestMetricsCollect_RandomValueInRange(t *testing.T) {
	a := NewAgent(DefaultServerAddress, DefaultPollInterval, DefaultReportInterval)
	a.MetricsCollect()

	v := a.gaugesMetrics["RandomValue"]
	assert.GreaterOrEqual(t, v, 0.0, "RandomValue should be >= 0")
	assert.Less(t, v, 1.0, "RandomValue should be < 1")
}

func TestMetricsSend_JSONFormat(t *testing.T) {
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
				assert.Equal(t, "/update", r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)

				var m models.Metrics
				require.NoError(t, json.Unmarshal(body, &m))
				received = append(received, m)

				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			a := NewAgent(srv.URL, DefaultPollInterval, DefaultReportInterval)
			a.mu.Lock()
			maps.Copy(a.gaugesMetrics, tt.gauges)
			maps.Copy(a.countersMetrics, tt.counters)
			a.mu.Unlock()

			a.MetricsSend()

			assert.Contains(t, received, tt.want)
		})
	}
}