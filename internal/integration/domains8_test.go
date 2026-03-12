package integration

import (
	"context"
	"testing"

	eventbreakpointspb "github.com/accretional/chromerpc/proto/cdp/eventbreakpoints"
	headlessexperimentalpb "github.com/accretional/chromerpc/proto/cdp/headlessexperimental"
	schemapb "github.com/accretional/chromerpc/proto/cdp/schema"
)

// =============================================================
// EventBreakpoints Domain Tests
// =============================================================

func TestEventBreakpointsSetRemove(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.eventBreakpointsClient.SetInstrumentationBreakpoint(ctx, &eventbreakpointspb.SetInstrumentationBreakpointRequest{
		EventName: "scriptFirstStatement",
	})
	if err != nil {
		t.Fatalf("SetInstrumentationBreakpoint: %v", err)
	}

	_, err = env.eventBreakpointsClient.RemoveInstrumentationBreakpoint(ctx, &eventbreakpointspb.RemoveInstrumentationBreakpointRequest{
		EventName: "scriptFirstStatement",
	})
	if err != nil {
		t.Fatalf("RemoveInstrumentationBreakpoint: %v", err)
	}
}

func TestEventBreakpointsDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.eventBreakpointsClient.Disable(ctx, &eventbreakpointspb.DisableRequest{})
	if err != nil {
		t.Fatalf("EventBreakpoints.Disable: %v", err)
	}
}

// =============================================================
// HeadlessExperimental Domain Tests
// =============================================================

func TestHeadlessExperimentalEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.headlessExperimentalClient.Enable(ctx, &headlessexperimentalpb.EnableRequest{})
	if err != nil {
		// HeadlessExperimental may not be available in all Chrome modes.
		t.Logf("HeadlessExperimental.Enable: %v", err)
		return
	}

	_, err = env.headlessExperimentalClient.Disable(ctx, &headlessexperimentalpb.DisableRequest{})
	if err != nil {
		t.Logf("HeadlessExperimental.Disable: %v", err)
	}
}

// =============================================================
// Schema Domain Tests
// =============================================================

func TestSchemaGetDomains(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	resp, err := env.schemaClient.GetDomains(ctx, &schemapb.GetDomainsRequest{})
	if err != nil {
		// Schema.getDomains may not be available in newer Chrome versions.
		t.Logf("Schema.GetDomains: %v", err)
		return
	}
	t.Logf("Schema domains: %d", len(resp.Domains))
	for i, d := range resp.Domains {
		if i < 5 {
			t.Logf("  %s v%s", d.Name, d.Version)
		}
	}
}
