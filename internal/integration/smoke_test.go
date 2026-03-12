package integration

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/accretional/chromerpc/internal/cdpclient"
)

// TestSmokeRawCDP tests the raw CDP client without gRPC to isolate issues.
func TestSmokeRawCDP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := cdpclient.Launch(ctx, cdpclient.LaunchConfig{
		Headless: true,
		ExtraArgs: []string{
			"--no-sandbox",
			"--disable-gpu",
			"--disable-dev-shm-usage",
		},
	})
	if err != nil {
		t.Skipf("Chrome not available: %v", err)
	}
	defer func() {
		result.Process.Kill()
		result.Cmd.Wait()
		if result.TempDir != "" {
			os.RemoveAll(result.TempDir)
		}
	}()

	t.Logf("Chrome WS URL: %s", result.WebSocketURL)

	// Small delay to let Chrome fully initialize.
	time.Sleep(500 * time.Millisecond)

	client, err := cdpclient.Dial(ctx, result.WebSocketURL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	// Try a simple command first: get browser version.
	verResult, err := client.Send(ctx, "Browser.getVersion", nil)
	if err != nil {
		t.Fatalf("Browser.getVersion: %v", err)
	}
	t.Logf("Browser version: %s", string(verResult))

	// Now get targets.
	targetsResult, err := client.Send(ctx, "Target.getTargets", nil)
	if err != nil {
		t.Fatalf("Target.getTargets: %v", err)
	}

	var targets struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfos"`
	}
	json.Unmarshal(targetsResult, &targets)
	t.Logf("Targets: %d", len(targets.TargetInfos))
	for _, ti := range targets.TargetInfos {
		t.Logf("  %s: %s (%s)", ti.Type, ti.TargetID, ti.URL)
	}

	// Attach to first page target.
	for _, ti := range targets.TargetInfos {
		if ti.Type == "page" {
			attachResult, err := client.Send(ctx, "Target.attachToTarget", map[string]interface{}{
				"targetId": ti.TargetID,
				"flatten":  true,
			})
			if err != nil {
				t.Fatalf("AttachToTarget: %v", err)
			}
			var ar struct {
				SessionID string `json:"sessionId"`
			}
			json.Unmarshal(attachResult, &ar)
			t.Logf("Attached with session: %s", ar.SessionID)
			client.SetSessionID(ar.SessionID)

			// Navigate.
			navResult, err := client.Send(ctx, "Page.navigate", map[string]interface{}{
				"url": "data:text/html,<h1>Hello</h1>",
			})
			if err != nil {
				t.Fatalf("Page.navigate: %v", err)
			}
			t.Logf("Navigate result: %s", string(navResult))

			time.Sleep(500 * time.Millisecond)

			// Screenshot.
			ssResult, err := client.Send(ctx, "Page.captureScreenshot", nil)
			if err != nil {
				t.Fatalf("Page.captureScreenshot: %v", err)
			}
			var ss struct {
				Data string `json:"data"`
			}
			json.Unmarshal(ssResult, &ss)
			t.Logf("Screenshot base64 length: %d", len(ss.Data))
			break
		}
	}
}
