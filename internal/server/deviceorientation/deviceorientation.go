// Package deviceorientation implements the gRPC DeviceOrientationService by bridging to CDP.
package deviceorientation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/deviceorientation"
)

type Server struct {
	pb.UnimplementedDeviceOrientationServiceServer
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


func (s *Server) ClearDeviceOrientationOverride(ctx context.Context, req *pb.ClearDeviceOrientationOverrideRequest) (*pb.ClearDeviceOrientationOverrideResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "DeviceOrientation.clearDeviceOrientationOverride", nil); err != nil {
		return nil, fmt.Errorf("DeviceOrientation.clearDeviceOrientationOverride: %w", err)
	}
	return &pb.ClearDeviceOrientationOverrideResponse{}, nil
}

func (s *Server) SetDeviceOrientationOverride(ctx context.Context, req *pb.SetDeviceOrientationOverrideRequest) (*pb.SetDeviceOrientationOverrideResponse, error) {
	params := map[string]interface{}{
		"alpha": req.Alpha,
		"beta":  req.Beta,
		"gamma": req.Gamma,
	}
	if _, err := s.send(ctx, req.SessionId, "DeviceOrientation.setDeviceOrientationOverride", params); err != nil {
		return nil, fmt.Errorf("DeviceOrientation.setDeviceOrientationOverride: %w", err)
	}
	return &pb.SetDeviceOrientationOverrideResponse{}, nil
}
