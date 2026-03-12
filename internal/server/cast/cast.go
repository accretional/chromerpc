// Package cast implements the gRPC CastService by bridging to CDP.
package cast

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/cast"
)

type Server struct {
	pb.UnimplementedCastServiceServer
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


func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	var params map[string]interface{}
	if req.PresentationUrl != nil {
		params = map[string]interface{}{
			"presentationUrl": *req.PresentationUrl,
		}
	}
	if _, err := s.send(ctx, req.SessionId, "Cast.enable", params); err != nil {
		return nil, fmt.Errorf("Cast.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Cast.disable", nil); err != nil {
		return nil, fmt.Errorf("Cast.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SetSinkToUse(ctx context.Context, req *pb.SetSinkToUseRequest) (*pb.SetSinkToUseResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.send(ctx, req.SessionId, "Cast.setSinkToUse", params); err != nil {
		return nil, fmt.Errorf("Cast.setSinkToUse: %w", err)
	}
	return &pb.SetSinkToUseResponse{}, nil
}

func (s *Server) StartDesktopMirroring(ctx context.Context, req *pb.StartDesktopMirroringRequest) (*pb.StartDesktopMirroringResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.send(ctx, req.SessionId, "Cast.startDesktopMirroring", params); err != nil {
		return nil, fmt.Errorf("Cast.startDesktopMirroring: %w", err)
	}
	return &pb.StartDesktopMirroringResponse{}, nil
}

func (s *Server) StartTabMirroring(ctx context.Context, req *pb.StartTabMirroringRequest) (*pb.StartTabMirroringResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.send(ctx, req.SessionId, "Cast.startTabMirroring", params); err != nil {
		return nil, fmt.Errorf("Cast.startTabMirroring: %w", err)
	}
	return &pb.StartTabMirroringResponse{}, nil
}

func (s *Server) StopCasting(ctx context.Context, req *pb.StopCastingRequest) (*pb.StopCastingResponse, error) {
	params := map[string]interface{}{
		"sinkName": req.SinkName,
	}
	if _, err := s.send(ctx, req.SessionId, "Cast.stopCasting", params); err != nil {
		return nil, fmt.Errorf("Cast.stopCasting: %w", err)
	}
	return &pb.StopCastingResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeCastEventsRequest, stream pb.CastService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.CastEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all Cast.* events.
	unregister := s.client.On("Cast.", func(method string, params json.RawMessage, sessionID string) {
		// Only forward events for the requested session (or all if empty).
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertCastEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	castEvents := []string{
		"Cast.sinksUpdated", "Cast.issueUpdated",
	}
	unregisters := make([]func(), 0, len(castEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range castEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertCastEvent(method, params)
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

func convertCastEvent(method string, params json.RawMessage) *pb.CastEvent {
	switch method {
	case "Cast.sinksUpdated":
		var d struct {
			Sinks []struct {
				Name    string `json:"name"`
				ID      string `json:"id"`
				Session string `json:"session"`
			} `json:"sinks"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		sinks := make([]*pb.Sink, len(d.Sinks))
		for i, s := range d.Sinks {
			sinks[i] = &pb.Sink{
				Name:    s.Name,
				Id:      s.ID,
				Session: s.Session,
			}
		}
		return &pb.CastEvent{Event: &pb.CastEvent_SinksUpdated{
			SinksUpdated: &pb.SinksUpdatedEvent{Sinks: sinks},
		}}

	case "Cast.issueUpdated":
		var d struct {
			IssueMessage string `json:"issueMessage"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.CastEvent{Event: &pb.CastEvent_IssueUpdated{
			IssueUpdated: &pb.IssueUpdatedEvent{IssueMessage: d.IssueMessage},
		}}
	}
	return nil
}
