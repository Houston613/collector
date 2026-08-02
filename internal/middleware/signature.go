package middleware

import (
	"bytes"
	"collector/pkg/signature"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var bufferPool = sync.Pool{
	New: func() any {
		return new(bytes.Buffer)
	},
}

type signatureWriter struct {
	http.ResponseWriter
	buffer     *bytes.Buffer
	hmac       hash.Hash
	writer     io.Writer
	statusCode int
	committed  bool
}

func newSignatureWriter(w http.ResponseWriter, key string) *signatureWriter {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	mac := hmac.New(sha256.New, []byte(key))
	return &signatureWriter{
		ResponseWriter: w,
		buffer:         buf,
		hmac:           mac,
		writer:         io.MultiWriter(buf, mac),
	}
}

func (w *signatureWriter) WriteHeader(code int) {
	if w.committed {
		return
	}
	w.statusCode = code
}

func (w *signatureWriter) Write(b []byte) (int, error) {
	return w.writer.Write(b)
}

func (w *signatureWriter) Flush() error {
	if w.committed {
		return nil
	}
	defer bufferPool.Put(w.buffer)

	hashStr := hex.EncodeToString(w.hmac.Sum(nil))
	w.Header().Set("HashSHA256", hashStr)

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

			// Skip GET requests as they don't have a body
			if c.Request().Method == http.MethodGet {
				return next(c)
			}

			reqHash := c.Request().Header.Get("HashSHA256")
			if reqHash == "" {
				reqHash = c.Request().Header.Get("Hash")
			}

			// If the header is missing, skip validation
			if reqHash == "" {
				return handleWithSignature(c, next, key, log)
			}

			body, err := io.ReadAll(c.Request().Body)
			if err != nil {
				log.Error("failed to read request body for signature verification", zap.Error(err))
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
			c.Request().Body = io.NopCloser(bytes.NewBuffer(body))

			if !signature.Verify(body, key, reqHash) {
				calculatedHash := signature.Sign(body, key)
				log.Warn("signature mismatch",
					zap.String("received", reqHash),
					zap.String("calculated", calculatedHash),
					zap.Int("body_len", len(body)),
					zap.String("method", c.Request().Method),
					zap.String("uri", c.Request().RequestURI),
				)
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "signature mismatch"})
			}

			return handleWithSignature(c, next, key, log)
		}
	}
}

func handleWithSignature(c echo.Context, next echo.HandlerFunc, key string, log *zap.Logger) error {
	originalWriter := c.Response().Writer
	sigWriter := newSignatureWriter(originalWriter, key)
	c.Response().Writer = sigWriter

	err := next(c)
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

