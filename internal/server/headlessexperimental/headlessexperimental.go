// Package headlessexperimental implements the gRPC HeadlessExperimentalService by bridging to CDP.
package headlessexperimental

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/headlessexperimental"
)

type Server struct {
	pb.UnimplementedHeadlessExperimentalServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "HeadlessExperimental.enable", nil); err != nil {
		return nil, fmt.Errorf("HeadlessExperimental.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "HeadlessExperimental.disable", nil); err != nil {
		return nil, fmt.Errorf("HeadlessExperimental.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) BeginFrame(ctx context.Context, req *pb.BeginFrameRequest) (*pb.BeginFrameResponse, error) {
	params := map[string]interface{}{}
	if req.FrameTimeTicks != nil {
		params["frameTimeTicks"] = *req.FrameTimeTicks
	}
	if req.Interval != nil {
		params["interval"] = *req.Interval
	}
	if req.NoDisplayUpdates {
		params["noDisplayUpdates"] = req.NoDisplayUpdates
	}
	if req.Screenshot != nil {
		ss := map[string]interface{}{}
		if req.Screenshot.Format != "" {
			ss["format"] = req.Screenshot.Format
		}
		if req.Screenshot.Quality != 0 {
			ss["quality"] = req.Screenshot.Quality
		}
		if req.Screenshot.OptimizeForSpeed {
			ss["optimizeForSpeed"] = req.Screenshot.OptimizeForSpeed
		}
		params["screenshot"] = ss
	}

	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.client.Send(ctx, "HeadlessExperimental.beginFrame", params)
	} else {
		result, err = s.client.Send(ctx, "HeadlessExperimental.beginFrame", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("HeadlessExperimental.beginFrame: %w", err)
	}

	var resp struct {
		HasDamage      bool   `json:"hasDamage"`
		ScreenshotData string `json:"screenshotData"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("HeadlessExperimental.beginFrame: unmarshal: %w", err)
	}

	out := &pb.BeginFrameResponse{
		HasDamage: resp.HasDamage,
	}
	if resp.ScreenshotData != "" {
		imageData, err := base64.StdEncoding.DecodeString(resp.ScreenshotData)
		if err != nil {
			return nil, fmt.Errorf("HeadlessExperimental.beginFrame: decode base64: %w", err)
		}
		out.ScreenshotData = imageData
	}

	return out, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeHeadlessExperimentalEventsRequest, stream pb.HeadlessExperimentalService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.HeadlessExperimentalEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all HeadlessExperimental.* events.
	unregister := s.client.On("HeadlessExperimental.", func(method string, params json.RawMessage, sessionID string) {
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertHeadlessExperimentalEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	heEvents := []string{
		"HeadlessExperimental.needsBeginFramesChanged",
	}
	unregisters := make([]func(), 0, len(heEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range heEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertHeadlessExperimentalEvent(method, params)
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

func convertHeadlessExperimentalEvent(method string, params json.RawMessage) *pb.HeadlessExperimentalEvent {
	switch method {
	case "HeadlessExperimental.needsBeginFramesChanged":
		var d struct {
			NeedsBeginFrames bool `json:"needsBeginFrames"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.HeadlessExperimentalEvent{Event: &pb.HeadlessExperimentalEvent_NeedsBeginFramesChanged{
			NeedsBeginFramesChanged: &pb.NeedsBeginFramesChangedEvent{
				NeedsBeginFrames: d.NeedsBeginFrames,
			},
		}}
	}
	return nil
}
