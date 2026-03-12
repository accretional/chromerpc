// Package console implements the gRPC ConsoleService by bridging to CDP.
package console

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/console"
)

type Server struct {
	pb.UnimplementedConsoleServiceServer
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
	if _, err := s.send(ctx, req.SessionId, "Console.enable", nil); err != nil {
		return nil, fmt.Errorf("Console.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Console.disable", nil); err != nil {
		return nil, fmt.Errorf("Console.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) ClearMessages(ctx context.Context, req *pb.ClearMessagesRequest) (*pb.ClearMessagesResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Console.clearMessages", nil); err != nil {
		return nil, fmt.Errorf("Console.clearMessages: %w", err)
	}
	return &pb.ClearMessagesResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.ConsoleService_SubscribeEventsServer) error {
	ch := make(chan *pb.ConsoleEvent, 64)
	defer close(ch)

	unsubscribe := s.client.On("Console.messageAdded", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Message cdpConsoleMessage `json:"message"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.ConsoleEvent{
			Event: &pb.ConsoleEvent_MessageAdded{
				MessageAdded: &pb.MessageAddedEvent{
					Message: raw.Message.toProto(),
				},
			},
		}
	})
	defer unsubscribe()

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

type cdpConsoleMessage struct {
	Source string `json:"source"`
	Level  string `json:"level"`
	Text   string `json:"text"`
	URL    string `json:"url"`
	Line   int32  `json:"line"`
	Column int32  `json:"column"`
}

func (m *cdpConsoleMessage) toProto() *pb.ConsoleMessage {
	return &pb.ConsoleMessage{
		Source: m.Source,
		Level:  m.Level,
		Text:   m.Text,
		Url:    m.URL,
		Line:   m.Line,
		Column: m.Column,
	}
}
