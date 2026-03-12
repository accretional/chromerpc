// Package deviceaccess implements the gRPC DeviceAccessService by bridging to CDP.
package deviceaccess

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/deviceaccess"
)

type Server struct {
	pb.UnimplementedDeviceAccessServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "DeviceAccess.enable", nil); err != nil {
		return nil, fmt.Errorf("DeviceAccess.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "DeviceAccess.disable", nil); err != nil {
		return nil, fmt.Errorf("DeviceAccess.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SelectPrompt(ctx context.Context, req *pb.SelectPromptRequest) (*pb.SelectPromptResponse, error) {
	params := map[string]interface{}{
		"id":       req.Id,
		"deviceId": req.DeviceId,
	}
	if _, err := s.client.Send(ctx, "DeviceAccess.selectPrompt", params); err != nil {
		return nil, fmt.Errorf("DeviceAccess.selectPrompt: %w", err)
	}
	return &pb.SelectPromptResponse{}, nil
}

func (s *Server) CancelPrompt(ctx context.Context, req *pb.CancelPromptRequest) (*pb.CancelPromptResponse, error) {
	params := map[string]interface{}{
		"id": req.Id,
	}
	if _, err := s.client.Send(ctx, "DeviceAccess.cancelPrompt", params); err != nil {
		return nil, fmt.Errorf("DeviceAccess.cancelPrompt: %w", err)
	}
	return &pb.CancelPromptResponse{}, nil
}
