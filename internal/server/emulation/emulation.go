// Package emulation implements the gRPC EmulationService by bridging to CDP.
package emulation

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/emulation"
)

type Server struct {
	pb.UnimplementedEmulationServiceServer
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


func (s *Server) SetDeviceMetricsOverride(ctx context.Context, req *pb.SetDeviceMetricsOverrideRequest) (*pb.SetDeviceMetricsOverrideResponse, error) {
	params := map[string]interface{}{
		"width":             req.Width,
		"height":            req.Height,
		"deviceScaleFactor": req.DeviceScaleFactor,
		"mobile":            req.Mobile,
	}
	if req.ScreenOrientation != nil {
		params["screenOrientation"] = map[string]interface{}{
			"type":  req.ScreenOrientation.Type,
			"angle": req.ScreenOrientation.Angle,
		}
	}
	if req.ScreenWidth > 0 {
		params["screenWidth"] = req.ScreenWidth
	}
	if req.ScreenHeight > 0 {
		params["screenHeight"] = req.ScreenHeight
	}
	if req.PositionX > 0 {
		params["positionX"] = req.PositionX
	}
	if req.PositionY > 0 {
		params["positionY"] = req.PositionY
	}
	if req.DontSetVisibleSize {
		params["dontSetVisibleSize"] = true
	}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setDeviceMetricsOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setDeviceMetricsOverride: %w", err)
	}
	return &pb.SetDeviceMetricsOverrideResponse{}, nil
}

func (s *Server) ClearDeviceMetricsOverride(ctx context.Context, req *pb.ClearDeviceMetricsOverrideRequest) (*pb.ClearDeviceMetricsOverrideResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Emulation.clearDeviceMetricsOverride", nil); err != nil {
		return nil, fmt.Errorf("Emulation.clearDeviceMetricsOverride: %w", err)
	}
	return &pb.ClearDeviceMetricsOverrideResponse{}, nil
}

func (s *Server) SetUserAgentOverride(ctx context.Context, req *pb.SetUserAgentOverrideRequest) (*pb.SetUserAgentOverrideResponse, error) {
	params := map[string]interface{}{
		"userAgent": req.UserAgent,
	}
	if req.AcceptLanguage != "" {
		params["acceptLanguage"] = req.AcceptLanguage
	}
	if req.Platform != "" {
		params["platform"] = req.Platform
	}
	if req.UserAgentMetadata != nil {
		meta := map[string]interface{}{}
		if len(req.UserAgentMetadata.Brands) > 0 {
			brands := make([]map[string]string, len(req.UserAgentMetadata.Brands))
			for i, b := range req.UserAgentMetadata.Brands {
				brands[i] = map[string]string{"brand": b.Brand, "version": b.Version}
			}
			meta["brands"] = brands
		}
		if req.UserAgentMetadata.Platform != "" {
			meta["platform"] = req.UserAgentMetadata.Platform
		}
		if req.UserAgentMetadata.PlatformVersion != "" {
			meta["platformVersion"] = req.UserAgentMetadata.PlatformVersion
		}
		if req.UserAgentMetadata.Architecture != "" {
			meta["architecture"] = req.UserAgentMetadata.Architecture
		}
		if req.UserAgentMetadata.Model != "" {
			meta["model"] = req.UserAgentMetadata.Model
		}
		if req.UserAgentMetadata.Mobile {
			meta["mobile"] = true
		}
		if req.UserAgentMetadata.Bitness != "" {
			meta["bitness"] = req.UserAgentMetadata.Bitness
		}
		params["userAgentMetadata"] = meta
	}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setUserAgentOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setUserAgentOverride: %w", err)
	}
	return &pb.SetUserAgentOverrideResponse{}, nil
}

func (s *Server) SetGeolocationOverride(ctx context.Context, req *pb.SetGeolocationOverrideRequest) (*pb.SetGeolocationOverrideResponse, error) {
	params := map[string]interface{}{}
	if req.Latitude != 0 {
		params["latitude"] = req.Latitude
	}
	if req.Longitude != 0 {
		params["longitude"] = req.Longitude
	}
	if req.Accuracy != 0 {
		params["accuracy"] = req.Accuracy
	}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setGeolocationOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setGeolocationOverride: %w", err)
	}
	return &pb.SetGeolocationOverrideResponse{}, nil
}

func (s *Server) ClearGeolocationOverride(ctx context.Context, req *pb.ClearGeolocationOverrideRequest) (*pb.ClearGeolocationOverrideResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Emulation.clearGeolocationOverride", nil); err != nil {
		return nil, fmt.Errorf("Emulation.clearGeolocationOverride: %w", err)
	}
	return &pb.ClearGeolocationOverrideResponse{}, nil
}

func (s *Server) SetTouchEmulationEnabled(ctx context.Context, req *pb.SetTouchEmulationEnabledRequest) (*pb.SetTouchEmulationEnabledResponse, error) {
	params := map[string]interface{}{"enabled": req.Enabled}
	if req.MaxTouchPoints > 0 {
		params["maxTouchPoints"] = req.MaxTouchPoints
	}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setTouchEmulationEnabled", params); err != nil {
		return nil, fmt.Errorf("Emulation.setTouchEmulationEnabled: %w", err)
	}
	return &pb.SetTouchEmulationEnabledResponse{}, nil
}

func (s *Server) SetEmulatedMedia(ctx context.Context, req *pb.SetEmulatedMediaRequest) (*pb.SetEmulatedMediaResponse, error) {
	params := map[string]interface{}{}
	if req.Media != "" {
		params["media"] = req.Media
	}
	if len(req.Features) > 0 {
		features := make([]map[string]string, len(req.Features))
		for i, f := range req.Features {
			features[i] = map[string]string{"name": f.Name, "value": f.Value}
		}
		params["features"] = features
	}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setEmulatedMedia", params); err != nil {
		return nil, fmt.Errorf("Emulation.setEmulatedMedia: %w", err)
	}
	return &pb.SetEmulatedMediaResponse{}, nil
}

func (s *Server) SetTimezoneOverride(ctx context.Context, req *pb.SetTimezoneOverrideRequest) (*pb.SetTimezoneOverrideResponse, error) {
	params := map[string]interface{}{"timezoneId": req.TimezoneId}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setTimezoneOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setTimezoneOverride: %w", err)
	}
	return &pb.SetTimezoneOverrideResponse{}, nil
}

func (s *Server) SetLocaleOverride(ctx context.Context, req *pb.SetLocaleOverrideRequest) (*pb.SetLocaleOverrideResponse, error) {
	params := map[string]interface{}{"locale": req.Locale}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setLocaleOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setLocaleOverride: %w", err)
	}
	return &pb.SetLocaleOverrideResponse{}, nil
}

func (s *Server) SetScrollbarsHidden(ctx context.Context, req *pb.SetScrollbarsHiddenRequest) (*pb.SetScrollbarsHiddenResponse, error) {
	params := map[string]interface{}{"hidden": req.Hidden}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setScrollbarsHidden", params); err != nil {
		return nil, fmt.Errorf("Emulation.setScrollbarsHidden: %w", err)
	}
	return &pb.SetScrollbarsHiddenResponse{}, nil
}

func (s *Server) SetDocumentCookieDisabled(ctx context.Context, req *pb.SetDocumentCookieDisabledRequest) (*pb.SetDocumentCookieDisabledResponse, error) {
	params := map[string]interface{}{"disabled": req.Disabled}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setDocumentCookieDisabled", params); err != nil {
		return nil, fmt.Errorf("Emulation.setDocumentCookieDisabled: %w", err)
	}
	return &pb.SetDocumentCookieDisabledResponse{}, nil
}

func (s *Server) SetEmulatedVisionDeficiency(ctx context.Context, req *pb.SetEmulatedVisionDeficiencyRequest) (*pb.SetEmulatedVisionDeficiencyResponse, error) {
	params := map[string]interface{}{"type": req.Type}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setEmulatedVisionDeficiency", params); err != nil {
		return nil, fmt.Errorf("Emulation.setEmulatedVisionDeficiency: %w", err)
	}
	return &pb.SetEmulatedVisionDeficiencyResponse{}, nil
}

func (s *Server) SetDisabledImageTypes(ctx context.Context, req *pb.SetDisabledImageTypesRequest) (*pb.SetDisabledImageTypesResponse, error) {
	params := map[string]interface{}{"imageTypes": req.ImageTypes}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setDisabledImageTypes", params); err != nil {
		return nil, fmt.Errorf("Emulation.setDisabledImageTypes: %w", err)
	}
	return &pb.SetDisabledImageTypesResponse{}, nil
}

func (s *Server) SetAutomationOverride(ctx context.Context, req *pb.SetAutomationOverrideRequest) (*pb.SetAutomationOverrideResponse, error) {
	params := map[string]interface{}{"enabled": req.Enabled}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setAutomationOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setAutomationOverride: %w", err)
	}
	return &pb.SetAutomationOverrideResponse{}, nil
}

func (s *Server) SetCPUThrottlingRate(ctx context.Context, req *pb.SetCPUThrottlingRateRequest) (*pb.SetCPUThrottlingRateResponse, error) {
	params := map[string]interface{}{"rate": req.Rate}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setCPUThrottlingRate", params); err != nil {
		return nil, fmt.Errorf("Emulation.setCPUThrottlingRate: %w", err)
	}
	return &pb.SetCPUThrottlingRateResponse{}, nil
}

func (s *Server) SetDefaultBackgroundColorOverride(ctx context.Context, req *pb.SetDefaultBackgroundColorOverrideRequest) (*pb.SetDefaultBackgroundColorOverrideResponse, error) {
	params := map[string]interface{}{}
	if req.Color != nil {
		params["color"] = map[string]interface{}{
			"r": req.Color.R,
			"g": req.Color.G,
			"b": req.Color.B,
			"a": req.Color.A,
		}
	}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setDefaultBackgroundColorOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setDefaultBackgroundColorOverride: %w", err)
	}
	return &pb.SetDefaultBackgroundColorOverrideResponse{}, nil
}

func (s *Server) ResetPageScaleFactor(ctx context.Context, req *pb.ResetPageScaleFactorRequest) (*pb.ResetPageScaleFactorResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Emulation.resetPageScaleFactor", nil); err != nil {
		return nil, fmt.Errorf("Emulation.resetPageScaleFactor: %w", err)
	}
	return &pb.ResetPageScaleFactorResponse{}, nil
}

func (s *Server) SetPageScaleFactor(ctx context.Context, req *pb.SetPageScaleFactorRequest) (*pb.SetPageScaleFactorResponse, error) {
	params := map[string]interface{}{"pageScaleFactor": req.PageScaleFactor}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setPageScaleFactor", params); err != nil {
		return nil, fmt.Errorf("Emulation.setPageScaleFactor: %w", err)
	}
	return &pb.SetPageScaleFactorResponse{}, nil
}

func (s *Server) SetHardwareConcurrencyOverride(ctx context.Context, req *pb.SetHardwareConcurrencyOverrideRequest) (*pb.SetHardwareConcurrencyOverrideResponse, error) {
	params := map[string]interface{}{"hardwareConcurrency": req.HardwareConcurrency}
	if _, err := s.send(ctx, req.SessionId, "Emulation.setHardwareConcurrencyOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setHardwareConcurrencyOverride: %w", err)
	}
	return &pb.SetHardwareConcurrencyOverrideResponse{}, nil
}
