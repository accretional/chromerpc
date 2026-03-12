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

// send routes a CDP command through the specified session, falling back
// to the client's default session if sessionID is empty.
func (s *Server) send(ctx context.Context, sessionID string, method string, params interface{}) (json.RawMessage, error) {
	if sessionID != "" {
		return s.client.SendWithSession(ctx, method, params, sessionID)
	}
	return s.client.Send(ctx, method, params)
}


func (s *Server) GetDOMCounters(ctx context.Context, req *pb.GetDOMCountersRequest) (*pb.GetDOMCountersResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Memory.getDOMCounters", nil)
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
	if _, err := s.send(ctx, req.SessionId, "Memory.prepareForLeakDetection", nil); err != nil {
		return nil, fmt.Errorf("Memory.prepareForLeakDetection: %w", err)
	}
	return &pb.PrepareForLeakDetectionResponse{}, nil
}

func (s *Server) ForciblyPurgeJavaScriptMemory(ctx context.Context, req *pb.ForciblyPurgeJavaScriptMemoryRequest) (*pb.ForciblyPurgeJavaScriptMemoryResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Memory.forciblyPurgeJavaScriptMemory", nil); err != nil {
		return nil, fmt.Errorf("Memory.forciblyPurgeJavaScriptMemory: %w", err)
	}
	return &pb.ForciblyPurgeJavaScriptMemoryResponse{}, nil
}

func (s *Server) SetPressureNotificationsSuppressed(ctx context.Context, req *pb.SetPressureNotificationsSuppressedRequest) (*pb.SetPressureNotificationsSuppressedResponse, error) {
	params := map[string]interface{}{
		"suppressed": req.Suppressed,
	}
	if _, err := s.send(ctx, req.SessionId, "Memory.setPressureNotificationsSuppressed", params); err != nil {
		return nil, fmt.Errorf("Memory.setPressureNotificationsSuppressed: %w", err)
	}
	return &pb.SetPressureNotificationsSuppressedResponse{}, nil
}

func (s *Server) SimulatePressureNotification(ctx context.Context, req *pb.SimulatePressureNotificationRequest) (*pb.SimulatePressureNotificationResponse, error) {
	params := map[string]interface{}{
		"level": req.Level,
	}
	if _, err := s.send(ctx, req.SessionId, "Memory.simulatePressureNotification", params); err != nil {
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
	if _, err := s.send(ctx, req.SessionId, "Memory.startSampling", params); err != nil {
		return nil, fmt.Errorf("Memory.startSampling: %w", err)
	}
	return &pb.StartSamplingResponse{}, nil
}

func (s *Server) StopSampling(ctx context.Context, req *pb.StopSamplingRequest) (*pb.StopSamplingResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Memory.stopSampling", nil); err != nil {
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
	result, err := s.send(ctx, req.SessionId, "Memory.getAllTimeSamplingProfile", nil)
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
	result, err := s.send(ctx, req.SessionId, "Memory.getSamplingProfile", nil)
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
	result, err := s.send(ctx, req.SessionId, "Memory.getBrowserSamplingProfile", nil)
	if err != nil {
		return nil, fmt.Errorf("Memory.getBrowserSamplingProfile: %w", err)
	}
	profile, err := unmarshalSamplingProfile(result)
	if err != nil {
		return nil, fmt.Errorf("Memory.getBrowserSamplingProfile: unmarshal: %w", err)
	}
	return &pb.GetBrowserSamplingProfileResponse{Profile: profile}, nil
}
