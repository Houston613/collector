package middleware

import (
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func RequestLogger(log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			duration := time.Since(start)
			res := c.Response()
			req := c.Request()

			log.Info("request",
				zap.String("uri", req.RequestURI),
				zap.String("method", req.Method),
				zap.Duration("duration", duration),
				zap.Int("status", res.Status),
				zap.Int64("size", res.Size),
			)

			return err
		}
	}
}
