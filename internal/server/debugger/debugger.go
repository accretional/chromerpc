// Package debugger implements the gRPC DebuggerService by bridging to CDP over WebSocket.
package debugger

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/accretional/chromerpc/proto/cdp/debugger"
	"github.com/accretional/chromerpc/internal/cdpclient"
	"google.golang.org/grpc"
)

// Server implements the cdp.debugger.DebuggerService gRPC service.
type Server struct {
	pb.UnimplementedDebuggerServiceServer
	client *cdpclient.Client
}

// New creates a new Debugger gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	result, err := s.client.Send(ctx, "Debugger.enable", nil)
	if err != nil {
		return nil, fmt.Errorf("Debugger.enable: %w", err)
	}
	var resp struct {
		DebuggerID string `json:"debuggerId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.enable: unmarshal: %w", err)
	}
	return &pb.EnableResponse{DebuggerId: resp.DebuggerID}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Debugger.disable", nil); err != nil {
		return nil, fmt.Errorf("Debugger.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) SetBreakpointByUrl(ctx context.Context, req *pb.SetBreakpointByUrlRequest) (*pb.SetBreakpointByUrlResponse, error) {
	params := map[string]interface{}{
		"lineNumber": req.LineNumber,
	}
	if req.Url != "" {
		params["url"] = req.Url
	}
	if req.UrlRegex != "" {
		params["urlRegex"] = req.UrlRegex
	}
	if req.ScriptHash != "" {
		params["scriptHash"] = req.ScriptHash
	}
	if req.ColumnNumber != 0 {
		params["columnNumber"] = req.ColumnNumber
	}
	if req.Condition != "" {
		params["condition"] = req.Condition
	}

	result, err := s.client.Send(ctx, "Debugger.setBreakpointByUrl", params)
	if err != nil {
		return nil, fmt.Errorf("Debugger.setBreakpointByUrl: %w", err)
	}

	var resp struct {
		BreakpointID string        `json:"breakpointId"`
		Locations    []cdpLocation `json:"locations"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.setBreakpointByUrl: unmarshal: %w", err)
	}

	locs := make([]*pb.Location, len(resp.Locations))
	for i, l := range resp.Locations {
		locs[i] = l.toProto()
	}
	return &pb.SetBreakpointByUrlResponse{
		BreakpointId: resp.BreakpointID,
		Locations:    locs,
	}, nil
}

func (s *Server) SetBreakpoint(ctx context.Context, req *pb.SetBreakpointRequest) (*pb.SetBreakpointResponse, error) {
	params := map[string]interface{}{}
	if req.Location != nil {
		params["location"] = locationToMap(req.Location)
	}
	if req.Condition != "" {
		params["condition"] = req.Condition
	}

	result, err := s.client.Send(ctx, "Debugger.setBreakpoint", params)
	if err != nil {
		return nil, fmt.Errorf("Debugger.setBreakpoint: %w", err)
	}

	var resp struct {
		BreakpointID   string      `json:"breakpointId"`
		ActualLocation cdpLocation `json:"actualLocation"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.setBreakpoint: unmarshal: %w", err)
	}

	return &pb.SetBreakpointResponse{
		BreakpointId:   resp.BreakpointID,
		ActualLocation: resp.ActualLocation.toProto(),
	}, nil
}

func (s *Server) RemoveBreakpoint(ctx context.Context, req *pb.RemoveBreakpointRequest) (*pb.RemoveBreakpointResponse, error) {
	params := map[string]interface{}{
		"breakpointId": req.BreakpointId,
	}
	if _, err := s.client.Send(ctx, "Debugger.removeBreakpoint", params); err != nil {
		return nil, fmt.Errorf("Debugger.removeBreakpoint: %w", err)
	}
	return &pb.RemoveBreakpointResponse{}, nil
}

func (s *Server) GetPossibleBreakpoints(ctx context.Context, req *pb.GetPossibleBreakpointsRequest) (*pb.GetPossibleBreakpointsResponse, error) {
	params := map[string]interface{}{}
	if req.Start != nil {
		params["start"] = locationToMap(req.Start)
	}
	if req.End != nil {
		params["end"] = locationToMap(req.End)
	}
	if req.RestrictToFunction {
		params["restrictToFunction"] = true
	}

	result, err := s.client.Send(ctx, "Debugger.getPossibleBreakpoints", params)
	if err != nil {
		return nil, fmt.Errorf("Debugger.getPossibleBreakpoints: %w", err)
	}

	var resp struct {
		Locations []cdpBreakLocation `json:"locations"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.getPossibleBreakpoints: unmarshal: %w", err)
	}

	locs := make([]*pb.BreakLocation, len(resp.Locations))
	for i, l := range resp.Locations {
		locs[i] = l.toProto()
	}
	return &pb.GetPossibleBreakpointsResponse{Locations: locs}, nil
}

func (s *Server) GetScriptSource(ctx context.Context, req *pb.GetScriptSourceRequest) (*pb.GetScriptSourceResponse, error) {
	params := map[string]interface{}{
		"scriptId": req.ScriptId,
	}

	result, err := s.client.Send(ctx, "Debugger.getScriptSource", params)
	if err != nil {
		return nil, fmt.Errorf("Debugger.getScriptSource: %w", err)
	}

	var resp struct {
		ScriptSource string `json:"scriptSource"`
		Bytecode     string `json:"bytecode"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.getScriptSource: unmarshal: %w", err)
	}
	return &pb.GetScriptSourceResponse{
		ScriptSource: resp.ScriptSource,
		Bytecode:     resp.Bytecode,
	}, nil
}

func (s *Server) SetScriptSource(ctx context.Context, req *pb.SetScriptSourceRequest) (*pb.SetScriptSourceResponse, error) {
	params := map[string]interface{}{
		"scriptId":     req.ScriptId,
		"scriptSource": req.ScriptSource,
	}
	if req.DryRun {
		params["dryRun"] = true
	}

	result, err := s.client.Send(ctx, "Debugger.setScriptSource", params)
	if err != nil {
		return nil, fmt.Errorf("Debugger.setScriptSource: %w", err)
	}

	var resp struct {
		CallFrames       []cdpCallFrame  `json:"callFrames"`
		StackChanged     bool            `json:"stackChanged"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.setScriptSource: unmarshal: %w", err)
	}

	frames := make([]*pb.CallFrame, len(resp.CallFrames))
	for i, cf := range resp.CallFrames {
		frames[i] = cf.toProto()
	}

	pbResp := &pb.SetScriptSourceResponse{
		CallFrames:   frames,
		StackChanged: resp.StackChanged,
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = string(resp.ExceptionDetails)
	}
	return pbResp, nil
}

func (s *Server) SearchInContent(ctx context.Context, req *pb.SearchInContentRequest) (*pb.SearchInContentResponse, error) {
	params := map[string]interface{}{
		"scriptId": req.ScriptId,
		"query":    req.Query,
	}
	if req.CaseSensitive {
		params["caseSensitive"] = true
	}
	if req.IsRegex {
		params["isRegex"] = true
	}

	result, err := s.client.Send(ctx, "Debugger.searchInContent", params)
	if err != nil {
		return nil, fmt.Errorf("Debugger.searchInContent: %w", err)
	}

	var resp struct {
		Result []cdpSearchMatch `json:"result"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.searchInContent: unmarshal: %w", err)
	}

	matches := make([]*pb.SearchMatch, len(resp.Result))
	for i, m := range resp.Result {
		matches[i] = m.toProto()
	}
	return &pb.SearchInContentResponse{Result: matches}, nil
}

func (s *Server) Pause(ctx context.Context, req *pb.PauseRequest) (*pb.PauseResponse, error) {
	if _, err := s.client.Send(ctx, "Debugger.pause", nil); err != nil {
		return nil, fmt.Errorf("Debugger.pause: %w", err)
	}
	return &pb.PauseResponse{}, nil
}

func (s *Server) Resume(ctx context.Context, req *pb.ResumeRequest) (*pb.ResumeResponse, error) {
	params := map[string]interface{}{}
	if req.TerminateOnResume {
		params["terminateOnResume"] = true
	}

	var err error
	if len(params) > 0 {
		_, err = s.client.Send(ctx, "Debugger.resume", params)
	} else {
		_, err = s.client.Send(ctx, "Debugger.resume", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Debugger.resume: %w", err)
	}
	return &pb.ResumeResponse{}, nil
}

func (s *Server) StepOver(ctx context.Context, req *pb.StepOverRequest) (*pb.StepOverResponse, error) {
	params := map[string]interface{}{}
	if len(req.SkipList) > 0 {
		skipList := make([]map[string]interface{}, len(req.SkipList))
		for i, loc := range req.SkipList {
			skipList[i] = locationToMap(loc)
		}
		params["skipList"] = skipList
	}

	var err error
	if len(params) > 0 {
		_, err = s.client.Send(ctx, "Debugger.stepOver", params)
	} else {
		_, err = s.client.Send(ctx, "Debugger.stepOver", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Debugger.stepOver: %w", err)
	}
	return &pb.StepOverResponse{}, nil
}

func (s *Server) StepInto(ctx context.Context, req *pb.StepIntoRequest) (*pb.StepIntoResponse, error) {
	params := map[string]interface{}{}
	if req.BreakOnAsyncCall {
		params["breakOnAsyncCall"] = true
	}
	if len(req.SkipList) > 0 {
		skipList := make([]map[string]interface{}, len(req.SkipList))
		for i, loc := range req.SkipList {
			skipList[i] = locationToMap(loc)
		}
		params["skipList"] = skipList
	}

	var err error
	if len(params) > 0 {
		_, err = s.client.Send(ctx, "Debugger.stepInto", params)
	} else {
		_, err = s.client.Send(ctx, "Debugger.stepInto", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Debugger.stepInto: %w", err)
	}
	return &pb.StepIntoResponse{}, nil
}

func (s *Server) StepOut(ctx context.Context, req *pb.StepOutRequest) (*pb.StepOutResponse, error) {
	if _, err := s.client.Send(ctx, "Debugger.stepOut", nil); err != nil {
		return nil, fmt.Errorf("Debugger.stepOut: %w", err)
	}
	return &pb.StepOutResponse{}, nil
}

func (s *Server) SetPauseOnExceptions(ctx context.Context, req *pb.SetPauseOnExceptionsRequest) (*pb.SetPauseOnExceptionsResponse, error) {
	params := map[string]interface{}{
		"state": req.State,
	}
	if _, err := s.client.Send(ctx, "Debugger.setPauseOnExceptions", params); err != nil {
		return nil, fmt.Errorf("Debugger.setPauseOnExceptions: %w", err)
	}
	return &pb.SetPauseOnExceptionsResponse{}, nil
}

func (s *Server) EvaluateOnCallFrame(ctx context.Context, req *pb.EvaluateOnCallFrameRequest) (*pb.EvaluateOnCallFrameResponse, error) {
	params := map[string]interface{}{
		"callFrameId": req.CallFrameId,
		"expression":  req.Expression,
	}
	if req.ObjectGroup != "" {
		params["objectGroup"] = req.ObjectGroup
	}
	if req.ReturnByValue {
		params["returnByValue"] = true
	}
	if req.GeneratePreview {
		params["generatePreview"] = true
	}
	if req.ThrowOnSideEffect {
		params["throwOnSideEffect"] = true
	}
	if req.Timeout != 0 {
		params["timeout"] = req.Timeout
	}

	result, err := s.client.Send(ctx, "Debugger.evaluateOnCallFrame", params)
	if err != nil {
		return nil, fmt.Errorf("Debugger.evaluateOnCallFrame: %w", err)
	}

	var resp struct {
		Result           json.RawMessage `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Debugger.evaluateOnCallFrame: unmarshal: %w", err)
	}

	pbResp := &pb.EvaluateOnCallFrameResponse{}
	if resp.Result != nil {
		pbResp.Result = string(resp.Result)
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = string(resp.ExceptionDetails)
	}
	return pbResp, nil
}

func (s *Server) SetBlackboxPatterns(ctx context.Context, req *pb.SetBlackboxPatternsRequest) (*pb.SetBlackboxPatternsResponse, error) {
	params := map[string]interface{}{
		"patterns": req.Patterns,
	}
	if _, err := s.client.Send(ctx, "Debugger.setBlackboxPatterns", params); err != nil {
		return nil, fmt.Errorf("Debugger.setBlackboxPatterns: %w", err)
	}
	return &pb.SetBlackboxPatternsResponse{}, nil
}

func (s *Server) SetAsyncCallStackDepth(ctx context.Context, req *pb.SetAsyncCallStackDepthRequest) (*pb.SetAsyncCallStackDepthResponse, error) {
	params := map[string]interface{}{
		"maxDepth": req.MaxDepth,
	}
	if _, err := s.client.Send(ctx, "Debugger.setAsyncCallStackDepth", params); err != nil {
		return nil, fmt.Errorf("Debugger.setAsyncCallStackDepth: %w", err)
	}
	return &pb.SetAsyncCallStackDepthResponse{}, nil
}

func (s *Server) SetBreakpointsActive(ctx context.Context, req *pb.SetBreakpointsActiveRequest) (*pb.SetBreakpointsActiveResponse, error) {
	params := map[string]interface{}{
		"active": req.Active,
	}
	if _, err := s.client.Send(ctx, "Debugger.setBreakpointsActive", params); err != nil {
		return nil, fmt.Errorf("Debugger.setBreakpointsActive: %w", err)
	}
	return &pb.SetBreakpointsActiveResponse{}, nil
}

// SubscribeEvents streams CDP Debugger events to the gRPC client.
func (s *Server) SubscribeEvents(req *pb.SubscribeDebuggerEventsRequest, stream grpc.ServerStreamingServer[pb.DebuggerEvent]) error {
	eventCh := make(chan *pb.DebuggerEvent, 128)
	ctx := stream.Context()

	events := []string{
		"Debugger.scriptParsed",
		"Debugger.scriptFailedToParse",
		"Debugger.paused",
		"Debugger.resumed",
	}

	unregisters := make([]func(), len(events))
	for i, method := range events {
		method := method
		unregisters[i] = s.client.On(method, func(_ string, params json.RawMessage, _ string) {
			if req.SessionId != "" {
				// session filtering would go here if needed
			}
			evt := convertDebuggerEvent(method, params)
			if evt != nil {
				select {
				case eventCh <- evt:
				default:
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

// --- internal CDP JSON structs and conversion helpers ---

type cdpLocation struct {
	ScriptID     string `json:"scriptId"`
	LineNumber   int32  `json:"lineNumber"`
	ColumnNumber int32  `json:"columnNumber"`
}

func (l *cdpLocation) toProto() *pb.Location {
	return &pb.Location{
		ScriptId:     l.ScriptID,
		LineNumber:   l.LineNumber,
		ColumnNumber: l.ColumnNumber,
	}
}

type cdpBreakLocation struct {
	ScriptID     string `json:"scriptId"`
	LineNumber   int32  `json:"lineNumber"`
	ColumnNumber int32  `json:"columnNumber"`
	Type         string `json:"type"`
}

func (l *cdpBreakLocation) toProto() *pb.BreakLocation {
	return &pb.BreakLocation{
		ScriptId:     l.ScriptID,
		LineNumber:   l.LineNumber,
		ColumnNumber: l.ColumnNumber,
		Type:         l.Type,
	}
}

type cdpScope struct {
	Type          string          `json:"type"`
	Object        json.RawMessage `json:"object"`
	Name          string          `json:"name"`
	StartLocation *cdpLocation    `json:"startLocation"`
	EndLocation   *cdpLocation    `json:"endLocation"`
}

func (sc *cdpScope) toProto() *pb.Scope {
	s := &pb.Scope{
		Type: sc.Type,
		Name: sc.Name,
	}
	if sc.Object != nil {
		s.Object = string(sc.Object)
	}
	if sc.StartLocation != nil {
		s.StartLocation = sc.StartLocation.toProto()
	}
	if sc.EndLocation != nil {
		s.EndLocation = sc.EndLocation.toProto()
	}
	return s
}

type cdpCallFrame struct {
	CallFrameID      string          `json:"callFrameId"`
	FunctionName     string          `json:"functionName"`
	FunctionLocation *cdpLocation    `json:"functionLocation"`
	Location         cdpLocation     `json:"location"`
	URL              string          `json:"url"`
	ScopeChain       []cdpScope      `json:"scopeChain"`
	This             json.RawMessage `json:"this"`
	ReturnValue      json.RawMessage `json:"returnValue"`
}

func (cf *cdpCallFrame) toProto() *pb.CallFrame {
	f := &pb.CallFrame{
		CallFrameId:  cf.CallFrameID,
		FunctionName: cf.FunctionName,
		Location:     cf.Location.toProto(),
		Url:          cf.URL,
	}
	if cf.FunctionLocation != nil {
		f.FunctionLocation = cf.FunctionLocation.toProto()
	}
	if len(cf.ScopeChain) > 0 {
		f.ScopeChain = make([]*pb.Scope, len(cf.ScopeChain))
		for i, sc := range cf.ScopeChain {
			f.ScopeChain[i] = sc.toProto()
		}
	}
	if cf.This != nil {
		f.This = string(cf.This)
	}
	if cf.ReturnValue != nil {
		f.ReturnValue = string(cf.ReturnValue)
	}
	return f
}

type cdpSearchMatch struct {
	LineNumber  float64 `json:"lineNumber"`
	LineContent string  `json:"lineContent"`
}

func (m *cdpSearchMatch) toProto() *pb.SearchMatch {
	return &pb.SearchMatch{
		LineNumber:  m.LineNumber,
		LineContent: m.LineContent,
	}
}

func locationToMap(loc *pb.Location) map[string]interface{} {
	m := map[string]interface{}{
		"scriptId":   loc.ScriptId,
		"lineNumber": loc.LineNumber,
	}
	if loc.ColumnNumber != 0 {
		m["columnNumber"] = loc.ColumnNumber
	}
	return m
}

// --- event conversion ---

func convertDebuggerEvent(method string, params json.RawMessage) *pb.DebuggerEvent {
	switch method {
	case "Debugger.scriptParsed":
		var data struct {
			ScriptID                 string          `json:"scriptId"`
			URL                      string          `json:"url"`
			StartLine                int32           `json:"startLine"`
			StartColumn              int32           `json:"startColumn"`
			EndLine                  int32           `json:"endLine"`
			EndColumn                int32           `json:"endColumn"`
			ExecutionContextID       int32           `json:"executionContextId"`
			Hash                     string          `json:"hash"`
			ExecutionContextAuxData  json.RawMessage `json:"executionContextAuxData"`
			IsLiveEdit               bool            `json:"isLiveEdit"`
			SourceMapURL             string          `json:"sourceMapURL"`
			HasSourceURL             bool            `json:"hasSourceURL"`
			IsModule                 bool            `json:"isModule"`
			Length                   int32           `json:"length"`
			StackTrace               json.RawMessage `json:"stackTrace"`
			CodeOffset               int32           `json:"codeOffset"`
			ScriptLanguage           string          `json:"scriptLanguage"`
			EmbedderName             string          `json:"embedderName"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		evt := &pb.ScriptParsedEvent{
			ScriptId:            data.ScriptID,
			Url:                 data.URL,
			StartLine:           data.StartLine,
			StartColumn:         data.StartColumn,
			EndLine:             data.EndLine,
			EndColumn:           data.EndColumn,
			ExecutionContextId:  data.ExecutionContextID,
			Hash:                data.Hash,
			IsLiveEdit:          data.IsLiveEdit,
			SourceMapUrl:        data.SourceMapURL,
			HasSourceUrl:        data.HasSourceURL,
			IsModule:            data.IsModule,
			Length:              data.Length,
			CodeOffset:          data.CodeOffset,
			ScriptLanguage:      data.ScriptLanguage,
			EmbedderName:        data.EmbedderName,
		}
		if data.ExecutionContextAuxData != nil {
			evt.ExecutionContextAuxData = string(data.ExecutionContextAuxData)
		}
		if data.StackTrace != nil {
			evt.StackTrace = string(data.StackTrace)
		}
		return &pb.DebuggerEvent{
			Event: &pb.DebuggerEvent_ScriptParsed{ScriptParsed: evt},
		}

	case "Debugger.scriptFailedToParse":
		var data struct {
			ScriptID                 string          `json:"scriptId"`
			URL                      string          `json:"url"`
			StartLine                int32           `json:"startLine"`
			StartColumn              int32           `json:"startColumn"`
			EndLine                  int32           `json:"endLine"`
			EndColumn                int32           `json:"endColumn"`
			ExecutionContextID       int32           `json:"executionContextId"`
			Hash                     string          `json:"hash"`
			ExecutionContextAuxData  json.RawMessage `json:"executionContextAuxData"`
			SourceMapURL             string          `json:"sourceMapURL"`
			HasSourceURL             bool            `json:"hasSourceURL"`
			IsModule                 bool            `json:"isModule"`
			Length                   int32           `json:"length"`
			StackTrace               json.RawMessage `json:"stackTrace"`
			CodeOffset               int32           `json:"codeOffset"`
			ScriptLanguage           string          `json:"scriptLanguage"`
			EmbedderName             string          `json:"embedderName"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		evt := &pb.ScriptFailedToParseEvent{
			ScriptId:            data.ScriptID,
			Url:                 data.URL,
			StartLine:           data.StartLine,
			StartColumn:         data.StartColumn,
			EndLine:             data.EndLine,
			EndColumn:           data.EndColumn,
			ExecutionContextId:  data.ExecutionContextID,
			Hash:                data.Hash,
			SourceMapUrl:        data.SourceMapURL,
			HasSourceUrl:        data.HasSourceURL,
			IsModule:            data.IsModule,
			Length:              data.Length,
			CodeOffset:          data.CodeOffset,
			ScriptLanguage:      data.ScriptLanguage,
			EmbedderName:        data.EmbedderName,
		}
		if data.ExecutionContextAuxData != nil {
			evt.ExecutionContextAuxData = string(data.ExecutionContextAuxData)
		}
		if data.StackTrace != nil {
			evt.StackTrace = string(data.StackTrace)
		}
		return &pb.DebuggerEvent{
			Event: &pb.DebuggerEvent_ScriptFailedToParse{ScriptFailedToParse: evt},
		}

	case "Debugger.paused":
		var data struct {
			CallFrames      []cdpCallFrame  `json:"callFrames"`
			Reason          string          `json:"reason"`
			Data            json.RawMessage `json:"data"`
			HitBreakpoints  []string        `json:"hitBreakpoints"`
			AsyncStackTrace json.RawMessage `json:"asyncStackTrace"`
			AsyncStackTraceID json.RawMessage `json:"asyncStackTraceId"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		frames := make([]*pb.CallFrame, len(data.CallFrames))
		for i, cf := range data.CallFrames {
			frames[i] = cf.toProto()
		}
		evt := &pb.PausedEvent{
			CallFrames:     frames,
			Reason:         data.Reason,
			HitBreakpoints: data.HitBreakpoints,
		}
		if data.Data != nil {
			evt.Data = string(data.Data)
		}
		if data.AsyncStackTrace != nil {
			evt.AsyncStackTrace = string(data.AsyncStackTrace)
		}
		if data.AsyncStackTraceID != nil {
			evt.AsyncStackTraceId = string(data.AsyncStackTraceID)
		}
		return &pb.DebuggerEvent{
			Event: &pb.DebuggerEvent_Paused{Paused: evt},
		}

	case "Debugger.resumed":
		return &pb.DebuggerEvent{
			Event: &pb.DebuggerEvent_Resumed{Resumed: &pb.ResumedEvent{}},
		}
	}
	return nil
}
