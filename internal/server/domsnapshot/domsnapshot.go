// Package domsnapshot implements the gRPC DOMSnapshotService by bridging to CDP.
package domsnapshot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/domsnapshot"
)

// Server implements the cdp.domsnapshot.DOMSnapshotService gRPC service.
type Server struct {
	pb.UnimplementedDOMSnapshotServiceServer
	client *cdpclient.Client
}

// New creates a new DOMSnapshot gRPC server backed by the given CDP client.
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
	if _, err := s.send(ctx, req.SessionId, "DOMSnapshot.enable", nil); err != nil {
		return nil, fmt.Errorf("DOMSnapshot.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "DOMSnapshot.disable", nil); err != nil {
		return nil, fmt.Errorf("DOMSnapshot.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) CaptureSnapshot(ctx context.Context, req *pb.CaptureSnapshotRequest) (*pb.CaptureSnapshotResponse, error) {
	params := map[string]interface{}{
		"computedStyles": req.ComputedStyles,
	}
	if req.IncludePaintOrder {
		params["includePaintOrder"] = true
	}
	if req.IncludeDomRects {
		params["includeDOMRects"] = true
	}
	if req.IncludeBlendedBackgroundColors {
		params["includeBlendedBackgroundColors"] = true
	}
	if req.IncludeTextColorOpacities {
		params["includeTextColorOpacities"] = true
	}

	var result json.RawMessage
	var err error
	result, err = s.send(ctx, req.SessionId, "DOMSnapshot.captureSnapshot", params)
	if err != nil {
		return nil, fmt.Errorf("DOMSnapshot.captureSnapshot: %w", err)
	}

	var raw struct {
		Documents json.RawMessage `json:"documents"`
		Strings   []string        `json:"strings"`
	}
	if err := json.Unmarshal(result, &raw); err != nil {
		return nil, fmt.Errorf("DOMSnapshot.captureSnapshot unmarshal: %w", err)
	}

	return &pb.CaptureSnapshotResponse{
		DocumentsJson: raw.Documents,
		Strings:       raw.Strings,
	}, nil
}
