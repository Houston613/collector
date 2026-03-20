package agent

import (
	"maps"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestMetricsSend_URLFormat(t *testing.T) {
	tests := []struct {
		name         string
		gauges       map[string]float64
		counters     map[string]int64
		expectedPath string
	}{
		{
			name:         "gauge",
			gauges:       map[string]float64{"TestGauge": 27.54},
			counters:     map[string]int64{},
			expectedPath: "/update/gauge/TestGauge/27.54",
		},
		{
			name:         "counter",
			gauges:       map[string]float64{},
			counters:     map[string]int64{"TestCounter": 17},
			expectedPath: "/update/counter/TestCounter/17",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var paths []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				paths = append(paths, r.URL.Path)
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			a := NewAgent(srv.URL, DefaultPollInterval, DefaultReportInterval)
			a.mu.Lock()
			//копируем тестовые данные в структуру агента, чтобы при отправке данных были именно эти данные
			maps.Copy(a.gaugesMetrics, tt.gauges)
			maps.Copy(a.countersMetrics, tt.counters)
			a.mu.Unlock()
			//сам агент не собирал метрики, поэтому при отправке данных будут именно эти данные
			a.MetricsSend()
			//ну и ссылки должны быть в правильном формате
			assert.Contains(t, paths, tt.expectedPath)
		})
	}
}