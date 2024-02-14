package server

import (
	"context"
	"net"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/sergeizaitcev/metrics/api/proto/metrics"
	"github.com/sergeizaitcev/metrics/internal/configs"
	"github.com/sergeizaitcev/metrics/internal/metrics"
	"github.com/sergeizaitcev/metrics/internal/storage"
	"github.com/sergeizaitcev/metrics/pkg/interceptors/md"
)

type updateServer struct {
	pb.UnimplementedMetricsServer
	subnet    *net.IPNet
	sha256key string
	storage   storage.Storage
}

func newUpdateServer(config *configs.Server, storage storage.Storage) *updateServer {
	return &updateServer{
		subnet:    config.CIDR(),
		sha256key: config.SHA256Key,
		storage:   storage,
	}
}

func (s *updateServer) Update(ctx context.Context, req *pb.UpdateRequest) (*emptypb.Empty, error) {
	values := make([]metrics.Metric, 0, len(req.Metrics))
	for _, metric := range req.Metrics {
		values = append(values, metrics.FromProto(metric))
	}

	if s.sha256key != "" {
		hash := md.GetHash256(ctx)
		currentHash := metrics.Sign(s.sha256key, values)
		if hash != currentHash {
			return nil, status.Error(codes.DataLoss, "metrics is corrupted")
		}
	}

	_, err := s.storage.Save(ctx, values...)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}
