package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"collector/pkg/crypto"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// CryptoMiddleware decrypts incoming request bodies using the provided RSA private key.
func CryptoMiddleware(privKey *rsa.PrivateKey, log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if privKey == nil || c.Request().Body == nil {
				return next(c)
			}

			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				log.Error("failed to read encrypted request body", zap.Error(err))
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to read body"})
			}

			if len(body) == 0 {
				c.Request().Body = io.NopCloser(bytes.NewBuffer(body))
				return next(c)
			}

			decrypted, err := crypto.Decrypt(privKey, body)
			if err != nil {
				log.Error("failed to decrypt request body", zap.Error(err))
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "failed to decrypt payload"})
			}

			c.Request().Body = io.NopCloser(bytes.NewBuffer(decrypted))
			return next(c)
		}
	}
}
