// Package pwa implements the gRPC PWAService by bridging to CDP.
package pwa

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/pwa"
)

type Server struct {
	pb.UnimplementedPWAServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) GetOsAppState(ctx context.Context, req *pb.GetOsAppStateRequest) (*pb.GetOsAppStateResponse, error) {
	params := map[string]interface{}{
		"manifestId": req.ManifestId,
	}
	result, err := s.client.Send(ctx, "PWA.getOsAppState", params)
	if err != nil {
		return nil, fmt.Errorf("PWA.getOsAppState: %w", err)
	}
	var resp struct {
		BadgeCount        int32  `json:"badgeCount"`
		InstallationState string `json:"installationState"`
		DisplayMode       string `json:"displayMode"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("PWA.getOsAppState: unmarshal: %w", err)
	}
	return &pb.GetOsAppStateResponse{
		BadgeCount:        resp.BadgeCount,
		InstallationState: resp.InstallationState,
		DisplayMode:       resp.DisplayMode,
	}, nil
}

func (s *Server) Install(ctx context.Context, req *pb.InstallRequest) (*pb.InstallResponse, error) {
	params := map[string]interface{}{
		"manifestId": req.ManifestId,
	}
	if req.InstallUrlOrBundleUrl != nil {
		params["installUrlOrBundleUrl"] = *req.InstallUrlOrBundleUrl
	}
	if _, err := s.client.Send(ctx, "PWA.install", params); err != nil {
		return nil, fmt.Errorf("PWA.install: %w", err)
	}
	return &pb.InstallResponse{}, nil
}

func (s *Server) Uninstall(ctx context.Context, req *pb.UninstallRequest) (*pb.UninstallResponse, error) {
	params := map[string]interface{}{
		"manifestId": req.ManifestId,
	}
	if _, err := s.client.Send(ctx, "PWA.uninstall", params); err != nil {
		return nil, fmt.Errorf("PWA.uninstall: %w", err)
	}
	return &pb.UninstallResponse{}, nil
}

func (s *Server) Launch(ctx context.Context, req *pb.LaunchRequest) (*pb.LaunchResponse, error) {
	params := map[string]interface{}{
		"manifestId": req.ManifestId,
	}
	if req.Url != nil {
		params["url"] = *req.Url
	}
	result, err := s.client.Send(ctx, "PWA.launch", params)
	if err != nil {
		return nil, fmt.Errorf("PWA.launch: %w", err)
	}
	var resp struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("PWA.launch: unmarshal: %w", err)
	}
	return &pb.LaunchResponse{
		TargetId: resp.TargetID,
	}, nil
}

func (s *Server) LaunchFilesInApp(ctx context.Context, req *pb.LaunchFilesInAppRequest) (*pb.LaunchFilesInAppResponse, error) {
	params := map[string]interface{}{
		"manifestId": req.ManifestId,
		"files":      req.Files,
	}
	result, err := s.client.Send(ctx, "PWA.launchFilesInApp", params)
	if err != nil {
		return nil, fmt.Errorf("PWA.launchFilesInApp: %w", err)
	}
	var resp struct {
		TargetIDs []string `json:"targetIds"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("PWA.launchFilesInApp: unmarshal: %w", err)
	}
	return &pb.LaunchFilesInAppResponse{
		TargetIds: resp.TargetIDs,
	}, nil
}

func (s *Server) ChangeAppUserSettings(ctx context.Context, req *pb.ChangeAppUserSettingsRequest) (*pb.ChangeAppUserSettingsResponse, error) {
	params := map[string]interface{}{
		"manifestId": req.ManifestId,
	}
	if req.LinkCapturing != nil {
		params["linkCapturing"] = *req.LinkCapturing
	}
	if req.DisplayMode != nil {
		params["displayMode"] = *req.DisplayMode
	}
	if _, err := s.client.Send(ctx, "PWA.changeAppUserSettings", params); err != nil {
		return nil, fmt.Errorf("PWA.changeAppUserSettings: %w", err)
	}
	return &pb.ChangeAppUserSettingsResponse{}, nil
}
