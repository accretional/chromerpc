// Package webaudio implements the gRPC WebAudioService by bridging to CDP.
package webaudio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/webaudio"
)

type Server struct {
	pb.UnimplementedWebAudioServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "WebAudio.enable", nil); err != nil {
		return nil, fmt.Errorf("WebAudio.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "WebAudio.disable", nil); err != nil {
		return nil, fmt.Errorf("WebAudio.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetRealtimeData(ctx context.Context, req *pb.GetRealtimeDataRequest) (*pb.GetRealtimeDataResponse, error) {
	params := map[string]interface{}{
		"contextId": req.ContextId,
	}
	result, err := s.client.Send(ctx, "WebAudio.getRealtimeData", params)
	if err != nil {
		return nil, fmt.Errorf("WebAudio.getRealtimeData: %w", err)
	}
	var resp struct {
		RealtimeData struct {
			CurrentTime              float64 `json:"currentTime"`
			RenderCapacity           float64 `json:"renderCapacity"`
			CallbackIntervalMean     float64 `json:"callbackIntervalMean"`
			CallbackIntervalVariance float64 `json:"callbackIntervalVariance"`
		} `json:"realtimeData"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("WebAudio.getRealtimeData: unmarshal: %w", err)
	}
	return &pb.GetRealtimeDataResponse{
		RealtimeData: &pb.ContextRealtimeData{
			CurrentTime:              resp.RealtimeData.CurrentTime,
			RenderCapacity:           resp.RealtimeData.RenderCapacity,
			CallbackIntervalMean:     resp.RealtimeData.CallbackIntervalMean,
			CallbackIntervalVariance: resp.RealtimeData.CallbackIntervalVariance,
		},
	}, nil
}
