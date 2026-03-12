// Package layertree implements the gRPC LayerTreeService by bridging to CDP.
package layertree

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/layertree"
)

type Server struct {
	pb.UnimplementedLayerTreeServiceServer
	client *cdpclient.Client
}

func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "LayerTree.enable", nil); err != nil {
		return nil, fmt.Errorf("LayerTree.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "LayerTree.disable", nil); err != nil {
		return nil, fmt.Errorf("LayerTree.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) CompositingReasons(ctx context.Context, req *pb.CompositingReasonsRequest) (*pb.CompositingReasonsResponse, error) {
	params := map[string]interface{}{
		"layerId": req.LayerId,
	}
	result, err := s.client.Send(ctx, "LayerTree.compositingReasons", params)
	if err != nil {
		return nil, fmt.Errorf("LayerTree.compositingReasons: %w", err)
	}
	var raw struct {
		CompositingReasons   []string `json:"compositingReasons"`
		CompositingReasonIds []string `json:"compositingReasonIds"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("LayerTree.compositingReasons unmarshal: %w", err)
	}
	return &pb.CompositingReasonsResponse{
		CompositingReasons:   raw.CompositingReasons,
		CompositingReasonIds: raw.CompositingReasonIds,
	}, nil
}

func (s *Server) MakeSnapshot(ctx context.Context, req *pb.MakeSnapshotRequest) (*pb.MakeSnapshotResponse, error) {
	params := map[string]interface{}{
		"layerId": req.LayerId,
	}
	result, err := s.client.Send(ctx, "LayerTree.makeSnapshot", params)
	if err != nil {
		return nil, fmt.Errorf("LayerTree.makeSnapshot: %w", err)
	}
	var raw struct {
		SnapshotId string `json:"snapshotId"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("LayerTree.makeSnapshot unmarshal: %w", err)
	}
	return &pb.MakeSnapshotResponse{SnapshotId: raw.SnapshotId}, nil
}

func (s *Server) LoadSnapshot(ctx context.Context, req *pb.LoadSnapshotRequest) (*pb.LoadSnapshotResponse, error) {
	tiles := make([]map[string]interface{}, len(req.Tiles))
	for i, t := range req.Tiles {
		tiles[i] = map[string]interface{}{
			"x":       t.X,
			"y":       t.Y,
			"picture": t.Picture,
			"scale":   t.Scale,
		}
	}
	params := map[string]interface{}{
		"tiles": tiles,
	}
	result, err := s.client.Send(ctx, "LayerTree.loadSnapshot", params)
	if err != nil {
		return nil, fmt.Errorf("LayerTree.loadSnapshot: %w", err)
	}
	var raw struct {
		SnapshotId string `json:"snapshotId"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("LayerTree.loadSnapshot unmarshal: %w", err)
	}
	return &pb.LoadSnapshotResponse{SnapshotId: raw.SnapshotId}, nil
}

func (s *Server) ReleaseSnapshot(ctx context.Context, req *pb.ReleaseSnapshotRequest) (*pb.ReleaseSnapshotResponse, error) {
	params := map[string]interface{}{
		"snapshotId": req.SnapshotId,
	}
	if _, err := s.client.Send(ctx, "LayerTree.releaseSnapshot", params); err != nil {
		return nil, fmt.Errorf("LayerTree.releaseSnapshot: %w", err)
	}
	return &pb.ReleaseSnapshotResponse{}, nil
}

func (s *Server) ReplaySnapshot(ctx context.Context, req *pb.ReplaySnapshotRequest) (*pb.ReplaySnapshotResponse, error) {
	params := map[string]interface{}{
		"snapshotId": req.SnapshotId,
	}
	if req.FromStep != 0 {
		params["fromStep"] = req.FromStep
	}
	if req.ToStep != 0 {
		params["toStep"] = req.ToStep
	}
	if req.Scale != 0 {
		params["scale"] = req.Scale
	}
	result, err := s.client.Send(ctx, "LayerTree.replaySnapshot", params)
	if err != nil {
		return nil, fmt.Errorf("LayerTree.replaySnapshot: %w", err)
	}
	var raw struct {
		DataURL string `json:"dataURL"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("LayerTree.replaySnapshot unmarshal: %w", err)
	}
	return &pb.ReplaySnapshotResponse{DataUrl: raw.DataURL}, nil
}

func (s *Server) ProfileSnapshot(ctx context.Context, req *pb.ProfileSnapshotRequest) (*pb.ProfileSnapshotResponse, error) {
	params := map[string]interface{}{
		"snapshotId": req.SnapshotId,
	}
	if req.MinRepeatCount != 0 {
		params["minRepeatCount"] = req.MinRepeatCount
	}
	if req.MinDuration != 0 {
		params["minDuration"] = req.MinDuration
	}
	if req.ClipRect != nil {
		params["clipRect"] = map[string]interface{}{
			"x":      req.ClipRect.X,
			"y":      req.ClipRect.Y,
			"width":  req.ClipRect.Width,
			"height": req.ClipRect.Height,
		}
	}
	result, err := s.client.Send(ctx, "LayerTree.profileSnapshot", params)
	if err != nil {
		return nil, fmt.Errorf("LayerTree.profileSnapshot: %w", err)
	}
	var raw struct {
		Timings json.RawMessage `json:"timings"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("LayerTree.profileSnapshot unmarshal: %w", err)
	}
	return &pb.ProfileSnapshotResponse{Timings: string(raw.Timings)}, nil
}

func (s *Server) SnapshotCommandLog(ctx context.Context, req *pb.SnapshotCommandLogRequest) (*pb.SnapshotCommandLogResponse, error) {
	params := map[string]interface{}{
		"snapshotId": req.SnapshotId,
	}
	result, err := s.client.Send(ctx, "LayerTree.snapshotCommandLog", params)
	if err != nil {
		return nil, fmt.Errorf("LayerTree.snapshotCommandLog: %w", err)
	}
	var raw struct {
		CommandLog json.RawMessage `json:"commandLog"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("LayerTree.snapshotCommandLog unmarshal: %w", err)
	}
	return &pb.SnapshotCommandLogResponse{CommandLog: string(raw.CommandLog)}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.LayerTreeService_SubscribeEventsServer) error {
	ch := make(chan *pb.LayerTreeEvent, 64)
	defer close(ch)

	events := []string{
		"LayerTree.layerTreeDidChange",
		"LayerTree.layerPainted",
	}
	var unsubs []func()
	for _, evt := range events {
		evt := evt
		unsub := s.client.On(evt, func(_ string, params json.RawMessage, _ string) {
			converted := convertLayerTreeEvent(evt, params)
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

func convertLayerTreeEvent(method string, params json.RawMessage) *pb.LayerTreeEvent {
	switch method {
	case "LayerTree.layerTreeDidChange":
		var raw struct {
			Layers []struct {
				LayerID                  string `json:"layerId"`
				ParentLayerID            string `json:"parentLayerId"`
				BackendNodeID            int32  `json:"backendNodeId"`
				OffsetX                  float64 `json:"offsetX"`
				OffsetY                  float64 `json:"offsetY"`
				Width                    float64 `json:"width"`
				Height                   float64 `json:"height"`
				PaintCount               int32  `json:"paintCount"`
				DrawsContent             bool   `json:"drawsContent"`
				Invisible                bool   `json:"invisible"`
				ScrollRects              []struct {
					Rect struct {
						X      float64 `json:"x"`
						Y      float64 `json:"y"`
						Width  float64 `json:"width"`
						Height float64 `json:"height"`
					} `json:"rect"`
					Type string `json:"type"`
				} `json:"scrollRects"`
				StickyPositionConstraint json.RawMessage `json:"stickyPositionConstraint"`
			} `json:"layers"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		var layers []*pb.Layer
		for _, l := range raw.Layers {
			layer := &pb.Layer{
				LayerId:       l.LayerID,
				ParentLayerId: l.ParentLayerID,
				BackendNodeId: l.BackendNodeID,
				OffsetX:       l.OffsetX,
				OffsetY:       l.OffsetY,
				Width:         l.Width,
				Height:        l.Height,
				PaintCount:    l.PaintCount,
				DrawsContent:  l.DrawsContent,
				Invisible:     l.Invisible,
			}
			for _, sr := range l.ScrollRects {
				layer.ScrollRects = append(layer.ScrollRects, &pb.ScrollRect{
					Rect: &pb.Rect{
						X: sr.Rect.X, Y: sr.Rect.Y,
						Width: sr.Rect.Width, Height: sr.Rect.Height,
					},
					Type: sr.Type,
				})
			}
			if l.StickyPositionConstraint != nil {
				layer.StickyPositionConstraint = string(l.StickyPositionConstraint)
			}
			layers = append(layers, layer)
		}
		return &pb.LayerTreeEvent{Event: &pb.LayerTreeEvent_LayerTreeDidChange{
			LayerTreeDidChange: &pb.LayerTreeDidChangeEvent{Layers: layers},
		}}
	case "LayerTree.layerPainted":
		var raw struct {
			LayerID string `json:"layerId"`
			Clip    struct {
				X      float64 `json:"x"`
				Y      float64 `json:"y"`
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
			} `json:"clip"`
		}
		if json.Unmarshal(params, &raw) != nil {
			return nil
		}
		return &pb.LayerTreeEvent{Event: &pb.LayerTreeEvent_LayerPainted{
			LayerPainted: &pb.LayerPaintedEvent{
				LayerId: raw.LayerID,
				Clip: &pb.Rect{
					X: raw.Clip.X, Y: raw.Clip.Y,
					Width: raw.Clip.Width, Height: raw.Clip.Height,
				},
			},
		}}
	}
	return nil
}
