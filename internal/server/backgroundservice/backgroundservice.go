// Package backgroundservice implements the gRPC BackgroundServiceService by bridging to CDP.
package backgroundservice

import (
	"context"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/backgroundservice"
)

type Server struct {
	pb.UnimplementedBackgroundServiceServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// serviceNameToString converts the proto enum to the CDP string value.
func serviceNameToString(s pb.ServiceName) string {
	switch s {
	case pb.ServiceName_BACKGROUND_FETCH:
		return "backgroundFetch"
	case pb.ServiceName_BACKGROUND_SYNC:
		return "backgroundSync"
	case pb.ServiceName_PUSH_MESSAGING:
		return "pushMessaging"
	case pb.ServiceName_NOTIFICATIONS:
		return "notifications"
	case pb.ServiceName_PAYMENT_HANDLER:
		return "paymentHandler"
	case pb.ServiceName_PERIODIC_BACKGROUND_SYNC:
		return "periodicBackgroundSync"
	default:
		return ""
	}
}

func (s *Server) StartObserving(ctx context.Context, req *pb.StartObservingRequest) (*pb.StartObservingResponse, error) {
	params := map[string]interface{}{
		"service": serviceNameToString(req.Service),
	}
	if _, err := s.client.Send(ctx, "BackgroundService.startObserving", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.startObserving: %w", err)
	}
	return &pb.StartObservingResponse{}, nil
}

func (s *Server) StopObserving(ctx context.Context, req *pb.StopObservingRequest) (*pb.StopObservingResponse, error) {
	params := map[string]interface{}{
		"service": serviceNameToString(req.Service),
	}
	if _, err := s.client.Send(ctx, "BackgroundService.stopObserving", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.stopObserving: %w", err)
	}
	return &pb.StopObservingResponse{}, nil
}

func (s *Server) SetRecording(ctx context.Context, req *pb.SetRecordingRequest) (*pb.SetRecordingResponse, error) {
	params := map[string]interface{}{
		"shouldRecord": req.ShouldRecord,
		"service":      serviceNameToString(req.Service),
	}
	if _, err := s.client.Send(ctx, "BackgroundService.setRecording", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.setRecording: %w", err)
	}
	return &pb.SetRecordingResponse{}, nil
}

func (s *Server) ClearEvents(ctx context.Context, req *pb.ClearEventsRequest) (*pb.ClearEventsResponse, error) {
	params := map[string]interface{}{
		"service": serviceNameToString(req.Service),
	}
	if _, err := s.client.Send(ctx, "BackgroundService.clearEvents", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.clearEvents: %w", err)
	}
	return &pb.ClearEventsResponse{}, nil
}
