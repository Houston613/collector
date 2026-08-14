package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTrustedSubnetMiddleware(t *testing.T) {
	e := echo.New()
	log := zap.NewNop()

	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	t.Run("empty trusted subnet allows all requests", func(t *testing.T) {
		mw := TrustedSubnetMiddleware("", log)(handler)

		req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := mw(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("ip within trusted subnet is allowed", func(t *testing.T) {
		mw := TrustedSubnetMiddleware("192.168.1.0/24", log)(handler)

		req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
		req.Header.Set("X-Real-IP", "192.168.1.50")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := mw(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("ip outside trusted subnet is forbidden", func(t *testing.T) {
		mw := TrustedSubnetMiddleware("192.168.1.0/24", log)(handler)

		req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
		req.Header.Set("X-Real-IP", "10.0.0.1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := mw(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("missing X-Real-IP header is forbidden when trusted subnet configured", func(t *testing.T) {
		mw := TrustedSubnetMiddleware("192.168.1.0/24", log)(handler)

		req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := mw(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("invalid IP in X-Real-IP is forbidden", func(t *testing.T) {
		mw := TrustedSubnetMiddleware("192.168.1.0/24", log)(handler)

		req := httptest.NewRequest(http.MethodPost, "/updates/", nil)
		req.Header.Set("X-Real-IP", "not-an-ip")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := mw(c)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}
