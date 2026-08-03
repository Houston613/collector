package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestGzipMiddleware(t *testing.T) {
	log := zap.NewNop()
	e := echo.New()

	e.Use(GzipMiddleware(log))

	e.POST("/test", func(c echo.Context) error {
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			return err
		}
		c.Response().Header().Set("Content-Type", "application/json")
		return c.JSON(http.StatusOK, map[string]string{"message": string(body)})
	})

	e.GET("/text", func(c echo.Context) error {
		c.Response().Header().Set("Content-Type", "text/plain")
		return c.String(http.StatusOK, "plain text")
	})

	t.Run("no compression", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("test_body")))
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEqual(t, "gzip", rec.Header().Get("Content-Encoding"))
		assert.JSONEq(t, `{"message":"test_body"}`, rec.Body.String())
	})

	t.Run("compress json response", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("test_body")))
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

		gr, err := gzip.NewReader(rec.Body)
		require.NoError(t, err)
		defer gr.Close()
		decompressed, err := io.ReadAll(gr)
		require.NoError(t, err)
		assert.JSONEq(t, `{"message":"test_body"}`, string(decompressed))
	})

	t.Run("do not compress uncompressible content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/text", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEqual(t, "gzip", rec.Header().Get("Content-Encoding"))
		assert.Equal(t, "plain text", rec.Body.String())
	})

	t.Run("decompress gzip request body", func(t *testing.T) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, err := gw.Write([]byte("gzipped_body"))
		require.NoError(t, err)
		gw.Close()

		req := httptest.NewRequest(http.MethodPost, "/test", &buf)
		req.Header.Set("Content-Encoding", "gzip")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.JSONEq(t, `{"message":"gzipped_body"}`, rec.Body.String())
	})

	t.Run("invalid gzip request body returns bad request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader([]byte("not_gzip")))
		req.Header.Set("Content-Encoding", "gzip")
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}
