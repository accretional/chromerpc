// Package performance implements the gRPC PerformanceService by bridging to CDP.
package performance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/performance"
)

type Server struct {
	pb.UnimplementedPerformanceServiceServer
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
	params := map[string]interface{}{}
	if req.TimeDomain != "" {
		params["timeDomain"] = req.TimeDomain
	}
	if len(params) > 0 {
		_, err := s.send(ctx, req.SessionId, "Performance.enable", params)
		if err != nil {
			return nil, fmt.Errorf("Performance.enable: %w", err)
		}
	} else {
		if _, err := s.send(ctx, req.SessionId, "Performance.enable", nil); err != nil {
			return nil, fmt.Errorf("Performance.enable: %w", err)
		}
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Performance.disable", nil); err != nil {
		return nil, fmt.Errorf("Performance.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Performance.getMetrics", nil)
	if err != nil {
		return nil, fmt.Errorf("Performance.getMetrics: %w", err)
	}
	var resp struct {
		Metrics []struct {
			Name  string  `json:"name"`
			Value float64 `json:"value"`
		} `json:"metrics"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Performance.getMetrics: unmarshal: %w", err)
	}
	metrics := make([]*pb.Metric, len(resp.Metrics))
	for i, m := range resp.Metrics {
		metrics[i] = &pb.Metric{Name: m.Name, Value: m.Value}
	}
	return &pb.GetMetricsResponse{Metrics: metrics}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.PerformanceService_SubscribeEventsServer) error {
	ch := make(chan *pb.PerformanceEvent, 64)
	defer close(ch)

	unsubscribe := s.client.On("Performance.metrics", func(_ string, params json.RawMessage, _ string) {
		var raw struct {
			Metrics []struct {
				Name  string  `json:"name"`
				Value float64 `json:"value"`
			} `json:"metrics"`
			Title string `json:"title"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		metrics := make([]*pb.Metric, len(raw.Metrics))
		for i, m := range raw.Metrics {
			metrics[i] = &pb.Metric{Name: m.Name, Value: m.Value}
		}
		ch <- &pb.PerformanceEvent{
			Event: &pb.PerformanceEvent_Metrics{
				Metrics: &pb.MetricsEvent{
					Metrics: metrics,
					Title:   raw.Title,
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
