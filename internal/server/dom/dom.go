// Package dom implements the gRPC DOMService by bridging to CDP over WebSocket.
package dom

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/dom"
	"google.golang.org/grpc"
)

// Server implements the cdp.dom.DOMService gRPC service.
type Server struct {
	pb.UnimplementedDOMServiceServer
	client *cdpclient.Client
}

// New creates a new DOM gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

// cdpNode mirrors the CDP DOM.Node JSON structure.
type cdpNode struct {
	NodeID            int32      `json:"nodeId"`
	ParentID          int32      `json:"parentId"`
	BackendNodeID     int32      `json:"backendNodeId"`
	NodeType          int32      `json:"nodeType"`
	NodeName          string     `json:"nodeName"`
	LocalName         string     `json:"localName"`
	NodeValue         string     `json:"nodeValue"`
	ChildNodeCount    int32      `json:"childNodeCount"`
	Children          []*cdpNode `json:"children"`
	Attributes        []string   `json:"attributes"`
	DocumentURL       string     `json:"documentURL"`
	BaseURL           string     `json:"baseURL"`
	PublicID          string     `json:"publicId"`
	SystemID          string     `json:"systemId"`
	InternalSubset    string     `json:"internalSubset"`
	XMLVersion        string     `json:"xmlVersion"`
	Name              string     `json:"name"`
	Value             string     `json:"value"`
	PseudoType        string     `json:"pseudoType"`
	PseudoIdentifier  string     `json:"pseudoIdentifier"`
	ShadowRootType    string     `json:"shadowRootType"`
	FrameID           string     `json:"frameId"`
	ContentDocument   *cdpNode   `json:"contentDocument"`
	ShadowRoots       []*cdpNode `json:"shadowRoots"`
	TemplateContent   *cdpNode   `json:"templateContent"`
	PseudoElements    []*cdpNode `json:"pseudoElements"`
	CompatibilityMode string     `json:"compatibilityMode"`
	AssignedSlot      *cdpNode   `json:"assignedSlot"`
	IsScrollable      bool       `json:"isScrollable"`
}

func (n *cdpNode) toProto() *pb.Node {
	if n == nil {
		return nil
	}
	node := &pb.Node{
		NodeId:            n.NodeID,
		ParentId:          n.ParentID,
		BackendNodeId:     n.BackendNodeID,
		NodeType:          n.NodeType,
		NodeName:          n.NodeName,
		LocalName:         n.LocalName,
		NodeValue:         n.NodeValue,
		ChildNodeCount:    n.ChildNodeCount,
		Attributes:        n.Attributes,
		DocumentUrl:       n.DocumentURL,
		BaseUrl:           n.BaseURL,
		PublicId:          n.PublicID,
		SystemId:          n.SystemID,
		InternalSubset:    n.InternalSubset,
		XmlVersion:        n.XMLVersion,
		Name:              n.Name,
		Value:             n.Value,
		PseudoType:        n.PseudoType,
		PseudoIdentifier:  n.PseudoIdentifier,
		ShadowRootType:    n.ShadowRootType,
		FrameId:           n.FrameID,
		ContentDocument:   n.ContentDocument.toProto(),
		TemplateContent:   n.TemplateContent.toProto(),
		CompatibilityMode: n.CompatibilityMode,
		AssignedSlot:      n.AssignedSlot.toProto(),
		IsScrollable:      n.IsScrollable,
	}
	if len(n.Children) > 0 {
		node.Children = make([]*pb.Node, len(n.Children))
		for i, c := range n.Children {
			node.Children[i] = c.toProto()
		}
	}
	if len(n.ShadowRoots) > 0 {
		node.ShadowRoots = make([]*pb.Node, len(n.ShadowRoots))
		for i, sr := range n.ShadowRoots {
			node.ShadowRoots[i] = sr.toProto()
		}
	}
	if len(n.PseudoElements) > 0 {
		node.PseudoElements = make([]*pb.Node, len(n.PseudoElements))
		for i, pe := range n.PseudoElements {
			node.PseudoElements[i] = pe.toProto()
		}
	}
	return node
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	params := map[string]interface{}{}
	if req.IncludeWhitespace != "" {
		params["includeWhitespace"] = req.IncludeWhitespace
	}
	var err error
	if len(params) > 0 {
		_, err = s.client.Send(ctx, "DOM.enable", params)
	} else {
		_, err = s.client.Send(ctx, "DOM.enable", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("DOM.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "DOM.disable", nil); err != nil {
		return nil, fmt.Errorf("DOM.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) GetDocument(ctx context.Context, req *pb.GetDocumentRequest) (*pb.GetDocumentResponse, error) {
	params := map[string]interface{}{}
	if req.Depth != 0 {
		params["depth"] = req.Depth
	}
	if req.Pierce {
		params["pierce"] = true
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.client.Send(ctx, "DOM.getDocument", params)
	} else {
		result, err = s.client.Send(ctx, "DOM.getDocument", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("DOM.getDocument: %w", err)
	}
	var resp struct {
		Root cdpNode `json:"root"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.getDocument: unmarshal: %w", err)
	}
	return &pb.GetDocumentResponse{Root: resp.Root.toProto()}, nil
}

func (s *Server) QuerySelector(ctx context.Context, req *pb.QuerySelectorRequest) (*pb.QuerySelectorResponse, error) {
	params := map[string]interface{}{
		"nodeId":   req.NodeId,
		"selector": req.Selector,
	}
	result, err := s.client.Send(ctx, "DOM.querySelector", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.querySelector: %w", err)
	}
	var resp struct {
		NodeID int32 `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.querySelector: unmarshal: %w", err)
	}
	return &pb.QuerySelectorResponse{NodeId: resp.NodeID}, nil
}

func (s *Server) QuerySelectorAll(ctx context.Context, req *pb.QuerySelectorAllRequest) (*pb.QuerySelectorAllResponse, error) {
	params := map[string]interface{}{
		"nodeId":   req.NodeId,
		"selector": req.Selector,
	}
	result, err := s.client.Send(ctx, "DOM.querySelectorAll", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.querySelectorAll: %w", err)
	}
	var resp struct {
		NodeIDs []int32 `json:"nodeIds"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.querySelectorAll: unmarshal: %w", err)
	}
	return &pb.QuerySelectorAllResponse{NodeIds: resp.NodeIDs}, nil
}

func (s *Server) GetOuterHTML(ctx context.Context, req *pb.GetOuterHTMLRequest) (*pb.GetOuterHTMLResponse, error) {
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
	result, err := s.client.Send(ctx, "DOM.getOuterHTML", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.getOuterHTML: %w", err)
	}
	var resp struct {
		OuterHTML string `json:"outerHTML"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.getOuterHTML: unmarshal: %w", err)
	}
	return &pb.GetOuterHTMLResponse{OuterHtml: resp.OuterHTML}, nil
}

func (s *Server) SetOuterHTML(ctx context.Context, req *pb.SetOuterHTMLRequest) (*pb.SetOuterHTMLResponse, error) {
	params := map[string]interface{}{
		"nodeId":    req.NodeId,
		"outerHTML": req.OuterHtml,
	}
	if _, err := s.client.Send(ctx, "DOM.setOuterHTML", params); err != nil {
		return nil, fmt.Errorf("DOM.setOuterHTML: %w", err)
	}
	return &pb.SetOuterHTMLResponse{}, nil
}

func (s *Server) GetAttributes(ctx context.Context, req *pb.GetAttributesRequest) (*pb.GetAttributesResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	result, err := s.client.Send(ctx, "DOM.getAttributes", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.getAttributes: %w", err)
	}
	var resp struct {
		Attributes []string `json:"attributes"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.getAttributes: unmarshal: %w", err)
	}
	return &pb.GetAttributesResponse{Attributes: resp.Attributes}, nil
}

func (s *Server) SetAttributeValue(ctx context.Context, req *pb.SetAttributeValueRequest) (*pb.SetAttributeValueResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
		"name":   req.Name,
		"value":  req.Value,
	}
	if _, err := s.client.Send(ctx, "DOM.setAttributeValue", params); err != nil {
		return nil, fmt.Errorf("DOM.setAttributeValue: %w", err)
	}
	return &pb.SetAttributeValueResponse{}, nil
}

func (s *Server) SetAttributesAsText(ctx context.Context, req *pb.SetAttributesAsTextRequest) (*pb.SetAttributesAsTextResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
		"text":   req.Text,
	}
	if req.Name != "" {
		params["name"] = req.Name
	}
	if _, err := s.client.Send(ctx, "DOM.setAttributesAsText", params); err != nil {
		return nil, fmt.Errorf("DOM.setAttributesAsText: %w", err)
	}
	return &pb.SetAttributesAsTextResponse{}, nil
}

func (s *Server) RemoveAttribute(ctx context.Context, req *pb.RemoveAttributeRequest) (*pb.RemoveAttributeResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
		"name":   req.Name,
	}
	if _, err := s.client.Send(ctx, "DOM.removeAttribute", params); err != nil {
		return nil, fmt.Errorf("DOM.removeAttribute: %w", err)
	}
	return &pb.RemoveAttributeResponse{}, nil
}

func (s *Server) RemoveNode(ctx context.Context, req *pb.RemoveNodeRequest) (*pb.RemoveNodeResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	if _, err := s.client.Send(ctx, "DOM.removeNode", params); err != nil {
		return nil, fmt.Errorf("DOM.removeNode: %w", err)
	}
	return &pb.RemoveNodeResponse{}, nil
}

func (s *Server) SetNodeName(ctx context.Context, req *pb.SetNodeNameRequest) (*pb.SetNodeNameResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
		"name":   req.Name,
	}
	result, err := s.client.Send(ctx, "DOM.setNodeName", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.setNodeName: %w", err)
	}
	var resp struct {
		NodeID int32 `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.setNodeName: unmarshal: %w", err)
	}
	return &pb.SetNodeNameResponse{NodeId: resp.NodeID}, nil
}

func (s *Server) SetNodeValue(ctx context.Context, req *pb.SetNodeValueRequest) (*pb.SetNodeValueResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
		"value":  req.Value,
	}
	if _, err := s.client.Send(ctx, "DOM.setNodeValue", params); err != nil {
		return nil, fmt.Errorf("DOM.setNodeValue: %w", err)
	}
	return &pb.SetNodeValueResponse{}, nil
}

func (s *Server) MoveTo(ctx context.Context, req *pb.MoveToRequest) (*pb.MoveToResponse, error) {
	params := map[string]interface{}{
		"nodeId":       req.NodeId,
		"targetNodeId": req.TargetNodeId,
	}
	if req.InsertBeforeNodeId != 0 {
		params["insertBeforeNodeId"] = req.InsertBeforeNodeId
	}
	result, err := s.client.Send(ctx, "DOM.moveTo", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.moveTo: %w", err)
	}
	var resp struct {
		NodeID int32 `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.moveTo: unmarshal: %w", err)
	}
	return &pb.MoveToResponse{NodeId: resp.NodeID}, nil
}

func (s *Server) RequestChildNodes(ctx context.Context, req *pb.RequestChildNodesRequest) (*pb.RequestChildNodesResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	if req.Depth != 0 {
		params["depth"] = req.Depth
	}
	if req.Pierce {
		params["pierce"] = true
	}
	if _, err := s.client.Send(ctx, "DOM.requestChildNodes", params); err != nil {
		return nil, fmt.Errorf("DOM.requestChildNodes: %w", err)
	}
	return &pb.RequestChildNodesResponse{}, nil
}

func (s *Server) RequestNode(ctx context.Context, req *pb.RequestNodeRequest) (*pb.RequestNodeResponse, error) {
	params := map[string]interface{}{
		"objectId": req.ObjectId,
	}
	result, err := s.client.Send(ctx, "DOM.requestNode", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.requestNode: %w", err)
	}
	var resp struct {
		NodeID int32 `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.requestNode: unmarshal: %w", err)
	}
	return &pb.RequestNodeResponse{NodeId: resp.NodeID}, nil
}

func (s *Server) ResolveNode(ctx context.Context, req *pb.ResolveNodeRequest) (*pb.ResolveNodeResponse, error) {
	params := map[string]interface{}{}
	if req.NodeId != 0 {
		params["nodeId"] = req.NodeId
	}
	if req.BackendNodeId != 0 {
		params["backendNodeId"] = req.BackendNodeId
	}
	if req.ObjectGroup != "" {
		params["objectGroup"] = req.ObjectGroup
	}
	if req.ExecutionContextId != 0 {
		params["executionContextId"] = req.ExecutionContextId
	}
	result, err := s.client.Send(ctx, "DOM.resolveNode", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.resolveNode: %w", err)
	}
	var resp struct {
		Object json.RawMessage `json:"object"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.resolveNode: unmarshal: %w", err)
	}
	return &pb.ResolveNodeResponse{ObjectJson: string(resp.Object)}, nil
}

func (s *Server) DescribeNode(ctx context.Context, req *pb.DescribeNodeRequest) (*pb.DescribeNodeResponse, error) {
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
	if req.Depth != 0 {
		params["depth"] = req.Depth
	}
	if req.Pierce {
		params["pierce"] = true
	}
	result, err := s.client.Send(ctx, "DOM.describeNode", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.describeNode: %w", err)
	}
	var resp struct {
		Node cdpNode `json:"node"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.describeNode: unmarshal: %w", err)
	}
	return &pb.DescribeNodeResponse{Node: resp.Node.toProto()}, nil
}

func (s *Server) GetBoxModel(ctx context.Context, req *pb.GetBoxModelRequest) (*pb.GetBoxModelResponse, error) {
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
	result, err := s.client.Send(ctx, "DOM.getBoxModel", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.getBoxModel: %w", err)
	}
	var resp struct {
		Model struct {
			Content      []float64 `json:"content"`
			Padding      []float64 `json:"padding"`
			Border       []float64 `json:"border"`
			Margin       []float64 `json:"margin"`
			Width        int32     `json:"width"`
			Height       int32     `json:"height"`
			ShapeOutside *struct {
				Bounds      []float64       `json:"bounds"`
				Shape       json.RawMessage `json:"shape"`
				MarginShape json.RawMessage `json:"marginShape"`
			} `json:"shapeOutside"`
		} `json:"model"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.getBoxModel: unmarshal: %w", err)
	}
	model := &pb.BoxModel{
		Content: resp.Model.Content,
		Padding: resp.Model.Padding,
		Border:  resp.Model.Border,
		Margin:  resp.Model.Margin,
		Width:   resp.Model.Width,
		Height:  resp.Model.Height,
	}
	if resp.Model.ShapeOutside != nil {
		model.ShapeOutside = &pb.ShapeOutsideInfo{
			Bounds:      resp.Model.ShapeOutside.Bounds,
			Shape:       string(resp.Model.ShapeOutside.Shape),
			MarginShape: string(resp.Model.ShapeOutside.MarginShape),
		}
	}
	return &pb.GetBoxModelResponse{Model: model}, nil
}

func (s *Server) GetNodeForLocation(ctx context.Context, req *pb.GetNodeForLocationRequest) (*pb.GetNodeForLocationResponse, error) {
	params := map[string]interface{}{
		"x": req.X,
		"y": req.Y,
	}
	if req.IncludeUserAgentShadowDom {
		params["includeUserAgentShadowDOM"] = true
	}
	if req.IgnorePointerEventsNone {
		params["ignorePointerEventsNone"] = true
	}
	result, err := s.client.Send(ctx, "DOM.getNodeForLocation", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.getNodeForLocation: %w", err)
	}
	var resp struct {
		BackendNodeID int32  `json:"backendNodeId"`
		FrameID       string `json:"frameId"`
		NodeID        int32  `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.getNodeForLocation: unmarshal: %w", err)
	}
	return &pb.GetNodeForLocationResponse{
		BackendNodeId: resp.BackendNodeID,
		FrameId:       resp.FrameID,
		NodeId:        resp.NodeID,
	}, nil
}

func (s *Server) GetContainerForNode(ctx context.Context, req *pb.GetContainerForNodeRequest) (*pb.GetContainerForNodeResponse, error) {
	params := map[string]interface{}{
		"nodeId": req.NodeId,
	}
	if req.ContainerName != "" {
		params["containerName"] = req.ContainerName
	}
	if req.PhysicalAxes != "" {
		params["physicalAxes"] = req.PhysicalAxes
	}
	if req.LogicalAxes != "" {
		params["logicalAxes"] = req.LogicalAxes
	}
	if req.QueriesScrollState {
		params["queriesScrollState"] = true
	}
	result, err := s.client.Send(ctx, "DOM.getContainerForNode", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.getContainerForNode: %w", err)
	}
	var resp struct {
		NodeID int32 `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.getContainerForNode: unmarshal: %w", err)
	}
	return &pb.GetContainerForNodeResponse{NodeId: resp.NodeID}, nil
}

func (s *Server) HighlightNode(ctx context.Context, req *pb.HighlightNodeRequest) (*pb.HighlightNodeResponse, error) {
	if _, err := s.client.Send(ctx, "DOM.highlightNode", nil); err != nil {
		return nil, fmt.Errorf("DOM.highlightNode: %w", err)
	}
	return &pb.HighlightNodeResponse{}, nil
}

func (s *Server) HideHighlight(ctx context.Context, req *pb.HideHighlightRequest) (*pb.HideHighlightResponse, error) {
	if _, err := s.client.Send(ctx, "DOM.hideHighlight", nil); err != nil {
		return nil, fmt.Errorf("DOM.hideHighlight: %w", err)
	}
	return &pb.HideHighlightResponse{}, nil
}

func (s *Server) MarkUndoableState(ctx context.Context, req *pb.MarkUndoableStateRequest) (*pb.MarkUndoableStateResponse, error) {
	if _, err := s.client.Send(ctx, "DOM.markUndoableState", nil); err != nil {
		return nil, fmt.Errorf("DOM.markUndoableState: %w", err)
	}
	return &pb.MarkUndoableStateResponse{}, nil
}

func (s *Server) Focus(ctx context.Context, req *pb.FocusRequest) (*pb.FocusResponse, error) {
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
	var err error
	if len(params) > 0 {
		_, err = s.client.Send(ctx, "DOM.focus", params)
	} else {
		_, err = s.client.Send(ctx, "DOM.focus", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("DOM.focus: %w", err)
	}
	return &pb.FocusResponse{}, nil
}

func (s *Server) GetFileInfo(ctx context.Context, req *pb.GetFileInfoRequest) (*pb.GetFileInfoResponse, error) {
	params := map[string]interface{}{
		"objectId": req.ObjectId,
	}
	result, err := s.client.Send(ctx, "DOM.getFileInfo", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.getFileInfo: %w", err)
	}
	var resp struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.getFileInfo: unmarshal: %w", err)
	}
	return &pb.GetFileInfoResponse{Path: resp.Path}, nil
}

func (s *Server) CopyTo(ctx context.Context, req *pb.CopyToRequest) (*pb.CopyToResponse, error) {
	params := map[string]interface{}{
		"nodeId":       req.NodeId,
		"targetNodeId": req.TargetNodeId,
	}
	if req.InsertBeforeNodeId != 0 {
		params["insertBeforeNodeId"] = req.InsertBeforeNodeId
	}
	result, err := s.client.Send(ctx, "DOM.copyTo", params)
	if err != nil {
		return nil, fmt.Errorf("DOM.copyTo: %w", err)
	}
	var resp struct {
		NodeID int32 `json:"nodeId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("DOM.copyTo: unmarshal: %w", err)
	}
	return &pb.CopyToResponse{NodeId: resp.NodeID}, nil
}

// SubscribeEvents streams CDP DOM events to the gRPC client.
func (s *Server) SubscribeEvents(req *pb.SubscribeDOMEventsRequest, stream grpc.ServerStreamingServer[pb.DOMEvent]) error {
	eventCh := make(chan *pb.DOMEvent, 128)
	ctx := stream.Context()

	domEvents := []string{
		"DOM.attributeModified",
		"DOM.attributeRemoved",
		"DOM.childNodeInserted",
		"DOM.childNodeRemoved",
		"DOM.setChildNodes",
		"DOM.documentUpdated",
		"DOM.childNodeCountUpdated",
	}

	unregisters := make([]func(), 0, len(domEvents))
	for _, method := range domEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertDOMEvent(method, params)
			if evt != nil {
				select {
				case eventCh <- evt:
				default:
				}
			}
		})
		unregisters = append(unregisters, unreg)
	}
	defer func() {
		for _, unreg := range unregisters {
			unreg()
		}
	}()

	for {
		select {
		case evt := <-eventCh:
			if err := stream.Send(evt); err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		case <-s.client.Done():
			return fmt.Errorf("CDP connection closed")
		}
	}
}

func convertDOMEvent(method string, params json.RawMessage) *pb.DOMEvent {
	switch method {
	case "DOM.attributeModified":
		var d struct {
			NodeID int32  `json:"nodeId"`
			Name   string `json:"name"`
			Value  string `json:"value"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.DOMEvent{Event: &pb.DOMEvent_AttributeModified{
			AttributeModified: &pb.AttributeModifiedEvent{
				NodeId: d.NodeID,
				Name:   d.Name,
				Value:  d.Value,
			},
		}}

	case "DOM.attributeRemoved":
		var d struct {
			NodeID int32  `json:"nodeId"`
			Name   string `json:"name"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.DOMEvent{Event: &pb.DOMEvent_AttributeRemoved{
			AttributeRemoved: &pb.AttributeRemovedEvent{
				NodeId: d.NodeID,
				Name:   d.Name,
			},
		}}

	case "DOM.childNodeInserted":
		var d struct {
			ParentNodeID   int32   `json:"parentNodeId"`
			PreviousNodeID int32   `json:"previousNodeId"`
			Node           cdpNode `json:"node"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.DOMEvent{Event: &pb.DOMEvent_ChildNodeInserted{
			ChildNodeInserted: &pb.ChildNodeInsertedEvent{
				ParentNodeId:   d.ParentNodeID,
				PreviousNodeId: d.PreviousNodeID,
				Node:           d.Node.toProto(),
			},
		}}

	case "DOM.childNodeRemoved":
		var d struct {
			ParentNodeID int32 `json:"parentNodeId"`
			NodeID       int32 `json:"nodeId"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.DOMEvent{Event: &pb.DOMEvent_ChildNodeRemoved{
			ChildNodeRemoved: &pb.ChildNodeRemovedEvent{
				ParentNodeId: d.ParentNodeID,
				NodeId:       d.NodeID,
			},
		}}

	case "DOM.setChildNodes":
		var d struct {
			ParentID int32     `json:"parentId"`
			Nodes    []cdpNode `json:"nodes"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		nodes := make([]*pb.Node, len(d.Nodes))
		for i := range d.Nodes {
			nodes[i] = d.Nodes[i].toProto()
		}
		return &pb.DOMEvent{Event: &pb.DOMEvent_SetChildNodes{
			SetChildNodes: &pb.SetChildNodesEvent{
				ParentId: d.ParentID,
				Nodes:    nodes,
			},
		}}

	case "DOM.documentUpdated":
		return &pb.DOMEvent{Event: &pb.DOMEvent_DocumentUpdated{
			DocumentUpdated: &pb.DocumentUpdatedEvent{},
		}}

	case "DOM.childNodeCountUpdated":
		var d struct {
			NodeID         int32 `json:"nodeId"`
			ChildNodeCount int32 `json:"childNodeCount"`
		}
		if json.Unmarshal(params, &d) != nil {
			return nil
		}
		return &pb.DOMEvent{Event: &pb.DOMEvent_ChildNodeCountUpdated{
			ChildNodeCountUpdated: &pb.ChildNodeCountUpdatedEvent{
				NodeId:         d.NodeID,
				ChildNodeCount: d.ChildNodeCount,
			},
		}}

	default:
		return nil
	}
}
