package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func BenchmarkGzipMiddleware(b *testing.B) {
	log := zap.NewNop()
	e := echo.New()
	e.Use(GzipMiddleware(log))

	e.POST("/test", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "application/json")
		return c.String(http.StatusOK, `{"status":"ok","message":"benchmark json data content description"}`)
	})

	reqBody := []byte(`{"data":"test payload"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(reqBody))
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
	}
}
