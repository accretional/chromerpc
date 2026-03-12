// Package domdebugger implements the gRPC DOMDebuggerService by bridging to CDP.
package domdebugger

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/domdebugger"
)

type Server struct {
	pb.UnimplementedDOMDebuggerServiceServer
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


func domBreakpointTypeToString(t pb.DOMBreakpointType) string {
	switch t {
	case pb.DOMBreakpointType_DOM_BREAKPOINT_TYPE_SUBTREE_MODIFIED:
		return "subtree-modified"
	case pb.DOMBreakpointType_DOM_BREAKPOINT_TYPE_ATTRIBUTE_MODIFIED:
		return "attribute-modified"
	case pb.DOMBreakpointType_DOM_BREAKPOINT_TYPE_NODE_REMOVED:
		return "node-removed"
	default:
		return ""
	}
}

func (s *Server) GetEventListeners(ctx context.Context, req *pb.GetEventListenersRequest) (*pb.GetEventListenersResponse, error) {
	params := map[string]interface{}{
		"objectId": req.ObjectId,
	}
	if req.Depth != 0 {
		params["depth"] = req.Depth
	}
	if req.Pierce {
		params["pierce"] = req.Pierce
	}
	result, err := s.send(ctx, req.SessionId, "DOMDebugger.getEventListeners", params)
	if err != nil {
		return nil, fmt.Errorf("DOMDebugger.getEventListeners: %w", err)
	}
	var resp struct {
		Listeners []struct {
			Type            string          `json:"type"`
			UseCapture      bool            `json:"useCapture"`
			Passive         bool            `json:"passive"`
			Once            bool            `json:"once"`
			ScriptId        string          `json:"scriptId"`
			LineNumber      int32           `json:"lineNumber"`
			ColumnNumber    int32           `json:"columnNumber"`
			Handler         json.RawMessage `json:"handler"`
			OriginalHandler json.RawMessage `json:"originalHandler"`
			BackendNodeId   int32           `json:"backendNodeId"`
		} `json:"listeners"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOMDebugger.getEventListeners: unmarshal: %w", err)
	}
	listeners := make([]*pb.EventListener, len(resp.Listeners))
	for i, l := range resp.Listeners {
		listeners[i] = &pb.EventListener{
			Type:         l.Type,
			UseCapture:   l.UseCapture,
			Passive:      l.Passive,
			Once:         l.Once,
			ScriptId:     l.ScriptId,
			LineNumber:   l.LineNumber,
			ColumnNumber: l.ColumnNumber,
		}
		if l.Handler != nil {
			listeners[i].Handler = string(l.Handler)
		}
		if l.OriginalHandler != nil {
			listeners[i].OriginalHandler = string(l.OriginalHandler)
		}
		if l.BackendNodeId != 0 {
			listeners[i].BackendNodeId = l.BackendNodeId
		}
	}
	return &pb.GetEventListenersResponse{Listeners: listeners}, nil
}

func (s *Server) RemoveDOMBreakpoint(ctx context.Context, req *pb.RemoveDOMBreakpointRequest) (*pb.RemoveDOMBreakpointResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
		"type":   domBreakpointTypeToString(req.Type),
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.removeDOMBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.removeDOMBreakpoint: %w", err)
	}
	return &pb.RemoveDOMBreakpointResponse{}, nil
}

func (s *Server) RemoveEventListenerBreakpoint(ctx context.Context, req *pb.RemoveEventListenerBreakpointRequest) (*pb.RemoveEventListenerBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if req.TargetName != "" {
		params["targetName"] = req.TargetName
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.removeEventListenerBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.removeEventListenerBreakpoint: %w", err)
	}
	return &pb.RemoveEventListenerBreakpointResponse{}, nil
}

func (s *Server) RemoveInstrumentationBreakpoint(ctx context.Context, req *pb.RemoveInstrumentationBreakpointRequest) (*pb.RemoveInstrumentationBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.removeInstrumentationBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.removeInstrumentationBreakpoint: %w", err)
	}
	return &pb.RemoveInstrumentationBreakpointResponse{}, nil
}

func (s *Server) RemoveXHRBreakpoint(ctx context.Context, req *pb.RemoveXHRBreakpointRequest) (*pb.RemoveXHRBreakpointResponse, error) {
	params := map[string]interface{}{
		"url": req.Url,
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.removeXHRBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.removeXHRBreakpoint: %w", err)
	}
	return &pb.RemoveXHRBreakpointResponse{}, nil
}

func (s *Server) SetDOMBreakpoint(ctx context.Context, req *pb.SetDOMBreakpointRequest) (*pb.SetDOMBreakpointResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
		"type":   domBreakpointTypeToString(req.Type),
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.setDOMBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.setDOMBreakpoint: %w", err)
	}
	return &pb.SetDOMBreakpointResponse{}, nil
}

func (s *Server) SetEventListenerBreakpoint(ctx context.Context, req *pb.SetEventListenerBreakpointRequest) (*pb.SetEventListenerBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if req.TargetName != "" {
		params["targetName"] = req.TargetName
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.setEventListenerBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.setEventListenerBreakpoint: %w", err)
	}
	return &pb.SetEventListenerBreakpointResponse{}, nil
}

func (s *Server) SetInstrumentationBreakpoint(ctx context.Context, req *pb.SetInstrumentationBreakpointRequest) (*pb.SetInstrumentationBreakpointResponse, error) {
	params := map[string]interface{}{
		"eventName": req.EventName,
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.setInstrumentationBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.setInstrumentationBreakpoint: %w", err)
	}
	return &pb.SetInstrumentationBreakpointResponse{}, nil
}

func (s *Server) SetXHRBreakpoint(ctx context.Context, req *pb.SetXHRBreakpointRequest) (*pb.SetXHRBreakpointResponse, error) {
	params := map[string]interface{}{
		"url": req.Url,
	}
	if _, err := s.send(ctx, req.SessionId, "DOMDebugger.setXHRBreakpoint", params); err != nil {
		return nil, fmt.Errorf("DOMDebugger.setXHRBreakpoint: %w", err)
	}
	return &pb.SetXHRBreakpointResponse{}, nil
}
