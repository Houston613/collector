package grpcserver

import (
	"collector/pkg/netutil"
	"context"
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TrustedSubnetInterceptor(trustedSubnet string, log *zap.Logger) grpc.UnaryServerInterceptor {
	var subnet *net.IPNet
	if trustedSubnet != "" {
		var err error
		subnet, err = netutil.ParseSubnet(trustedSubnet)
		if err != nil && log != nil {
			log.Error("failed to parse trusted_subnet for gRPC interceptor", zap.String("subnet", trustedSubnet), zap.Error(err))
		}
	}

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if trustedSubnet == "" || subnet == nil {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "missing metadata")
		}

		ips := md.Get("x-real-ip")
		if len(ips) == 0 || strings.TrimSpace(ips[0]) == "" {
			return nil, status.Error(codes.PermissionDenied, "missing x-real-ip metadata")
		}

		ip := net.ParseIP(strings.TrimSpace(ips[0]))
		if ip == nil || !subnet.Contains(ip) {
			return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("IP %s is not in trusted subnet", ips[0]))
		}

		return handler(ctx, req)
	}
}
