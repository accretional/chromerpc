// Package page implements the gRPC PageService by bridging to CDP over WebSocket.
package page

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	pb "github.com/accretional/chromerpc/proto/cdp/page"
	"github.com/accretional/chromerpc/internal/cdpclient"
)

// Server implements the cdp.page.PageService gRPC service.
type Server struct {
	pb.UnimplementedPageServiceServer
	client *cdpclient.Client
}

// New creates a new Page gRPC server backed by the given CDP client.
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
	params := map[string]interface{}{}
	if req.EnableFileChooserOpenedEvent {
		params["enableFileChooserOpenedEvent"] = true
	}
	var err error
	if len(params) > 0 {
		_, err = s.send(ctx, req.SessionId, "Page.enable", params)
	} else {
		_, err = s.send(ctx, req.SessionId, "Page.enable", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Page.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.disable", nil); err != nil {
		return nil, fmt.Errorf("Page.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) Navigate(ctx context.Context, req *pb.NavigateRequest) (*pb.NavigateResponse, error) {
	params := map[string]interface{}{
		"url": req.Url,
	}
	if req.Referrer != "" {
		params["referrer"] = req.Referrer
	}
	if req.TransitionType != pb.TransitionType_TRANSITION_TYPE_UNSPECIFIED {
		params["transitionType"] = transitionTypeToString(req.TransitionType)
	}
	if req.FrameId != "" {
		params["frameId"] = req.FrameId
	}
	if req.ReferrerPolicy != pb.ReferrerPolicy_REFERRER_POLICY_UNSPECIFIED {
		params["referrerPolicy"] = referrerPolicyToString(req.ReferrerPolicy)
	}

	result, err := s.send(ctx, req.SessionId, "Page.navigate", params)
	if err != nil {
		return nil, fmt.Errorf("Page.navigate: %w", err)
	}

	var resp struct {
		FrameID   string `json:"frameId"`
		LoaderID  string `json:"loaderId"`
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.navigate: unmarshal: %w", err)
	}
	return &pb.NavigateResponse{
		FrameId:   resp.FrameID,
		LoaderId:  resp.LoaderID,
		ErrorText: resp.ErrorText,
	}, nil
}

func (s *Server) Reload(ctx context.Context, req *pb.ReloadRequest) (*pb.ReloadResponse, error) {
	params := map[string]interface{}{}
	if req.IgnoreCache {
		params["ignoreCache"] = true
	}
	if req.ScriptToEvaluateOnLoad != "" {
		params["scriptToEvaluateOnLoad"] = req.ScriptToEvaluateOnLoad
	}
	if req.LoaderId != "" {
		params["loaderId"] = req.LoaderId
	}
	if len(params) > 0 {
		_, err := s.send(ctx, req.SessionId, "Page.reload", params)
		if err != nil {
			return nil, fmt.Errorf("Page.reload: %w", err)
		}
	} else {
		_, err := s.send(ctx, req.SessionId, "Page.reload", nil)
		if err != nil {
			return nil, fmt.Errorf("Page.reload: %w", err)
		}
	}
	return &pb.ReloadResponse{}, nil
}

func (s *Server) StopLoading(ctx context.Context, req *pb.StopLoadingRequest) (*pb.StopLoadingResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.stopLoading", nil); err != nil {
		return nil, fmt.Errorf("Page.stopLoading: %w", err)
	}
	return &pb.StopLoadingResponse{}, nil
}

func (s *Server) GetNavigationHistory(ctx context.Context, req *pb.GetNavigationHistoryRequest) (*pb.GetNavigationHistoryResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Page.getNavigationHistory", nil)
	if err != nil {
		return nil, fmt.Errorf("Page.getNavigationHistory: %w", err)
	}
	var resp struct {
		CurrentIndex int32 `json:"currentIndex"`
		Entries      []struct {
			ID             int32  `json:"id"`
			URL            string `json:"url"`
			UserTypedURL   string `json:"userTypedURL"`
			Title          string `json:"title"`
			TransitionType string `json:"transitionType"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getNavigationHistory: unmarshal: %w", err)
	}
	entries := make([]*pb.NavigationEntry, len(resp.Entries))
	for i, e := range resp.Entries {
		entries[i] = &pb.NavigationEntry{
			Id:             e.ID,
			Url:            e.URL,
			UserTypedUrl:   e.UserTypedURL,
			Title:          e.Title,
			TransitionType: stringToTransitionType(e.TransitionType),
		}
	}
	return &pb.GetNavigationHistoryResponse{
		CurrentIndex: resp.CurrentIndex,
		Entries:      entries,
	}, nil
}

func (s *Server) NavigateToHistoryEntry(ctx context.Context, req *pb.NavigateToHistoryEntryRequest) (*pb.NavigateToHistoryEntryResponse, error) {
	params := map[string]interface{}{
		"entryId": req.EntryId,
	}
	if _, err := s.send(ctx, req.SessionId, "Page.navigateToHistoryEntry", params); err != nil {
		return nil, fmt.Errorf("Page.navigateToHistoryEntry: %w", err)
	}
	return &pb.NavigateToHistoryEntryResponse{}, nil
}

func (s *Server) ResetNavigationHistory(ctx context.Context, req *pb.ResetNavigationHistoryRequest) (*pb.ResetNavigationHistoryResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.resetNavigationHistory", nil); err != nil {
		return nil, fmt.Errorf("Page.resetNavigationHistory: %w", err)
	}
	return &pb.ResetNavigationHistoryResponse{}, nil
}

func (s *Server) CaptureScreenshot(ctx context.Context, req *pb.CaptureScreenshotRequest) (*pb.CaptureScreenshotResponse, error) {
	params := map[string]interface{}{}

	if req.Format != pb.ScreenshotFormat_SCREENSHOT_FORMAT_UNSPECIFIED {
		params["format"] = screenshotFormatToString(req.Format)
	}
	if req.Quality > 0 {
		params["quality"] = req.Quality
	}
	if req.Clip != nil {
		params["clip"] = map[string]interface{}{
			"x":      req.Clip.X,
			"y":      req.Clip.Y,
			"width":  req.Clip.Width,
			"height": req.Clip.Height,
			"scale":  req.Clip.Scale,
		}
	}
	if req.FromSurface {
		params["fromSurface"] = true
	}
	if req.CaptureBeyondViewport {
		params["captureBeyondViewport"] = true
	}
	if req.OptimizeForSpeed {
		params["optimizeForSpeed"] = true
	}

	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Page.captureScreenshot", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Page.captureScreenshot", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Page.captureScreenshot: %w", err)
	}

	var resp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.captureScreenshot: unmarshal: %w", err)
	}

	// CDP returns base64 string; decode to raw bytes for the protobuf bytes field.
	imageData, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("Page.captureScreenshot: decode base64: %w", err)
	}

	return &pb.CaptureScreenshotResponse{Data: imageData}, nil
}

func (s *Server) CaptureSnapshot(ctx context.Context, req *pb.CaptureSnapshotRequest) (*pb.CaptureSnapshotResponse, error) {
	params := map[string]interface{}{}
	if req.Format != pb.CaptureSnapshotFormat_CAPTURE_SNAPSHOT_FORMAT_UNSPECIFIED {
		params["format"] = "mhtml"
	}

	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Page.captureSnapshot", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Page.captureSnapshot", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Page.captureSnapshot: %w", err)
	}

	var resp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.captureSnapshot: unmarshal: %w", err)
	}
	return &pb.CaptureSnapshotResponse{Data: resp.Data}, nil
}

func (s *Server) PrintToPDF(ctx context.Context, req *pb.PrintToPDFRequest) (*pb.PrintToPDFResponse, error) {
	params := map[string]interface{}{}

	if req.Landscape {
		params["landscape"] = true
	}
	if req.DisplayHeaderFooter {
		params["displayHeaderFooter"] = true
	}
	if req.PrintBackground {
		params["printBackground"] = true
	}
	if req.Scale > 0 {
		params["scale"] = req.Scale
	}
	if req.PaperWidth > 0 {
		params["paperWidth"] = req.PaperWidth
	}
	if req.PaperHeight > 0 {
		params["paperHeight"] = req.PaperHeight
	}
	if req.MarginTop > 0 {
		params["marginTop"] = req.MarginTop
	}
	if req.MarginBottom > 0 {
		params["marginBottom"] = req.MarginBottom
	}
	if req.MarginLeft > 0 {
		params["marginLeft"] = req.MarginLeft
	}
	if req.MarginRight > 0 {
		params["marginRight"] = req.MarginRight
	}
	if req.PageRanges != "" {
		params["pageRanges"] = req.PageRanges
	}
	if req.HeaderTemplate != "" {
		params["headerTemplate"] = req.HeaderTemplate
	}
	if req.FooterTemplate != "" {
		params["footerTemplate"] = req.FooterTemplate
	}
	if req.PreferCssPageSize {
		params["preferCSSPageSize"] = true
	}
	if req.TransferMode != pb.PrintToPDFTransferMode_PRINT_TO_PDF_TRANSFER_MODE_UNSPECIFIED {
		switch req.TransferMode {
		case pb.PrintToPDFTransferMode_PRINT_TO_PDF_TRANSFER_MODE_RETURN_AS_BASE64:
			params["transferMode"] = "ReturnAsBase64"
		case pb.PrintToPDFTransferMode_PRINT_TO_PDF_TRANSFER_MODE_RETURN_AS_STREAM:
			params["transferMode"] = "ReturnAsStream"
		}
	}
	if req.GenerateTaggedPdf {
		params["generateTaggedPDF"] = true
	}
	if req.GenerateDocumentOutline {
		params["generateDocumentOutline"] = true
	}

	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Page.printToPDF", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Page.printToPDF", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Page.printToPDF: %w", err)
	}

	var resp struct {
		Data   string `json:"data"`
		Stream string `json:"stream"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.printToPDF: unmarshal: %w", err)
	}

	var pdfData []byte
	if resp.Data != "" {
		pdfData, err = base64.StdEncoding.DecodeString(resp.Data)
		if err != nil {
			return nil, fmt.Errorf("Page.printToPDF: decode base64: %w", err)
		}
	}

	return &pb.PrintToPDFResponse{
		Data:   pdfData,
		Stream: resp.Stream,
	}, nil
}

func (s *Server) GetFrameTree(ctx context.Context, req *pb.GetFrameTreeRequest) (*pb.GetFrameTreeResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Page.getFrameTree", nil)
	if err != nil {
		return nil, fmt.Errorf("Page.getFrameTree: %w", err)
	}
	var resp struct {
		FrameTree cdpFrameTree `json:"frameTree"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getFrameTree: unmarshal: %w", err)
	}
	return &pb.GetFrameTreeResponse{FrameTree: resp.FrameTree.toProto()}, nil
}

func (s *Server) GetLayoutMetrics(ctx context.Context, req *pb.GetLayoutMetricsRequest) (*pb.GetLayoutMetricsResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Page.getLayoutMetrics", nil)
	if err != nil {
		return nil, fmt.Errorf("Page.getLayoutMetrics: %w", err)
	}
	var resp struct {
		CSSLayoutViewport struct {
			PageX       int32 `json:"pageX"`
			PageY       int32 `json:"pageY"`
			ClientWidth int32 `json:"clientWidth"`
			ClientHeight int32 `json:"clientHeight"`
		} `json:"cssLayoutViewport"`
		CSSVisualViewport struct {
			OffsetX     float64 `json:"offsetX"`
			OffsetY     float64 `json:"offsetY"`
			PageX       float64 `json:"pageX"`
			PageY       float64 `json:"pageY"`
			ClientWidth float64 `json:"clientWidth"`
			ClientHeight float64 `json:"clientHeight"`
			Scale       float64 `json:"scale"`
			Zoom        float64 `json:"zoom"`
		} `json:"cssVisualViewport"`
		CSSContentSize struct {
			X      float64 `json:"x"`
			Y      float64 `json:"y"`
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"cssContentSize"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getLayoutMetrics: unmarshal: %w", err)
	}
	return &pb.GetLayoutMetricsResponse{
		CssLayoutViewport: &pb.LayoutViewport{
			PageX:        resp.CSSLayoutViewport.PageX,
			PageY:        resp.CSSLayoutViewport.PageY,
			ClientWidth:  resp.CSSLayoutViewport.ClientWidth,
			ClientHeight: resp.CSSLayoutViewport.ClientHeight,
		},
		CssVisualViewport: &pb.VisualViewport{
			OffsetX:      resp.CSSVisualViewport.OffsetX,
			OffsetY:      resp.CSSVisualViewport.OffsetY,
			PageX:        resp.CSSVisualViewport.PageX,
			PageY:        resp.CSSVisualViewport.PageY,
			ClientWidth:  resp.CSSVisualViewport.ClientWidth,
			ClientHeight: resp.CSSVisualViewport.ClientHeight,
			Scale:        resp.CSSVisualViewport.Scale,
			Zoom:         resp.CSSVisualViewport.Zoom,
		},
		CssContentSize: &pb.Rect{
			X:      resp.CSSContentSize.X,
			Y:      resp.CSSContentSize.Y,
			Width:  resp.CSSContentSize.Width,
			Height: resp.CSSContentSize.Height,
		},
	}, nil
}

func (s *Server) SetDocumentContent(ctx context.Context, req *pb.SetDocumentContentRequest) (*pb.SetDocumentContentResponse, error) {
	params := map[string]interface{}{
		"frameId": req.FrameId,
		"html":    req.Html,
	}
	if _, err := s.send(ctx, req.SessionId, "Page.setDocumentContent", params); err != nil {
		return nil, fmt.Errorf("Page.setDocumentContent: %w", err)
	}
	return &pb.SetDocumentContentResponse{}, nil
}

func (s *Server) BringToFront(ctx context.Context, req *pb.BringToFrontRequest) (*pb.BringToFrontResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.bringToFront", nil); err != nil {
		return nil, fmt.Errorf("Page.bringToFront: %w", err)
	}
	return &pb.BringToFrontResponse{}, nil
}

func (s *Server) Close(ctx context.Context, req *pb.CloseRequest) (*pb.CloseResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.close", nil); err != nil {
		return nil, fmt.Errorf("Page.close: %w", err)
	}
	return &pb.CloseResponse{}, nil
}

func (s *Server) AddScriptToEvaluateOnNewDocument(ctx context.Context, req *pb.AddScriptToEvaluateOnNewDocumentRequest) (*pb.AddScriptToEvaluateOnNewDocumentResponse, error) {
	params := map[string]interface{}{
		"source": req.Source,
	}
	if req.WorldName != "" {
		params["worldName"] = req.WorldName
	}
	if req.IncludeCommandLineApi {
		params["includeCommandLineAPI"] = true
	}
	if req.RunImmediately {
		params["runImmediately"] = true
	}
	result, err := s.send(ctx, req.SessionId, "Page.addScriptToEvaluateOnNewDocument", params)
	if err != nil {
		return nil, fmt.Errorf("Page.addScriptToEvaluateOnNewDocument: %w", err)
	}
	var resp struct {
		Identifier string `json:"identifier"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.addScriptToEvaluateOnNewDocument: unmarshal: %w", err)
	}
	return &pb.AddScriptToEvaluateOnNewDocumentResponse{Identifier: resp.Identifier}, nil
}

func (s *Server) RemoveScriptToEvaluateOnNewDocument(ctx context.Context, req *pb.RemoveScriptToEvaluateOnNewDocumentRequest) (*pb.RemoveScriptToEvaluateOnNewDocumentResponse, error) {
	params := map[string]interface{}{
		"identifier": req.Identifier,
	}
	if _, err := s.send(ctx, req.SessionId, "Page.removeScriptToEvaluateOnNewDocument", params); err != nil {
		return nil, fmt.Errorf("Page.removeScriptToEvaluateOnNewDocument: %w", err)
	}
	return &pb.RemoveScriptToEvaluateOnNewDocumentResponse{}, nil
}

func (s *Server) SetAdBlockingEnabled(ctx context.Context, req *pb.SetAdBlockingEnabledRequest) (*pb.SetAdBlockingEnabledResponse, error) {
	params := map[string]interface{}{"enabled": req.Enabled}
	if _, err := s.send(ctx, req.SessionId, "Page.setAdBlockingEnabled", params); err != nil {
		return nil, fmt.Errorf("Page.setAdBlockingEnabled: %w", err)
	}
	return &pb.SetAdBlockingEnabledResponse{}, nil
}

func (s *Server) SetBypassCSP(ctx context.Context, req *pb.SetBypassCSPRequest) (*pb.SetBypassCSPResponse, error) {
	params := map[string]interface{}{"enabled": req.Enabled}
	if _, err := s.send(ctx, req.SessionId, "Page.setBypassCSP", params); err != nil {
		return nil, fmt.Errorf("Page.setBypassCSP: %w", err)
	}
	return &pb.SetBypassCSPResponse{}, nil
}

func (s *Server) SetLifecycleEventsEnabled(ctx context.Context, req *pb.SetLifecycleEventsEnabledRequest) (*pb.SetLifecycleEventsEnabledResponse, error) {
	params := map[string]interface{}{"enabled": req.Enabled}
	if _, err := s.send(ctx, req.SessionId, "Page.setLifecycleEventsEnabled", params); err != nil {
		return nil, fmt.Errorf("Page.setLifecycleEventsEnabled: %w", err)
	}
	return &pb.SetLifecycleEventsEnabledResponse{}, nil
}

func (s *Server) HandleJavaScriptDialog(ctx context.Context, req *pb.HandleJavaScriptDialogRequest) (*pb.HandleJavaScriptDialogResponse, error) {
	params := map[string]interface{}{"accept": req.Accept}
	if req.PromptText != "" {
		params["promptText"] = req.PromptText
	}
	if _, err := s.send(ctx, req.SessionId, "Page.handleJavaScriptDialog", params); err != nil {
		return nil, fmt.Errorf("Page.handleJavaScriptDialog: %w", err)
	}
	return &pb.HandleJavaScriptDialogResponse{}, nil
}

func (s *Server) GetResourceContent(ctx context.Context, req *pb.GetResourceContentRequest) (*pb.GetResourceContentResponse, error) {
	params := map[string]interface{}{
		"frameId": req.FrameId,
		"url":     req.Url,
	}
	result, err := s.send(ctx, req.SessionId, "Page.getResourceContent", params)
	if err != nil {
		return nil, fmt.Errorf("Page.getResourceContent: %w", err)
	}
	var resp struct {
		Content       string `json:"content"`
		Base64Encoded bool   `json:"base64Encoded"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getResourceContent: unmarshal: %w", err)
	}
	return &pb.GetResourceContentResponse{
		Content:       resp.Content,
		Base64Encoded: resp.Base64Encoded,
	}, nil
}

func (s *Server) GetResourceTree(ctx context.Context, req *pb.GetResourceTreeRequest) (*pb.GetResourceTreeResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Page.getResourceTree", nil)
	if err != nil {
		return nil, fmt.Errorf("Page.getResourceTree: %w", err)
	}
	var resp struct {
		FrameTree cdpFrameResourceTree `json:"frameTree"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getResourceTree: unmarshal: %w", err)
	}
	return &pb.GetResourceTreeResponse{FrameTree: resp.FrameTree.toProto()}, nil
}

func (s *Server) SearchInResource(ctx context.Context, req *pb.SearchInResourceRequest) (*pb.SearchInResourceResponse, error) {
	params := map[string]interface{}{
		"frameId": req.FrameId,
		"url":     req.Url,
		"query":   req.Query,
	}
	if req.CaseSensitive {
		params["caseSensitive"] = true
	}
	if req.IsRegex {
		params["isRegex"] = true
	}
	result, err := s.send(ctx, req.SessionId, "Page.searchInResource", params)
	if err != nil {
		return nil, fmt.Errorf("Page.searchInResource: %w", err)
	}
	var resp struct {
		Result []struct {
			LineNumber  float64 `json:"lineNumber"`
			LineContent string  `json:"lineContent"`
		} `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.searchInResource: unmarshal: %w", err)
	}
	matches := make([]*pb.SearchMatch, len(resp.Result))
	for i, m := range resp.Result {
		matches[i] = &pb.SearchMatch{
			LineNumber:  m.LineNumber,
			LineContent: m.LineContent,
		}
	}
	return &pb.SearchInResourceResponse{Result: matches}, nil
}

func (s *Server) CreateIsolatedWorld(ctx context.Context, req *pb.CreateIsolatedWorldRequest) (*pb.CreateIsolatedWorldResponse, error) {
	params := map[string]interface{}{
		"frameId": req.FrameId,
	}
	if req.WorldName != "" {
		params["worldName"] = req.WorldName
	}
	if req.GrantUniversalAccess {
		params["grantUniveralAccess"] = true // note: CDP has the typo
	}
	result, err := s.send(ctx, req.SessionId, "Page.createIsolatedWorld", params)
	if err != nil {
		return nil, fmt.Errorf("Page.createIsolatedWorld: %w", err)
	}
	var resp struct {
		ExecutionContextID int32 `json:"executionContextId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.createIsolatedWorld: unmarshal: %w", err)
	}
	return &pb.CreateIsolatedWorldResponse{ExecutionContextId: resp.ExecutionContextID}, nil
}

func (s *Server) StartScreencast(ctx context.Context, req *pb.StartScreencastRequest) (*pb.StartScreencastResponse, error) {
	params := map[string]interface{}{}
	if req.Format != pb.ScreencastFormat_SCREENCAST_FORMAT_UNSPECIFIED {
		switch req.Format {
		case pb.ScreencastFormat_SCREENCAST_FORMAT_JPEG:
			params["format"] = "jpeg"
		case pb.ScreencastFormat_SCREENCAST_FORMAT_PNG:
			params["format"] = "png"
		}
	}
	if req.Quality > 0 {
		params["quality"] = req.Quality
	}
	if req.MaxWidth > 0 {
		params["maxWidth"] = req.MaxWidth
	}
	if req.MaxHeight > 0 {
		params["maxHeight"] = req.MaxHeight
	}
	if req.EveryNthFrame > 0 {
		params["everyNthFrame"] = req.EveryNthFrame
	}
	if len(params) > 0 {
		_, err := s.send(ctx, req.SessionId, "Page.startScreencast", params)
		if err != nil {
			return nil, fmt.Errorf("Page.startScreencast: %w", err)
		}
	} else {
		_, err := s.send(ctx, req.SessionId, "Page.startScreencast", nil)
		if err != nil {
			return nil, fmt.Errorf("Page.startScreencast: %w", err)
		}
	}
	return &pb.StartScreencastResponse{}, nil
}

func (s *Server) StopScreencast(ctx context.Context, req *pb.StopScreencastRequest) (*pb.StopScreencastResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.stopScreencast", nil); err != nil {
		return nil, fmt.Errorf("Page.stopScreencast: %w", err)
	}
	return &pb.StopScreencastResponse{}, nil
}

func (s *Server) ScreencastFrameAck(ctx context.Context, req *pb.ScreencastFrameAckRequest) (*pb.ScreencastFrameAckResponse, error) {
	params := map[string]interface{}{"sessionId": req.SessionId}
	if _, err := s.send(ctx, req.TargetSessionId, "Page.screencastFrameAck", params); err != nil {
		return nil, fmt.Errorf("Page.screencastFrameAck: %w", err)
	}
	return &pb.ScreencastFrameAckResponse{}, nil
}

func (s *Server) SetFontFamilies(ctx context.Context, req *pb.SetFontFamiliesRequest) (*pb.SetFontFamiliesResponse, error) {
	params := map[string]interface{}{}
	if req.FontFamilies != nil {
		ff := map[string]interface{}{}
		if req.FontFamilies.Standard != "" {
			ff["standard"] = req.FontFamilies.Standard
		}
		if req.FontFamilies.Fixed != "" {
			ff["fixed"] = req.FontFamilies.Fixed
		}
		if req.FontFamilies.Serif != "" {
			ff["serif"] = req.FontFamilies.Serif
		}
		if req.FontFamilies.SansSerif != "" {
			ff["sansSerif"] = req.FontFamilies.SansSerif
		}
		if req.FontFamilies.Cursive != "" {
			ff["cursive"] = req.FontFamilies.Cursive
		}
		if req.FontFamilies.Fantasy != "" {
			ff["fantasy"] = req.FontFamilies.Fantasy
		}
		if req.FontFamilies.Math != "" {
			ff["math"] = req.FontFamilies.Math
		}
		params["fontFamilies"] = ff
	}
	if _, err := s.send(ctx, req.SessionId, "Page.setFontFamilies", params); err != nil {
		return nil, fmt.Errorf("Page.setFontFamilies: %w", err)
	}
	return &pb.SetFontFamiliesResponse{}, nil
}

func (s *Server) SetFontSizes(ctx context.Context, req *pb.SetFontSizesRequest) (*pb.SetFontSizesResponse, error) {
	params := map[string]interface{}{}
	if req.FontSizes != nil {
		fs := map[string]interface{}{}
		if req.FontSizes.Standard > 0 {
			fs["standard"] = req.FontSizes.Standard
		}
		if req.FontSizes.Fixed > 0 {
			fs["fixed"] = req.FontSizes.Fixed
		}
		params["fontSizes"] = fs
	}
	if _, err := s.send(ctx, req.SessionId, "Page.setFontSizes", params); err != nil {
		return nil, fmt.Errorf("Page.setFontSizes: %w", err)
	}
	return &pb.SetFontSizesResponse{}, nil
}

func (s *Server) SetInterceptFileChooserDialog(ctx context.Context, req *pb.SetInterceptFileChooserDialogRequest) (*pb.SetInterceptFileChooserDialogResponse, error) {
	params := map[string]interface{}{"enabled": req.Enabled}
	if _, err := s.send(ctx, req.SessionId, "Page.setInterceptFileChooserDialog", params); err != nil {
		return nil, fmt.Errorf("Page.setInterceptFileChooserDialog: %w", err)
	}
	return &pb.SetInterceptFileChooserDialogResponse{}, nil
}

func (s *Server) Crash(ctx context.Context, req *pb.CrashRequest) (*pb.CrashResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.crash", nil); err != nil {
		return nil, fmt.Errorf("Page.crash: %w", err)
	}
	return &pb.CrashResponse{}, nil
}

func (s *Server) GenerateTestReport(ctx context.Context, req *pb.GenerateTestReportRequest) (*pb.GenerateTestReportResponse, error) {
	params := map[string]interface{}{"message": req.Message}
	if req.Group != "" {
		params["group"] = req.Group
	}
	if _, err := s.send(ctx, req.SessionId, "Page.generateTestReport", params); err != nil {
		return nil, fmt.Errorf("Page.generateTestReport: %w", err)
	}
	return &pb.GenerateTestReportResponse{}, nil
}

func (s *Server) WaitForDebugger(ctx context.Context, req *pb.WaitForDebuggerRequest) (*pb.WaitForDebuggerResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.waitForDebugger", nil); err != nil {
		return nil, fmt.Errorf("Page.waitForDebugger: %w", err)
	}
	return &pb.WaitForDebuggerResponse{}, nil
}

func (s *Server) SetWebLifecycleState(ctx context.Context, req *pb.SetWebLifecycleStateRequest) (*pb.SetWebLifecycleStateResponse, error) {
	stateStr := "active"
	if req.State == pb.WebLifecycleState_WEB_LIFECYCLE_STATE_FROZEN {
		stateStr = "frozen"
	}
	params := map[string]interface{}{"state": stateStr}
	if _, err := s.send(ctx, req.SessionId, "Page.setWebLifecycleState", params); err != nil {
		return nil, fmt.Errorf("Page.setWebLifecycleState: %w", err)
	}
	return &pb.SetWebLifecycleStateResponse{}, nil
}

func (s *Server) GetInstallabilityErrors(ctx context.Context, req *pb.GetInstallabilityErrorsRequest) (*pb.GetInstallabilityErrorsResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Page.getInstallabilityErrors", nil)
	if err != nil {
		return nil, fmt.Errorf("Page.getInstallabilityErrors: %w", err)
	}
	var resp struct {
		InstallabilityErrors []struct {
			ErrorID        string `json:"errorId"`
			ErrorArguments []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"errorArguments"`
		} `json:"installabilityErrors"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getInstallabilityErrors: unmarshal: %w", err)
	}
	errors := make([]*pb.InstallabilityError, len(resp.InstallabilityErrors))
	for i, e := range resp.InstallabilityErrors {
		args := make([]*pb.InstallabilityErrorArgument, len(e.ErrorArguments))
		for j, a := range e.ErrorArguments {
			args[j] = &pb.InstallabilityErrorArgument{Name: a.Name, Value: a.Value}
		}
		errors[i] = &pb.InstallabilityError{ErrorId: e.ErrorID, ErrorArguments: args}
	}
	return &pb.GetInstallabilityErrorsResponse{InstallabilityErrors: errors}, nil
}

func (s *Server) GetAppManifest(ctx context.Context, req *pb.GetAppManifestRequest) (*pb.GetAppManifestResponse, error) {
	params := map[string]interface{}{}
	if req.ManifestId != "" {
		params["manifestId"] = req.ManifestId
	}
	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.send(ctx, req.SessionId, "Page.getAppManifest", params)
	} else {
		result, err = s.send(ctx, req.SessionId, "Page.getAppManifest", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Page.getAppManifest: %w", err)
	}
	var resp struct {
		URL    string `json:"url"`
		Errors []struct {
			Message  string `json:"message"`
			Critical int32  `json:"critical"`
			Line     int32  `json:"line"`
			Column   int32  `json:"column"`
		} `json:"errors"`
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getAppManifest: unmarshal: %w", err)
	}
	errs := make([]*pb.AppManifestError, len(resp.Errors))
	for i, e := range resp.Errors {
		errs[i] = &pb.AppManifestError{
			Message:  e.Message,
			Critical: e.Critical,
			Line:     e.Line,
			Column:   e.Column,
		}
	}
	return &pb.GetAppManifestResponse{
		Url:    resp.URL,
		Errors: errs,
		Data:   resp.Data,
	}, nil
}

func (s *Server) GetAppId(ctx context.Context, req *pb.GetAppIdRequest) (*pb.GetAppIdResponse, error) {
	result, err := s.send(ctx, req.SessionId, "Page.getAppId", nil)
	if err != nil {
		return nil, fmt.Errorf("Page.getAppId: %w", err)
	}
	var resp struct {
		AppID         string `json:"appId"`
		RecommendedID string `json:"recommendedId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getAppId: unmarshal: %w", err)
	}
	return &pb.GetAppIdResponse{AppId: resp.AppID, RecommendedId: resp.RecommendedID}, nil
}

func (s *Server) GetPermissionsPolicyState(ctx context.Context, req *pb.GetPermissionsPolicyStateRequest) (*pb.GetPermissionsPolicyStateResponse, error) {
	params := map[string]interface{}{"frameId": req.FrameId}
	result, err := s.send(ctx, req.SessionId, "Page.getPermissionsPolicyState", params)
	if err != nil {
		return nil, fmt.Errorf("Page.getPermissionsPolicyState: %w", err)
	}
	var resp struct {
		States []struct {
			Feature string `json:"feature"`
			Allowed bool   `json:"allowed"`
			Locator *struct {
				FrameID     string `json:"frameId"`
				BlockReason string `json:"blockReason"`
			} `json:"locator"`
		} `json:"states"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getPermissionsPolicyState: unmarshal: %w", err)
	}
	states := make([]*pb.PermissionsPolicyFeatureState, len(resp.States))
	for i, st := range resp.States {
		s := &pb.PermissionsPolicyFeatureState{
			Feature: st.Feature,
			Allowed: st.Allowed,
		}
		if st.Locator != nil {
			s.Locator = &pb.PermissionsPolicyBlockLocator{
				FrameId:     st.Locator.FrameID,
				BlockReason: st.Locator.BlockReason,
			}
		}
		states[i] = s
	}
	return &pb.GetPermissionsPolicyStateResponse{States: states}, nil
}

func (s *Server) GetOriginTrials(ctx context.Context, req *pb.GetOriginTrialsRequest) (*pb.GetOriginTrialsResponse, error) {
	params := map[string]interface{}{"frameId": req.FrameId}
	result, err := s.send(ctx, req.SessionId, "Page.getOriginTrials", params)
	if err != nil {
		return nil, fmt.Errorf("Page.getOriginTrials: %w", err)
	}
	var resp struct {
		OriginTrials []struct {
			TrialName        string `json:"trialName"`
			Status           string `json:"status"`
			TokensWithStatus []struct {
				RawTokenText string `json:"rawTokenText"`
				ParsedToken  *struct {
					Origin           string  `json:"origin"`
					MatchSubDomains  bool    `json:"matchSubDomains"`
					TrialName        string  `json:"trialName"`
					ExpiryTime       float64 `json:"expiryTime"`
					IsThirdParty     bool    `json:"isThirdParty"`
					UsageRestriction string  `json:"usageRestriction"`
				} `json:"parsedToken"`
				Status string `json:"status"`
			} `json:"tokensWithStatus"`
		} `json:"originTrials"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.getOriginTrials: unmarshal: %w", err)
	}
	trials := make([]*pb.OriginTrial, len(resp.OriginTrials))
	for i, t := range resp.OriginTrials {
		tokens := make([]*pb.OriginTrialTokenWithStatus, len(t.TokensWithStatus))
		for j, tok := range t.TokensWithStatus {
			ts := &pb.OriginTrialTokenWithStatus{
				RawTokenText: tok.RawTokenText,
				Status:       tok.Status,
			}
			if tok.ParsedToken != nil {
				ts.ParsedToken = &pb.OriginTrialToken{
					Origin:           tok.ParsedToken.Origin,
					MatchSubDomains:  tok.ParsedToken.MatchSubDomains,
					TrialName:        tok.ParsedToken.TrialName,
					ExpiryTime:       tok.ParsedToken.ExpiryTime,
					IsThirdParty:     tok.ParsedToken.IsThirdParty,
					UsageRestriction: tok.ParsedToken.UsageRestriction,
				}
			}
			tokens[j] = ts
		}
		trials[i] = &pb.OriginTrial{
			TrialName:        t.TrialName,
			Status:           t.Status,
			TokensWithStatus: tokens,
		}
	}
	return &pb.GetOriginTrialsResponse{OriginTrials: trials}, nil
}

func (s *Server) ProduceCompilationCache(ctx context.Context, req *pb.ProduceCompilationCacheRequest) (*pb.ProduceCompilationCacheResponse, error) {
	scripts := make([]map[string]interface{}, len(req.Scripts))
	for i, sc := range req.Scripts {
		s := map[string]interface{}{"url": sc.Url}
		if sc.Eager {
			s["eager"] = true
		}
		scripts[i] = s
	}
	params := map[string]interface{}{"scripts": scripts}
	if _, err := s.send(ctx, req.SessionId, "Page.produceCompilationCache", params); err != nil {
		return nil, fmt.Errorf("Page.produceCompilationCache: %w", err)
	}
	return &pb.ProduceCompilationCacheResponse{}, nil
}

func (s *Server) AddCompilationCache(ctx context.Context, req *pb.AddCompilationCacheRequest) (*pb.AddCompilationCacheResponse, error) {
	params := map[string]interface{}{
		"url":  req.Url,
		"data": base64.StdEncoding.EncodeToString(req.Data),
	}
	if _, err := s.send(ctx, req.SessionId, "Page.addCompilationCache", params); err != nil {
		return nil, fmt.Errorf("Page.addCompilationCache: %w", err)
	}
	return &pb.AddCompilationCacheResponse{}, nil
}

func (s *Server) ClearCompilationCache(ctx context.Context, req *pb.ClearCompilationCacheRequest) (*pb.ClearCompilationCacheResponse, error) {
	if _, err := s.send(ctx, req.SessionId, "Page.clearCompilationCache", nil); err != nil {
		return nil, fmt.Errorf("Page.clearCompilationCache: %w", err)
	}
	return &pb.ClearCompilationCacheResponse{}, nil
}

func (s *Server) SetPrerenderingAllowed(ctx context.Context, req *pb.SetPrerenderingAllowedRequest) (*pb.SetPrerenderingAllowedResponse, error) {
	params := map[string]interface{}{"isAllowed": req.IsAllowed}
	if _, err := s.send(ctx, req.SessionId, "Page.setPrerenderingAllowed", params); err != nil {
		return nil, fmt.Errorf("Page.setPrerenderingAllowed: %w", err)
	}
	return &pb.SetPrerenderingAllowedResponse{}, nil
}

// SubscribeEvents streams CDP Page events to the gRPC client.
func (s *Server) SubscribeEvents(req *pb.SubscribePageEventsRequest, stream pb.PageService_SubscribeEventsServer) error {
	eventCh := make(chan *pb.PageEvent, 128)
	ctx := stream.Context()

	// Register a wildcard handler for all Page.* events.
	unregister := s.client.On("Page.", func(method string, params json.RawMessage, sessionID string) {
		// Only forward events for the requested session (or all if empty).
		if req.SessionId != "" && sessionID != req.SessionId {
			return
		}
		evt := convertPageEvent(method, params)
		if evt != nil {
			select {
			case eventCh <- evt:
			default:
			}
		}
	})

	// Also register specific events since the On() matching is exact.
	pageEvents := []string{
		"Page.domContentEventFired", "Page.loadEventFired",
		"Page.frameAttached", "Page.frameDetached", "Page.frameNavigated",
		"Page.javascriptDialogOpening", "Page.javascriptDialogClosed",
		"Page.lifecycleEvent", "Page.windowOpen",
		"Page.frameStartedLoading", "Page.frameStoppedLoading",
		"Page.frameStartedNavigating", "Page.frameRequestedNavigation",
		"Page.navigatedWithinDocument",
		"Page.interstitialShown", "Page.interstitialHidden",
		"Page.fileChooserOpened",
		"Page.screencastFrame", "Page.screencastVisibilityChanged",
		"Page.documentOpened", "Page.frameResized",
		"Page.backForwardCacheNotUsed", "Page.compilationCacheProduced",
	}
	unregisters := make([]func(), 0, len(pageEvents)+1)
	unregisters = append(unregisters, unregister)
	for _, method := range pageEvents {
		method := method
		unreg := s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertPageEvent(method, params)
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
