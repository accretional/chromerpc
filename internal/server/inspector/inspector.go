// Package inspector implements the gRPC InspectorService by bridging to CDP.
package inspector

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/inspector"
)

type Server struct {
	pb.UnimplementedInspectorServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Inspector.enable", nil); err != nil {
		return nil, fmt.Errorf("Inspector.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Inspector.disable", nil); err != nil {
		return nil, fmt.Errorf("Inspector.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}
