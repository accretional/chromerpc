// Package inspector implements the gRPC InspectorService by bridging to CDP.
package inspector

import (
	"context"
	"encoding/json"
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

func (s *Server) SubscribeEvents(req *pb.SubscribeInspectorEventsRequest, stream pb.InspectorService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.InspectorEvent, 128)
	ctx := stream.Context()

	events := []string{
		"Inspector.detached",
		"Inspector.targetCrashed",
		"Inspector.targetReloadedAfterCrash",
	}

	var unregisters []func()
	for _, method := range events {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertInspectorEvent(method, params)
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

func convertInspectorEvent(method string, params json.RawMessage) *pb.InspectorEvent {
	switch method {
	case "Inspector.detached":
		var p struct {
			Reason string `json:"reason"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil
		}
		return &pb.InspectorEvent{
			Event: &pb.InspectorEvent_Detached{
				Detached: &pb.DetachedEvent{Reason: p.Reason},
			},
		}
	case "Inspector.targetCrashed":
		return &pb.InspectorEvent{
			Event: &pb.InspectorEvent_TargetCrashed{
				TargetCrashed: &pb.TargetCrashedEvent{},
			},
		}
	case "Inspector.targetReloadedAfterCrash":
		return &pb.InspectorEvent{
			Event: &pb.InspectorEvent_TargetReloadedAfterCrash{
				TargetReloadedAfterCrash: &pb.TargetReloadedAfterCrashEvent{},
			},
		}
	}
	return nil
}
