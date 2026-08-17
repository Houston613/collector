package middleware

import (
	"collector/pkg/netutil"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

// TrustedSubnetMiddleware returns Echo middleware that checks if incoming requests
// have an X-Real-IP header containing an IP address within the configured trusted subnet.
// If trustedSubnet is empty, no restriction is enforced.
func TrustedSubnetMiddleware(trustedSubnet string, log *zap.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		if trustedSubnet == "" {
			return next
		}

		subnet, err := netutil.ParseSubnet(trustedSubnet)
		if err != nil && log != nil {
			log.Error("failed to parse trusted_subnet CIDR", zap.String("subnet", trustedSubnet), zap.Error(err))
		}

		return func(c echo.Context) error {
			if subnet == nil {
				return next(c)
			}

			ipStr := strings.TrimSpace(c.Request().Header.Get("X-Real-IP"))
			if ipStr == "" {
				return c.String(http.StatusForbidden, "Forbidden: missing X-Real-IP header")
			}

			ip := net.ParseIP(ipStr)
			if ip == nil || !subnet.Contains(ip) {
				return c.String(http.StatusForbidden, fmt.Sprintf("Forbidden: IP %s is not in trusted subnet", ipStr))
			}

			return next(c)
		}
	}
}
