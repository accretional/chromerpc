package integration

import (
	"context"
	"testing"

	performancetimelinepb "github.com/accretional/chromerpc/proto/cdp/performancetimeline"
	preloadpb "github.com/accretional/chromerpc/proto/cdp/preload"
	webauthnpb "github.com/accretional/chromerpc/proto/cdp/webauthn"
)

// =============================================================
// WebAuthn Domain Tests
// =============================================================

func TestWebAuthnEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.webAuthnClient.Enable(ctx, &webauthnpb.EnableRequest{})
	if err != nil {
		t.Fatalf("WebAuthn.Enable: %v", err)
	}

	_, err = env.webAuthnClient.Disable(ctx, &webauthnpb.DisableRequest{})
	if err != nil {
		t.Fatalf("WebAuthn.Disable: %v", err)
	}
}

func TestWebAuthnAddRemoveVirtualAuthenticator(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.webAuthnClient.Enable(ctx, &webauthnpb.EnableRequest{})
	if err != nil {
		t.Fatalf("WebAuthn.Enable: %v", err)
	}
	defer env.webAuthnClient.Disable(ctx, &webauthnpb.DisableRequest{})

	addResp, err := env.webAuthnClient.AddVirtualAuthenticator(ctx, &webauthnpb.AddVirtualAuthenticatorRequest{
		Options: &webauthnpb.VirtualAuthenticatorOptions{
			Protocol:  webauthnpb.AuthenticatorProtocol_CTAP2,
			Transport: webauthnpb.AuthenticatorTransport_INTERNAL,
		},
	})
	if err != nil {
		t.Fatalf("AddVirtualAuthenticator: %v", err)
	}
	if addResp.AuthenticatorId == "" {
		t.Fatal("expected non-empty authenticator ID")
	}
	t.Logf("Added authenticator: %s", addResp.AuthenticatorId)

	_, err = env.webAuthnClient.RemoveVirtualAuthenticator(ctx, &webauthnpb.RemoveVirtualAuthenticatorRequest{
		AuthenticatorId: addResp.AuthenticatorId,
	})
	if err != nil {
		t.Fatalf("RemoveVirtualAuthenticator: %v", err)
	}
}

func TestWebAuthnGetCredentials(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.webAuthnClient.Enable(ctx, &webauthnpb.EnableRequest{})
	defer env.webAuthnClient.Disable(ctx, &webauthnpb.DisableRequest{})

	addResp, err := env.webAuthnClient.AddVirtualAuthenticator(ctx, &webauthnpb.AddVirtualAuthenticatorRequest{
		Options: &webauthnpb.VirtualAuthenticatorOptions{
			Protocol:           webauthnpb.AuthenticatorProtocol_CTAP2,
			Transport:          webauthnpb.AuthenticatorTransport_INTERNAL,
			HasResidentKey:     true,
			HasUserVerification: true,
			IsUserVerified:     true,
		},
	})
	if err != nil {
		t.Fatalf("AddVirtualAuthenticator: %v", err)
	}
	defer env.webAuthnClient.RemoveVirtualAuthenticator(ctx, &webauthnpb.RemoveVirtualAuthenticatorRequest{
		AuthenticatorId: addResp.AuthenticatorId,
	})

	resp, err := env.webAuthnClient.GetCredentials(ctx, &webauthnpb.GetCredentialsRequest{
		AuthenticatorId: addResp.AuthenticatorId,
	})
	if err != nil {
		t.Fatalf("GetCredentials: %v", err)
	}
	t.Logf("Credentials: %d", len(resp.Credentials))
}

// =============================================================
// PerformanceTimeline Domain Tests
// =============================================================

func TestPerformanceTimelineEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.performanceTimelineClient.Enable(ctx, &performancetimelinepb.EnableRequest{
		EventTypes: []string{"largest-contentful-paint", "layout-shift"},
	})
	if err != nil {
		t.Logf("PerformanceTimeline.Enable: %v", err)
	}
}

// =============================================================
// Preload Domain Tests
// =============================================================

func TestPreloadEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.preloadClient.Enable(ctx, &preloadpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Preload.Enable: %v", err)
	}

	_, err = env.preloadClient.Disable(ctx, &preloadpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Preload.Disable: %v", err)
	}
}
