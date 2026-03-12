// Package eventbreakpoints implements the gRPC EventBreakpointsService by bridging to CDP.
package eventbreakpoints

import (
	"context"
	"encoding/json"
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

// send routes a CDP command through the specified session, falling back
// to the client's default session if sessionID is empty.
func (s *Server) send(ctx context.Context, sessionID string, method string, params interface{}) (json.RawMessage, error) {
	if sessionID != "" {
		return s.client.SendWithSession(ctx, method, params, sessionID)
	}
	return s.client.Send(ctx, method, params)
}


func (s *Server) SetInstrumentationBreakpoint(ctx context.Context, req *pb.SetInstrumentationBreakpointRequest) (*pb.SetInstrumentationBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if _, err := s.send(ctx, req.SessionId, "EventBreakpoints.setInstrumentationBreakpoint", params); err != nil {
		return nil, fmt.Errorf("EventBreakpoints.setInstrumentationBreakpoint: %w", err)
	}
	return &pb.SetInstrumentationBreakpointResponse{}, nil
}

func (s *Server) RemoveInstrumentationBreakpoint(ctx context.Context, req *pb.RemoveInstrumentationBreakpointRequest) (*pb.RemoveInstrumentationBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if _, err := s.send(ctx, req.SessionId, "EventBreakpoints.removeInstrumentationBreakpoint", params); err != nil {
		return nil, fmt.Errorf("EventBreakpoints.removeInstrumentationBreakpoint: %w", err)
	}
	return &pb.RemoveInstrumentationBreakpointResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "EventBreakpoints.disable", nil); err != nil {
		return nil, fmt.Errorf("EventBreakpoints.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}
