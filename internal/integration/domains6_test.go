package integration

import (
	"context"
	"testing"
	"time"

	backgroundservicepb "github.com/accretional/chromerpc/proto/cdp/backgroundservice"
	databasepb "github.com/accretional/chromerpc/proto/cdp/database"
	deviceorientationpb "github.com/accretional/chromerpc/proto/cdp/deviceorientation"
	domdebuggerpb "github.com/accretional/chromerpc/proto/cdp/domdebugger"
	inspectorpb "github.com/accretional/chromerpc/proto/cdp/inspector"
	memorypb "github.com/accretional/chromerpc/proto/cdp/memory"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	runtimepb "github.com/accretional/chromerpc/proto/cdp/runtime"
	webaudiopb "github.com/accretional/chromerpc/proto/cdp/webaudio"
)

// =============================================================
// Memory Domain Tests
// =============================================================

func TestMemoryGetDOMCounters(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<div>Memory test</div>",
	})
	time.Sleep(300 * time.Millisecond)

	resp, err := env.memoryClient.GetDOMCounters(ctx, &memorypb.GetDOMCountersRequest{})
	if err != nil {
		t.Fatalf("GetDOMCounters: %v", err)
	}
	t.Logf("DOM counters: documents=%d nodes=%d jsEventListeners=%d",
		resp.Documents, resp.Nodes, resp.JsEventListeners)
}

func TestMemoryForciblyPurgeJavaScriptMemory(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.memoryClient.ForciblyPurgeJavaScriptMemory(ctx, &memorypb.ForciblyPurgeJavaScriptMemoryRequest{})
	if err != nil {
		t.Fatalf("ForciblyPurgeJavaScriptMemory: %v", err)
	}
}

func TestMemorySetPressureNotificationsSuppressed(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.memoryClient.SetPressureNotificationsSuppressed(ctx, &memorypb.SetPressureNotificationsSuppressedRequest{
		Suppressed: true,
	})
	if err != nil {
		t.Fatalf("SetPressureNotificationsSuppressed: %v", err)
	}

	// Reset.
	env.memoryClient.SetPressureNotificationsSuppressed(ctx, &memorypb.SetPressureNotificationsSuppressedRequest{
		Suppressed: false,
	})
}

func TestMemorySimulatePressureNotification(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.memoryClient.SimulatePressureNotification(ctx, &memorypb.SimulatePressureNotificationRequest{
		Level: "moderate",
	})
	if err != nil {
		t.Logf("SimulatePressureNotification: %v", err)
	}
}

func TestMemorySampling(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.memoryClient.StartSampling(ctx, &memorypb.StartSamplingRequest{})
	if err != nil {
		t.Fatalf("StartSampling: %v", err)
	}

	// Do some work.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<script>var arr=[]; for(var i=0;i<100;i++) arr.push(new Array(100));</script>",
	})
	time.Sleep(300 * time.Millisecond)

	resp, err := env.memoryClient.GetSamplingProfile(ctx, &memorypb.GetSamplingProfileRequest{})
	if err != nil {
		t.Logf("GetSamplingProfile: %v", err)
	} else if resp.Profile != nil {
		t.Logf("Sampling profile: %d samples", len(resp.Profile.Samples))
	}

	_, err = env.memoryClient.StopSampling(ctx, &memorypb.StopSamplingRequest{})
	if err != nil {
		t.Fatalf("StopSampling: %v", err)
	}
}

// =============================================================
// DOMDebugger Domain Tests
// =============================================================

func TestDOMDebuggerGetEventListeners(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<button onclick='alert()'>Click</button>",
	})
	time.Sleep(300 * time.Millisecond)

	// Get document object.
	evalResp, err := env.runtimeClient.Evaluate(ctx, &runtimepb.EvaluateRequest{
		Expression: "document",
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	resp, err := env.domDebuggerClient.GetEventListeners(ctx, &domdebuggerpb.GetEventListenersRequest{
		ObjectId: evalResp.Result.ObjectId,
	})
	if err != nil {
		t.Fatalf("GetEventListeners: %v", err)
	}
	t.Logf("Event listeners: %d", len(resp.Listeners))
}

func TestDOMDebuggerSetXHRBreakpoint(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.domDebuggerClient.SetXHRBreakpoint(ctx, &domdebuggerpb.SetXHRBreakpointRequest{
		Url: "example.com",
	})
	if err != nil {
		t.Fatalf("SetXHRBreakpoint: %v", err)
	}

	_, err = env.domDebuggerClient.RemoveXHRBreakpoint(ctx, &domdebuggerpb.RemoveXHRBreakpointRequest{
		Url: "example.com",
	})
	if err != nil {
		t.Fatalf("RemoveXHRBreakpoint: %v", err)
	}
}

func TestDOMDebuggerSetEventListenerBreakpoint(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.domDebuggerClient.SetEventListenerBreakpoint(ctx, &domdebuggerpb.SetEventListenerBreakpointRequest{
		EventName: "click",
	})
	if err != nil {
		t.Fatalf("SetEventListenerBreakpoint: %v", err)
	}

	_, err = env.domDebuggerClient.RemoveEventListenerBreakpoint(ctx, &domdebuggerpb.RemoveEventListenerBreakpointRequest{
		EventName: "click",
	})
	if err != nil {
		t.Fatalf("RemoveEventListenerBreakpoint: %v", err)
	}
}

// =============================================================
// WebAudio Domain Tests
// =============================================================

func TestWebAudioEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.webAudioClient.Enable(ctx, &webaudiopb.EnableRequest{})
	if err != nil {
		t.Fatalf("WebAudio.Enable: %v", err)
	}

	_, err = env.webAudioClient.Disable(ctx, &webaudiopb.DisableRequest{})
	if err != nil {
		t.Fatalf("WebAudio.Disable: %v", err)
	}
}

// =============================================================
// Inspector Domain Tests
// =============================================================

func TestInspectorEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.inspectorClient.Enable(ctx, &inspectorpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Inspector.Enable: %v", err)
	}

	_, err = env.inspectorClient.Disable(ctx, &inspectorpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Inspector.Disable: %v", err)
	}
}

// =============================================================
// Database Domain Tests
// =============================================================

func TestDatabaseEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.databaseClient.Enable(ctx, &databasepb.EnableRequest{})
	if err != nil {
		t.Fatalf("Database.Enable: %v", err)
	}

	_, err = env.databaseClient.Disable(ctx, &databasepb.DisableRequest{})
	if err != nil {
		t.Fatalf("Database.Disable: %v", err)
	}
}

// =============================================================
// BackgroundService Domain Tests
// =============================================================

func TestBackgroundServiceStartStopObserving(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.backgroundServiceClient.StartObserving(ctx, &backgroundservicepb.StartObservingRequest{
		Service: backgroundservicepb.ServiceName_BACKGROUND_FETCH,
	})
	if err != nil {
		// BackgroundService may not be available on all Chrome versions.
		t.Logf("StartObserving: %v", err)
		return
	}

	_, err = env.backgroundServiceClient.StopObserving(ctx, &backgroundservicepb.StopObservingRequest{
		Service: backgroundservicepb.ServiceName_BACKGROUND_FETCH,
	})
	if err != nil {
		t.Logf("StopObserving: %v", err)
	}
}

func TestBackgroundServiceSetRecording(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.backgroundServiceClient.SetRecording(ctx, &backgroundservicepb.SetRecordingRequest{
		ShouldRecord: true,
		Service:      backgroundservicepb.ServiceName_PUSH_MESSAGING,
	})
	if err != nil {
		t.Logf("SetRecording: %v", err)
		return
	}

	// Stop recording.
	env.backgroundServiceClient.SetRecording(ctx, &backgroundservicepb.SetRecordingRequest{
		ShouldRecord: false,
		Service:      backgroundservicepb.ServiceName_PUSH_MESSAGING,
	})
}

// =============================================================
// DeviceOrientation Domain Tests
// =============================================================

func TestDeviceOrientationSetAndClear(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.deviceOrientationClient.SetDeviceOrientationOverride(ctx, &deviceorientationpb.SetDeviceOrientationOverrideRequest{
		Alpha: 90.0,
		Beta:  45.0,
		Gamma: 30.0,
	})
	if err != nil {
		t.Fatalf("SetDeviceOrientationOverride: %v", err)
	}

	_, err = env.deviceOrientationClient.ClearDeviceOrientationOverride(ctx, &deviceorientationpb.ClearDeviceOrientationOverrideRequest{})
	if err != nil {
		t.Fatalf("ClearDeviceOrientationOverride: %v", err)
	}
}
