// Package overlay implements the gRPC OverlayService by bridging to CDP.
package overlay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/overlay"
)

type Server struct {
	pb.UnimplementedOverlayServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Overlay.enable", nil); err != nil {
		return nil, fmt.Errorf("Overlay.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Overlay.disable", nil); err != nil {
		return nil, fmt.Errorf("Overlay.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) HighlightNode(ctx context.Context, req *pb.HighlightNodeRequest) (*pb.HighlightNodeResponse, error) {
	params := map[string]interface{}{}
	if req.HighlightConfig != nil {
		params["highlightConfig"] = highlightConfigToMap(req.HighlightConfig)
	}
	if req.NodeId != 0 {
		params["nodeId"] = req.NodeId
	}
	if req.BackendNodeId != 0 {
		params["backendNodeId"] = req.BackendNodeId
	}
	if req.ObjectId != "" {
		params["objectId"] = req.ObjectId
	}
	if req.Selector != "" {
		params["selector"] = req.Selector
	}
	if _, err := s.client.Send(ctx, "Overlay.highlightNode", params); err != nil {
		return nil, fmt.Errorf("Overlay.highlightNode: %w", err)
	}
	return &pb.HighlightNodeResponse{}, nil
}

func (s *Server) HighlightRect(ctx context.Context, req *pb.HighlightRectRequest) (*pb.HighlightRectResponse, error) {
	params := map[string]interface{}{
		"x":      req.X,
		"y":      req.Y,
		"width":  req.Width,
		"height": req.Height,
	}
	if req.Color != nil {
		params["color"] = rgbaToMap(req.Color)
	}
	if req.OutlineColor != nil {
		params["outlineColor"] = rgbaToMap(req.OutlineColor)
	}
	if _, err := s.client.Send(ctx, "Overlay.highlightRect", params); err != nil {
		return nil, fmt.Errorf("Overlay.highlightRect: %w", err)
	}
	return &pb.HighlightRectResponse{}, nil
}

func (s *Server) HighlightQuad(ctx context.Context, req *pb.HighlightQuadRequest) (*pb.HighlightQuadResponse, error) {
	params := map[string]interface{}{
		"quad": req.Quad,
	}
	if req.Color != nil {
		params["color"] = rgbaToMap(req.Color)
	}
	if req.OutlineColor != nil {
		params["outlineColor"] = rgbaToMap(req.OutlineColor)
	}
	if _, err := s.client.Send(ctx, "Overlay.highlightQuad", params); err != nil {
		return nil, fmt.Errorf("Overlay.highlightQuad: %w", err)
	}
	return &pb.HighlightQuadResponse{}, nil
}

func (s *Server) HideHighlight(ctx context.Context, req *pb.HideHighlightRequest) (*pb.HideHighlightResponse, error) {
	if _, err := s.client.Send(ctx, "Overlay.hideHighlight", nil); err != nil {
		return nil, fmt.Errorf("Overlay.hideHighlight: %w", err)
	}
	return &pb.HideHighlightResponse{}, nil
}

func (s *Server) SetInspectMode(ctx context.Context, req *pb.SetInspectModeRequest) (*pb.SetInspectModeResponse, error) {
	params := map[string]interface{}{
		"mode": req.Mode,
	}
	if req.HighlightConfig != nil {
		params["highlightConfig"] = highlightConfigToMap(req.HighlightConfig)
	}
	if _, err := s.client.Send(ctx, "Overlay.setInspectMode", params); err != nil {
		return nil, fmt.Errorf("Overlay.setInspectMode: %w", err)
	}
	return &pb.SetInspectModeResponse{}, nil
}

func (s *Server) SetShowPaintRects(ctx context.Context, req *pb.SetShowPaintRectsRequest) (*pb.SetShowPaintRectsResponse, error) {
	params := map[string]interface{}{"result": req.Result}
	if _, err := s.client.Send(ctx, "Overlay.setShowPaintRects", params); err != nil {
		return nil, fmt.Errorf("Overlay.setShowPaintRects: %w", err)
	}
	return &pb.SetShowPaintRectsResponse{}, nil
}

func (s *Server) SetShowLayoutShiftRegions(ctx context.Context, req *pb.SetShowLayoutShiftRegionsRequest) (*pb.SetShowLayoutShiftRegionsResponse, error) {
	params := map[string]interface{}{"result": req.Result}
	if _, err := s.client.Send(ctx, "Overlay.setShowLayoutShiftRegions", params); err != nil {
		return nil, fmt.Errorf("Overlay.setShowLayoutShiftRegions: %w", err)
	}
	return &pb.SetShowLayoutShiftRegionsResponse{}, nil
}

func (s *Server) SetShowScrollBottleneckRects(ctx context.Context, req *pb.SetShowScrollBottleneckRectsRequest) (*pb.SetShowScrollBottleneckRectsResponse, error) {
	params := map[string]interface{}{"show": req.Show}
	if _, err := s.client.Send(ctx, "Overlay.setShowScrollBottleneckRects", params); err != nil {
		return nil, fmt.Errorf("Overlay.setShowScrollBottleneckRects: %w", err)
	}
	return &pb.SetShowScrollBottleneckRectsResponse{}, nil
}

func (s *Server) SetShowFPSCounter(ctx context.Context, req *pb.SetShowFPSCounterRequest) (*pb.SetShowFPSCounterResponse, error) {
	params := map[string]interface{}{"show": req.Show}
	if _, err := s.client.Send(ctx, "Overlay.setShowFPSCounter", params); err != nil {
		return nil, fmt.Errorf("Overlay.setShowFPSCounter: %w", err)
	}
	return &pb.SetShowFPSCounterResponse{}, nil
}

func (s *Server) SetShowDebugBorders(ctx context.Context, req *pb.SetShowDebugBordersRequest) (*pb.SetShowDebugBordersResponse, error) {
	params := map[string]interface{}{"show": req.Show}
	if _, err := s.client.Send(ctx, "Overlay.setShowDebugBorders", params); err != nil {
		return nil, fmt.Errorf("Overlay.setShowDebugBorders: %w", err)
	}
	return &pb.SetShowDebugBordersResponse{}, nil
}

func (s *Server) SetPausedInDebuggerMessage(ctx context.Context, req *pb.SetPausedInDebuggerMessageRequest) (*pb.SetPausedInDebuggerMessageResponse, error) {
	params := map[string]interface{}{}
	if req.Message != "" {
		params["message"] = req.Message
	}
	if len(params) > 0 {
		_, err := s.client.Send(ctx, "Overlay.setPausedInDebuggerMessage", params)
		if err != nil {
			return nil, fmt.Errorf("Overlay.setPausedInDebuggerMessage: %w", err)
		}
	} else {
		if _, err := s.client.Send(ctx, "Overlay.setPausedInDebuggerMessage", nil); err != nil {
			return nil, fmt.Errorf("Overlay.setPausedInDebuggerMessage: %w", err)
		}
	}
	return &pb.SetPausedInDebuggerMessageResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.OverlayService_SubscribeEventsServer) error {
	ch := make(chan *pb.OverlayEvent, 64)
	defer close(ch)

	events := []string{
		"Overlay.inspectNodeRequested",
		"Overlay.nodeHighlightRequested",
		"Overlay.screenshotRequested",
	}
	var unsubs []func()
	for _, evt := range events {
		evt := evt
		unsub := s.client.On(evt, func(_ string, params json.RawMessage, _ string) {
			converted := convertOverlayEvent(evt, params)
			if converted != nil {
				ch <- converted
			}
		})
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

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

func convertOverlayEvent(method string, params json.RawMessage) *pb.OverlayEvent {
	switch method {
	case "Overlay.inspectNodeRequested":
		var raw struct {
			BackendNodeID int32 `json:"backendNodeId"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.OverlayEvent{Event: &pb.OverlayEvent_InspectNodeRequested{
			InspectNodeRequested: &pb.InspectNodeRequestedEvent{BackendNodeId: raw.BackendNodeID},
		}}
	case "Overlay.nodeHighlightRequested":
		var raw struct {
			NodeID int32 `json:"nodeId"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.OverlayEvent{Event: &pb.OverlayEvent_NodeHighlightRequested{
			NodeHighlightRequested: &pb.NodeHighlightRequestedEvent{NodeId: raw.NodeID},
		}}
	case "Overlay.screenshotRequested":
		var raw struct {
			Viewport struct {
				X      float64 `json:"x"`
				Y      float64 `json:"y"`
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
				Scale  float64 `json:"scale"`
			} `json:"viewport"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.OverlayEvent{Event: &pb.OverlayEvent_ScreenshotRequested{
			ScreenshotRequested: &pb.ScreenshotRequestedEvent{
				X: raw.Viewport.X, Y: raw.Viewport.Y,
				Width: raw.Viewport.Width, Height: raw.Viewport.Height,
				Scale: raw.Viewport.Scale,
			},
		}}
	}
	return nil
}

func rgbaToMap(c *pb.RGBA) map[string]interface{} {
	m := map[string]interface{}{
		"r": c.R, "g": c.G, "b": c.B,
	}
	if c.A != 0 {
		m["a"] = c.A
	}
	return m
}

func highlightConfigToMap(cfg *pb.HighlightConfig) map[string]interface{} {
	m := map[string]interface{}{}
	if cfg.ShowInfo {
		m["showInfo"] = true
	}
	if cfg.ShowStyles {
		m["showStyles"] = true
	}
	if cfg.ShowRulers {
		m["showRulers"] = true
	}
	if cfg.ShowAccessibilityInfo {
		m["showAccessibilityInfo"] = true
	}
	if cfg.ShowExtensionLines {
		m["showExtensionLines"] = true
	}
	if cfg.ContentColor != nil {
		m["contentColor"] = rgbaToMap(cfg.ContentColor)
	}
	if cfg.PaddingColor != nil {
		m["paddingColor"] = rgbaToMap(cfg.PaddingColor)
	}
	if cfg.BorderColor != nil {
		m["borderColor"] = rgbaToMap(cfg.BorderColor)
	}
	if cfg.MarginColor != nil {
		m["marginColor"] = rgbaToMap(cfg.MarginColor)
	}
	if cfg.EventTargetColor != nil {
		m["eventTargetColor"] = rgbaToMap(cfg.EventTargetColor)
	}
	if cfg.ShapeColor != nil {
		m["shapeColor"] = rgbaToMap(cfg.ShapeColor)
	}
	if cfg.ShapeMarginColor != nil {
		m["shapeMarginColor"] = rgbaToMap(cfg.ShapeMarginColor)
	}
	if cfg.CssGridColor != nil {
		m["cssGridColor"] = rgbaToMap(cfg.CssGridColor)
	}
	if cfg.ColorFormat != "" {
		m["colorFormat"] = cfg.ColorFormat
	}
	return m
}
