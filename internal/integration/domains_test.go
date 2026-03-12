package integration

import (
	"context"
	"testing"
	"time"

	browserpb "github.com/accretional/chromerpc/proto/cdp/browser"
	dompb "github.com/accretional/chromerpc/proto/cdp/dom"
	emulationpb "github.com/accretional/chromerpc/proto/cdp/emulation"
	inputpb "github.com/accretional/chromerpc/proto/cdp/input"
	networkpb "github.com/accretional/chromerpc/proto/cdp/network"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	runtimepb "github.com/accretional/chromerpc/proto/cdp/runtime"
	targetpb "github.com/accretional/chromerpc/proto/cdp/target"
)

// =============================================================
// Runtime Domain Tests
// =============================================================

func TestRuntimeEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.runtimeClient.Enable(ctx, &runtimepb.EnableRequest{})
	if err != nil {
		t.Fatalf("Runtime.Enable: %v", err)
	}
}

func TestRuntimeEvaluate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.runtimeClient.Enable(ctx, &runtimepb.EnableRequest{})
	if err != nil {
		t.Fatalf("Runtime.Enable: %v", err)
	}

	resp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression: "1 + 2",
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if resp.Result.Type != "number" {
		t.Errorf("expected type 'number', got %q", resp.Result.Type)
	}
	t.Logf("Evaluate result: type=%s value=%s", resp.Result.Type, resp.Result.Value)
}

func TestRuntimeEvaluateString(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression:    "'hello' + ' world'",
		ReturnByValue: true,
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if resp.Result.Type != "string" {
		t.Errorf("expected type 'string', got %q", resp.Result.Type)
	}
	t.Logf("Evaluate string result: type=%s value=%s", resp.Result.Type, resp.Result.Value)
}

func TestRuntimeEvaluateObject(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression:    "({name: 'test', count: 42})",
		ReturnByValue: true,
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
	if resp.Result.Type != "object" {
		t.Errorf("expected type 'object', got %q", resp.Result.Type)
	}
	t.Logf("Evaluate object result: type=%s value=%s", resp.Result.Type, resp.Result.Value)
}

func TestRuntimeCallFunctionOn(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// First create an object to call function on.
	evalResp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression: "({x: 10, y: 20})",
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}
	objectId := evalResp.Result.ObjectId

	resp, err := env.runtimeClient.CallFunctionOn(ctx, &runtimepb.CallFunctionOnRequest{
		FunctionDeclaration: "function() { return this.x + this.y; }",
		ObjectId:            objectId,
		ReturnByValue:       true,
	})
	if err != nil {
		t.Fatalf("Runtime.CallFunctionOn: %v", err)
	}
	if resp.Result == nil {
		t.Fatal("expected non-nil result")
	}
	t.Logf("CallFunctionOn result: type=%s value=%s", resp.Result.Type, resp.Result.Value)
}

func TestRuntimeGetProperties(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	evalResp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression: "({a: 1, b: 'hello'})",
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}

	resp, err := env.runtimeClient.GetProperties(ctx, &runtimepb.GetPropertiesRequest{
		ObjectId:             evalResp.Result.ObjectId,
		OwnProperties:       true,
		GeneratePreview:      false,
		NonIndexedPropertiesOnly: false,
	})
	if err != nil {
		t.Fatalf("Runtime.GetProperties: %v", err)
	}
	if len(resp.Result) == 0 {
		t.Fatal("expected at least one property")
	}
	for _, prop := range resp.Result {
		if prop.Value != nil {
			t.Logf("Property: %s = %s (%s)", prop.Name, prop.Value.Value, prop.Value.Type)
		}
	}
}

func TestRuntimeCompileAndRunScript(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Get an execution context first.
	env.runtimeClient.Enable(ctx, &runtimepb.EnableRequest{})

	// Navigate to get a fresh context.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{Url: "about:blank"})
	time.Sleep(300 * time.Millisecond)

	compileResp, err := env.runtimeClient.CompileScript(ctx, &runtimepb.CompileScriptRequest{
		Expression:    "2 * 21",
		SourceUrl:     "test.js",
		PersistScript: true,
	})
	if err != nil {
		t.Fatalf("Runtime.CompileScript: %v", err)
	}
	if compileResp.ScriptId == "" {
		t.Fatal("expected non-empty script ID")
	}
	t.Logf("Compiled script: %s", compileResp.ScriptId)

	runResp, err := env.runtimeClient.RunScript(ctx, &runtimepb.RunScriptRequest{
		ScriptId:      compileResp.ScriptId,
		ReturnByValue: true,
	})
	if err != nil {
		t.Fatalf("Runtime.RunScript: %v", err)
	}
	if runResp.Result == nil {
		t.Fatal("expected non-nil result")
	}
	t.Logf("RunScript result: type=%s value=%s", runResp.Result.Type, runResp.Result.Value)
}

func TestRuntimeReleaseObject(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	evalResp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression: "({temporary: true})",
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}

	_, err = env.runtimeClient.ReleaseObject(ctx, &runtimepb.ReleaseObjectRequest{
		ObjectId: evalResp.Result.ObjectId,
	})
	if err != nil {
		t.Fatalf("Runtime.ReleaseObject: %v", err)
	}
}

func TestRuntimeDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.runtimeClient.Enable(ctx, &runtimepb.EnableRequest{})

	_, err := env.runtimeClient.Disable(ctx, &runtimepb.DisableRequest{})
	if err != nil {
		t.Fatalf("Runtime.Disable: %v", err)
	}
}

// =============================================================
// Network Domain Tests
// =============================================================

func TestNetworkEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.networkClient.Enable(ctx, &networkpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Network.Enable: %v", err)
	}
}

func TestNetworkSetCacheDisabled(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.networkClient.Enable(ctx, &networkpb.EnableRequest{})

	_, err := env.networkClient.SetCacheDisabled(ctx, &networkpb.SetCacheDisabledRequest{
		CacheDisabled: true,
	})
	if err != nil {
		t.Fatalf("Network.SetCacheDisabled: %v", err)
	}

	// Re-enable cache.
	_, err = env.networkClient.SetCacheDisabled(ctx, &networkpb.SetCacheDisabledRequest{
		CacheDisabled: false,
	})
	if err != nil {
		t.Fatalf("Network.SetCacheDisabled (re-enable): %v", err)
	}
}

func TestNetworkGetCookies(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.networkClient.Enable(ctx, &networkpb.EnableRequest{})

	resp, err := env.networkClient.GetCookies(ctx, &networkpb.GetCookiesRequest{})
	if err != nil {
		t.Fatalf("Network.GetCookies: %v", err)
	}
	t.Logf("Cookies: %d", len(resp.Cookies))
}

func TestNetworkSetExtraHTTPHeaders(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.networkClient.Enable(ctx, &networkpb.EnableRequest{})

	_, err := env.networkClient.SetExtraHTTPHeaders(ctx, &networkpb.SetExtraHTTPHeadersRequest{
		Headers: map[string]string{
			"X-Custom-Header": "chromerpc-test",
		},
	})
	if err != nil {
		t.Fatalf("Network.SetExtraHTTPHeaders: %v", err)
	}
}

func TestNetworkSetUserAgentOverride(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.networkClient.Enable(ctx, &networkpb.EnableRequest{})

	_, err := env.networkClient.SetUserAgentOverride(ctx, &networkpb.SetUserAgentOverrideRequest{
		UserAgent: "ChromeRPC-Test/1.0",
	})
	if err != nil {
		t.Fatalf("Network.SetUserAgentOverride: %v", err)
	}
}

func TestNetworkDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.networkClient.Enable(ctx, &networkpb.EnableRequest{})

	_, err := env.networkClient.Disable(ctx, &networkpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Network.Disable: %v", err)
	}
}

// =============================================================
// DOM Domain Tests
// =============================================================

func TestDOMGetDocument(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<html><body><h1 id='title'>DOM Test</h1><p class='content'>Hello</p></body></html>",
	})
	time.Sleep(500 * time.Millisecond)

	resp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{
		Depth: 3,
	})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}
	if resp.Root == nil {
		t.Fatal("expected non-nil root node")
	}
	t.Logf("Root node: id=%d name=%s children=%d", resp.Root.NodeId, resp.Root.NodeName, len(resp.Root.Children))
}

func TestDOMQuerySelector(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<html><body><h1 id='title'>QuerySelector Test</h1></body></html>",
	})
	time.Sleep(500 * time.Millisecond)

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	resp, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#title",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector: %v", err)
	}
	if resp.NodeId == 0 {
		t.Fatal("expected non-zero node ID")
	}
	t.Logf("Found node: %d", resp.NodeId)
}

func TestDOMQuerySelectorAll(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<html><body><p class='item'>One</p><p class='item'>Two</p><p class='item'>Three</p></body></html>",
	})
	time.Sleep(500 * time.Millisecond)

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	resp, err := env.domClient.QuerySelectorAll(ctx, &dompb.QuerySelectorAllRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: ".item",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelectorAll: %v", err)
	}
	if len(resp.NodeIds) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(resp.NodeIds))
	}
	t.Logf("Found %d nodes: %v", len(resp.NodeIds), resp.NodeIds)
}

func TestDOMGetOuterHTML(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<html><body><div id='target'>Hello World</div></body></html>",
	})
	time.Sleep(500 * time.Millisecond)

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	selResp, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#target",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector: %v", err)
	}

	resp, err := env.domClient.GetOuterHTML(ctx, &dompb.GetOuterHTMLRequest{
		NodeId: selResp.NodeId,
	})
	if err != nil {
		t.Fatalf("DOM.GetOuterHTML: %v", err)
	}
	if resp.OuterHtml == "" {
		t.Fatal("expected non-empty outer HTML")
	}
	t.Logf("OuterHTML: %s", resp.OuterHtml)
}

func TestDOMGetBoxModel(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<html><body><div id='box' style='width:100px;height:50px;margin:10px;padding:5px'>Box</div></body></html>",
	})
	time.Sleep(500 * time.Millisecond)

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	selResp, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#box",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector: %v", err)
	}

	resp, err := env.domClient.GetBoxModel(ctx, &dompb.GetBoxModelRequest{
		NodeId: selResp.NodeId,
	})
	if err != nil {
		t.Fatalf("DOM.GetBoxModel: %v", err)
	}
	if resp.Model == nil {
		t.Fatal("expected non-nil box model")
	}
	t.Logf("Box model: width=%d height=%d content=%v",
		resp.Model.Width, resp.Model.Height, resp.Model.Content)
}

func TestDOMSetOuterHTML(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<html><body><div id='replace'>Old Content</div></body></html>",
	})
	time.Sleep(500 * time.Millisecond)

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	selResp, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#replace",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector: %v", err)
	}

	_, err = env.domClient.SetOuterHTML(ctx, &dompb.SetOuterHTMLRequest{
		NodeId:    selResp.NodeId,
		OuterHtml: "<div id='replace'>New Content</div>",
	})
	if err != nil {
		t.Fatalf("DOM.SetOuterHTML: %v", err)
	}
}

func TestDOMDescribeNode(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<html><body><div id='desc'>Describe Me</div></body></html>",
	})
	time.Sleep(500 * time.Millisecond)

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	selResp, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#desc",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector: %v", err)
	}

	resp, err := env.domClient.DescribeNode(ctx, &dompb.DescribeNodeRequest{
		NodeId: selResp.NodeId,
		Depth:  1,
	})
	if err != nil {
		t.Fatalf("DOM.DescribeNode: %v", err)
	}
	if resp.Node == nil {
		t.Fatal("expected non-nil node")
	}
	t.Logf("Described node: name=%s type=%d children=%d",
		resp.Node.NodeName, resp.Node.NodeType, len(resp.Node.Children))
}

// =============================================================
// Emulation Domain Tests
// =============================================================

func TestEmulationSetDeviceMetricsOverride(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetDeviceMetricsOverride(ctx, &emulationpb.SetDeviceMetricsOverrideRequest{
		Width:             375,
		Height:            812,
		DeviceScaleFactor: 3,
		Mobile:            true,
	})
	if err != nil {
		t.Fatalf("Emulation.SetDeviceMetricsOverride: %v", err)
	}

	// Verify by taking a screenshot.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Mobile View</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	ssResp, err := env.pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{})
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}
	t.Logf("Mobile screenshot: %d bytes", len(ssResp.Data))

	// Clear override.
	_, err = env.emulationClient.ClearDeviceMetricsOverride(ctx, &emulationpb.ClearDeviceMetricsOverrideRequest{})
	if err != nil {
		t.Fatalf("Emulation.ClearDeviceMetricsOverride: %v", err)
	}
}

func TestEmulationSetUserAgentOverride(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetUserAgentOverride(ctx, &emulationpb.SetUserAgentOverrideRequest{
		UserAgent:      "ChromeRPC-Test/1.0 (Emulation)",
		AcceptLanguage: "en-US",
		Platform:       "Linux",
	})
	if err != nil {
		t.Fatalf("Emulation.SetUserAgentOverride: %v", err)
	}

	// Verify by evaluating navigator.userAgent.
	resp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression:    "navigator.userAgent",
		ReturnByValue: true,
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}
	t.Logf("User agent: %s", resp.Result.Value)
}

func TestEmulationSetGeolocationOverride(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetGeolocationOverride(ctx, &emulationpb.SetGeolocationOverrideRequest{
		Latitude:  37.7749,
		Longitude: -122.4194,
		Accuracy:  100,
	})
	if err != nil {
		t.Fatalf("Emulation.SetGeolocationOverride: %v", err)
	}

	_, err = env.emulationClient.ClearGeolocationOverride(ctx, &emulationpb.ClearGeolocationOverrideRequest{})
	if err != nil {
		t.Fatalf("Emulation.ClearGeolocationOverride: %v", err)
	}
}

func TestEmulationSetTimezoneOverride(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetTimezoneOverride(ctx, &emulationpb.SetTimezoneOverrideRequest{
		TimezoneId: "America/New_York",
	})
	if err != nil {
		t.Fatalf("Emulation.SetTimezoneOverride: %v", err)
	}
}

func TestEmulationSetTouchEmulationEnabled(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetTouchEmulationEnabled(ctx, &emulationpb.SetTouchEmulationEnabledRequest{
		Enabled:        true,
		MaxTouchPoints: 5,
	})
	if err != nil {
		t.Fatalf("Emulation.SetTouchEmulationEnabled: %v", err)
	}

	_, err = env.emulationClient.SetTouchEmulationEnabled(ctx, &emulationpb.SetTouchEmulationEnabledRequest{
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("Emulation.SetTouchEmulationEnabled (disable): %v", err)
	}
}

func TestEmulationSetEmulatedMedia(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetEmulatedMedia(ctx, &emulationpb.SetEmulatedMediaRequest{
		Media: "print",
		Features: []*emulationpb.MediaFeature{
			{Name: "prefers-color-scheme", Value: "dark"},
		},
	})
	if err != nil {
		t.Fatalf("Emulation.SetEmulatedMedia: %v", err)
	}
}

func TestEmulationSetEmulatedVisionDeficiency(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetEmulatedVisionDeficiency(ctx, &emulationpb.SetEmulatedVisionDeficiencyRequest{
		Type: "deuteranopia",
	})
	if err != nil {
		t.Fatalf("Emulation.SetEmulatedVisionDeficiency: %v", err)
	}

	// Reset.
	_, err = env.emulationClient.SetEmulatedVisionDeficiency(ctx, &emulationpb.SetEmulatedVisionDeficiencyRequest{
		Type: "none",
	})
	if err != nil {
		t.Fatalf("Emulation.SetEmulatedVisionDeficiency (reset): %v", err)
	}
}

func TestEmulationSetCPUThrottlingRate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.emulationClient.SetCPUThrottlingRate(ctx, &emulationpb.SetCPUThrottlingRateRequest{
		Rate: 4,
	})
	if err != nil {
		t.Fatalf("Emulation.SetCPUThrottlingRate: %v", err)
	}

	// Reset throttling.
	_, err = env.emulationClient.SetCPUThrottlingRate(ctx, &emulationpb.SetCPUThrottlingRateRequest{
		Rate: 1,
	})
	if err != nil {
		t.Fatalf("Emulation.SetCPUThrottlingRate (reset): %v", err)
	}
}

// =============================================================
// Input Domain Tests
// =============================================================

func TestInputDispatchKeyEvent(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<input id='inp' autofocus>",
	})
	time.Sleep(500 * time.Millisecond)

	// Key down.
	_, err := env.inputClient.DispatchKeyEvent(ctx, &inputpb.DispatchKeyEventRequest{
		Type:                  "keyDown",
		Key:                   "a",
		Code:                  "KeyA",
		Text:                  "a",
		WindowsVirtualKeyCode: 65,
	})
	if err != nil {
		t.Fatalf("Input.DispatchKeyEvent (keyDown): %v", err)
	}

	// Key up.
	_, err = env.inputClient.DispatchKeyEvent(ctx, &inputpb.DispatchKeyEventRequest{
		Type:                  "keyUp",
		Key:                   "a",
		Code:                  "KeyA",
		WindowsVirtualKeyCode: 65,
	})
	if err != nil {
		t.Fatalf("Input.DispatchKeyEvent (keyUp): %v", err)
	}
}

func TestInputDispatchMouseEvent(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<button onclick='document.title=\"clicked\"'>Click me</button>",
	})
	time.Sleep(500 * time.Millisecond)

	// Mouse move.
	_, err := env.inputClient.DispatchMouseEvent(ctx, &inputpb.DispatchMouseEventRequest{
		Type: "mouseMoved",
		X:    50,
		Y:    20,
	})
	if err != nil {
		t.Fatalf("Input.DispatchMouseEvent (mouseMoved): %v", err)
	}

	// Mouse press.
	_, err = env.inputClient.DispatchMouseEvent(ctx, &inputpb.DispatchMouseEventRequest{
		Type:       "mousePressed",
		X:          50,
		Y:          20,
		Button:     "left",
		ClickCount: 1,
	})
	if err != nil {
		t.Fatalf("Input.DispatchMouseEvent (mousePressed): %v", err)
	}

	// Mouse release.
	_, err = env.inputClient.DispatchMouseEvent(ctx, &inputpb.DispatchMouseEventRequest{
		Type:       "mouseReleased",
		X:          50,
		Y:          20,
		Button:     "left",
		ClickCount: 1,
	})
	if err != nil {
		t.Fatalf("Input.DispatchMouseEvent (mouseReleased): %v", err)
	}
}

func TestInputInsertText(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<input id='inp' autofocus>",
	})
	time.Sleep(500 * time.Millisecond)

	_, err := env.inputClient.InsertText(ctx, &inputpb.InsertTextRequest{
		Text: "Hello ChromeRPC",
	})
	if err != nil {
		t.Fatalf("Input.InsertText: %v", err)
	}

	// Verify the text was inserted.
	resp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression:    "document.getElementById('inp').value",
		ReturnByValue: true,
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}
	t.Logf("Input value: %s", resp.Result.Value)
}

func TestInputSetIgnoreInputEvents(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.inputClient.SetIgnoreInputEvents(ctx, &inputpb.SetIgnoreInputEventsRequest{
		Ignore: true,
	})
	if err != nil {
		t.Fatalf("Input.SetIgnoreInputEvents: %v", err)
	}

	// Re-enable.
	_, err = env.inputClient.SetIgnoreInputEvents(ctx, &inputpb.SetIgnoreInputEventsRequest{
		Ignore: false,
	})
	if err != nil {
		t.Fatalf("Input.SetIgnoreInputEvents (re-enable): %v", err)
	}
}

// =============================================================
// Browser Domain Tests
// =============================================================

func TestBrowserGetVersion(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.browserClient.GetVersion(ctx, &browserpb.GetVersionRequest{})
	if err != nil {
		t.Fatalf("Browser.GetVersion: %v", err)
	}
	if resp.Product == "" {
		t.Fatal("expected non-empty product")
	}
	if resp.UserAgent == "" {
		t.Fatal("expected non-empty user agent")
	}
	t.Logf("Browser: product=%s protocol=%s revision=%s userAgent=%s jsVersion=%s",
		resp.Product, resp.ProtocolVersion, resp.Revision, resp.UserAgent, resp.JsVersion)
}

func TestBrowserGetBrowserCommandLine(t *testing.T) {
	// This command requires --enable-automation flag on Chrome.
	// Our test Chrome doesn't have it, so just verify the RPC round-trips.
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.browserClient.GetBrowserCommandLine(ctx, &browserpb.GetBrowserCommandLineRequest{})
	if err != nil {
		// Expected: "Command line not returned because --enable-automation not set."
		t.Logf("Browser.GetBrowserCommandLine (expected error without --enable-automation): %v", err)
		return
	}
	t.Logf("Command line: %v", resp.Arguments)
}

func TestBrowserGetWindowForTarget(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Get a page target ID to pass to GetWindowForTarget.
	targets, err := env.targetClient.GetTargets(ctx, &targetpb.GetTargetsRequest{})
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	var pageTargetId string
	for _, ti := range targets.TargetInfos {
		if ti.Type == "page" {
			pageTargetId = ti.TargetId
			break
		}
	}
	if pageTargetId == "" {
		t.Skip("no page target found")
	}

	resp, err := env.browserClient.GetWindowForTarget(ctx, &browserpb.GetWindowForTargetRequest{
		TargetId: pageTargetId,
	})
	if err != nil {
		t.Fatalf("Browser.GetWindowForTarget: %v", err)
	}
	t.Logf("Window ID: %d", resp.WindowId)
	if resp.Bounds != nil {
		t.Logf("Bounds: left=%d top=%d width=%d height=%d state=%s",
			resp.Bounds.Left, resp.Bounds.Top, resp.Bounds.Width, resp.Bounds.Height, resp.Bounds.WindowState)
	}
}

func TestBrowserGetWindowBounds(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Get a page target ID to find its window.
	targets, err := env.targetClient.GetTargets(ctx, &targetpb.GetTargetsRequest{})
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	var pageTargetId string
	for _, ti := range targets.TargetInfos {
		if ti.Type == "page" {
			pageTargetId = ti.TargetId
			break
		}
	}
	if pageTargetId == "" {
		t.Skip("no page target found")
	}

	wResp, err := env.browserClient.GetWindowForTarget(ctx, &browserpb.GetWindowForTargetRequest{
		TargetId: pageTargetId,
	})
	if err != nil {
		t.Fatalf("Browser.GetWindowForTarget: %v", err)
	}

	resp, err := env.browserClient.GetWindowBounds(ctx, &browserpb.GetWindowBoundsRequest{
		WindowId: wResp.WindowId,
	})
	if err != nil {
		t.Fatalf("Browser.GetWindowBounds: %v", err)
	}
	if resp.Bounds == nil {
		t.Fatal("expected non-nil bounds")
	}
	t.Logf("Window bounds: width=%d height=%d", resp.Bounds.Width, resp.Bounds.Height)
}

func TestBrowserSetWindowBounds(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	targets, err := env.targetClient.GetTargets(ctx, &targetpb.GetTargetsRequest{})
	if err != nil {
		t.Fatalf("GetTargets: %v", err)
	}
	var pageTargetId string
	for _, ti := range targets.TargetInfos {
		if ti.Type == "page" {
			pageTargetId = ti.TargetId
			break
		}
	}
	if pageTargetId == "" {
		t.Skip("no page target found")
	}

	wResp, err := env.browserClient.GetWindowForTarget(ctx, &browserpb.GetWindowForTargetRequest{
		TargetId: pageTargetId,
	})
	if err != nil {
		t.Fatalf("Browser.GetWindowForTarget: %v", err)
	}

	_, err = env.browserClient.SetWindowBounds(ctx, &browserpb.SetWindowBoundsRequest{
		WindowId: wResp.WindowId,
		Bounds: &browserpb.Bounds{
			Width:  1024,
			Height: 768,
		},
	})
	if err != nil {
		t.Fatalf("Browser.SetWindowBounds: %v", err)
	}
}

func TestBrowserGetHistograms(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.browserClient.GetHistograms(ctx, &browserpb.GetHistogramsRequest{})
	if err != nil {
		t.Fatalf("Browser.GetHistograms: %v", err)
	}
	t.Logf("Histograms: %d total", len(resp.Histograms))
}

func TestBrowserSetDownloadBehavior(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.browserClient.SetDownloadBehavior(ctx, &browserpb.SetDownloadBehaviorRequest{
		Behavior: "deny",
	})
	if err != nil {
		t.Fatalf("Browser.SetDownloadBehavior: %v", err)
	}
}

// =============================================================
// Cross-Domain Workflow Test
// =============================================================

func TestCrossDomainWorkflow(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// 1. Set mobile device emulation.
	_, err := env.emulationClient.SetDeviceMetricsOverride(ctx, &emulationpb.SetDeviceMetricsOverrideRequest{
		Width:             390,
		Height:            844,
		DeviceScaleFactor: 3,
		Mobile:            true,
	})
	if err != nil {
		t.Fatalf("SetDeviceMetricsOverride: %v", err)
	}

	// 2. Navigate to a page.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: `data:text/html,<!DOCTYPE html>
<html>
<head><meta name="viewport" content="width=device-width"></head>
<body>
<h1 id="heading">Cross-Domain Test</h1>
<p class="info">Mobile emulated page</p>
<input id="search" placeholder="Search...">
</body></html>`,
	})
	time.Sleep(500 * time.Millisecond)

	// 3. Use Runtime to evaluate JavaScript.
	evalResp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression:    "document.getElementById('heading').textContent",
		ReturnByValue: true,
	})
	if err != nil {
		t.Fatalf("Runtime.Evaluate: %v", err)
	}
	t.Logf("Heading text: %s", evalResp.Result.Value)

	// 4. Use DOM to inspect the page.
	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{Depth: 3})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}
	t.Logf("Document root: %s", docResp.Root.NodeName)

	// 5. Use Input to type into the search field.
	// First click on the input.
	_, err = env.inputClient.DispatchMouseEvent(ctx, &inputpb.DispatchMouseEventRequest{
		Type: "mousePressed", X: 100, Y: 100, Button: "left", ClickCount: 1,
	})
	if err != nil {
		t.Fatalf("DispatchMouseEvent: %v", err)
	}
	_, err = env.inputClient.DispatchMouseEvent(ctx, &inputpb.DispatchMouseEventRequest{
		Type: "mouseReleased", X: 100, Y: 100, Button: "left", ClickCount: 1,
	})
	if err != nil {
		t.Fatalf("DispatchMouseEvent: %v", err)
	}

	// Focus the input via JS then type.
	env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression: "document.getElementById('search').focus()",
	})
	time.Sleep(100 * time.Millisecond)

	_, err = env.inputClient.InsertText(ctx, &inputpb.InsertTextRequest{Text: "chromerpc"})
	if err != nil {
		t.Fatalf("InsertText: %v", err)
	}

	// 6. Capture screenshot of the emulated mobile page.
	ssResp, err := env.pageClient.CaptureScreenshot(ctx, &pagepb.CaptureScreenshotRequest{})
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}
	t.Logf("Mobile screenshot: %d bytes", len(ssResp.Data))

	// 7. Get browser version.
	verResp, err := env.browserClient.GetVersion(ctx, &browserpb.GetVersionRequest{})
	if err != nil {
		t.Fatalf("Browser.GetVersion: %v", err)
	}
	t.Logf("Browser: %s", verResp.Product)

	// 8. Clear emulation.
	_, err = env.emulationClient.ClearDeviceMetricsOverride(ctx, &emulationpb.ClearDeviceMetricsOverrideRequest{})
	if err != nil {
		t.Fatalf("ClearDeviceMetricsOverride: %v", err)
	}
}
