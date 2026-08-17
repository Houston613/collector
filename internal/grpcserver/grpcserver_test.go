package grpcserver

import (
	pb "collector/internal/proto/gen"
	"collector/internal/repository"
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

func TestGRPCMetricsServer(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	log := zap.NewNop()
	repo := repository.NewStructMem()

	s := grpc.NewServer(
		grpc.UnaryInterceptor(TrustedSubnetInterceptor("192.168.1.0/24", log)),
	)
	metricsServer := NewMetricsServer(repo, log)
	pb.RegisterMetricsServer(s, metricsServer)

	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	conn, err := grpc.NewClient("passthrough://bufconn",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	defer conn.Close()

	client := pb.NewMetricsClient(conn)

	t.Run("allowed IP updates metrics", func(t *testing.T) {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.100"))
		req := &pb.UpdateMetricsRequest{
			Metrics: []*pb.Metric{
				{
					Id:    "gauge1",
					Type:  pb.Metric_GAUGE,
					Value: 42.5,
				},
				{
					Id:    "counter1",
					Type:  pb.Metric_COUNTER,
					Delta: 10,
				},
			},
		}

		resp, err := client.UpdateMetrics(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		v, ok, err := repo.GetGauge(context.Background(), "gauge1")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, 42.5, v)

		c, ok, err := repo.GetCounter(context.Background(), "counter1")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, int64(10), c)
	})

	t.Run("forbidden IP returns PermissionDenied", func(t *testing.T) {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-real-ip", "10.0.0.1"))
		req := &pb.UpdateMetricsRequest{
			Metrics: []*pb.Metric{
				{
					Id:    "gauge2",
					Type:  pb.Metric_GAUGE,
					Value: 1.0,
				},
			},
		}

		_, err := client.UpdateMetrics(ctx, req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("missing metadata returns PermissionDenied", func(t *testing.T) {
		req := &pb.UpdateMetricsRequest{
			Metrics: []*pb.Metric{
				{
					Id:    "gauge3",
					Type:  pb.Metric_GAUGE,
					Value: 1.0,
				},
			},
		}

		_, err := client.UpdateMetrics(context.Background(), req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.PermissionDenied, st.Code())
	})

	t.Run("invalid metric type returns InvalidArgument", func(t *testing.T) {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.100"))
		req := &pb.UpdateMetricsRequest{
			Metrics: []*pb.Metric{
				{
					Id:   "badMetric",
					Type: pb.Metric_MType(999),
				},
			},
		}

		_, err := client.UpdateMetrics(ctx, req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("empty metric ID returns InvalidArgument", func(t *testing.T) {
		ctx := metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-real-ip", "192.168.1.100"))
		req := &pb.UpdateMetricsRequest{
			Metrics: []*pb.Metric{
				{
					Id:    "",
					Type:  pb.Metric_GAUGE,
					Value: 5.0,
				},
			},
		}

		_, err := client.UpdateMetrics(ctx, req)
		require.Error(t, err)
		st, ok := status.FromError(err)
		assert.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}
