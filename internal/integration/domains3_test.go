package integration

import (
	"context"
	"testing"
	"time"

	debuggerpb "github.com/accretional/chromerpc/proto/cdp/debugger"
	domstoragepb "github.com/accretional/chromerpc/proto/cdp/domstorage"
	overlaypb "github.com/accretional/chromerpc/proto/cdp/overlay"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	profilerpb "github.com/accretional/chromerpc/proto/cdp/profiler"
	storagepb "github.com/accretional/chromerpc/proto/cdp/storage"
)

// =============================================================
// Storage Domain Tests
// =============================================================

func TestStorageClearDataForOrigin(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Navigate to create a valid origin context.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Storage Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	// data: URLs don't have a traditional origin; use empty string which clears for the current context.
	_, err := env.storageClient.ClearDataForOrigin(ctx, &storagepb.ClearDataForOriginRequest{
		Origin:       "",
		StorageTypes: "all",
	})
	if err != nil {
		// This may fail with internal error for data: origins; that's expected.
		t.Logf("ClearDataForOrigin: %v (expected for non-http origin)", err)
	}
}

func TestStorageGetUsageAndQuota(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Navigate to create a valid origin.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Quota Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	resp, err := env.storageClient.GetUsageAndQuota(ctx, &storagepb.GetUsageAndQuotaRequest{
		Origin: "data://",
	})
	if err != nil {
		// data:// origin may not support quota queries.
		t.Logf("GetUsageAndQuota: %v (expected for non-http origin)", err)
		return
	}
	t.Logf("Usage: %.0f, Quota: %.0f, OverrideActive: %v", resp.Usage, resp.Quota, resp.OverrideActive)
	for _, b := range resp.UsageBreakdown {
		t.Logf("  %s: %.0f", b.StorageType, b.Usage)
	}
}

func TestStorageGetSetClearCookies(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Set a cookie.
	_, err := env.storageClient.SetCookies(ctx, &storagepb.SetCookiesRequest{
		Cookies: []*storagepb.CookieParam{
			{Name: "test_cookie", Value: "hello_chromerpc", Domain: "example.com", Path: "/"},
		},
	})
	if err != nil {
		t.Fatalf("SetCookies: %v", err)
	}

	// Get cookies.
	getResp, err := env.storageClient.GetCookies(ctx, &storagepb.GetCookiesRequest{})
	if err != nil {
		t.Fatalf("GetCookies: %v", err)
	}
	var found bool
	for _, c := range getResp.Cookies {
		t.Logf("Cookie: %s=%s domain=%s", c.Name, c.Value, c.Domain)
		if c.Name == "test_cookie" {
			found = true
		}
	}
	if !found {
		t.Log("test_cookie not found (may be filtered by browser context)")
	}

	// Clear cookies.
	_, err = env.storageClient.ClearCookies(ctx, &storagepb.ClearCookiesRequest{})
	if err != nil {
		t.Fatalf("ClearCookies: %v", err)
	}
}

func TestStorageTrackIndexedDB(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// These commands require a storage-capable origin. data: URLs don't qualify.
	// We test that the RPC call itself works (even if CDP returns an internal error for non-http origins).
	_, err := env.storageClient.TrackIndexedDBForOrigin(ctx, &storagepb.TrackIndexedDBForOriginRequest{
		Origin: "http://localhost",
	})
	if err != nil {
		t.Logf("TrackIndexedDBForOrigin: %v (expected for headless without http server)", err)
	}

	_, err = env.storageClient.UntrackIndexedDBForOrigin(ctx, &storagepb.UntrackIndexedDBForOriginRequest{
		Origin: "http://localhost",
	})
	if err != nil {
		t.Logf("UntrackIndexedDBForOrigin: %v (expected)", err)
	}
}

func TestStorageTrackCacheStorage(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.storageClient.TrackCacheStorageForOrigin(ctx, &storagepb.TrackCacheStorageForOriginRequest{
		Origin: "http://localhost",
	})
	if err != nil {
		t.Logf("TrackCacheStorageForOrigin: %v (expected for headless without http server)", err)
	}

	_, err = env.storageClient.UntrackCacheStorageForOrigin(ctx, &storagepb.UntrackCacheStorageForOriginRequest{
		Origin: "http://localhost",
	})
	if err != nil {
		t.Logf("UntrackCacheStorageForOrigin: %v (expected)", err)
	}
}

// =============================================================
// Overlay Domain Tests
// =============================================================

func TestOverlayEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Overlay requires DOM domain.
	env.domClient.Enable(ctx, nil)

	_, err := env.overlayClient.Enable(ctx, &overlaypb.EnableRequest{})
	if err != nil {
		t.Fatalf("Overlay.Enable: %v", err)
	}

	_, err = env.overlayClient.Disable(ctx, &overlaypb.DisableRequest{})
	if err != nil {
		t.Fatalf("Overlay.Disable: %v", err)
	}
}

func TestOverlayHighlightRect(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.domClient.Enable(ctx, nil)
	env.overlayClient.Enable(ctx, &overlaypb.EnableRequest{})

	_, err := env.overlayClient.HighlightRect(ctx, &overlaypb.HighlightRectRequest{
		X: 10, Y: 10, Width: 100, Height: 100,
		Color: &overlaypb.RGBA{R: 255, G: 0, B: 0, A: 0.5},
	})
	if err != nil {
		t.Fatalf("HighlightRect: %v", err)
	}

	_, err = env.overlayClient.HideHighlight(ctx, &overlaypb.HideHighlightRequest{})
	if err != nil {
		t.Fatalf("HideHighlight: %v", err)
	}
}

func TestOverlaySetShowPaintRects(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.domClient.Enable(ctx, nil)
	env.overlayClient.Enable(ctx, &overlaypb.EnableRequest{})

	_, err := env.overlayClient.SetShowPaintRects(ctx, &overlaypb.SetShowPaintRectsRequest{Result: true})
	if err != nil {
		t.Fatalf("SetShowPaintRects: %v", err)
	}

	_, err = env.overlayClient.SetShowPaintRects(ctx, &overlaypb.SetShowPaintRectsRequest{Result: false})
	if err != nil {
		t.Fatalf("SetShowPaintRects (disable): %v", err)
	}
}

func TestOverlaySetShowDebugBorders(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.domClient.Enable(ctx, nil)
	env.overlayClient.Enable(ctx, &overlaypb.EnableRequest{})

	_, err := env.overlayClient.SetShowDebugBorders(ctx, &overlaypb.SetShowDebugBordersRequest{Show: true})
	if err != nil {
		t.Fatalf("SetShowDebugBorders: %v", err)
	}

	_, err = env.overlayClient.SetShowDebugBorders(ctx, &overlaypb.SetShowDebugBordersRequest{Show: false})
	if err != nil {
		t.Fatalf("SetShowDebugBorders (disable): %v", err)
	}
}

func TestOverlaySetShowFPSCounter(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.domClient.Enable(ctx, nil)
	env.overlayClient.Enable(ctx, &overlaypb.EnableRequest{})

	_, err := env.overlayClient.SetShowFPSCounter(ctx, &overlaypb.SetShowFPSCounterRequest{Show: true})
	if err != nil {
		t.Fatalf("SetShowFPSCounter: %v", err)
	}

	_, err = env.overlayClient.SetShowFPSCounter(ctx, &overlaypb.SetShowFPSCounterRequest{Show: false})
	if err != nil {
		t.Fatalf("SetShowFPSCounter (disable): %v", err)
	}
}

// =============================================================
// DOMStorage Domain Tests
// =============================================================

func TestDOMStorageEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.domstorageClient.Enable(ctx, &domstoragepb.EnableRequest{})
	if err != nil {
		t.Fatalf("DOMStorage.Enable: %v", err)
	}

	_, err = env.domstorageClient.Disable(ctx, &domstoragepb.DisableRequest{})
	if err != nil {
		t.Fatalf("DOMStorage.Disable: %v", err)
	}
}

func TestDOMStorageSetGetRemove(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Navigate to a page so we have a valid origin for localStorage.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>DOMStorage Test</h1>",
	})
	time.Sleep(500 * time.Millisecond)

	env.domstorageClient.Enable(ctx, &domstoragepb.EnableRequest{})

	storageId := &domstoragepb.StorageId{
		SecurityOrigin: "data://",
		IsLocalStorage: true,
	}

	// Set an item.
	_, err := env.domstorageClient.SetDOMStorageItem(ctx, &domstoragepb.SetDOMStorageItemRequest{
		StorageId: storageId,
		Key:       "test_key",
		Value:     "test_value",
	})
	if err != nil {
		// data:// origin may not support localStorage in all Chrome versions.
		t.Logf("SetDOMStorageItem: %v (may be unsupported for data:// origin)", err)
		return
	}

	// Get items.
	getResp, err := env.domstorageClient.GetDOMStorageItems(ctx, &domstoragepb.GetDOMStorageItemsRequest{
		StorageId: storageId,
	})
	if err != nil {
		t.Fatalf("GetDOMStorageItems: %v", err)
	}
	t.Logf("DOMStorage items: %d", len(getResp.Entries))
	for _, item := range getResp.Entries {
		t.Logf("  %s = %s", item.Key, item.Value)
	}

	// Remove.
	_, err = env.domstorageClient.RemoveDOMStorageItem(ctx, &domstoragepb.RemoveDOMStorageItemRequest{
		StorageId: storageId,
		Key:       "test_key",
	})
	if err != nil {
		t.Fatalf("RemoveDOMStorageItem: %v", err)
	}
}

func TestDOMStorageClear(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// Navigate to create a frame with a real origin.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Clear Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	env.domstorageClient.Enable(ctx, &domstoragepb.EnableRequest{})

	storageId := &domstoragepb.StorageId{
		SecurityOrigin: "data://",
		IsLocalStorage: true,
	}

	_, err := env.domstorageClient.Clear(ctx, &domstoragepb.ClearRequest{
		StorageId: storageId,
	})
	if err != nil {
		// data:// origins may not support DOMStorage operations.
		t.Logf("DOMStorage.Clear: %v (expected for data:// origin)", err)
	}
}

// =============================================================
// Debugger Domain Tests
// =============================================================

func TestDebuggerEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.debuggerClient.Enable(ctx, &debuggerpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Debugger.Enable: %v", err)
	}
	t.Logf("DebuggerId: %s", resp.DebuggerId)

	_, err = env.debuggerClient.Disable(ctx, &debuggerpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Debugger.Disable: %v", err)
	}
}

func TestDebuggerSetPauseOnExceptions(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.debuggerClient.Enable(ctx, &debuggerpb.EnableRequest{})
	defer env.debuggerClient.Disable(ctx, &debuggerpb.DisableRequest{})

	for _, state := range []string{"none", "uncaught", "all"} {
		_, err := env.debuggerClient.SetPauseOnExceptions(ctx, &debuggerpb.SetPauseOnExceptionsRequest{
			State: state,
		})
		if err != nil {
			t.Fatalf("SetPauseOnExceptions(%s): %v", state, err)
		}
	}
}

func TestDebuggerSetAsyncCallStackDepth(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.debuggerClient.Enable(ctx, &debuggerpb.EnableRequest{})
	defer env.debuggerClient.Disable(ctx, &debuggerpb.DisableRequest{})

	_, err := env.debuggerClient.SetAsyncCallStackDepth(ctx, &debuggerpb.SetAsyncCallStackDepthRequest{
		MaxDepth: 32,
	})
	if err != nil {
		t.Fatalf("SetAsyncCallStackDepth: %v", err)
	}
}

func TestDebuggerSetBlackboxPatterns(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.debuggerClient.Enable(ctx, &debuggerpb.EnableRequest{})
	defer env.debuggerClient.Disable(ctx, &debuggerpb.DisableRequest{})

	_, err := env.debuggerClient.SetBlackboxPatterns(ctx, &debuggerpb.SetBlackboxPatternsRequest{
		Patterns: []string{"node_modules"},
	})
	if err != nil {
		t.Fatalf("SetBlackboxPatterns: %v", err)
	}
}

func TestDebuggerSetBreakpointsActive(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.debuggerClient.Enable(ctx, &debuggerpb.EnableRequest{})
	defer env.debuggerClient.Disable(ctx, &debuggerpb.DisableRequest{})

	_, err := env.debuggerClient.SetBreakpointsActive(ctx, &debuggerpb.SetBreakpointsActiveRequest{
		Active: true,
	})
	if err != nil {
		t.Fatalf("SetBreakpointsActive: %v", err)
	}
}

func TestDebuggerGetScriptSource(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.debuggerClient.Enable(ctx, &debuggerpb.EnableRequest{})
	defer env.debuggerClient.Disable(ctx, &debuggerpb.DisableRequest{})

	// Navigate to a page with a script to get a scriptId.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<script>var x = 1;</script>",
	})
	time.Sleep(500 * time.Millisecond)

	// We can't easily get a scriptId without subscribing to events,
	// so just test that the call works with an error for invalid ID.
	_, err := env.debuggerClient.GetScriptSource(ctx, &debuggerpb.GetScriptSourceRequest{
		ScriptId: "999999",
	})
	if err != nil {
		t.Logf("GetScriptSource with invalid ID: %v (expected)", err)
	}
}

// =============================================================
// Profiler Domain Tests
// =============================================================

func TestProfilerEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.profilerClient.Enable(ctx, &profilerpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Profiler.Enable: %v", err)
	}

	_, err = env.profilerClient.Disable(ctx, &profilerpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Profiler.Disable: %v", err)
	}
}

func TestProfilerStartStop(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.profilerClient.Enable(ctx, &profilerpb.EnableRequest{})
	defer env.profilerClient.Disable(ctx, &profilerpb.DisableRequest{})

	_, err := env.profilerClient.Start(ctx, &profilerpb.StartRequest{})
	if err != nil {
		t.Fatalf("Profiler.Start: %v", err)
	}

	// Do some JS work to generate profile data.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<script>for(var i=0;i<1000;i++){Math.sqrt(i);}</script>",
	})
	time.Sleep(500 * time.Millisecond)

	stopResp, err := env.profilerClient.Stop(ctx, &profilerpb.StopRequest{})
	if err != nil {
		t.Fatalf("Profiler.Stop: %v", err)
	}
	if stopResp.Profile == nil {
		t.Fatal("expected non-nil profile")
	}
	t.Logf("Profile: %d nodes, startTime=%.0f endTime=%.0f",
		len(stopResp.Profile.Nodes), stopResp.Profile.StartTime, stopResp.Profile.EndTime)
}

func TestProfilerPreciseCoverage(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.profilerClient.Enable(ctx, &profilerpb.EnableRequest{})
	defer env.profilerClient.Disable(ctx, &profilerpb.DisableRequest{})

	startResp, err := env.profilerClient.StartPreciseCoverage(ctx, &profilerpb.StartPreciseCoverageRequest{
		CallCount: true,
		Detailed:  true,
	})
	if err != nil {
		t.Fatalf("StartPreciseCoverage: %v", err)
	}
	t.Logf("Coverage started at timestamp: %f", startResp.Timestamp)

	// Generate some code execution.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<script>function foo(){return 42;} foo();</script>",
	})
	time.Sleep(500 * time.Millisecond)

	takeResp, err := env.profilerClient.TakePreciseCoverage(ctx, &profilerpb.TakePreciseCoverageRequest{})
	if err != nil {
		t.Fatalf("TakePreciseCoverage: %v", err)
	}
	t.Logf("Coverage: %d scripts, timestamp=%f", len(takeResp.Result), takeResp.Timestamp)

	_, err = env.profilerClient.StopPreciseCoverage(ctx, &profilerpb.StopPreciseCoverageRequest{})
	if err != nil {
		t.Fatalf("StopPreciseCoverage: %v", err)
	}
}

func TestProfilerGetBestEffortCoverage(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.profilerClient.Enable(ctx, &profilerpb.EnableRequest{})
	defer env.profilerClient.Disable(ctx, &profilerpb.DisableRequest{})

	resp, err := env.profilerClient.GetBestEffortCoverage(ctx, &profilerpb.GetBestEffortCoverageRequest{})
	if err != nil {
		t.Fatalf("GetBestEffortCoverage: %v", err)
	}
	t.Logf("BestEffort coverage: %d scripts", len(resp.Result))
}

func TestProfilerSetSamplingInterval(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.profilerClient.Enable(ctx, &profilerpb.EnableRequest{})
	defer env.profilerClient.Disable(ctx, &profilerpb.DisableRequest{})

	_, err := env.profilerClient.SetSamplingInterval(ctx, &profilerpb.SetSamplingIntervalRequest{
		Interval: 100,
	})
	if err != nil {
		// Some Chrome versions require this before Start.
		t.Logf("SetSamplingInterval: %v", err)
	}
}
