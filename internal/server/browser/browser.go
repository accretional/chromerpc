// Package browser implements the gRPC BrowserService by bridging to CDP.
// Browser domain commands operate at browser level (no session ID).
package browser

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/browser"
)

type Server struct {
	pb.UnimplementedBrowserServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// sendBrowser sends at browser level (no session ID).
func (s *Server) sendBrowser(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.SendWithSession(ctx, method, params, "")
}

func (s *Server) GetVersion(ctx context.Context, req *pb.GetVersionRequest) (*pb.GetVersionResponse, error) {
	result, err := s.sendBrowser(ctx, "Browser.getVersion", nil)
	if err != nil {
		return nil, fmt.Errorf("Browser.getVersion: %w", err)
	}
	var resp struct {
		ProtocolVersion string `json:"protocolVersion"`
		Product         string `json:"product"`
		Revision        string `json:"revision"`
		UserAgent       string `json:"userAgent"`
		JSVersion       string `json:"jsVersion"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Browser.getVersion: unmarshal: %w", err)
	}
	return &pb.GetVersionResponse{
		ProtocolVersion: resp.ProtocolVersion,
		Product:         resp.Product,
		Revision:        resp.Revision,
		UserAgent:       resp.UserAgent,
		JsVersion:       resp.JSVersion,
	}, nil
}

func (s *Server) Close(ctx context.Context, req *pb.CloseRequest) (*pb.CloseResponse, error) {
	if _, err := s.sendBrowser(ctx, "Browser.close", nil); err != nil {
		return nil, fmt.Errorf("Browser.close: %w", err)
	}
	return &pb.CloseResponse{}, nil
}

func (s *Server) GetBrowserCommandLine(ctx context.Context, req *pb.GetBrowserCommandLineRequest) (*pb.GetBrowserCommandLineResponse, error) {
	result, err := s.sendBrowser(ctx, "Browser.getBrowserCommandLine", nil)
	if err != nil {
		return nil, fmt.Errorf("Browser.getBrowserCommandLine: %w", err)
	}
	var resp struct {
		Arguments []string `json:"arguments"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Browser.getBrowserCommandLine: unmarshal: %w", err)
	}
	return &pb.GetBrowserCommandLineResponse{Arguments: resp.Arguments}, nil
}

func (s *Server) GetHistograms(ctx context.Context, req *pb.GetHistogramsRequest) (*pb.GetHistogramsResponse, error) {
	params := map[string]interface{}{}
	if req.Query != "" {
		params["query"] = req.Query
	}
	if req.Delta {
		params["delta"] = true
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.sendBrowser(ctx, "Browser.getHistograms", params)
	} else {
		result, err = s.sendBrowser(ctx, "Browser.getHistograms", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Browser.getHistograms: %w", err)
	}
	var resp struct {
		Histograms []cdpHistogram `json:"histograms"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Browser.getHistograms: unmarshal: %w", err)
	}
	histograms := make([]*pb.Histogram, len(resp.Histograms))
	for i, h := range resp.Histograms {
		histograms[i] = h.toProto()
	}
	return &pb.GetHistogramsResponse{Histograms: histograms}, nil
}

func (s *Server) GetHistogram(ctx context.Context, req *pb.GetHistogramRequest) (*pb.GetHistogramResponse, error) {
	params := map[string]interface{}{"name": req.Name}
	if req.Delta {
		params["delta"] = true
	}
	result, err := s.sendBrowser(ctx, "Browser.getHistogram", params)
	if err != nil {
		return nil, fmt.Errorf("Browser.getHistogram: %w", err)
	}
	var resp struct {
		Histogram cdpHistogram `json:"histogram"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Browser.getHistogram: unmarshal: %w", err)
	}
	return &pb.GetHistogramResponse{Histogram: resp.Histogram.toProto()}, nil
}

func (s *Server) GetWindowBounds(ctx context.Context, req *pb.GetWindowBoundsRequest) (*pb.GetWindowBoundsResponse, error) {
	params := map[string]interface{}{"windowId": req.WindowId}
	result, err := s.sendBrowser(ctx, "Browser.getWindowBounds", params)
	if err != nil {
		return nil, fmt.Errorf("Browser.getWindowBounds: %w", err)
	}
	var resp struct {
		Bounds cdpBounds `json:"bounds"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Browser.getWindowBounds: unmarshal: %w", err)
	}
	return &pb.GetWindowBoundsResponse{Bounds: resp.Bounds.toProto()}, nil
}

func (s *Server) GetWindowForTarget(ctx context.Context, req *pb.GetWindowForTargetRequest) (*pb.GetWindowForTargetResponse, error) {
	params := map[string]interface{}{}
	if req.TargetId != "" {
		params["targetId"] = req.TargetId
	}
	result, err := s.sendBrowser(ctx, "Browser.getWindowForTarget", params)
	if err != nil {
		return nil, fmt.Errorf("Browser.getWindowForTarget: %w", err)
	}
	var resp struct {
		WindowID int32     `json:"windowId"`
		Bounds   cdpBounds `json:"bounds"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Browser.getWindowForTarget: unmarshal: %w", err)
	}
	return &pb.GetWindowForTargetResponse{
		WindowId: resp.WindowID,
		Bounds:   resp.Bounds.toProto(),
	}, nil
}

func (s *Server) SetWindowBounds(ctx context.Context, req *pb.SetWindowBoundsRequest) (*pb.SetWindowBoundsResponse, error) {
	params := map[string]interface{}{
		"windowId": req.WindowId,
	}
	if req.Bounds != nil {
		bounds := map[string]interface{}{}
		if req.Bounds.Left != 0 {
			bounds["left"] = req.Bounds.Left
		}
		if req.Bounds.Top != 0 {
			bounds["top"] = req.Bounds.Top
		}
		if req.Bounds.Width != 0 {
			bounds["width"] = req.Bounds.Width
		}
		if req.Bounds.Height != 0 {
			bounds["height"] = req.Bounds.Height
		}
		if req.Bounds.WindowState != "" {
			bounds["windowState"] = req.Bounds.WindowState
		}
		params["bounds"] = bounds
	}
	if _, err := s.sendBrowser(ctx, "Browser.setWindowBounds", params); err != nil {
		return nil, fmt.Errorf("Browser.setWindowBounds: %w", err)
	}
	return &pb.SetWindowBoundsResponse{}, nil
}

func (s *Server) SetPermission(ctx context.Context, req *pb.SetPermissionRequest) (*pb.SetPermissionResponse, error) {
	params := map[string]interface{}{
		"setting": req.Setting,
	}
	if req.Permission != nil {
		perm := map[string]interface{}{"name": req.Permission.Name}
		if req.Permission.Sysex {
			perm["sysex"] = true
		}
		if req.Permission.UserVisibleOnly {
			perm["userVisibleOnly"] = true
		}
		if req.Permission.AllowWithoutSanitization {
			perm["allowWithoutSanitization"] = true
		}
		if req.Permission.AllowWithoutGesture {
			perm["allowWithoutGesture"] = true
		}
		if req.Permission.PanTiltZoom {
			perm["panTiltZoom"] = true
		}
		params["permission"] = perm
	}
	if req.Origin != "" {
		params["origin"] = req.Origin
	}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if _, err := s.sendBrowser(ctx, "Browser.setPermission", params); err != nil {
		return nil, fmt.Errorf("Browser.setPermission: %w", err)
	}
	return &pb.SetPermissionResponse{}, nil
}

func (s *Server) GrantPermissions(ctx context.Context, req *pb.GrantPermissionsRequest) (*pb.GrantPermissionsResponse, error) {
	params := map[string]interface{}{
		"permissions": req.Permissions,
	}
	if req.Origin != "" {
		params["origin"] = req.Origin
	}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if _, err := s.sendBrowser(ctx, "Browser.grantPermissions", params); err != nil {
		return nil, fmt.Errorf("Browser.grantPermissions: %w", err)
	}
	return &pb.GrantPermissionsResponse{}, nil
}

func (s *Server) ResetPermissions(ctx context.Context, req *pb.ResetPermissionsRequest) (*pb.ResetPermissionsResponse, error) {
	params := map[string]interface{}{}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if len(params) > 0 {
		_, err := s.sendBrowser(ctx, "Browser.resetPermissions", params)
		if err != nil {
			return nil, fmt.Errorf("Browser.resetPermissions: %w", err)
		}
	} else {
		_, err := s.sendBrowser(ctx, "Browser.resetPermissions", nil)
		if err != nil {
			return nil, fmt.Errorf("Browser.resetPermissions: %w", err)
		}
	}
	return &pb.ResetPermissionsResponse{}, nil
}

func (s *Server) SetDownloadBehavior(ctx context.Context, req *pb.SetDownloadBehaviorRequest) (*pb.SetDownloadBehaviorResponse, error) {
	params := map[string]interface{}{
		"behavior": req.Behavior,
	}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if req.DownloadPath != "" {
		params["downloadPath"] = req.DownloadPath
	}
	if req.EventsEnabled {
		params["eventsEnabled"] = true
	}
	if _, err := s.sendBrowser(ctx, "Browser.setDownloadBehavior", params); err != nil {
		return nil, fmt.Errorf("Browser.setDownloadBehavior: %w", err)
	}
	return &pb.SetDownloadBehaviorResponse{}, nil
}

func (s *Server) CancelDownload(ctx context.Context, req *pb.CancelDownloadRequest) (*pb.CancelDownloadResponse, error) {
	params := map[string]interface{}{
		"guid": req.Guid,
	}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if _, err := s.sendBrowser(ctx, "Browser.cancelDownload", params); err != nil {
		return nil, fmt.Errorf("Browser.cancelDownload: %w", err)
	}
	return &pb.CancelDownloadResponse{}, nil
}

// --- internal helpers ---

type cdpBounds struct {
	Left        int32  `json:"left"`
	Top         int32  `json:"top"`
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
	WindowState string `json:"windowState"`
}

func (b *cdpBounds) toProto() *pb.Bounds {
	return &pb.Bounds{
		Left:        b.Left,
		Top:         b.Top,
		Width:       b.Width,
		Height:      b.Height,
		WindowState: b.WindowState,
	}
}

type cdpHistogram struct {
	Name    string      `json:"name"`
	Sum     int32       `json:"sum"`
	Count   int32       `json:"count"`
	Buckets []cdpBucket `json:"buckets"`
}

type cdpBucket struct {
	Low   int32 `json:"low"`
	High  int32 `json:"high"`
	Count int32 `json:"count"`
}

func (h *cdpHistogram) toProto() *pb.Histogram {
	buckets := make([]*pb.Bucket, len(h.Buckets))
	for i, b := range h.Buckets {
		buckets[i] = &pb.Bucket{Low: b.Low, High: b.High, Count: b.Count}
	}
	return &pb.Histogram{
		Name:    h.Name,
		Sum:     h.Sum,
		Count:   h.Count,
		Buckets: buckets,
	}
}
