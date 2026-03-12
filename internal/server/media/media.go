// Package media implements the gRPC MediaService by bridging to CDP.
package media

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/media"
)

type Server struct {
	pb.UnimplementedMediaServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Media.enable", nil); err != nil {
		return nil, fmt.Errorf("Media.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Media.disable", nil); err != nil {
		return nil, fmt.Errorf("Media.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.MediaService_SubscribeEventsServer) error {
	ch := make(chan *pb.MediaEvent, 64)
	defer close(ch)

	unsub1 := s.client.On("Media.playerPropertiesChanged", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			PlayerID   string            `json:"playerId"`
			Properties []cdpPlayerProp   `json:"properties"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		props := make([]*pb.PlayerProperty, len(raw.Properties))
		for i, p := range raw.Properties {
			props[i] = &pb.PlayerProperty{Name: p.Name, Value: p.Value}
		}
		ch <- &pb.MediaEvent{
			Event: &pb.MediaEvent_PlayerPropertiesChanged{
				PlayerPropertiesChanged: &pb.PlayerPropertiesChangedEvent{
					PlayerId:   raw.PlayerID,
					Properties: props,
				},
			},
		}
	})
	defer unsub1()

	unsub2 := s.client.On("Media.playerEventsAdded", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			PlayerID string           `json:"playerId"`
			Events   []cdpPlayerEvent `json:"events"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		events := make([]*pb.PlayerEvent, len(raw.Events))
		for i, e := range raw.Events {
			events[i] = &pb.PlayerEvent{Timestamp: e.Timestamp, Value: e.Value}
		}
		ch <- &pb.MediaEvent{
			Event: &pb.MediaEvent_PlayerEventsAdded{
				PlayerEventsAdded: &pb.PlayerEventsAddedEvent{
					PlayerId: raw.PlayerID,
					Events:   events,
				},
			},
		}
	})
	defer unsub2()

	unsub3 := s.client.On("Media.playerMessagesLogged", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			PlayerID string             `json:"playerId"`
			Messages []cdpPlayerMessage `json:"messages"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		msgs := make([]*pb.PlayerMessage, len(raw.Messages))
		for i, m := range raw.Messages {
			msgs[i] = &pb.PlayerMessage{Level: m.Level, Message: m.Message}
		}
		ch <- &pb.MediaEvent{
			Event: &pb.MediaEvent_PlayerMessagesLogged{
				PlayerMessagesLogged: &pb.PlayerMessagesLoggedEvent{
					PlayerId: raw.PlayerID,
					Messages: msgs,
				},
			},
		}
	})
	defer unsub3()

	unsub4 := s.client.On("Media.playerErrorsRaised", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			PlayerID string           `json:"playerId"`
			Errors   []cdpPlayerError `json:"errors"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		errs := make([]*pb.PlayerError, len(raw.Errors))
		for i, e := range raw.Errors {
			stackJSON := ""
			if len(e.Stack) > 0 {
				stackJSON = string(e.Stack)
			}
			errs[i] = &pb.PlayerError{
				Type:      e.Type,
				ErrorCode: e.ErrorCode,
				Stack:     stackJSON,
			}
		}
		ch <- &pb.MediaEvent{
			Event: &pb.MediaEvent_PlayerErrorsRaised{
				PlayerErrorsRaised: &pb.PlayerErrorsRaisedEvent{
					PlayerId: raw.PlayerID,
					Errors:   errs,
				},
			},
		}
	})
	defer unsub4()

	unsub5 := s.client.On("Media.playersCreated", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Players []struct {
				PlayerID string `json:"playerId"`
			} `json:"players"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		players := make([]*pb.PlayerInfo, len(raw.Players))
		for i, p := range raw.Players {
			players[i] = &pb.PlayerInfo{PlayerId: p.PlayerID}
		}
		ch <- &pb.MediaEvent{
			Event: &pb.MediaEvent_PlayersCreated{
				PlayersCreated: &pb.PlayersCreatedEvent{
					Players: players,
				},
			},
		}
	})
	defer unsub5()

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

// --- internal helpers ---

type cdpPlayerProp struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type cdpPlayerEvent struct {
	Timestamp float64 `json:"timestamp"`
	Value     string  `json:"value"`
}

type cdpPlayerMessage struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}

type cdpPlayerError struct {
	Type      string          `json:"type"`
	ErrorCode string          `json:"errorCode"`
	Stack     json.RawMessage `json:"stack"`
}
