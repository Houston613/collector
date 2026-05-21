package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// compressWriter — ленивый gzip-writer.
// Решение о сжатии принимается в WriteHeader (когда Content-Type уже известен,
// но заголовки ещё не отправлены клиенту).
type compressWriter struct {
	http.ResponseWriter
	gzWriter *gzip.Writer
	compress bool // нужно ли сжимать — выясняется в WriteHeader
	written  bool // были ли байты записаны через gzip
}

// compressibleContentType возвращает true для типов контента, которые нужно сжимать.
func compressibleContentType(ct string) bool {
	return strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/html")
}

// WriteHeader перехватывает отправку статус-кода.
// Echo устанавливает Content-Type ДО вызова WriteHeader, поэтому здесь
// мы можем принять решение о сжатии и добавить Content-Encoding в заголовки
// до их фиксации.
func (w *compressWriter) WriteHeader(code int) {
	if compressibleContentType(w.Header().Get("Content-Type")) {
		w.compress = true
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *compressWriter) Write(b []byte) (int, error) {
	if !w.compress {
		return w.ResponseWriter.Write(b)
	}
	w.written = true
	return w.gzWriter.Write(b)
}

// Close завершает gzip-поток только если через него что-то записывалось.
func (w *compressWriter) Close() {
	if w.written {
		w.gzWriter.Close()
	}
}

// GzipMiddleware распаковывает входящие gzip-запросы (Content-Encoding: gzip)
// и сжимает ответы для клиентов, поддерживающих gzip (Accept-Encoding: gzip).
// Сжатие применяется только для application/json и text/html.
func GzipMiddleware(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if c.Request().Header.Get("Content-Encoding") == "gzip" {
				gr, err := gzip.NewReader(c.Request().Body)
				if err != nil {
					log.Error("failed to create gzip reader for request body", zap.Error(err))
					return c.String(http.StatusBadRequest, "failed to decompress gzip body")
				}
				c.Request().Body = io.NopCloser(gr)
				c.Request().Header.Del("Content-Encoding")
			}
			if !strings.Contains(c.Request().Header.Get("Accept-Encoding"), "gzip") {
				return next(c)
			}

			res := c.Response()
			res.Header().Set("Vary", "Accept-Encoding")

			origWriter := res.Writer

			gz, err := gzip.NewWriterLevel(origWriter, gzip.BestCompression)
			if err != nil {
				log.Error("failed to create gzip writer", zap.Error(err))
				return err
			}

			grw := &compressWriter{
				ResponseWriter: origWriter,
				gzWriter:       gz,
			}
			res.Writer = grw

			defer func() {
				// Закрываем gzip-поток (если данные записывались).
				grw.Close()
				// Восстанавливаем оригинальный writer, чтобы Echo's HTTPErrorHandler,
				// который запускается ПОСЛЕ возврата из middleware, мог писать
				// ответ напрямую — без незакрытого gzip-потока.
				// Если ответ уже зафиксирован (c.Response().Committed == true),
				// HTTPErrorHandler сам пропустит запись, так что восстановление
				// writer безопасно в обоих случаях.
				res.Writer = origWriter
			}()

			return next(c)
		}
	}
}
