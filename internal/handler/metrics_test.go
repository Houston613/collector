package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "collector/internal/model"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	gauges   map[string]float64
	counters map[string]int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (m *mockRepo) UpdateGauge(ctx context.Context, name string, value float64) error {
	m.gauges[name] = value
	return nil
}
func (m *mockRepo) UpdateCounter(ctx context.Context, name string, value int64) error {
	m.counters[name] += value
	return nil
}
func (m *mockRepo) UpdateMetrics(ctx context.Context, metrics []models.Metrics) error {
	for _, metric := range metrics {
		switch metric.MType {
		case models.Gauge:
			if metric.Value != nil {
				m.gauges[metric.ID] = *metric.Value
			}
		case models.Counter:
			if metric.Delta != nil {
				m.counters[metric.ID] += *metric.Delta
			}
		}
	}
	return nil
}
func (m *mockRepo) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	v, ok := m.gauges[name]
	return v, ok, nil
}
func (m *mockRepo) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	v, ok := m.counters[name]
	return v, ok, nil
}
func (m *mockRepo) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	return m.gauges, nil
}
func (m *mockRepo) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	return m.counters, nil
}
func newEcho(repo *mockRepo) *echo.Echo {
	e := echo.New()
	//в тестаз просто пока репозиторий
	metricsHandler := NewMetricsHandler(repo, "")
	metricsHandler.RegisterRoutes(e)
	return e
}

func TestUpdateMetrics_GaugeOK(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/27.54", nil)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	v, ok := repo.gauges["Alloc"]
	require.True(t, ok, "gauge Alloc should exist")
	assert.Equal(t, 27.54, v)
}

func TestUpdateMetrics_CounterOK(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/10", nil)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	v, ok := repo.counters["PollCount"]
	require.True(t, ok, "counter PollCount should exist")
	assert.Equal(t, int64(10), v)
}

func TestUpdateMetrics_CounterAccumulates(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	for _, val := range []string{"10", "20", "5"} {
		req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/"+val, nil)
		req.Header.Set("Content-Type", "text/plain")
		e.ServeHTTP(httptest.NewRecorder(), req)
	}

	assert.Equal(t, int64(35), repo.counters["PollCount"])
}

func TestUpdateMetrics_InvalidGaugeValue(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/not-a-number", nil)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetrics_InvalidCounterValue(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/27.54", nil)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetrics_InvalidType(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/histogram/myMetric/1", nil)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetrics_WrongContentType(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rec.Code)
}

func TestUpdateMetrics_WrongMethod(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodGet, "/update/gauge/Alloc/1", nil)
	req.Header.Set("Content-Type", "text/plain")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestGetMetric_GaugeFound(t *testing.T) {
	repo := newMockRepo()
	repo.gauges["Alloc"] = 54.27
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodGet, "/value/gauge/Alloc", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "54.27", rec.Body.String())
}

func TestGetMetric_CounterFound(t *testing.T) {
	repo := newMockRepo()
	repo.counters["PollCount"] = 10
	e := newEcho(repo)

	req := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "10", rec.Body.String())
}

func jsonBody(t *testing.T, v any) *strings.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return strings.NewReader(string(b))
}

func jsonRequest(method, target string, body *strings.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestUpdateMetricJSON_GaugeOK(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	val := 27.54
	body := jsonBody(t, models.Metrics{ID: "Alloc", MType: models.Gauge, Value: &val})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/update", body))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	stored, ok := repo.gauges["Alloc"]
	require.True(t, ok)
	assert.Equal(t, 27.54, stored)
}

func TestUpdateMetricJSON_CounterOK(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	delta := int64(10)
	body := jsonBody(t, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/update", body))

	assert.Equal(t, http.StatusOK, rec.Code)
	stored, ok := repo.counters["PollCount"]
	require.True(t, ok)
	assert.Equal(t, int64(10), stored)
}

func TestUpdateMetricJSON_CounterAccumulates(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	for _, d := range []int64{10, 20, 5} {
		delta := d
		body := jsonBody(t, models.Metrics{ID: "PollCount", MType: models.Counter, Delta: &delta})
		e.ServeHTTP(httptest.NewRecorder(), jsonRequest(http.MethodPost, "/update", body))
	}

	assert.Equal(t, int64(35), repo.counters["PollCount"])
}

func TestUpdateMetricJSON_ResponseContainsMetric(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	val := 123.456
	body := jsonBody(t, models.Metrics{ID: "HeapAlloc", MType: models.Gauge, Value: &val})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/update", body))

	require.Equal(t, http.StatusOK, rec.Code)
	var got models.Metrics
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "HeapAlloc", got.ID)
	assert.Equal(t, models.Gauge, got.MType)
	require.NotNil(t, got.Value)
	assert.Equal(t, 123.456, *got.Value)
}

func TestUpdateMetricJSON_MissingGaugeValue(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	body := jsonBody(t, models.Metrics{ID: "Alloc", MType: models.Gauge})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/update", body))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetricJSON_MissingCounterDelta(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	body := jsonBody(t, models.Metrics{ID: "PollCount", MType: models.Counter})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/update", body))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetricJSON_InvalidType(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	body := jsonBody(t, models.Metrics{ID: "x", MType: "histogram"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/update", body))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdateMetricJSON_InvalidJSON(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := jsonRequest(http.MethodPost, "/update", strings.NewReader(`not-json`))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestGetMetricJSON_GaugeFound(t *testing.T) {
	repo := newMockRepo()
	repo.gauges["LastGC"] = 1744184459
	e := newEcho(repo)

	body := jsonBody(t, models.Metrics{ID: "LastGC", MType: models.Gauge})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/value", body))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got models.Metrics
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "LastGC", got.ID)
	assert.Equal(t, models.Gauge, got.MType)
	require.NotNil(t, got.Value)
	assert.Equal(t, float64(1744184459), *got.Value)
}

func TestGetMetricJSON_CounterFound(t *testing.T) {
	repo := newMockRepo()
	repo.counters["PollCount"] = 42
	e := newEcho(repo)

	body := jsonBody(t, models.Metrics{ID: "PollCount", MType: models.Counter})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/value", body))

	require.Equal(t, http.StatusOK, rec.Code)
	var got models.Metrics
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	assert.Equal(t, "PollCount", got.ID)
	require.NotNil(t, got.Delta)
	assert.Equal(t, int64(42), *got.Delta)
}

func TestGetMetricJSON_NotFound(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	body := jsonBody(t, models.Metrics{ID: "Unknown", MType: models.Gauge})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/value", body))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetMetricJSON_InvalidType(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	body := jsonBody(t, models.Metrics{ID: "x", MType: "histogram"})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/value", body))

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetMetricJSON_InvalidJSON(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	req := jsonRequest(http.MethodPost, "/value", strings.NewReader(`not-json`))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUpdatesMetricsJSON_OK(t *testing.T) {
	repo := newMockRepo()
	e := newEcho(repo)

	gaugeVal := 123.45
	counterDelta := int64(10)
	metrics := []models.Metrics{
		{ID: "Gauge1", MType: models.Gauge, Value: &gaugeVal},
		{ID: "Counter1", MType: models.Counter, Delta: &counterDelta},
	}

	body := jsonBody(t, metrics)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, jsonRequest(http.MethodPost, "/updates/", body))

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, 123.45, repo.gauges["Gauge1"])
	assert.Equal(t, int64(10), repo.counters["Counter1"])
}
