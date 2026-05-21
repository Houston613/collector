package middleware

import (
	"bytes"
	"collector/pkg/signature"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSignatureMiddleware(t *testing.T) {
	key := "test_key"
	log := zap.NewNop()
	e := echo.New()

	e.Use(SignatureMiddleware(key, log))

	e.POST("/test", func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	})

	t.Run("valid signature", func(t *testing.T) {
		body := []byte("test body")
		hash := signature.Sign(body, key)

		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		req.Header.Set("HashSHA256", hash)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
		assert.NotEmpty(t, rec.Header().Get("HashSHA256"))
		
		expectedRespHash := signature.Sign([]byte("OK"), key)
		assert.Equal(t, expectedRespHash, rec.Header().Get("HashSHA256"))
	})

	t.Run("invalid signature", func(t *testing.T) {
		body := []byte("test body")
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		req.Header.Set("HashSHA256", "invalid_hash")
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})

	t.Run("missing signature", func(t *testing.T) {
		body := []byte("test body")
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEmpty(t, rec.Header().Get("HashSHA256"))
	})

	t.Run("empty body valid signature", func(t *testing.T) {
		body := []byte("")
		hash := signature.Sign(body, key)

		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
		req.Header.Set("HashSHA256", hash)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
