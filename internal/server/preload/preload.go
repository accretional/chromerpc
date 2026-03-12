// Package preload implements the gRPC PreloadService by bridging to CDP.
package preload

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/preload"
)

type Server struct {
	pb.UnimplementedPreloadServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Preload.enable", nil); err != nil {
		return nil, fmt.Errorf("Preload.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Preload.disable", nil); err != nil {
		return nil, fmt.Errorf("Preload.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}
