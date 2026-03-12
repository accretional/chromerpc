// Package tethering implements the gRPC TetheringService by bridging to CDP.
// Tethering domain commands operate at browser level (no session ID).
package tethering

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/tethering"
)

type Server struct {
	pb.UnimplementedTetheringServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// sendBrowser sends at browser level (no session ID).
func (s *Server) sendBrowser(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.SendWithSession(ctx, method, params, "")
}

func (s *Server) Bind(ctx context.Context, req *pb.BindRequest) (*pb.BindResponse, error) {
	params := map[string]interface{}{
		"port": req.Port,
	}
	if _, err := s.sendBrowser(ctx, "Tethering.bind", params); err != nil {
		return nil, fmt.Errorf("Tethering.bind: %w", err)
	}
	return &pb.BindResponse{}, nil
}

func (s *Server) Unbind(ctx context.Context, req *pb.UnbindRequest) (*pb.UnbindResponse, error) {
	params := map[string]interface{}{
		"port": req.Port,
	}
	if _, err := s.sendBrowser(ctx, "Tethering.unbind", params); err != nil {
		return nil, fmt.Errorf("Tethering.unbind: %w", err)
	}
	return &pb.UnbindResponse{}, nil
}
