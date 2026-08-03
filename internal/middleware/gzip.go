package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

var gzipWriterPool = sync.Pool{
	New: func() interface{} {
		gz, err := gzip.NewWriterLevel(io.Discard, gzip.BestCompression)
		if err != nil {
			panic(err)
		}
		return gz
	},
}

// compressWriter is a lazy gzip-writer.
// The compression decision is made in WriteHeader (when the Content-Type is already known,
// but the headers have not yet been sent to the client).
type compressWriter struct {
	http.ResponseWriter
	gzWriter *gzip.Writer
	compress bool // whether to compress (determined in WriteHeader)
	written  bool // whether any bytes were written via gzip
}

// compressibleContentType returns true for content types that should be compressed.
func compressibleContentType(ct string) bool {
	return strings.HasPrefix(ct, "application/json") || strings.HasPrefix(ct, "text/html")
}

// WriteHeader intercepts the sending of the status code.
// Echo sets the Content-Type BEFORE calling WriteHeader, so here
// we can decide whether to compress and add Content-Encoding to headers
// before they are committed.
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

// Close finishes the gzip stream only if something was written through it.
func (w *compressWriter) Close() {
	if w.written {
		w.gzWriter.Close()
	}
}

// GzipMiddleware decompresses incoming gzip requests (Content-Encoding: gzip)
// and compresses responses for clients supporting gzip (Accept-Encoding: gzip).
// Compression is only applied to application/json and text/html.
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

			gz := gzipWriterPool.Get().(*gzip.Writer)
			gz.Reset(origWriter)

			grw := &compressWriter{
				ResponseWriter: origWriter,
				gzWriter:       gz,
			}
			res.Writer = grw

			defer func() {
				// Close the gzip stream (if data was written).
				grw.Close()
				// Restore the original writer so that Echo's HTTPErrorHandler,
				// which runs AFTER returning from middleware, can write
				// the response directly without an unclosed gzip stream.
				// If the response is already committed (c.Response().Committed == true),
				// HTTPErrorHandler will skip writing, so restoring the
				// writer is safe in both cases.
				res.Writer = origWriter
				// Return gzip.Writer to the pool
				gzipWriterPool.Put(gz)
			}()

			return next(c)
		}
	}
}
