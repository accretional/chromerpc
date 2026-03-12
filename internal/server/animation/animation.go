// Package animation implements the gRPC AnimationService by bridging to CDP.
package animation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/animation"
)

type Server struct {
	pb.UnimplementedAnimationServiceServer
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
	if _, err := s.send(ctx, req.SessionId, "Animation.enable", nil); err != nil {
		return nil, fmt.Errorf("Animation.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Animation.disable", nil); err != nil {
		return nil, fmt.Errorf("Animation.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetPlaybackRate(ctx context.Context, req *pb.GetPlaybackRateRequest) (*pb.GetPlaybackRateResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Animation.getPlaybackRate", nil)
	if err != nil {
		return nil, fmt.Errorf("Animation.getPlaybackRate: %w", err)
	}
	var resp struct {
		PlaybackRate float64 `json:"playbackRate"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Animation.getPlaybackRate: unmarshal: %w", err)
	}
	return &pb.GetPlaybackRateResponse{PlaybackRate: resp.PlaybackRate}, nil
}

func (s *Server) SetPlaybackRate(ctx context.Context, req *pb.SetPlaybackRateRequest) (*pb.SetPlaybackRateResponse, error) {
	params := map[string]interface{}{
		"playbackRate": req.PlaybackRate,
	}
	if _, err := s.send(ctx, req.SessionId, "Animation.setPlaybackRate", params); err != nil {
		return nil, fmt.Errorf("Animation.setPlaybackRate: %w", err)
	}
	return &pb.SetPlaybackRateResponse{}, nil
}

func (s *Server) GetCurrentTime(ctx context.Context, req *pb.GetCurrentTimeRequest) (*pb.GetCurrentTimeResponse, error) {
	params := map[string]interface{}{
		"id": req.Id,
	}
	result, err := s.send(ctx, req.SessionId, "Animation.getCurrentTime", params)
	if err != nil {
		return nil, fmt.Errorf("Animation.getCurrentTime: %w", err)
	}
	var resp struct {
		CurrentTime float64 `json:"currentTime"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Animation.getCurrentTime: unmarshal: %w", err)
	}
	return &pb.GetCurrentTimeResponse{CurrentTime: resp.CurrentTime}, nil
}

func (s *Server) SetPaused(ctx context.Context, req *pb.SetPausedRequest) (*pb.SetPausedResponse, error) {
	params := map[string]interface{}{
		"animations": req.Animations,
		"paused":     req.Paused,
	}
	if _, err := s.send(ctx, req.SessionId, "Animation.setPaused", params); err != nil {
		return nil, fmt.Errorf("Animation.setPaused: %w", err)
	}
	return &pb.SetPausedResponse{}, nil
}

func (s *Server) SetTiming(ctx context.Context, req *pb.SetTimingRequest) (*pb.SetTimingResponse, error) {
	params := map[string]interface{}{
		"animationId": req.AnimationId,
		"duration":    req.Duration,
		"delay":       req.Delay,
	}
	if _, err := s.send(ctx, req.SessionId, "Animation.setTiming", params); err != nil {
		return nil, fmt.Errorf("Animation.setTiming: %w", err)
	}
	return &pb.SetTimingResponse{}, nil
}

func (s *Server) SeekAnimations(ctx context.Context, req *pb.SeekAnimationsRequest) (*pb.SeekAnimationsResponse, error) {
	params := map[string]interface{}{
		"animations":  req.Animations,
		"currentTime": req.CurrentTime,
	}
	if _, err := s.send(ctx, req.SessionId, "Animation.seekAnimations", params); err != nil {
		return nil, fmt.Errorf("Animation.seekAnimations: %w", err)
	}
	return &pb.SeekAnimationsResponse{}, nil
}

func (s *Server) ReleaseAnimations(ctx context.Context, req *pb.ReleaseAnimationsRequest) (*pb.ReleaseAnimationsResponse, error) {
	params := map[string]interface{}{
		"animations": req.Animations,
	}
	if _, err := s.send(ctx, req.SessionId, "Animation.releaseAnimations", params); err != nil {
		return nil, fmt.Errorf("Animation.releaseAnimations: %w", err)
	}
	return &pb.ReleaseAnimationsResponse{}, nil
}

func (s *Server) ResolveAnimation(ctx context.Context, req *pb.ResolveAnimationRequest) (*pb.ResolveAnimationResponse, error) {
	params := map[string]interface{}{
		"animationId": req.AnimationId,
	}
	result, err := s.send(ctx, req.SessionId, "Animation.resolveAnimation", params)
	if err != nil {
		return nil, fmt.Errorf("Animation.resolveAnimation: %w", err)
	}
	var resp struct {
		RemoteObject json.RawMessage `json:"remoteObject"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Animation.resolveAnimation: unmarshal: %w", err)
	}
	return &pb.ResolveAnimationResponse{RemoteObject: string(resp.RemoteObject)}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.AnimationService_SubscribeEventsServer) error {
	ch := make(chan *pb.AnimationEvent, 64)
	defer close(ch)

	unsubCreated := s.client.On("Animation.animationCreated", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.AnimationEvent{
			Event: &pb.AnimationEvent_AnimationCreated{
				AnimationCreated: &pb.AnimationCreatedEvent{Id: raw.ID},
			},
		}
	})
	defer unsubCreated()

	unsubStarted := s.client.On("Animation.animationStarted", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Animation struct {
				ID           string  `json:"id"`
				Name         string  `json:"name"`
				PausedState  bool    `json:"pausedState"`
				PlayState    string  `json:"playState"`
				PlaybackRate float64 `json:"playbackRate"`
				StartTime    float64 `json:"startTime"`
				CurrentTime  float64 `json:"currentTime"`
				Type         string  `json:"type"`
				Source       *struct {
					Delay          float64 `json:"delay"`
					EndDelay       float64 `json:"endDelay"`
					IterationStart float64 `json:"iterationStart"`
					Iterations     float64 `json:"iterations"`
					Duration       float64 `json:"duration"`
					Direction      string  `json:"direction"`
					Fill           string  `json:"fill"`
					BackendNodeId  int32   `json:"backendNodeId"`
					Easing         string  `json:"easing"`
				} `json:"source"`
				CssID string `json:"cssId"`
			} `json:"animation"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		anim := &pb.AnimationObj{
			Id:           raw.Animation.ID,
			Name:         raw.Animation.Name,
			PausedState:  raw.Animation.PausedState,
			PlayState:    raw.Animation.PlayState,
			PlaybackRate: raw.Animation.PlaybackRate,
			StartTime:    raw.Animation.StartTime,
			CurrentTime:  raw.Animation.CurrentTime,
			Type:         raw.Animation.Type,
			CssId:        raw.Animation.CssID,
		}
		if raw.Animation.Source != nil {
			anim.Source = &pb.AnimationEffect{
				Delay:          raw.Animation.Source.Delay,
				EndDelay:       raw.Animation.Source.EndDelay,
				IterationStart: raw.Animation.Source.IterationStart,
				Iterations:     raw.Animation.Source.Iterations,
				Duration:       raw.Animation.Source.Duration,
				Direction:      raw.Animation.Source.Direction,
				Fill:           raw.Animation.Source.Fill,
				BackendNodeId:  raw.Animation.Source.BackendNodeId,
				Easing:         raw.Animation.Source.Easing,
			}
		}
		ch <- &pb.AnimationEvent{
			Event: &pb.AnimationEvent_AnimationStarted{
				AnimationStarted: &pb.AnimationStartedEvent{Animation: anim},
			},
		}
	})
	defer unsubStarted()

	unsubCanceled := s.client.On("Animation.animationCanceled", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.AnimationEvent{
			Event: &pb.AnimationEvent_AnimationCanceled{
				AnimationCanceled: &pb.AnimationCanceledEvent{Id: raw.ID},
			},
		}
	})
	defer unsubCanceled()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case evt := <-ch:
			if err := stream.Send(evt); err != nil {
				return err
			}
		}
	}
}
