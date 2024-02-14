package senders

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding/gzip"

	pb "github.com/sergeizaitcev/metrics/api/proto/metrics"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/pkg/interceptors/md"
)

// SenderGRPC определяет агент для отправки метрик на gRPC-сервер.
type SenderGRPC struct {
	client pb.MetricsClient
	opts   commonOptions
}

// GRPC возвращает новый экземпляр Sender для gRPC-сервера.
func GRPC(conn *grpc.ClientConn, opts ...Option) *SenderGRPC {
	sender := &SenderGRPC{
		client: pb.NewMetricsClient(conn),
	}
	for _, opt := range opts {
		opt(&sender.opts)
	}
	return sender
}

func (s *SenderGRPC) Send(ctx context.Context, values []metrics.Metric) error {
	ctx = s.setMetadata(ctx, values)

	req := &pb.UpdateRequest{
		Metrics: make([]*pb.Metric, 0, len(values)),
	}
	for _, value := range values {
		req.Metrics = append(req.Metrics, value.Proto())
	}

	_, err := s.client.Update(ctx, req, grpc.UseCompressor(gzip.Name))
	if err != nil {
		return err
	}

	return nil
}

func (s *SenderGRPC) setMetadata(ctx context.Context, values []metrics.Metric) context.Context {
	ctx = md.SetRealIP(ctx, s.opts.ip)
	if s.opts.sha256key == "" {
		ctx = md.SetHash256(ctx, metrics.Sign(s.opts.sha256key, values))
	}
	return ctx
}
