package integration

import (
	"context"
	"testing"
	"time"

	animationpb "github.com/accretional/chromerpc/proto/cdp/animation"
	auditspb "github.com/accretional/chromerpc/proto/cdp/audits"
	layertreepb "github.com/accretional/chromerpc/proto/cdp/layertree"
	mediapb "github.com/accretional/chromerpc/proto/cdp/media"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	systeminfopb "github.com/accretional/chromerpc/proto/cdp/systeminfo"
	tracingpb "github.com/accretional/chromerpc/proto/cdp/tracing"
)

// =============================================================
// Audits Domain Tests
// =============================================================

func TestAuditsEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.auditsClient.Enable(ctx, &auditspb.EnableRequest{})
	if err != nil {
		t.Fatalf("Audits.Enable: %v", err)
	}

	_, err = env.auditsClient.Disable(ctx, &auditspb.DisableRequest{})
	if err != nil {
		t.Fatalf("Audits.Disable: %v", err)
	}
}

func TestAuditsCheckContrast(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.auditsClient.Enable(ctx, &auditspb.EnableRequest{})
	defer env.auditsClient.Disable(ctx, &auditspb.DisableRequest{})

	// Navigate to a page with content for contrast checking.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<p style='color:#999;background:#fff'>Low contrast</p>",
	})
	time.Sleep(300 * time.Millisecond)

	_, err := env.auditsClient.CheckContrast(ctx, &auditspb.CheckContrastRequest{})
	if err != nil {
		t.Logf("CheckContrast: %v", err)
	}
}

// =============================================================
// LayerTree Domain Tests
// =============================================================

func TestLayerTreeEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.layerTreeClient.Enable(ctx, &layertreepb.EnableRequest{})
	if err != nil {
		t.Fatalf("LayerTree.Enable: %v", err)
	}

	_, err = env.layerTreeClient.Disable(ctx, &layertreepb.DisableRequest{})
	if err != nil {
		t.Fatalf("LayerTree.Disable: %v", err)
	}
}

func TestLayerTreeCompositingReasons(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<div style='transform:translateZ(0);width:100px;height:100px;background:red'>Layer</div>",
	})
	time.Sleep(300 * time.Millisecond)

	env.layerTreeClient.Enable(ctx, &layertreepb.EnableRequest{})
	defer env.layerTreeClient.Disable(ctx, &layertreepb.DisableRequest{})

	// We need a valid layerId; just test with a placeholder.
	_, err := env.layerTreeClient.CompositingReasons(ctx, &layertreepb.CompositingReasonsRequest{
		LayerId: "1",
	})
	if err != nil {
		t.Logf("CompositingReasons: %v (expected for invalid layerId)", err)
	}
}

// =============================================================
// Animation Domain Tests
// =============================================================

func TestAnimationEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.animationClient.Enable(ctx, &animationpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Animation.Enable: %v", err)
	}

	_, err = env.animationClient.Disable(ctx, &animationpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Animation.Disable: %v", err)
	}
}

func TestAnimationGetPlaybackRate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.animationClient.Enable(ctx, &animationpb.EnableRequest{})
	defer env.animationClient.Disable(ctx, &animationpb.DisableRequest{})

	resp, err := env.animationClient.GetPlaybackRate(ctx, &animationpb.GetPlaybackRateRequest{})
	if err != nil {
		t.Fatalf("GetPlaybackRate: %v", err)
	}
	t.Logf("Playback rate: %f", resp.PlaybackRate)
}

func TestAnimationSetPlaybackRate(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.animationClient.Enable(ctx, &animationpb.EnableRequest{})
	defer env.animationClient.Disable(ctx, &animationpb.DisableRequest{})

	_, err := env.animationClient.SetPlaybackRate(ctx, &animationpb.SetPlaybackRateRequest{
		PlaybackRate: 2.0,
	})
	if err != nil {
		t.Fatalf("SetPlaybackRate: %v", err)
	}

	// Reset.
	env.animationClient.SetPlaybackRate(ctx, &animationpb.SetPlaybackRateRequest{PlaybackRate: 1.0})
}

// =============================================================
// Media Domain Tests
// =============================================================

func TestMediaEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.mediaClient.Enable(ctx, &mediapb.EnableRequest{})
	if err != nil {
		t.Fatalf("Media.Enable: %v", err)
	}

	_, err = env.mediaClient.Disable(ctx, &mediapb.DisableRequest{})
	if err != nil {
		t.Fatalf("Media.Disable: %v", err)
	}
}

// =============================================================
// Tracing Domain Tests
// =============================================================

func TestTracingGetCategories(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.tracingClient.GetCategories(ctx, &tracingpb.GetCategoriesRequest{})
	if err != nil {
		t.Fatalf("GetCategories: %v", err)
	}
	if len(resp.Categories) == 0 {
		t.Fatal("expected at least one tracing category")
	}
	t.Logf("Tracing categories: %d (first: %s)", len(resp.Categories), resp.Categories[0])
}

func TestTracingStartEnd(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.tracingClient.Start(ctx, &tracingpb.StartRequest{
		Categories: "-*,devtools.timeline",
	})
	if err != nil {
		t.Fatalf("Tracing.Start: %v", err)
	}

	// Do some work.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<script>for(var i=0;i<100;i++) document.createElement('div');</script>",
	})
	time.Sleep(300 * time.Millisecond)

	_, err = env.tracingClient.End(ctx, &tracingpb.EndRequest{})
	if err != nil {
		t.Fatalf("Tracing.End: %v", err)
	}
}

// =============================================================
// SystemInfo Domain Tests
// =============================================================

func TestSystemInfoGetInfo(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.systemInfoClient.GetInfo(ctx, &systeminfopb.GetInfoRequest{})
	if err != nil {
		t.Fatalf("SystemInfo.GetInfo: %v", err)
	}
	t.Logf("Model: %s %s, CommandLine: %.80s...", resp.ModelName, resp.ModelVersion, resp.CommandLine)
}

func TestSystemInfoGetProcessInfo(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.systemInfoClient.GetProcessInfo(ctx, &systeminfopb.GetProcessInfoRequest{})
	if err != nil {
		t.Fatalf("GetProcessInfo: %v", err)
	}
	t.Logf("Processes: %d", len(resp.ProcessInfo))
	for _, p := range resp.ProcessInfo {
		t.Logf("  type=%s id=%d cpuTime=%.2f", p.Type, p.Id, p.CpuTime)
	}
}
