package integration

import (
	"context"
	"testing"
	"time"

	autofillpb "github.com/accretional/chromerpc/proto/cdp/autofill"
	castpb "github.com/accretional/chromerpc/proto/cdp/cast"
	domsnapshotpb "github.com/accretional/chromerpc/proto/cdp/domsnapshot"
	fedcmpb "github.com/accretional/chromerpc/proto/cdp/fedcm"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
)

// =============================================================
// Cast Domain Tests
// =============================================================

func TestCastEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.castClient.Enable(ctx, &castpb.EnableRequest{})
	if err != nil {
		// Cast may not be available in headless mode.
		t.Logf("Cast.Enable: %v", err)
		return
	}

	_, err = env.castClient.Disable(ctx, &castpb.DisableRequest{})
	if err != nil {
		t.Logf("Cast.Disable: %v", err)
	}
}

// =============================================================
// DOMSnapshot Domain Tests
// =============================================================

func TestDOMSnapshotEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.domSnapshotClient.Enable(ctx, &domsnapshotpb.EnableRequest{})
	if err != nil {
		t.Fatalf("DOMSnapshot.Enable: %v", err)
	}

	_, err = env.domSnapshotClient.Disable(ctx, &domsnapshotpb.DisableRequest{})
	if err != nil {
		t.Fatalf("DOMSnapshot.Disable: %v", err)
	}
}

func TestDOMSnapshotCaptureSnapshot(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<div style='color:red'>Snapshot test</div>",
	})
	time.Sleep(300 * time.Millisecond)

	resp, err := env.domSnapshotClient.CaptureSnapshot(ctx, &domsnapshotpb.CaptureSnapshotRequest{
		ComputedStyles: []string{"color", "background-color"},
	})
	if err != nil {
		t.Fatalf("CaptureSnapshot: %v", err)
	}
	if len(resp.DocumentsJson) == 0 {
		t.Error("expected non-empty documents JSON")
	}
	t.Logf("DOMSnapshot: %d bytes documents, %d strings", len(resp.DocumentsJson), len(resp.Strings))
}

// =============================================================
// FedCm Domain Tests
// =============================================================

func TestFedCmEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.fedCmClient.Enable(ctx, &fedcmpb.EnableRequest{})
	if err != nil {
		// FedCm may not be available in all Chrome versions.
		t.Logf("FedCm.Enable: %v", err)
		return
	}

	_, err = env.fedCmClient.Disable(ctx, &fedcmpb.DisableRequest{})
	if err != nil {
		t.Logf("FedCm.Disable: %v", err)
	}
}

func TestFedCmResetCooldown(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.fedCmClient.ResetCooldown(ctx, &fedcmpb.ResetCooldownRequest{})
	if err != nil {
		t.Logf("FedCm.ResetCooldown: %v", err)
	}
}

// =============================================================
// Autofill Domain Tests
// =============================================================

func TestAutofillEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.autofillClient.Enable(ctx, &autofillpb.EnableRequest{})
	if err != nil {
		t.Logf("Autofill.Enable: %v", err)
		return
	}

	_, err = env.autofillClient.Disable(ctx, &autofillpb.DisableRequest{})
	if err != nil {
		t.Logf("Autofill.Disable: %v", err)
	}
}
