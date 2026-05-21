package middleware

import (
	"bytes"
	"collector/pkg/signature"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type signatureWriter struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	key        string
	statusCode int
	committed  bool
}

func (w *signatureWriter) WriteHeader(code int) {
	if w.committed {
		return
	}
	w.statusCode = code
}

func (w *signatureWriter) Write(b []byte) (int, error) {
	return w.buffer.Write(b)
}

func (w *signatureWriter) Flush() error {
	if w.committed {
		return nil
	}

	hash := signature.Sign(w.buffer.Bytes(), w.key)
	w.Header().Set("HashSHA256", hash)

	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}

	w.ResponseWriter.WriteHeader(w.statusCode)
	_, err := w.ResponseWriter.Write(w.buffer.Bytes())
	w.committed = true
	return err
}

func SignatureMiddleware(key string, log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if key == "" {
				return next(c)
			}

			// попробуем пропускат GET запросы, так как у них нет тела
			if c.Request().Method == http.MethodGet {
				return next(c)
			}

			reqHash := c.Request().Header.Get("HashSHA256")
			if reqHash == "" {
				reqHash = c.Request().Header.Get("Hash")
			}

			// Если заголовок "none", пропускаем проверку и подпись
			if reqHash == "none" {
				return next(c)
			}

			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				log.Error("failed to read request body for signature verification", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(body))

			if !signature.Verify(body, key, reqHash) {
				calculatedHash := signature.Sign(body, key)
				//разошьем логи на всякий
				log.Warn("signature mismatch",
					zap.String("received", reqHash),
					zap.String("calculated", calculatedHash),
					zap.Int("body_len", len(body)),
					zap.String("method", c.Request().Method),
					zap.String("uri", c.Request().RequestURI),
				)
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "signature mismatch"})
			}

			// Перехват ответа для подписи
			originalWriter := c.Response().Writer
			sigWriter := &signatureWriter{
				ResponseWriter: originalWriter,
				buffer:         new(bytes.Buffer),
				key:            key,
			}
			c.Response().Writer = sigWriter

			err = next(c)
			if err != nil {
				c.Error(err)
				err = nil
			}

			if flushErr := sigWriter.Flush(); flushErr != nil {
				log.Error("failed to flush signature writer", zap.Error(flushErr))
				if err == nil {
					err = flushErr
				}
			}
			c.Response().Writer = originalWriter

			return err
		}
	}
}
