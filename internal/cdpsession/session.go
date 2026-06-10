// Package cdpsession provides helpers for establishing a default CDP page
// session on a client, shared by the server entrypoint and the interactive
// session manager.
package cdpsession

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/accretional/chromerpc/internal/cdpclient"
)

// SetupDefault discovers page targets and attaches to the first one with
// flatten=true, setting the resulting session ID as the client's default so
// that page-scoped commands route to that target.
func SetupDefault(ctx context.Context, client *cdpclient.Client) error {
	result, err := client.Send(ctx, "Target.getTargets", nil)
	if err != nil {
		return fmt.Errorf("getTargets: %w", err)
	}

	type targetInfo struct {
		TargetID string `json:"targetId"`
		Type     string `json:"type"`
		URL      string `json:"url"`
	}
	var resp struct {
		TargetInfos []targetInfo `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return fmt.Errorf("unmarshal targets: %w", err)
	}

	var pageTarget *targetInfo
	for _, t := range resp.TargetInfos {
		if t.Type == "page" {
			t := t
			pageTarget = &t
			break
		}
	}
	if pageTarget == nil {
		return fmt.Errorf("no page target found")
	}

	log.Printf("Attaching to page target %s (%s)", pageTarget.TargetID, pageTarget.URL)

	attachResult, err := client.Send(ctx, "Target.attachToTarget", map[string]interface{}{
		"targetId": pageTarget.TargetID,
		"flatten":  true,
	})
	if err != nil {
		return fmt.Errorf("attachToTarget: %w", err)
	}

	var attachResp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attachResult, &attachResp); err != nil {
		return fmt.Errorf("unmarshal attach: %w", err)
	}

	client.SetSessionID(attachResp.SessionID)
	log.Printf("Session established: %s", attachResp.SessionID)
	return nil
}
