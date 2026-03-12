// Package log implements the gRPC LogService by bridging to CDP.
package log

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/log"
)

type Server struct {
	pb.UnimplementedLogServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Log.enable", nil); err != nil {
		return nil, fmt.Errorf("Log.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Log.disable", nil); err != nil {
		return nil, fmt.Errorf("Log.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) Clear(ctx context.Context, req *pb.ClearRequest) (*pb.ClearResponse, error) {
	if _, err := s.client.Send(ctx, "Log.clear", nil); err != nil {
		return nil, fmt.Errorf("Log.clear: %w", err)
	}
	return &pb.ClearResponse{}, nil
}

func (s *Server) StartViolationsReport(ctx context.Context, req *pb.StartViolationsReportRequest) (*pb.StartViolationsReportResponse, error) {
	config := make([]map[string]interface{}, len(req.Config))
	for i, v := range req.Config {
		config[i] = map[string]interface{}{
			"name":      v.Name,
			"threshold": v.Threshold,
		}
	}
	params := map[string]interface{}{"config": config}
	if _, err := s.client.Send(ctx, "Log.startViolationsReport", params); err != nil {
		return nil, fmt.Errorf("Log.startViolationsReport: %w", err)
	}
	return &pb.StartViolationsReportResponse{}, nil
}

func (s *Server) StopViolationsReport(ctx context.Context, req *pb.StopViolationsReportRequest) (*pb.StopViolationsReportResponse, error) {
	if _, err := s.client.Send(ctx, "Log.stopViolationsReport", nil); err != nil {
		return nil, fmt.Errorf("Log.stopViolationsReport: %w", err)
	}
	return &pb.StopViolationsReportResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.LogService_SubscribeEventsServer) error {
	ch := make(chan *pb.LogEvent, 64)
	defer close(ch)

	unsubscribe := s.client.On("Log.entryAdded", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Entry cdpLogEntry `json:"entry"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.LogEvent{
			Event: &pb.LogEvent_EntryAdded{
				EntryAdded: &pb.EntryAddedEvent{
					Entry: raw.Entry.toProto(),
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

type cdpLogEntry struct {
	Source           string          `json:"source"`
	Level            string          `json:"level"`
	Text             string          `json:"text"`
	Timestamp        float64         `json:"timestamp"`
	URL              string          `json:"url"`
	LineNumber       int32           `json:"lineNumber"`
	StackTrace       json.RawMessage `json:"stackTrace"`
	NetworkRequestID string          `json:"networkRequestId"`
	WorkerID         string          `json:"workerId"`
	Args             json.RawMessage `json:"args"`
	Category         string          `json:"category"`
}

func (e *cdpLogEntry) toProto() *pb.LogEntry {
	entry := &pb.LogEntry{
		Source:           e.Source,
		Level:            e.Level,
		Text:             e.Text,
		Timestamp:        e.Timestamp,
		Url:              e.URL,
		LineNumber:       e.LineNumber,
		NetworkRequestId: e.NetworkRequestID,
		WorkerId:         e.WorkerID,
		Category:         e.Category,
	}
	if len(e.StackTrace) > 0 {
		entry.StackTrace = string(e.StackTrace)
	}
	if len(e.Args) > 0 {
		entry.Args = string(e.Args)
	}
	return entry
}
