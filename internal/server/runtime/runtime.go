// Package runtime implements the gRPC RuntimeService by bridging to CDP over WebSocket.
package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	pb "github.com/accretional/chromerpc/proto/cdp/runtime"
	"github.com/accretional/chromerpc/internal/cdpclient"
	"google.golang.org/grpc"
)

// Server implements the cdp.runtime.RuntimeService gRPC service.
type Server struct {
	pb.UnimplementedRuntimeServiceServer
	client *cdpclient.Client
}

// New creates a new Runtime gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) Enable(ctx context.Context, req *pb.EnableRequest) (*pb.EnableResponse, error) {
	if _, err := s.client.Send(ctx, "Runtime.enable", nil); err != nil {
		return nil, fmt.Errorf("Runtime.enable: %w", err)
	}
	return &pb.EnableResponse{}, nil
}

func (s *Server) Disable(ctx context.Context, req *pb.DisableRequest) (*pb.DisableResponse, error) {
	if _, err := s.client.Send(ctx, "Runtime.disable", nil); err != nil {
		return nil, fmt.Errorf("Runtime.disable: %w", err)
	}
	return &pb.DisableResponse{}, nil
}

func (s *Server) Evaluate(ctx context.Context, req *pb.EvaluateRequest) (*pb.EvaluateResponse, error) {
	params := map[string]interface{}{
		"expression": req.Expression,
	}
	if req.ObjectGroup != "" {
		params["objectGroup"] = req.ObjectGroup
	}
	if req.IncludeCommandLineApi {
		params["includeCommandLineAPI"] = true
	}
	if req.Silent {
		params["silent"] = true
	}
	if req.ContextId != 0 {
		params["contextId"] = req.ContextId
	}
	if req.ReturnByValue {
		params["returnByValue"] = true
	}
	if req.GeneratePreview {
		params["generatePreview"] = true
	}
	if req.UserGesture {
		params["userGesture"] = true
	}
	if req.AwaitPromise {
		params["awaitPromise"] = true
	}
	if req.ThrowOnSideEffect {
		params["throwOnSideEffect"] = true
	}
	if req.Timeout != 0 {
		params["timeout"] = req.Timeout
	}
	if req.DisableBreaks {
		params["disableBreaks"] = true
	}
	if req.ReplMode {
		params["replMode"] = true
	}
	if req.AllowUnsafeEvalBlockedByCsp {
		params["allowUnsafeEvalBlockedByCSP"] = true
	}
	if req.UniqueContextId != "" {
		params["uniqueContextId"] = req.UniqueContextId
	}
	if req.SerializationOptions != nil {
		so := map[string]interface{}{
			"serialization": req.SerializationOptions.Serialization,
		}
		if req.SerializationOptions.MaxDepth != 0 {
			so["maxDepth"] = req.SerializationOptions.MaxDepth
		}
		if req.SerializationOptions.AdditionalParameters != "" {
			var ap interface{}
			if err := json.Unmarshal([]byte(req.SerializationOptions.AdditionalParameters), &ap); err == nil {
				so["additionalParameters"] = ap
			}
		}
		params["serializationOptions"] = so
	}

	result, err := s.client.Send(ctx, "Runtime.evaluate", params)
	if err != nil {
		return nil, fmt.Errorf("Runtime.evaluate: %w", err)
	}

	var resp struct {
		Result           cdpRemoteObject   `json:"result"`
		ExceptionDetails *cdpExceptionDetails `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.evaluate: unmarshal: %w", err)
	}

	pbResp := &pb.EvaluateResponse{
		Result: resp.Result.toProto(),
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = resp.ExceptionDetails.toProto()
	}
	return pbResp, nil
}

func (s *Server) CallFunctionOn(ctx context.Context, req *pb.CallFunctionOnRequest) (*pb.CallFunctionOnResponse, error) {
	params := map[string]interface{}{
		"functionDeclaration": req.FunctionDeclaration,
	}
	if req.ObjectId != "" {
		params["objectId"] = req.ObjectId
	}
	if len(req.Arguments) > 0 {
		args := make([]map[string]interface{}, len(req.Arguments))
		for i, a := range req.Arguments {
			args[i] = callArgumentToMap(a)
		}
		params["arguments"] = args
	}
	if req.Silent {
		params["silent"] = true
	}
	if req.ReturnByValue {
		params["returnByValue"] = true
	}
	if req.GeneratePreview {
		params["generatePreview"] = true
	}
	if req.UserGesture {
		params["userGesture"] = true
	}
	if req.AwaitPromise {
		params["awaitPromise"] = true
	}
	if req.ExecutionContextId != 0 {
		params["executionContextId"] = req.ExecutionContextId
	}
	if req.ObjectGroup != "" {
		params["objectGroup"] = req.ObjectGroup
	}
	if req.ThrowOnSideEffect {
		params["throwOnSideEffect"] = true
	}
	if req.UniqueContextId != "" {
		params["uniqueContextId"] = req.UniqueContextId
	}
	if req.SerializationOptions != nil {
		so := map[string]interface{}{
			"serialization": req.SerializationOptions.Serialization,
		}
		if req.SerializationOptions.MaxDepth != 0 {
			so["maxDepth"] = req.SerializationOptions.MaxDepth
		}
		if req.SerializationOptions.AdditionalParameters != "" {
			var ap interface{}
			if err := json.Unmarshal([]byte(req.SerializationOptions.AdditionalParameters), &ap); err == nil {
				so["additionalParameters"] = ap
			}
		}
		params["serializationOptions"] = so
	}

	result, err := s.client.Send(ctx, "Runtime.callFunctionOn", params)
	if err != nil {
		return nil, fmt.Errorf("Runtime.callFunctionOn: %w", err)
	}

	var resp struct {
		Result           cdpRemoteObject      `json:"result"`
		ExceptionDetails *cdpExceptionDetails `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.callFunctionOn: unmarshal: %w", err)
	}

	pbResp := &pb.CallFunctionOnResponse{
		Result: resp.Result.toProto(),
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = resp.ExceptionDetails.toProto()
	}
	return pbResp, nil
}

func (s *Server) GetProperties(ctx context.Context, req *pb.GetPropertiesRequest) (*pb.GetPropertiesResponse, error) {
	params := map[string]interface{}{
		"objectId": req.ObjectId,
	}
	if req.OwnProperties {
		params["ownProperties"] = true
	}
	if req.AccessorPropertiesOnly {
		params["accessorPropertiesOnly"] = true
	}
	if req.GeneratePreview {
		params["generatePreview"] = true
	}
	if req.NonIndexedPropertiesOnly {
		params["nonIndexedPropertiesOnly"] = true
	}

	result, err := s.client.Send(ctx, "Runtime.getProperties", params)
	if err != nil {
		return nil, fmt.Errorf("Runtime.getProperties: %w", err)
	}

	var resp struct {
		Result             []cdpPropertyDescriptor         `json:"result"`
		InternalProperties []cdpInternalPropertyDescriptor `json:"internalProperties"`
		PrivateProperties  []cdpPrivatePropertyDescriptor  `json:"privateProperties"`
		ExceptionDetails   *cdpExceptionDetails            `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.getProperties: unmarshal: %w", err)
	}

	pbResult := make([]*pb.PropertyDescriptor, len(resp.Result))
	for i, p := range resp.Result {
		pbResult[i] = p.toProto()
	}

	pbInternal := make([]*pb.InternalPropertyDescriptor, len(resp.InternalProperties))
	for i, p := range resp.InternalProperties {
		pbInternal[i] = p.toProto()
	}

	pbPrivate := make([]*pb.PrivatePropertyDescriptor, len(resp.PrivateProperties))
	for i, p := range resp.PrivateProperties {
		pbPrivate[i] = p.toProto()
	}

	pbResp := &pb.GetPropertiesResponse{
		Result:             pbResult,
		InternalProperties: pbInternal,
		PrivateProperties:  pbPrivate,
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = resp.ExceptionDetails.toProto()
	}
	return pbResp, nil
}

func (s *Server) AwaitPromise(ctx context.Context, req *pb.AwaitPromiseRequest) (*pb.AwaitPromiseResponse, error) {
	params := map[string]interface{}{
		"promiseObjectId": req.PromiseObjectId,
	}
	if req.ReturnByValue {
		params["returnByValue"] = true
	}
	if req.GeneratePreview {
		params["generatePreview"] = true
	}

	result, err := s.client.Send(ctx, "Runtime.awaitPromise", params)
	if err != nil {
		return nil, fmt.Errorf("Runtime.awaitPromise: %w", err)
	}

	var resp struct {
		Result           cdpRemoteObject      `json:"result"`
		ExceptionDetails *cdpExceptionDetails `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.awaitPromise: unmarshal: %w", err)
	}

	pbResp := &pb.AwaitPromiseResponse{
		Result: resp.Result.toProto(),
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = resp.ExceptionDetails.toProto()
	}
	return pbResp, nil
}

func (s *Server) ReleaseObject(ctx context.Context, req *pb.ReleaseObjectRequest) (*pb.ReleaseObjectResponse, error) {
	params := map[string]interface{}{
		"objectId": req.ObjectId,
	}
	if _, err := s.client.Send(ctx, "Runtime.releaseObject", params); err != nil {
		return nil, fmt.Errorf("Runtime.releaseObject: %w", err)
	}
	return &pb.ReleaseObjectResponse{}, nil
}

func (s *Server) ReleaseObjectGroup(ctx context.Context, req *pb.ReleaseObjectGroupRequest) (*pb.ReleaseObjectGroupResponse, error) {
	params := map[string]interface{}{
		"objectGroup": req.ObjectGroup,
	}
	if _, err := s.client.Send(ctx, "Runtime.releaseObjectGroup", params); err != nil {
		return nil, fmt.Errorf("Runtime.releaseObjectGroup: %w", err)
	}
	return &pb.ReleaseObjectGroupResponse{}, nil
}

func (s *Server) CompileScript(ctx context.Context, req *pb.CompileScriptRequest) (*pb.CompileScriptResponse, error) {
	params := map[string]interface{}{
		"expression":    req.Expression,
		"sourceURL":     req.SourceUrl,
		"persistScript": req.PersistScript,
	}
	if req.ExecutionContextId != 0 {
		params["executionContextId"] = req.ExecutionContextId
	}

	result, err := s.client.Send(ctx, "Runtime.compileScript", params)
	if err != nil {
		return nil, fmt.Errorf("Runtime.compileScript: %w", err)
	}

	var resp struct {
		ScriptID         string               `json:"scriptId"`
		ExceptionDetails *cdpExceptionDetails `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.compileScript: unmarshal: %w", err)
	}

	pbResp := &pb.CompileScriptResponse{
		ScriptId: resp.ScriptID,
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = resp.ExceptionDetails.toProto()
	}
	return pbResp, nil
}

func (s *Server) RunScript(ctx context.Context, req *pb.RunScriptRequest) (*pb.RunScriptResponse, error) {
	params := map[string]interface{}{
		"scriptId": req.ScriptId,
	}
	if req.ExecutionContextId != 0 {
		params["executionContextId"] = req.ExecutionContextId
	}
	if req.ObjectGroup != "" {
		params["objectGroup"] = req.ObjectGroup
	}
	if req.Silent {
		params["silent"] = true
	}
	if req.IncludeCommandLineApi {
		params["includeCommandLineAPI"] = true
	}
	if req.ReturnByValue {
		params["returnByValue"] = true
	}
	if req.GeneratePreview {
		params["generatePreview"] = true
	}
	if req.AwaitPromise {
		params["awaitPromise"] = true
	}

	result, err := s.client.Send(ctx, "Runtime.runScript", params)
	if err != nil {
		return nil, fmt.Errorf("Runtime.runScript: %w", err)
	}

	var resp struct {
		Result           cdpRemoteObject      `json:"result"`
		ExceptionDetails *cdpExceptionDetails `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.runScript: unmarshal: %w", err)
	}

	pbResp := &pb.RunScriptResponse{
		Result: resp.Result.toProto(),
	}
	if resp.ExceptionDetails != nil {
		pbResp.ExceptionDetails = resp.ExceptionDetails.toProto()
	}
	return pbResp, nil
}

func (s *Server) RunIfWaitingForDebugger(ctx context.Context, req *pb.RunIfWaitingForDebuggerRequest) (*pb.RunIfWaitingForDebuggerResponse, error) {
	if _, err := s.client.Send(ctx, "Runtime.runIfWaitingForDebugger", nil); err != nil {
		return nil, fmt.Errorf("Runtime.runIfWaitingForDebugger: %w", err)
	}
	return &pb.RunIfWaitingForDebuggerResponse{}, nil
}

func (s *Server) DiscardConsoleEntries(ctx context.Context, req *pb.DiscardConsoleEntriesRequest) (*pb.DiscardConsoleEntriesResponse, error) {
	if _, err := s.client.Send(ctx, "Runtime.discardConsoleEntries", nil); err != nil {
		return nil, fmt.Errorf("Runtime.discardConsoleEntries: %w", err)
	}
	return &pb.DiscardConsoleEntriesResponse{}, nil
}

func (s *Server) AddBinding(ctx context.Context, req *pb.AddBindingRequest) (*pb.AddBindingResponse, error) {
	params := map[string]interface{}{
		"name": req.Name,
	}
	if req.ExecutionContextId != 0 {
		params["executionContextId"] = req.ExecutionContextId
	}
	if req.ExecutionContextName != "" {
		params["executionContextName"] = req.ExecutionContextName
	}

	if _, err := s.client.Send(ctx, "Runtime.addBinding", params); err != nil {
		return nil, fmt.Errorf("Runtime.addBinding: %w", err)
	}
	return &pb.AddBindingResponse{}, nil
}

func (s *Server) RemoveBinding(ctx context.Context, req *pb.RemoveBindingRequest) (*pb.RemoveBindingResponse, error) {
	params := map[string]interface{}{
		"name": req.Name,
	}
	if _, err := s.client.Send(ctx, "Runtime.removeBinding", params); err != nil {
		return nil, fmt.Errorf("Runtime.removeBinding: %w", err)
	}
	return &pb.RemoveBindingResponse{}, nil
}

func (s *Server) GlobalLexicalScopeNames(ctx context.Context, req *pb.GlobalLexicalScopeNamesRequest) (*pb.GlobalLexicalScopeNamesResponse, error) {
	params := map[string]interface{}{}
	if req.ExecutionContextId != 0 {
		params["executionContextId"] = req.ExecutionContextId
	}

	var result json.RawMessage
	var err error
	if len(params) > 0 {
		result, err = s.client.Send(ctx, "Runtime.globalLexicalScopeNames", params)
	} else {
		result, err = s.client.Send(ctx, "Runtime.globalLexicalScopeNames", nil)
	}
	if err != nil {
		return nil, fmt.Errorf("Runtime.globalLexicalScopeNames: %w", err)
	}

	var resp struct {
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.globalLexicalScopeNames: unmarshal: %w", err)
	}
	return &pb.GlobalLexicalScopeNamesResponse{Names: resp.Names}, nil
}

// SubscribeEvents streams CDP Runtime events to the gRPC client.
func (s *Server) SubscribeEvents(req *pb.SubscribeRuntimeEventsRequest, stream grpc.ServerStreamingServer[pb.RuntimeEvent]) error {
	eventCh := make(chan *pb.RuntimeEvent, 128)
	ctx := stream.Context()

	events := []string{
		"Runtime.consoleAPICalled",
		"Runtime.exceptionThrown",
		"Runtime.exceptionRevoked",
		"Runtime.executionContextCreated",
		"Runtime.executionContextDestroyed",
		"Runtime.executionContextsCleared",
		"Runtime.bindingCalled",
	}

	unregisters := make([]func(), len(events))
	for i, method := range events {
		method := method
		unregisters[i] = s.client.On(method, func(m string, params json.RawMessage, sessionID string) {
			if req.SessionId != "" && sessionID != req.SessionId {
				return
			}
			evt := convertRuntimeEvent(method, params)
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

type cdpRemoteObject struct {
	Type                string                   `json:"type"`
	Subtype             string                   `json:"subtype"`
	ClassName           string                   `json:"className"`
	Value               json.RawMessage          `json:"value"`
	UnserializableValue string                   `json:"unserializableValue"`
	Description         string                   `json:"description"`
	ObjectID            string                   `json:"objectId"`
	DeepSerializedValue *cdpDeepSerializedValue  `json:"deepSerializedValue"`
}

func (r *cdpRemoteObject) toProto() *pb.RemoteObject {
	obj := &pb.RemoteObject{
		Type:                r.Type,
		Subtype:             r.Subtype,
		ClassName:           r.ClassName,
		UnserializableValue: r.UnserializableValue,
		Description:         r.Description,
		ObjectId:            r.ObjectID,
	}
	if r.Value != nil {
		obj.Value = string(r.Value)
	}
	if r.DeepSerializedValue != nil {
		obj.DeepSerializedValue = r.DeepSerializedValue.toProto()
	}
	return obj
}

type cdpDeepSerializedValue struct {
	Type                     string          `json:"type"`
	Value                    json.RawMessage `json:"value"`
	ObjectID                 string          `json:"objectId"`
	WeakLocalObjectReference int32           `json:"weakLocalObjectReference"`
}

func (d *cdpDeepSerializedValue) toProto() *pb.DeepSerializedValue {
	v := &pb.DeepSerializedValue{
		Type:                     d.Type,
		ObjectId:                 d.ObjectID,
		WeakLocalObjectReference: d.WeakLocalObjectReference,
	}
	if d.Value != nil {
		v.Value = string(d.Value)
	}
	return v
}

type cdpExceptionDetails struct {
	ExceptionID        int32            `json:"exceptionId"`
	Text               string           `json:"text"`
	LineNumber         int32            `json:"lineNumber"`
	ColumnNumber       int32            `json:"columnNumber"`
	ScriptID           string           `json:"scriptId"`
	URL                string           `json:"url"`
	StackTrace         *cdpStackTrace   `json:"stackTrace"`
	Exception          *cdpRemoteObject `json:"exception"`
	ExecutionContextID int32            `json:"executionContextId"`
}

func (e *cdpExceptionDetails) toProto() *pb.ExceptionDetails {
	d := &pb.ExceptionDetails{
		ExceptionId:        e.ExceptionID,
		Text:               e.Text,
		LineNumber:         e.LineNumber,
		ColumnNumber:       e.ColumnNumber,
		ScriptId:           e.ScriptID,
		Url:                e.URL,
		ExecutionContextId: e.ExecutionContextID,
	}
	if e.StackTrace != nil {
		d.StackTrace = e.StackTrace.toProto()
	}
	if e.Exception != nil {
		d.Exception = e.Exception.toProto()
	}
	return d
}

type cdpStackTrace struct {
	Description string          `json:"description"`
	CallFrames  []cdpCallFrame  `json:"callFrames"`
	Parent      *cdpStackTrace  `json:"parent"`
}

func (st *cdpStackTrace) toProto() *pb.StackTrace {
	t := &pb.StackTrace{
		Description: st.Description,
	}
	t.CallFrames = make([]*pb.CallFrame, len(st.CallFrames))
	for i, cf := range st.CallFrames {
		t.CallFrames[i] = cf.toProto()
	}
	if st.Parent != nil {
		t.Parent = st.Parent.toProto()
	}
	return t
}

type cdpCallFrame struct {
	FunctionName string `json:"functionName"`
	ScriptID     string `json:"scriptId"`
	URL          string `json:"url"`
	LineNumber   int32  `json:"lineNumber"`
	ColumnNumber int32  `json:"columnNumber"`
}

func (cf *cdpCallFrame) toProto() *pb.CallFrame {
	return &pb.CallFrame{
		FunctionName: cf.FunctionName,
		ScriptId:     cf.ScriptID,
		Url:          cf.URL,
		LineNumber:   cf.LineNumber,
		ColumnNumber: cf.ColumnNumber,
	}
}

type cdpPropertyDescriptor struct {
	Name         string           `json:"name"`
	Value        *cdpRemoteObject `json:"value"`
	Writable     bool             `json:"writable"`
	Get          *cdpRemoteObject `json:"get"`
	Set          *cdpRemoteObject `json:"set"`
	Configurable bool             `json:"configurable"`
	Enumerable   bool             `json:"enumerable"`
	WasThrown    bool             `json:"wasThrown"`
	IsOwn        bool             `json:"isOwn"`
	Symbol       *cdpRemoteObject `json:"symbol"`
}

func (p *cdpPropertyDescriptor) toProto() *pb.PropertyDescriptor {
	d := &pb.PropertyDescriptor{
		Name:         p.Name,
		Writable:     p.Writable,
		Configurable: p.Configurable,
		Enumerable:   p.Enumerable,
		WasThrown:    p.WasThrown,
		IsOwn:        p.IsOwn,
	}
	if p.Value != nil {
		d.Value = p.Value.toProto()
	}
	if p.Get != nil {
		d.Get = p.Get.toProto()
	}
	if p.Set != nil {
		d.Set = p.Set.toProto()
	}
	if p.Symbol != nil {
		d.Symbol = p.Symbol.toProto()
	}
	return d
}

type cdpInternalPropertyDescriptor struct {
	Name  string           `json:"name"`
	Value *cdpRemoteObject `json:"value"`
}

func (p *cdpInternalPropertyDescriptor) toProto() *pb.InternalPropertyDescriptor {
	d := &pb.InternalPropertyDescriptor{
		Name: p.Name,
	}
	if p.Value != nil {
		d.Value = p.Value.toProto()
	}
	return d
}

type cdpPrivatePropertyDescriptor struct {
	Name  string           `json:"name"`
	Value *cdpRemoteObject `json:"value"`
	Get   *cdpRemoteObject `json:"get"`
	Set   *cdpRemoteObject `json:"set"`
}

func (p *cdpPrivatePropertyDescriptor) toProto() *pb.PrivatePropertyDescriptor {
	d := &pb.PrivatePropertyDescriptor{
		Name: p.Name,
	}
	if p.Value != nil {
		d.Value = p.Value.toProto()
	}
	if p.Get != nil {
		d.Get = p.Get.toProto()
	}
	if p.Set != nil {
		d.Set = p.Set.toProto()
	}
	return d
}

type cdpExecutionContextDescription struct {
	ID       int32           `json:"id"`
	Origin   string          `json:"origin"`
	Name     string          `json:"name"`
	UniqueID string          `json:"uniqueId"`
	AuxData  json.RawMessage `json:"auxData"`
}

func (c *cdpExecutionContextDescription) toProto() *pb.ExecutionContextDescription {
	d := &pb.ExecutionContextDescription{
		Id:       c.ID,
		Origin:   c.Origin,
		Name:     c.Name,
		UniqueId: c.UniqueID,
	}
	if c.AuxData != nil {
		d.AuxData = string(c.AuxData)
	}
	return d
}

// callArgumentToMap converts a proto CallArgument to a CDP JSON map.
func callArgumentToMap(a *pb.CallArgument) map[string]interface{} {
	m := map[string]interface{}{}
	if a.Value != "" {
		// Value is JSON-encoded; parse it back to an interface{} so it serializes correctly.
		var v interface{}
		if err := json.Unmarshal([]byte(a.Value), &v); err == nil {
			m["value"] = v
		} else {
			m["value"] = a.Value
		}
	}
	if a.UnserializableValue != "" {
		m["unserializableValue"] = a.UnserializableValue
	}
	if a.ObjectId != "" {
		m["objectId"] = a.ObjectId
	}
	return m
}

// --- event conversion ---

func convertRuntimeEvent(method string, params json.RawMessage) *pb.RuntimeEvent {
	switch method {
	case "Runtime.consoleAPICalled":
		var data struct {
			Type               string            `json:"type"`
			Args               []cdpRemoteObject `json:"args"`
			ExecutionContextID int32             `json:"executionContextId"`
			Timestamp          float64           `json:"timestamp"`
			StackTrace         *cdpStackTrace    `json:"stackTrace"`
			Context            string            `json:"context"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		args := make([]*pb.RemoteObject, len(data.Args))
		for i := range data.Args {
			args[i] = data.Args[i].toProto()
		}
		evt := &pb.ConsoleAPICalledEvent{
			Type:               data.Type,
			Args:               args,
			ExecutionContextId: data.ExecutionContextID,
			Timestamp:          data.Timestamp,
			Context:            data.Context,
		}
		if data.StackTrace != nil {
			evt.StackTrace = data.StackTrace.toProto()
		}
		return &pb.RuntimeEvent{
			Event: &pb.RuntimeEvent_ConsoleApiCalled{
				ConsoleApiCalled: evt,
			},
		}

	case "Runtime.exceptionThrown":
		var data struct {
			Timestamp        float64              `json:"timestamp"`
			ExceptionDetails cdpExceptionDetails  `json:"exceptionDetails"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.RuntimeEvent{
			Event: &pb.RuntimeEvent_ExceptionThrown{
				ExceptionThrown: &pb.ExceptionThrownEvent{
					Timestamp:        data.Timestamp,
					ExceptionDetails: data.ExceptionDetails.toProto(),
				},
			},
		}

	case "Runtime.exceptionRevoked":
		var data struct {
			Reason      string `json:"reason"`
			ExceptionID int32  `json:"exceptionId"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.RuntimeEvent{
			Event: &pb.RuntimeEvent_ExceptionRevoked{
				ExceptionRevoked: &pb.ExceptionRevokedEvent{
					Reason:      data.Reason,
					ExceptionId: data.ExceptionID,
				},
			},
		}

	case "Runtime.executionContextCreated":
		var data struct {
			Context cdpExecutionContextDescription `json:"context"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.RuntimeEvent{
			Event: &pb.RuntimeEvent_ExecutionContextCreated{
				ExecutionContextCreated: &pb.ExecutionContextCreatedEvent{
					Context: data.Context.toProto(),
				},
			},
		}

	case "Runtime.executionContextDestroyed":
		var data struct {
			ExecutionContextID       int32  `json:"executionContextId"`
			ExecutionContextUniqueID string `json:"executionContextUniqueId"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.RuntimeEvent{
			Event: &pb.RuntimeEvent_ExecutionContextDestroyed{
				ExecutionContextDestroyed: &pb.ExecutionContextDestroyedEvent{
					ExecutionContextId:       data.ExecutionContextID,
					ExecutionContextUniqueId: data.ExecutionContextUniqueID,
				},
			},
		}

	case "Runtime.executionContextsCleared":
		return &pb.RuntimeEvent{
			Event: &pb.RuntimeEvent_ExecutionContextsCleared{
				ExecutionContextsCleared: &pb.ExecutionContextsClearedEvent{},
			},
		}

	case "Runtime.bindingCalled":
		var data struct {
			Name               string `json:"name"`
			Payload            string `json:"payload"`
			ExecutionContextID int32  `json:"executionContextId"`
		}
		if json.Unmarshal(params, &data) != nil {
			return nil
		}
		return &pb.RuntimeEvent{
			Event: &pb.RuntimeEvent_BindingCalled{
				BindingCalled: &pb.BindingCalledEvent{
					Name:               data.Name,
					Payload:            data.Payload,
					ExecutionContextId: data.ExecutionContextID,
				},
			},
		}
	}
	return nil
}
