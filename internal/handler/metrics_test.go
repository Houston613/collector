package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

func (m *mockRepo) UpdateGauge(name string, value float64) {
	m.gauges[name] = value
}
func (m *mockRepo) UpdateCounter(name string, value int64) {
	m.counters[name] += value
}
func (m *mockRepo) GetGauge(name string) (float64, bool) {
	v, ok := m.gauges[name]
	return v, ok
}
func (m *mockRepo) GetCounter(name string) (int64, bool) {
	v, ok := m.counters[name]
	return v, ok
}
func (m *mockRepo) GetAllGauges() map[string]float64 {
	return m.gauges
}
func (m *mockRepo) GetAllCounters() map[string]int64 {
	return m.counters
}
func newEcho(repo *mockRepo) *echo.Echo {
	e := echo.New()
	metricsHandler := NewMetricsHandler(repo)
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
