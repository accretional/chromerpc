// Package eventbreakpoints implements the gRPC EventBreakpointsService by bridging to CDP.
package eventbreakpoints

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/eventbreakpoints"
)

type Server struct {
	pb.UnimplementedEventBreakpointsServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) SetInstrumentationBreakpoint(ctx context.Context, req *pb.SetInstrumentationBreakpointRequest) (*pb.SetInstrumentationBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if _, err := s.client.Send(ctx, "EventBreakpoints.setInstrumentationBreakpoint", params); err != nil {
		return nil, fmt.Errorf("EventBreakpoints.setInstrumentationBreakpoint: %w", err)
	}
	return &pb.SetInstrumentationBreakpointResponse{}, nil
}

func (s *Server) RemoveInstrumentationBreakpoint(ctx context.Context, req *pb.RemoveInstrumentationBreakpointRequest) (*pb.RemoveInstrumentationBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if _, err := s.client.Send(ctx, "EventBreakpoints.removeInstrumentationBreakpoint", params); err != nil {
		return nil, fmt.Errorf("EventBreakpoints.removeInstrumentationBreakpoint: %w", err)
	}
	return &pb.RemoveInstrumentationBreakpointResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "EventBreakpoints.disable", nil); err != nil {
		return nil, fmt.Errorf("EventBreakpoints.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}
