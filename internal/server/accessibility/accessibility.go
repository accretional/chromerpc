// Package accessibility implements the gRPC AccessibilityService by bridging to CDP.
package accessibility

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/accessibility"
)

type Server struct {
	pb.UnimplementedAccessibilityServiceServer
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


func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Accessibility.enable", nil); err != nil {
		return nil, fmt.Errorf("Accessibility.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Accessibility.disable", nil); err != nil {
		return nil, fmt.Errorf("Accessibility.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetPartialAXTree(ctx context.Context, req *pb.GetPartialAXTreeRequest) (*pb.GetPartialAXTreeResponse, error) {
	params := map[string]interface{}{}
	if req.NodeId != 0 {
		params["nodeId"] = req.NodeId
	}
	if req.BackendNodeId != 0 {
		params["backendNodeId"] = req.BackendNodeId
	}
	if req.ObjectId != "" {
		params["objectId"] = req.ObjectId
	}
	if req.FetchRelatives {
		params["fetchRelatives"] = true
	}
	if req.Depth != 0 {
		params["depth"] = req.Depth
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getPartialAXTree", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getPartialAXTree", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Accessibility.getPartialAXTree: %w", err)
	}
	var resp struct {
		Nodes []cdpAXNode `json:"nodes"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Accessibility.getPartialAXTree: unmarshal: %w", err)
	}
	nodes := make([]*pb.AXNode, len(resp.Nodes))
	for i, n := range resp.Nodes {
		nodes[i] = n.toProto()
	}
	return &pb.GetPartialAXTreeResponse{Nodes: nodes}, nil
}

func (s *Server) GetFullAXTree(ctx context.Context, req *pb.GetFullAXTreeRequest) (*pb.GetFullAXTreeResponse, error) {
	params := map[string]interface{}{}
	if req.Depth != 0 {
		params["depth"] = req.Depth
	}
	if req.FrameId != "" {
		params["frameId"] = req.FrameId
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getFullAXTree", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getFullAXTree", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Accessibility.getFullAXTree: %w", err)
	}
	var resp struct {
		Nodes []cdpAXNode `json:"nodes"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Accessibility.getFullAXTree: unmarshal: %w", err)
	}
	nodes := make([]*pb.AXNode, len(resp.Nodes))
	for i, n := range resp.Nodes {
		nodes[i] = n.toProto()
	}
	return &pb.GetFullAXTreeResponse{Nodes: nodes}, nil
}

func (s *Server) GetRootAXNode(ctx context.Context, req *pb.GetRootAXNodeRequest) (*pb.GetRootAXNodeResponse, error) {
	params := map[string]interface{}{}
	if req.FrameId != "" {
		params["frameId"] = req.FrameId
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getRootAXNode", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getRootAXNode", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Accessibility.getRootAXNode: %w", err)
	}
	var resp struct {
		Node cdpAXNode `json:"node"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Accessibility.getRootAXNode: unmarshal: %w", err)
	}
	return &pb.GetRootAXNodeResponse{Node: resp.Node.toProto()}, nil
}

func (s *Server) GetAXNodeAndAncestors(ctx context.Context, req *pb.GetAXNodeAndAncestorsRequest) (*pb.GetAXNodeAndAncestorsResponse, error) {
	params := map[string]interface{}{}
	if req.NodeId != 0 {
		params["nodeId"] = req.NodeId
	}
	if req.BackendNodeId != 0 {
		params["backendNodeId"] = req.BackendNodeId
	}
	if req.ObjectId != "" {
		params["objectId"] = req.ObjectId
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getAXNodeAndAncestors", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Accessibility.getAXNodeAndAncestors", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Accessibility.getAXNodeAndAncestors: %w", err)
	}
	var resp struct {
		Nodes []cdpAXNode `json:"nodes"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Accessibility.getAXNodeAndAncestors: unmarshal: %w", err)
	}
	nodes := make([]*pb.AXNode, len(resp.Nodes))
	for i, n := range resp.Nodes {
		nodes[i] = n.toProto()
	}
	return &pb.GetAXNodeAndAncestorsResponse{Nodes: nodes}, nil
}

func (s *Server) GetChildAXNodes(ctx context.Context, req *pb.GetChildAXNodesRequest) (*pb.GetChildAXNodesResponse, error) {
	params := map[string]interface{}{
		"id": req.Id,
	}
	if req.FrameId != "" {
		params["frameId"] = req.FrameId
	}
	result, err := s.send(ctx, req.SessionId, "Accessibility.getChildAXNodes", params)
	if err != nil {
		return nil, fmt.Errorf("Accessibility.getChildAXNodes: %w", err)
	}
	var resp struct {
		Nodes []cdpAXNode `json:"nodes"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Accessibility.getChildAXNodes: unmarshal: %w", err)
	}
	nodes := make([]*pb.AXNode, len(resp.Nodes))
	for i, n := range resp.Nodes {
		nodes[i] = n.toProto()
	}
	return &pb.GetChildAXNodesResponse{Nodes: nodes}, nil
}

func (s *Server) QueryAXTree(ctx context.Context, req *pb.QueryAXTreeRequest) (*pb.QueryAXTreeResponse, error) {
	params := map[string]interface{}{}
	if req.NodeId != 0 {
		params["nodeId"] = req.NodeId
	}
	if req.BackendNodeId != 0 {
		params["backendNodeId"] = req.BackendNodeId
	}
	if req.ObjectId != "" {
		params["objectId"] = req.ObjectId
	}
	if req.AccessibleName != "" {
		params["accessibleName"] = req.AccessibleName
	}
	if req.Role != "" {
		params["role"] = req.Role
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Accessibility.queryAXTree", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Accessibility.queryAXTree", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Accessibility.queryAXTree: %w", err)
	}
	var resp struct {
		Nodes []cdpAXNode `json:"nodes"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Accessibility.queryAXTree: unmarshal: %w", err)
	}
	nodes := make([]*pb.AXNode, len(resp.Nodes))
	for i, n := range resp.Nodes {
		nodes[i] = n.toProto()
	}
	return &pb.QueryAXTreeResponse{Nodes: nodes}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeEventsRequest, stream pb.AccessibilityService_SubscribeEventsServer) error {
	ch := make(chan *pb.AccessibilityEvent, 64)
	defer close(ch)

	unsubLoadComplete := s.client.On("Accessibility.loadComplete", func(method string, params json.RawMessage, sessionID string) {
		var raw struct {
			Root cdpAXNode `json:"root"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		ch <- &pb.AccessibilityEvent{
			Event: &pb.AccessibilityEvent_LoadComplete{
				LoadComplete: &pb.LoadCompleteEvent{
					Root: raw.Root.toProto(),
				},
			},
		}
	})
	defer unsubLoadComplete()

	unsubNodesUpdated := s.client.On("Accessibility.nodesUpdated", func(method string, params json.RawMessage, sessionID string) {
		var raw struct {
			Nodes []cdpAXNode `json:"nodes"`
		}
		if err := json.Unmarshal(params, &raw); err != nil {
			return
		}
		nodes := make([]*pb.AXNode, len(raw.Nodes))
		for i, n := range raw.Nodes {
			nodes[i] = n.toProto()
		}
		ch <- &pb.AccessibilityEvent{
			Event: &pb.AccessibilityEvent_NodesUpdated{
				NodesUpdated: &pb.NodesUpdatedEvent{
					Nodes: nodes,
				},
			},
		}
	})
	defer unsubNodesUpdated()

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

// --- internal helpers ---

type cdpAXNode struct {
	NodeID           string          `json:"nodeId"`
	Ignored          bool            `json:"ignored"`
	IgnoredReasons   []cdpAXProperty `json:"ignoredReasons"`
	Role             *cdpAXValue     `json:"role"`
	Name             *cdpAXValue     `json:"name"`
	Description      *cdpAXValue     `json:"description"`
	Value            *cdpAXValue     `json:"value"`
	Properties       []cdpAXProperty `json:"properties"`
	ChildIDs         []string        `json:"childIds"`
	BackendDOMNodeID int32           `json:"backendDOMNodeId"`
	FrameID          string          `json:"frameId"`
	ParentID         string          `json:"parentId"`
}

func (n *cdpAXNode) toProto() *pb.AXNode {
	node := &pb.AXNode{
		NodeId:           n.NodeID,
		Ignored:          n.Ignored,
		ChildIds:         n.ChildIDs,
		BackendDomNodeId: n.BackendDOMNodeID,
		FrameId:          n.FrameID,
		ParentId:         n.ParentID,
	}
	if n.Role != nil {
		node.Role = n.Role.toProto()
	}
	if n.Name != nil {
		node.Name = n.Name.toProto()
	}
	if n.Description != nil {
		node.Description = n.Description.toProto()
	}
	if n.Value != nil {
		node.Value = n.Value.toProto()
	}
	if len(n.IgnoredReasons) > 0 {
		node.IgnoredReasons = make([]*pb.AXProperty, len(n.IgnoredReasons))
		for i, p := range n.IgnoredReasons {
			node.IgnoredReasons[i] = p.toProto()
		}
	}
	if len(n.Properties) > 0 {
		node.Properties = make([]*pb.AXProperty, len(n.Properties))
		for i, p := range n.Properties {
			node.Properties[i] = p.toProto()
		}
	}
	return node
}

type cdpAXValue struct {
	Type         string             `json:"type"`
	Value        json.RawMessage    `json:"value"`
	RelatedNodes []cdpAXRelatedNode `json:"relatedNodes"`
	Sources      []cdpAXValueSource `json:"sources"`
}

func (v *cdpAXValue) toProto() *pb.AXValue {
	pv := &pb.AXValue{
		Type: v.Type,
	}
	if len(v.Value) > 0 {
		// CDP can return any type (bool, string, number); stringify it.
		var s string
		if err := json.Unmarshal(v.Value, &s); err == nil {
			pv.Value = s
		} else {
			pv.Value = string(v.Value)
		}
	}
	if len(v.RelatedNodes) > 0 {
		pv.RelatedNodes = make([]*pb.AXRelatedNode, len(v.RelatedNodes))
		for i, rn := range v.RelatedNodes {
			pv.RelatedNodes[i] = rn.toProto()
		}
	}
	if len(v.Sources) > 0 {
		pv.Sources = make([]*pb.AXValueSource, len(v.Sources))
		for i, src := range v.Sources {
			pv.Sources[i] = src.toProto()
		}
	}
	return pv
}

type cdpAXRelatedNode struct {
	BackendDOMNodeID int32  `json:"backendDOMNodeId"`
	Idref            string `json:"idref"`
	Text             string `json:"text"`
}

func (rn *cdpAXRelatedNode) toProto() *pb.AXRelatedNode {
	return &pb.AXRelatedNode{
		BackendDomNodeId: rn.BackendDOMNodeID,
		Idref:            rn.Idref,
		Text:             rn.Text,
	}
}

type cdpAXValueSource struct {
	Type              string      `json:"type"`
	Value             *cdpAXValue `json:"value"`
	Attribute         string      `json:"attribute"`
	AttributeValue    *cdpAXValue `json:"attributeValue"`
	Superseded        bool        `json:"superseded"`
	NativeSource      string      `json:"nativeSource"`
	NativeSourceValue *cdpAXValue `json:"nativeSourceValue"`
	Invalid           bool        `json:"invalid"`
	InvalidReason     string      `json:"invalidReason"`
}

func (src *cdpAXValueSource) toProto() *pb.AXValueSource {
	ps := &pb.AXValueSource{
		Type:          src.Type,
		Attribute:     src.Attribute,
		Superseded:    src.Superseded,
		NativeSource:  src.NativeSource,
		Invalid:       src.Invalid,
		InvalidReason: src.InvalidReason,
	}
	if src.Value != nil {
		ps.Value = src.Value.toProto()
	}
	if src.AttributeValue != nil {
		ps.AttributeValue = src.AttributeValue.toProto()
	}
	if src.NativeSourceValue != nil {
		ps.NativeSourceValue = src.NativeSourceValue.toProto()
	}
	return ps
}

type cdpAXProperty struct {
	Name  string      `json:"name"`
	Value *cdpAXValue `json:"value"`
}

func (p *cdpAXProperty) toProto() *pb.AXProperty {
	pp := &pb.AXProperty{
		Name: p.Name,
	}
	if p.Value != nil {
		pp.Value = p.Value.toProto()
	}
	return pp
}
