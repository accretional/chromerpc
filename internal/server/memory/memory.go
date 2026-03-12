// Package memory implements the gRPC MemoryService by bridging to CDP.
package memory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/memory"
)

type Server struct {
	pb.UnimplementedMemoryServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) GetDOMCounters(ctx context.Context, req *pb.GetDOMCountersRequest) (*pb.GetDOMCountersResponse, error) {
	result, err := s.client.Send(ctx, "Memory.getDOMCounters", nil)
	if err != nil {
		return nil, fmt.Errorf("Memory.getDOMCounters: %w", err)
	}
	var resp struct {
		Documents        int32 `json:"documents"`
		Nodes            int32 `json:"nodes"`
		JsEventListeners int32 `json:"jsEventListeners"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Memory.getDOMCounters: unmarshal: %w", err)
	}
	return &pb.GetDOMCountersResponse{
		Documents:        resp.Documents,
		Nodes:            resp.Nodes,
		JsEventListeners: resp.JsEventListeners,
	}, nil
}

func (s *Server) PrepareForLeakDetection(ctx context.Context, req *pb.PrepareForLeakDetectionRequest) (*pb.PrepareForLeakDetectionResponse, error) {
	if _, err := s.client.Send(ctx, "Memory.prepareForLeakDetection", nil); err != nil {
		return nil, fmt.Errorf("Memory.prepareForLeakDetection: %w", err)
	}
	return &pb.PrepareForLeakDetectionResponse{}, nil
}

func (s *Server) ForciblyPurgeJavaScriptMemory(ctx context.Context, req *pb.ForciblyPurgeJavaScriptMemoryRequest) (*pb.ForciblyPurgeJavaScriptMemoryResponse, error) {
	if _, err := s.client.Send(ctx, "Memory.forciblyPurgeJavaScriptMemory", nil); err != nil {
		return nil, fmt.Errorf("Memory.forciblyPurgeJavaScriptMemory: %w", err)
	}
	return &pb.ForciblyPurgeJavaScriptMemoryResponse{}, nil
}

func (s *Server) SetPressureNotificationsSuppressed(ctx context.Context, req *pb.SetPressureNotificationsSuppressedRequest) (*pb.SetPressureNotificationsSuppressedResponse, error) {
	params := map[string]interface{}{
		"suppressed": req.Suppressed,
	}
	if _, err := s.client.Send(ctx, "Memory.setPressureNotificationsSuppressed", params); err != nil {
		return nil, fmt.Errorf("Memory.setPressureNotificationsSuppressed: %w", err)
	}
	return &pb.SetPressureNotificationsSuppressedResponse{}, nil
}

func (s *Server) SimulatePressureNotification(ctx context.Context, req *pb.SimulatePressureNotificationRequest) (*pb.SimulatePressureNotificationResponse, error) {
	params := map[string]interface{}{
		"level": req.Level,
	}
	if _, err := s.client.Send(ctx, "Memory.simulatePressureNotification", params); err != nil {
		return nil, fmt.Errorf("Memory.simulatePressureNotification: %w", err)
	}
	return &pb.SimulatePressureNotificationResponse{}, nil
}

func (s *Server) StartSampling(ctx context.Context, req *pb.StartSamplingRequest) (*pb.StartSamplingResponse, error) {
	params := map[string]interface{}{}
	if req.SamplingInterval != nil {
		params["samplingInterval"] = *req.SamplingInterval
	}
	if req.SuppressRandomness != nil {
		params["suppressRandomness"] = *req.SuppressRandomness
	}
	if _, err := s.client.Send(ctx, "Memory.startSampling", params); err != nil {
		return nil, fmt.Errorf("Memory.startSampling: %w", err)
	}
	return &pb.StartSamplingResponse{}, nil
}

func (s *Server) StopSampling(ctx context.Context, req *pb.StopSamplingRequest) (*pb.StopSamplingResponse, error) {
	if _, err := s.client.Send(ctx, "Memory.stopSampling", nil); err != nil {
		return nil, fmt.Errorf("Memory.stopSampling: %w", err)
	}
	return &pb.StopSamplingResponse{}, nil
}

func unmarshalSamplingProfile(result json.RawMessage) (*pb.SamplingProfile, error) {
	var resp struct {
		Profile struct {
			Samples []struct {
				Size  float64  `json:"size"`
				Total float64  `json:"total"`
				Stack []string `json:"stack"`
			} `json:"samples"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, err
	}
	profile := &pb.SamplingProfile{}
	for _, s := range resp.Profile.Samples {
		profile.Samples = append(profile.Samples, &pb.SamplingProfileNode{
			Size:  s.Size,
			Total: s.Total,
			Stack: s.Stack,
		})
	}
	return profile, nil
}

func (s *Server) GetAllTimeSamplingProfile(ctx context.Context, req *pb.GetAllTimeSamplingProfileRequest) (*pb.GetAllTimeSamplingProfileResponse, error) {
	result, err := s.client.Send(ctx, "Memory.getAllTimeSamplingProfile", nil)
	if err != nil {
		return nil, fmt.Errorf("Memory.getAllTimeSamplingProfile: %w", err)
	}
	profile, err := unmarshalSamplingProfile(result)
	if err != nil {
		return nil, fmt.Errorf("Memory.getAllTimeSamplingProfile: unmarshal: %w", err)
	}
	return &pb.GetAllTimeSamplingProfileResponse{Profile: profile}, nil
}

func (s *Server) GetSamplingProfile(ctx context.Context, req *pb.GetSamplingProfileRequest) (*pb.GetSamplingProfileResponse, error) {
	result, err := s.client.Send(ctx, "Memory.getSamplingProfile", nil)
	if err != nil {
		return nil, fmt.Errorf("Memory.getSamplingProfile: %w", err)
	}
	profile, err := unmarshalSamplingProfile(result)
	if err != nil {
		return nil, fmt.Errorf("Memory.getSamplingProfile: unmarshal: %w", err)
	}
	return &pb.GetSamplingProfileResponse{Profile: profile}, nil
}

func (s *Server) GetBrowserSamplingProfile(ctx context.Context, req *pb.GetBrowserSamplingProfileRequest) (*pb.GetBrowserSamplingProfileResponse, error) {
	result, err := s.client.Send(ctx, "Memory.getBrowserSamplingProfile", nil)
	if err != nil {
		return nil, fmt.Errorf("Memory.getBrowserSamplingProfile: %w", err)
	}
	profile, err := unmarshalSamplingProfile(result)
	if err != nil {
		return nil, fmt.Errorf("Memory.getBrowserSamplingProfile: unmarshal: %w", err)
	}
	return &pb.GetBrowserSamplingProfileResponse{Profile: profile}, nil
}
