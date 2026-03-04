package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	gauges   map[string]float64
	counters map[string]int64
}

func newmockRepo() *mockRepo {
	return &mockRepo{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func (f *mockRepo) UpdateGauge(name string, value float64) {
	f.gauges[name] = value
}

func (f *mockRepo) UpdateCounter(name string, value int64) {
	f.counters[name] += value
}

// newTestMux wires UpdateMetrics into the same routing pattern as production.
func newTestMux(repo *mockRepo) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /update/{type}/{name}/{value}", UpdateMetrics(repo))
	return mux
}

func TestUpdateMetrics_GaugeOK(t *testing.T) {
	repo := newmockRepo()
	mux := newTestMux(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/27.54", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, 27.54, repo.gauges["Alloc"])
}

func TestUpdateMetrics_CounterOK(t *testing.T) {
	repo := newmockRepo()
	mux := newTestMux(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/10", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, int64(10), repo.counters["PollCount"])
}

func TestUpdateMetrics_CounterAccumulates(t *testing.T) {
	repo := newmockRepo()
	mux := newTestMux(repo)

	for _, val := range []string{"10", "20", "5"} {
		req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/"+val, nil)
		req.Header.Set("Content-Type", "text/plain")
		mux.ServeHTTP(httptest.NewRecorder(), req)
	}

	assert.Equal(t, int64(35), repo.counters["PollCount"])
}

func TestUpdateMetrics_InvalidGaugeValue(t *testing.T) {
	repo := newmockRepo()
	mux := newTestMux(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/not-a-number", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateMetrics_InvalidCounterValue(t *testing.T) {
	repo := newmockRepo()
	mux := newTestMux(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/3.14", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateMetrics_InvalidType(t *testing.T) {
	repo := newmockRepo()
	mux := newTestMux(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/histogram/myMetric/1", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestUpdateMetrics_WrongContentType(t *testing.T) {
	repo := newmockRepo()
	mux := newTestMux(repo)

	req := httptest.NewRequest(http.MethodPost, "/update/gauge/Alloc/1", nil)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, rr.Code)
}

func TestUpdateMetrics_WrongMethod(t *testing.T) {
	repo := newmockRepo()
	handler := UpdateMetrics(repo)

	req := httptest.NewRequest(http.MethodGet, "/update/gauge/Alloc/1", nil)
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rr.Code)
}
