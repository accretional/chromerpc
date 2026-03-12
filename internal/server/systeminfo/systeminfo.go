// Package systeminfo implements the gRPC SystemInfoService by bridging to CDP.
package systeminfo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/systeminfo"
)

type Server struct {
	pb.UnimplementedSystemInfoServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// sendBrowser sends at browser level (no session ID).
func (s *Server) sendBrowser(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.SendWithSession(ctx, method, params, "")
}

func (s *Server) GetInfo(ctx context.Context, req *pb.GetInfoRequest) (*pb.GetInfoResponse, error) {
	result, err := s.sendBrowser(ctx, "SystemInfo.getInfo", nil)
	if err != nil {
		return nil, fmt.Errorf("SystemInfo.getInfo: %w", err)
	}
	var resp struct {
		GPU          json.RawMessage `json:"gpu"`
		ModelName    string          `json:"modelName"`
		ModelVersion string          `json:"modelVersion"`
		CommandLine  string          `json:"commandLine"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("SystemInfo.getInfo: unmarshal: %w", err)
	}
	gpuStr := ""
	if resp.GPU != nil {
		gpuStr = string(resp.GPU)
	}
	return &pb.GetInfoResponse{
		Gpu:          gpuStr,
		ModelName:    resp.ModelName,
		ModelVersion: resp.ModelVersion,
		CommandLine:  resp.CommandLine,
	}, nil
}

func (s *Server) GetFeatureState(ctx context.Context, req *pb.GetFeatureStateRequest) (*pb.GetFeatureStateResponse, error) {
	params := map[string]interface{}{
		"featureState": req.FeatureState,
	}
	result, err := s.sendBrowser(ctx, "SystemInfo.getFeatureState", params)
	if err != nil {
		return nil, fmt.Errorf("SystemInfo.getFeatureState: %w", err)
	}
	var resp struct {
		FeatureEnabled bool `json:"featureEnabled"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("SystemInfo.getFeatureState: unmarshal: %w", err)
	}
	return &pb.GetFeatureStateResponse{FeatureEnabled: resp.FeatureEnabled}, nil
}

func (s *Server) GetProcessInfo(ctx context.Context, req *pb.GetProcessInfoRequest) (*pb.GetProcessInfoResponse, error) {
	result, err := s.sendBrowser(ctx, "SystemInfo.getProcessInfo", nil)
	if err != nil {
		return nil, fmt.Errorf("SystemInfo.getProcessInfo: %w", err)
	}
	var resp struct {
		ProcessInfo []struct {
			Type    string  `json:"type"`
			ID      int32   `json:"id"`
			CPUTime float64 `json:"cpuTime"`
		} `json:"processInfo"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("SystemInfo.getProcessInfo: unmarshal: %w", err)
	}
	infos := make([]*pb.ProcessInfo, len(resp.ProcessInfo))
	for i, p := range resp.ProcessInfo {
		infos[i] = &pb.ProcessInfo{
			Type:    p.Type,
			Id:      p.ID,
			CpuTime: p.CPUTime,
		}
	}
	return &pb.GetProcessInfoResponse{ProcessInfo: infos}, nil
}
