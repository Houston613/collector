package agent

import (
	"bytes"
	models "collector/internal/model"
	pb "collector/internal/proto/gen"
	"collector/pkg/crypto"
	"collector/pkg/retry"
	"collector/pkg/signature"
	"compress/gzip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type MetricsSender interface {
	Send(ctx context.Context, metrics []models.Metrics) error
	Close() error
}

// HTTPSender sends metrics via HTTP/REST JSON payloads.
type HTTPSender struct {
	addr      string
	key       string
	cryptoKey *rsa.PublicKey
	hostIP    string
	client    *http.Client
	log       *zap.Logger
}

// NewHTTPSender creates a new HTTPSender.
func NewHTTPSender(addr, key string, cryptoKey *rsa.PublicKey, hostIP string, log *zap.Logger) *HTTPSender {
	return &HTTPSender{
		addr:      addr,
		key:       key,
		cryptoKey: cryptoKey,
		hostIP:    hostIP,
		client:    &http.Client{},
		log:       log,
	}
}

// Send serializes and ships a batch of metrics using HTTP POST.
func (s *HTTPSender) Send(ctx context.Context, metrics []models.Metrics) error {
	body, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("gzip writer: %w", err)
	}
	if _, err = gz.Write(body); err != nil {
		return fmt.Errorf("gzip write: %w", err)
	}
	if err = gz.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	compressedData := buf.Bytes()
	payload := compressedData

	if s.cryptoKey != nil {
		enc, err := crypto.Encrypt(s.cryptoKey, compressedData)
		if err != nil {
			return fmt.Errorf("encrypt payload: %w", err)
		}
		payload = enc
	}

	return retry.Do(ctx, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.addr+"/updates/", bytes.NewReader(payload))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Accept-Encoding", "gzip")
		if s.hostIP != "" {
			req.Header.Set("X-Real-IP", s.hostIP)
		}

		if s.key != "" {
			hash := signature.Sign(body, s.key)
			req.Header.Set("HashSHA256", hash)
		}

		resp, err := s.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}

		return nil
	}, func(err error) bool {
		var netErr net.Error
		return errors.As(err, &netErr)
	})
}

// Close implements MetricsSender.
func (s *HTTPSender) Close() error {
	return nil
}

// GRPCSender sends metrics via gRPC Protobuf payloads.
type GRPCSender struct {
	grpcAddr   string
	hostIP     string
	grpcConn   *grpc.ClientConn
	grpcClient pb.MetricsClient
	log        *zap.Logger
}

// NewGRPCSender creates a new GRPCSender.
func NewGRPCSender(grpcAddr, hostIP string, log *zap.Logger) *GRPCSender {
	return &GRPCSender{
		grpcAddr: grpcAddr,
		hostIP:   hostIP,
		log:      log,
	}
}

// Send serializes and ships a batch of metrics using gRPC.
func (s *GRPCSender) Send(ctx context.Context, metrics []models.Metrics) error {
	if s.grpcClient == nil {
		conn, err := grpc.NewClient(s.grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("failed to connect to gRPC server %s: %w", s.grpcAddr, err)
		}
		s.grpcConn = conn
		s.grpcClient = pb.NewMetricsClient(conn)
	}

	pbMetrics := make([]*pb.Metric, 0, len(metrics))
	for _, m := range metrics {
		pbM := &pb.Metric{Id: m.ID}
		switch m.MType {
		case models.Gauge:
			pbM.Type = pb.Metric_GAUGE
			if m.Value != nil {
				pbM.Value = *m.Value
			}
		case models.Counter:
			pbM.Type = pb.Metric_COUNTER
			if m.Delta != nil {
				pbM.Delta = *m.Delta
			}
		}
		pbMetrics = append(pbMetrics, pbM)
	}

	return retry.Do(ctx, func() error {
		reqCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs("x-real-ip", s.hostIP))
		_, err := s.grpcClient.UpdateMetrics(reqCtx, &pb.UpdateMetricsRequest{Metrics: pbMetrics})
		return err
	}, func(err error) bool {
		st, ok := status.FromError(err)
		if ok {
			code := st.Code()
			return code == codes.Unavailable || code == codes.ResourceExhausted
		}
		return false
	})
}

// Close closes the gRPC connection if active.
func (s *GRPCSender) Close() error {
	if s.grpcConn != nil {
		return s.grpcConn.Close()
	}
	return nil
}
