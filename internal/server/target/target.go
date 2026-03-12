// Package target implements the gRPC TargetService by bridging to CDP over WebSocket.
package target

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/anthropics/chromerpc/proto/cdp/target"
	"github.com/anthropics/chromerpc/internal/cdpclient"
)

// Server implements the cdp.target.TargetService gRPC service.
type Server struct {
	pb.UnimplementedTargetServiceServer
	client *cdpclient.Client
}

// New creates a new Target gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) ActivateTarget(ctx context.Context, req *pb.ActivateTargetRequest) (*pb.ActivateTargetResponse, error) {
	params := map[string]interface{}{
		"targetId": req.TargetId,
	}
	if _, err := s.client.Send(ctx, "Target.activateTarget", params); err != nil {
		return nil, fmt.Errorf("Target.activateTarget: %w", err)
	}
	return &pb.ActivateTargetResponse{}, nil
}

func (s *Server) AttachToTarget(ctx context.Context, req *pb.AttachToTargetRequest) (*pb.AttachToTargetResponse, error) {
	params := map[string]interface{}{
		"targetId": req.TargetId,
		"flatten":  req.Flatten,
	}
	result, err := s.client.Send(ctx, "Target.attachToTarget", params)
	if err != nil {
		return nil, fmt.Errorf("Target.attachToTarget: %w", err)
	}
	var resp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.attachToTarget: unmarshal: %w", err)
	}
	return &pb.AttachToTargetResponse{SessionId: resp.SessionID}, nil
}

func (s *Server) AttachToBrowserTarget(ctx context.Context, req *pb.AttachToBrowserTargetRequest) (*pb.AttachToBrowserTargetResponse, error) {
	result, err := s.client.Send(ctx, "Target.attachToBrowserTarget", nil)
	if err != nil {
		return nil, fmt.Errorf("Target.attachToBrowserTarget: %w", err)
	}
	var resp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.attachToBrowserTarget: unmarshal: %w", err)
	}
	return &pb.AttachToBrowserTargetResponse{SessionId: resp.SessionID}, nil
}

func (s *Server) CloseTarget(ctx context.Context, req *pb.CloseTargetRequest) (*pb.CloseTargetResponse, error) {
	params := map[string]interface{}{
		"targetId": req.TargetId,
	}
	result, err := s.client.Send(ctx, "Target.closeTarget", params)
	if err != nil {
		return nil, fmt.Errorf("Target.closeTarget: %w", err)
	}
	var resp struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.closeTarget: unmarshal: %w", err)
	}
	return &pb.CloseTargetResponse{Success: resp.Success}, nil
}

func (s *Server) CreateBrowserContext(ctx context.Context, req *pb.CreateBrowserContextRequest) (*pb.CreateBrowserContextResponse, error) {
	params := map[string]interface{}{}
	if req.DisposeOnDetach {
		params["disposeOnDetach"] = true
	}
	if req.ProxyServer != "" {
		params["proxyServer"] = req.ProxyServer
	}
	if req.ProxyBypassList != "" {
		params["proxyBypassList"] = req.ProxyBypassList
	}
	if len(req.OriginsWithUniversalNetworkAccess) > 0 {
		params["originsWithUniversalNetworkAccess"] = req.OriginsWithUniversalNetworkAccess
	}

	result, err := s.client.Send(ctx, "Target.createBrowserContext", params)
	if err != nil {
		return nil, fmt.Errorf("Target.createBrowserContext: %w", err)
	}
	var resp struct {
		BrowserContextID string `json:"browserContextId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.createBrowserContext: unmarshal: %w", err)
	}
	return &pb.CreateBrowserContextResponse{BrowserContextId: resp.BrowserContextID}, nil
}

func (s *Server) CreateTarget(ctx context.Context, req *pb.CreateTargetRequest) (*pb.CreateTargetResponse, error) {
	params := map[string]interface{}{
		"url": req.Url,
	}
	if req.Width > 0 {
		params["width"] = req.Width
	}
	if req.Height > 0 {
		params["height"] = req.Height
	}
	if req.BrowserContextId != "" {
		params["browserContextId"] = req.BrowserContextId
	}
	if req.EnableBeginFrameControl {
		params["enableBeginFrameControl"] = true
	}
	if req.NewWindow {
		params["newWindow"] = true
	}
	if req.Background {
		params["background"] = true
	}
	if req.ForTab {
		params["forTab"] = true
	}

	result, err := s.client.Send(ctx, "Target.createTarget", params)
	if err != nil {
		return nil, fmt.Errorf("Target.createTarget: %w", err)
	}
	var resp struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.createTarget: unmarshal: %w", err)
	}
	return &pb.CreateTargetResponse{TargetId: resp.TargetID}, nil
}

func (s *Server) DetachFromTarget(ctx context.Context, req *pb.DetachFromTargetRequest) (*pb.DetachFromTargetResponse, error) {
	params := map[string]interface{}{}
	if req.SessionId != "" {
		params["sessionId"] = req.SessionId
	}
	if req.TargetId != "" {
		params["targetId"] = req.TargetId
	}
	if _, err := s.client.Send(ctx, "Target.detachFromTarget", params); err != nil {
		return nil, fmt.Errorf("Target.detachFromTarget: %w", err)
	}
	return &pb.DetachFromTargetResponse{}, nil
}

func (s *Server) DisposeBrowserContext(ctx context.Context, req *pb.DisposeBrowserContextRequest) (*pb.DisposeBrowserContextResponse, error) {
	params := map[string]interface{}{
		"browserContextId": req.BrowserContextId,
	}
	if _, err := s.client.Send(ctx, "Target.disposeBrowserContext", params); err != nil {
		return nil, fmt.Errorf("Target.disposeBrowserContext: %w", err)
	}
	return &pb.DisposeBrowserContextResponse{}, nil
}

func (s *Server) GetBrowserContexts(ctx context.Context, req *pb.GetBrowserContextsRequest) (*pb.GetBrowserContextsResponse, error) {
	result, err := s.client.Send(ctx, "Target.getBrowserContexts", nil)
	if err != nil {
		return nil, fmt.Errorf("Target.getBrowserContexts: %w", err)
	}
	var resp struct {
		BrowserContextIDs []string `json:"browserContextIds"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.getBrowserContexts: unmarshal: %w", err)
	}
	return &pb.GetBrowserContextsResponse{BrowserContextIds: resp.BrowserContextIDs}, nil
}

func (s *Server) GetTargets(ctx context.Context, req *pb.GetTargetsRequest) (*pb.GetTargetsResponse, error) {
	params := map[string]interface{}{}
	if len(req.Filter) > 0 {
		filter := make([]map[string]interface{}, len(req.Filter))
		for i, f := range req.Filter {
			entry := map[string]interface{}{}
			if f.Exclude {
				entry["exclude"] = true
			}
			if f.Type != "" {
				entry["type"] = f.Type
			}
			filter[i] = entry
		}
		params["filter"] = filter
	}

	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.client.Send(ctx, "Target.getTargets", params)
	} else {
		result, err = s.client.Send(ctx, "Target.getTargets", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Target.getTargets: %w", err)
	}

	var resp struct {
		TargetInfos []cdpTargetInfo `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.getTargets: unmarshal: %w", err)
	}

	infos := make([]*pb.TargetInfo, len(resp.TargetInfos))
	for i, t := range resp.TargetInfos {
		infos[i] = t.toProto()
	}
	return &pb.GetTargetsResponse{TargetInfos: infos}, nil
}

func (s *Server) GetTargetInfo(ctx context.Context, req *pb.GetTargetInfoRequest) (*pb.GetTargetInfoResponse, error) {
	params := map[string]interface{}{
		"targetId": req.TargetId,
	}
	result, err := s.client.Send(ctx, "Target.getTargetInfo", params)
	if err != nil {
		return nil, fmt.Errorf("Target.getTargetInfo: %w", err)
	}
	var resp struct {
		TargetInfo cdpTargetInfo `json:"targetInfo"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Target.getTargetInfo: unmarshal: %w", err)
	}
	return &pb.GetTargetInfoResponse{TargetInfo: resp.TargetInfo.toProto()}, nil
}

func (s *Server) SetAutoAttach(ctx context.Context, req *pb.SetAutoAttachRequest) (*pb.SetAutoAttachResponse, error) {
	params := map[string]interface{}{
		"autoAttach":             req.AutoAttach,
		"waitForDebuggerOnStart": req.WaitForDebuggerOnStart,
	}
	if req.Flatten {
		params["flatten"] = true
	}
	if len(req.Filter) > 0 {
		filter := make([]map[string]interface{}, len(req.Filter))
		for i, f := range req.Filter {
			entry := map[string]interface{}{}
			if f.Exclude {
				entry["exclude"] = true
			}
			if f.Type != "" {
				entry["type"] = f.Type
			}
			filter[i] = entry
		}
		params["filter"] = filter
	}
	if _, err := s.client.Send(ctx, "Target.setAutoAttach", params); err != nil {
		return nil, fmt.Errorf("Target.setAutoAttach: %w", err)
	}
	return &pb.SetAutoAttachResponse{}, nil
}

func (s *Server) SetDiscoverTargets(ctx context.Context, req *pb.SetDiscoverTargetsRequest) (*pb.SetDiscoverTargetsResponse, error) {
	params := map[string]interface{}{
		"discover": req.Discover,
	}
	if len(req.Filter) > 0 {
		filter := make([]map[string]interface{}, len(req.Filter))
		for i, f := range req.Filter {
			entry := map[string]interface{}{}
			if f.Exclude {
				entry["exclude"] = true
			}
			if f.Type != "" {
				entry["type"] = f.Type
			}
			filter[i] = entry
		}
		params["filter"] = filter
	}
	if _, err := s.client.Send(ctx, "Target.setDiscoverTargets", params); err != nil {
		return nil, fmt.Errorf("Target.setDiscoverTargets: %w", err)
	}
	return &pb.SetDiscoverTargetsResponse{}, nil
}

func (s *Server) SubscribeEvents(req *pb.SubscribeTargetEventsRequest, stream pb.TargetService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.TargetEvent, 64)
	ctx := stream.Context()

	// Register handlers for all target events.
	events := []string{
		"Target.targetCreated",
		"Target.targetDestroyed",
		"Target.targetInfoChanged",
		"Target.targetCrashed",
		"Target.attachedToTarget",
		"Target.detachedFromTarget",
	}

	unregisters := make([]func(), len(events))
	for i, method := range events {
		method := method
		unregisters[i] = s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			evt := convertTargetEvent(method, params)
			if evt != nil {
				select {
				case eventCh <- evt:
				default:
					// Drop event if buffer full.
				}
			}
		})
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

// --- internal helpers ---

type cdpTargetInfo struct {
	TargetID         string `json:"targetId"`
	Type             string `json:"type"`
	Title            string `json:"title"`
	URL              string `json:"url"`
	Attached         bool   `json:"attached"`
	OpenerID         string `json:"openerId"`
	CanAccessOpener  bool   `json:"canAccessOpener"`
	BrowserContextID string `json:"browserContextId"`
	Subtype          string `json:"subtype"`
}

func (t *cdpTargetInfo) toProto() *pb.TargetInfo {
	return &pb.TargetInfo{
		TargetId:         t.TargetID,
		Type:             t.Type,
		Title:            t.Title,
		Url:              t.URL,
		Attached:         t.Attached,
		OpenerId:         t.OpenerID,
		CanAccessOpener:  t.CanAccessOpener,
		BrowserContextId: t.BrowserContextID,
		Subtype:          t.Subtype,
	}
}

func convertTargetEvent(method string, params json.RawMessage) *pb.TargetEvent {
	switch method {
	case "Target.targetCreated":
		var data struct {
			TargetInfo cdpTargetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.TargetEvent{
			Event: &pb.TargetEvent_TargetCreated{
				TargetCreated: &pb.TargetCreatedEvent{TargetInfo: data.TargetInfo.toProto()},
			},
		}
	case "Target.targetDestroyed":
		var data struct {
			TargetID string `json:"targetId"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.TargetEvent{
			Event: &pb.TargetEvent_TargetDestroyed{
				TargetDestroyed: &pb.TargetDestroyedEvent{TargetId: data.TargetID},
			},
		}
	case "Target.targetInfoChanged":
		var data struct {
			TargetInfo cdpTargetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.TargetEvent{
			Event: &pb.TargetEvent_TargetInfoChanged{
				TargetInfoChanged: &pb.TargetInfoChangedEvent{TargetInfo: data.TargetInfo.toProto()},
			},
		}
	case "Target.targetCrashed":
		var data struct {
			TargetID  string `json:"targetId"`
			Status    string `json:"status"`
			ErrorCode int32  `json:"errorCode"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.TargetEvent{
			Event: &pb.TargetEvent_TargetCrashed{
				TargetCrashed: &pb.TargetCrashedEvent{
					TargetId:  data.TargetID,
					Status:    data.Status,
					ErrorCode: data.ErrorCode,
				},
			},
		}
	case "Target.attachedToTarget":
		var data struct {
			SessionID          string        `json:"sessionId"`
			TargetInfo         cdpTargetInfo `json:"targetInfo"`
			WaitingForDebugger bool          `json:"waitingForDebugger"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.TargetEvent{
			Event: &pb.TargetEvent_AttachedToTarget{
				AttachedToTarget: &pb.AttachedToTargetEvent{
					SessionId:          data.SessionID,
					TargetInfo:         data.TargetInfo.toProto(),
					WaitingForDebugger: data.WaitingForDebugger,
				},
			},
		}
	case "Target.detachedFromTarget":
		var data struct {
			SessionID string `json:"sessionId"`
			TargetID  string `json:"targetId"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.TargetEvent{
			Event: &pb.TargetEvent_DetachedFromTarget{
				DetachedFromTarget: &pb.DetachedFromTargetEvent{
					SessionId: data.SessionID,
					TargetId:  data.TargetID,
				},
			},
		}
	}
	return nil
}
