// Package backgroundservice implements the gRPC BackgroundServiceService by bridging to CDP.
package backgroundservice

import (
	"context"
	"encoding/json"
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

// send routes a CDP command through the specified session, falling back
// to the client's default session if sessionID is empty.
func (s *Server) send(ctx context.Context, sessionID string, method string, params interface{}) (json.RawMessage, error) {
	if sessionID != "" {
		return s.client.SendWithSession(ctx, method, params, sessionID)
	}
	return s.client.Send(ctx, method, params)
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
	if _, err := s.send(ctx, req.SessionId, "BackgroundService.startObserving", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.startObserving: %w", err)
	}
	return &pb.StartObservingResponse{}, nil
}

func (s *Server) StopObserving(ctx context.Context, req *pb.StopObservingRequest) (*pb.StopObservingResponse, error) {
	params := map[string]interface{}{
		"service": serviceNameToString(req.Service),
	}
	if _, err := s.send(ctx, req.SessionId, "BackgroundService.stopObserving", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.stopObserving: %w", err)
	}
	return &pb.StopObservingResponse{}, nil
}

func (s *Server) SetRecording(ctx context.Context, req *pb.SetRecordingRequest) (*pb.SetRecordingResponse, error) {
	params := map[string]interface{}{
		"shouldRecord": req.ShouldRecord,
		"service":      serviceNameToString(req.Service),
	}
	if _, err := s.send(ctx, req.SessionId, "BackgroundService.setRecording", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.setRecording: %w", err)
	}
	return &pb.SetRecordingResponse{}, nil
}

func (s *Server) ClearEvents(ctx context.Context, req *pb.ClearEventsRequest) (*pb.ClearEventsResponse, error) {
	params := map[string]interface{}{
		"service": serviceNameToString(req.Service),
	}
	if _, err := s.send(ctx, req.SessionId, "BackgroundService.clearEvents", params); err != nil {
		return nil, fmt.Errorf("BackgroundService.clearEvents: %w", err)
	}
	return &pb.ClearEventsResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeBackgroundServiceEventsRequest, stream pb.BackgroundServiceService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.BackgroundServiceEvent, 128)
	ctx := stream.Context()

	events := []string{
		"BackgroundService.recordingStateChanged",
		"BackgroundService.backgroundServiceEventReceived",
	}

	var unregisters []func()
	for _, method := range events {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertBackgroundServiceEvent(method, params)
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

func convertBackgroundServiceEvent(method string, params json.RawMessage) *pb.BackgroundServiceEvent {
	switch method {
	case "BackgroundService.recordingStateChanged":
		var d struct {
			IsRecording bool   `json:"isRecording"`
			Service     string `json:"service"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.BackgroundServiceEvent{Event: &pb.BackgroundServiceEvent_RecordingStateChanged{
			RecordingStateChanged: &pb.RecordingStateChangedEvent{
				IsRecording: d.IsRecording,
				Service:     d.Service,
			},
		}}

	case "BackgroundService.backgroundServiceEventReceived":
		var d struct {
			BackgroundServiceEvent struct {
				Timestamp                  float64 `json:"timestamp"`
				Origin                     string  `json:"origin"`
				ServiceWorkerRegistrationID string `json:"serviceWorkerRegistrationId"`
				Service                    string  `json:"service"`
				EventName                  string  `json:"eventName"`
				InstanceID                 string  `json:"instanceId"`
				EventMetadata              []struct {
					Key   string `json:"key"`
					Value string `json:"value"`
				} `json:"eventMetadata"`
				StorageKey string `json:"storageKey"`
			} `json:"backgroundServiceEvent"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		evt := d.BackgroundServiceEvent
		metadata := make([]*pb.EventMetadata, len(evt.EventMetadata))
		for i, m := range evt.EventMetadata {
			metadata[i] = &pb.EventMetadata{Key: m.Key, Value: m.Value}
		}
		return &pb.BackgroundServiceEvent{Event: &pb.BackgroundServiceEvent_BackgroundServiceEventReceived{
			BackgroundServiceEventReceived: &pb.BackgroundServiceEventReceivedEvent{
				BackgroundServiceEvent: &pb.BackgroundServiceEventDetail{
					Timestamp:                   evt.Timestamp,
					Origin:                      evt.Origin,
					ServiceWorkerRegistrationId: evt.ServiceWorkerRegistrationID,
					Service:                     evt.Service,
					EventName:                   evt.EventName,
					InstanceId:                  evt.InstanceID,
					EventMetadata:               metadata,
					StorageKey:                  evt.StorageKey,
				},
			},
		}}

	default:
		return nil
	}
}
