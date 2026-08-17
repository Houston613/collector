package grpcserver

import (
	models "collector/internal/model"
	pb "collector/internal/proto/gen"
	"collector/internal/repository"
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MetricsServer struct {
	pb.UnimplementedMetricsServer
	repo repository.MemRepository
	log  *zap.Logger
}

func NewMetricsServer(repo repository.MemRepository, log *zap.Logger) *MetricsServer {
	return &MetricsServer{
		repo: repo,
		log:  log,
	}
}

func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is nil")
	}

	if len(req.GetMetrics()) == 0 {
		return &pb.UpdateMetricsResponse{}, nil
	}

	metrics := make([]models.Metrics, 0, len(req.GetMetrics()))
	for _, m := range req.GetMetrics() {
		if m == nil {
			return nil, status.Error(codes.InvalidArgument, "metric is nil")
		}
		if m.GetId() == "" {
			return nil, status.Error(codes.InvalidArgument, "metric ID cannot be empty")
		}

		var metric models.Metrics
		metric.ID = m.GetId()

		switch m.GetType() {
		case pb.Metric_GAUGE:
			metric.MType = models.Gauge
			val := m.GetValue()
			metric.Value = &val
		case pb.Metric_COUNTER:
			metric.MType = models.Counter
			delta := m.GetDelta()
			metric.Delta = &delta
		default:
			return nil, status.Errorf(codes.InvalidArgument, "invalid or unspecified metric type for metric %s", m.GetId())
		}
		metrics = append(metrics, metric)
	}

	if err := s.repo.UpdateMetrics(ctx, metrics); err != nil {
		if s.log != nil {
			s.log.Error("failed to update metrics via gRPC", zap.Error(err))
		}
		return nil, status.Errorf(codes.Internal, "failed to update metrics: %v", err)
	}

	return &pb.UpdateMetricsResponse{}, nil
}
