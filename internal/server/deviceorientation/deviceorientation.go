// Package deviceorientation implements the gRPC DeviceOrientationService by bridging to CDP.
package deviceorientation

import (
	"context"
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

func (s *Server) ClearDeviceOrientationOverride(ctx context.Context, req *pb.ClearDeviceOrientationOverrideRequest) (*pb.ClearDeviceOrientationOverrideResponse, error) {
	if _, err := s.client.Send(ctx, "DeviceOrientation.clearDeviceOrientationOverride", nil); err != nil {
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
	if _, err := s.client.Send(ctx, "DeviceOrientation.setDeviceOrientationOverride", params); err != nil {
		return nil, fmt.Errorf("DeviceOrientation.setDeviceOrientationOverride: %w", err)
	}
	return &pb.SetDeviceOrientationOverrideResponse{}, nil
}
