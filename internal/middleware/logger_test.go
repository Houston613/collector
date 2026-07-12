package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestLogger(t *testing.T) {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	log := zap.New(observedZapCore)

	e := echo.New()
	e.Use(RequestLogger(log))

	e.GET("/test", func(c echo.Context) error {
		return c.String(http.StatusAccepted, "logged")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Equal(t, 1, observedLogs.Len())

	logEntry := observedLogs.All()[0]
	assert.Equal(t, "request", logEntry.Message)

	fields := logEntry.ContextMap()
	assert.Equal(t, "/test", fields["uri"])
	assert.Equal(t, "GET", fields["method"])
	assert.Equal(t, int64(http.StatusAccepted), fields["status"])
}
