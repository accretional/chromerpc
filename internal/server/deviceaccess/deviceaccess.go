// Package deviceaccess implements the gRPC DeviceAccessService by bridging to CDP.
package deviceaccess

import (
	"context"
	"encoding/json"
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

func (s *Server) SubscribeEvents(req *pb.SubscribeDeviceAccessEventsRequest, stream pb.DeviceAccessService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.DeviceAccessEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all DeviceAccess.* events.
	unregister := s.client.On("DeviceAccess.", func(method string, params json.RawMessage, sessionID string) {
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertDeviceAccessEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	deviceAccessEvents := []string{
		"DeviceAccess.deviceRequestPrompted",
	}
	unregisters := make([]func(), 0, len(deviceAccessEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range deviceAccessEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertDeviceAccessEvent(method, params)
			if evt != nil {
				select {
				case eventCh <- evt:
				default:
				}
			}
		})
		unregisters = append(unregisters, unreg)
	}
	defer func() {
		for _, unreg := range unregisters {
			unreg()
		}
	}()

	for {
		select {
		case evt := <-eventCh:
			if err := stream.Send(evt); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-s.client.Done():
			return fmt.Errorf("CDP connection closed")
		}
	}
}

func convertDeviceAccessEvent(method string, params json.RawMessage) *pb.DeviceAccessEvent {
	switch method {
	case "DeviceAccess.deviceRequestPrompted":
		var d struct {
			ID      string `json:"id"`
			Devices []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"devices"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		devices := make([]*pb.PromptDevice, len(d.Devices))
		for i, dev := range d.Devices {
			devices[i] = &pb.PromptDevice{
				Id:   dev.ID,
				Name: dev.Name,
			}
		}
		return &pb.DeviceAccessEvent{Event: &pb.DeviceAccessEvent_DeviceRequestPrompted{
			DeviceRequestPrompted: &pb.DeviceRequestPromptedEvent{
				Id:      d.ID,
				Devices: devices,
			},
		}}

	default:
		return nil
	}
}
