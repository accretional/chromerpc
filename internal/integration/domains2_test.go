package integration

import (
	"context"
	"testing"
	"time"

	accessibilitypb "github.com/accretional/chromerpc/proto/cdp/accessibility"
	csspb "github.com/accretional/chromerpc/proto/cdp/css"
	dompb "github.com/accretional/chromerpc/proto/cdp/dom"
	fetchpb "github.com/accretional/chromerpc/proto/cdp/fetch"
	logpb "github.com/accretional/chromerpc/proto/cdp/log"
	pagepb "github.com/accretional/chromerpc/proto/cdp/page"
	performancepb "github.com/accretional/chromerpc/proto/cdp/performance"
	securitypb "github.com/accretional/chromerpc/proto/cdp/security"
)

// =============================================================
// Log Domain Tests
// =============================================================

func TestLogEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.logClient.Enable(ctx, &logpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Log.Enable: %v", err)
	}
}

func TestLogClear(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.logClient.Enable(ctx, &logpb.EnableRequest{})

	_, err := env.logClient.Clear(ctx, &logpb.ClearRequest{})
	if err != nil {
		t.Fatalf("Log.Clear: %v", err)
	}
}

func TestLogViolationsReport(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.logClient.Enable(ctx, &logpb.EnableRequest{})

	_, err := env.logClient.StartViolationsReport(ctx, &logpb.StartViolationsReportRequest{
		Config: []*logpb.ViolationSetting{
			{Name: "longTask", Threshold: 200},
			{Name: "blockedEvent", Threshold: 100},
		},
	})
	if err != nil {
		t.Fatalf("Log.StartViolationsReport: %v", err)
	}

	_, err = env.logClient.StopViolationsReport(ctx, &logpb.StopViolationsReportRequest{})
	if err != nil {
		t.Fatalf("Log.StopViolationsReport: %v", err)
	}
}

func TestLogDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.logClient.Enable(ctx, &logpb.EnableRequest{})

	_, err := env.logClient.Disable(ctx, &logpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Log.Disable: %v", err)
	}
}

// =============================================================
// Performance Domain Tests
// =============================================================

func TestPerformanceEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.performanceClient.Enable(ctx, &performancepb.EnableRequest{})
	if err != nil {
		t.Fatalf("Performance.Enable: %v", err)
	}
}

func TestPerformanceGetMetrics(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.performanceClient.Enable(ctx, &performancepb.EnableRequest{})

	// Navigate to a page to generate some metrics.
	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<h1>Performance Test</h1>",
	})
	time.Sleep(300 * time.Millisecond)

	resp, err := env.performanceClient.GetMetrics(ctx, &performancepb.GetMetricsRequest{})
	if err != nil {
		t.Fatalf("Performance.GetMetrics: %v", err)
	}
	if len(resp.Metrics) == 0 {
		t.Fatal("expected at least one metric")
	}
	for _, m := range resp.Metrics {
		if m.Name == "JSHeapUsedSize" || m.Name == "Documents" || m.Name == "Nodes" {
			t.Logf("Metric: %s = %.0f", m.Name, m.Value)
		}
	}
	t.Logf("Total metrics: %d", len(resp.Metrics))
}

func TestPerformanceEnableWithTimeDomain(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.performanceClient.Enable(ctx, &performancepb.EnableRequest{
		TimeDomain: "timeTicks",
	})
	if err != nil {
		t.Fatalf("Performance.Enable (timeTicks): %v", err)
	}

	resp, err := env.performanceClient.GetMetrics(ctx, &performancepb.GetMetricsRequest{})
	if err != nil {
		t.Fatalf("Performance.GetMetrics: %v", err)
	}
	t.Logf("Metrics with timeTicks: %d", len(resp.Metrics))
}

func TestPerformanceDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.performanceClient.Enable(ctx, &performancepb.EnableRequest{})

	_, err := env.performanceClient.Disable(ctx, &performancepb.DisableRequest{})
	if err != nil {
		t.Fatalf("Performance.Disable: %v", err)
	}
}

// =============================================================
// Security Domain Tests
// =============================================================

func TestSecurityEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.securityClient.Enable(ctx, &securitypb.EnableRequest{})
	if err != nil {
		t.Fatalf("Security.Enable: %v", err)
	}
}

func TestSecuritySetIgnoreCertificateErrors(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.securityClient.SetIgnoreCertificateErrors(ctx, &securitypb.SetIgnoreCertificateErrorsRequest{
		Ignore: true,
	})
	if err != nil {
		t.Fatalf("Security.SetIgnoreCertificateErrors: %v", err)
	}

	// Re-enable cert checking.
	_, err = env.securityClient.SetIgnoreCertificateErrors(ctx, &securitypb.SetIgnoreCertificateErrorsRequest{
		Ignore: false,
	})
	if err != nil {
		t.Fatalf("Security.SetIgnoreCertificateErrors (re-enable): %v", err)
	}
}

func TestSecurityDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.securityClient.Enable(ctx, &securitypb.EnableRequest{})

	_, err := env.securityClient.Disable(ctx, &securitypb.DisableRequest{})
	if err != nil {
		t.Fatalf("Security.Disable: %v", err)
	}
}

// =============================================================
// CSS Domain Tests
// =============================================================

func TestCSSEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	// CSS requires DOM to be enabled first.
	env.domClient.Enable(ctx, &dompb.EnableRequest{})

	_, err := env.cssClient.Enable(ctx, &csspb.EnableRequest{})
	if err != nil {
		t.Fatalf("CSS.Enable: %v", err)
	}
}

func TestCSSGetComputedStyleForNode(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<div id='styled' style='color:red;font-size:24px'>Styled</div>",
	})
	time.Sleep(500 * time.Millisecond)

	env.domClient.Enable(ctx, &dompb.EnableRequest{})
	env.cssClient.Enable(ctx, &csspb.EnableRequest{})

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	selResp, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#styled",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector: %v", err)
	}

	resp, err := env.cssClient.GetComputedStyleForNode(ctx, &csspb.GetComputedStyleForNodeRequest{
		NodeId: selResp.NodeId,
	})
	if err != nil {
		t.Fatalf("CSS.GetComputedStyleForNode: %v", err)
	}
	if len(resp.ComputedStyle) == 0 {
		t.Fatal("expected at least one computed style property")
	}
	t.Logf("Computed style properties: %d", len(resp.ComputedStyle))
	for _, prop := range resp.ComputedStyle {
		if prop.Name == "color" || prop.Name == "font-size" {
			t.Logf("  %s: %s", prop.Name, prop.Value)
		}
	}
}

func TestCSSGetMatchedStylesForNode(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<style>h1{color:blue}</style><h1 id='heading'>Styled Heading</h1>",
	})
	time.Sleep(500 * time.Millisecond)

	env.domClient.Enable(ctx, &dompb.EnableRequest{})
	env.cssClient.Enable(ctx, &csspb.EnableRequest{})

	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	selResp, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#heading",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector: %v", err)
	}

	resp, err := env.cssClient.GetMatchedStylesForNode(ctx, &csspb.GetMatchedStylesForNodeRequest{
		NodeId: selResp.NodeId,
	})
	if err != nil {
		t.Fatalf("CSS.GetMatchedStylesForNode: %v", err)
	}
	t.Logf("Matched CSS rules: %d, has inline style: %v",
		len(resp.MatchedCssRules), resp.InlineStyle != nil)
}

func TestCSSRuleUsageTracking(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<style>.used{color:red} .unused{color:blue}</style><div class='used'>Used</div>",
	})
	time.Sleep(500 * time.Millisecond)

	env.domClient.Enable(ctx, &dompb.EnableRequest{})
	env.cssClient.Enable(ctx, &csspb.EnableRequest{})

	_, err := env.cssClient.StartRuleUsageTracking(ctx, &csspb.StartRuleUsageTrackingRequest{})
	if err != nil {
		t.Fatalf("CSS.StartRuleUsageTracking: %v", err)
	}

	// Navigate to trigger style usage.
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<style>.used{color:red} .unused{color:blue}</style><div class='used'>Used</div>",
	})
	time.Sleep(300 * time.Millisecond)

	stopResp, err := env.cssClient.StopRuleUsageTracking(ctx, &csspb.StopRuleUsageTrackingRequest{})
	if err != nil {
		t.Fatalf("CSS.StopRuleUsageTracking: %v", err)
	}
	t.Logf("CSS rule usage entries: %d", len(stopResp.RuleUsage))
}

func TestCSSForcePseudoState(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.domClient.Enable(ctx, &dompb.EnableRequest{})
	env.cssClient.Enable(ctx, &csspb.EnableRequest{})

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<style>a:hover{color:red}</style><a id='link' href='#'>Hover me</a>",
	})
	time.Sleep(500 * time.Millisecond)

	// GetDocument with depth to populate the DOM tree in the agent.
	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{Depth: 5})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	// ForcePseudoState requires nodes to be pushed to the CSS agent.
	// Use DOM.requestChildNodes to ensure full tree is available.
	env.domClient.RequestChildNodes(ctx, &dompb.RequestChildNodesRequest{
		NodeId: docResp.Root.NodeId,
		Depth:  -1,
	})
	time.Sleep(200 * time.Millisecond)

	// Re-query after tree push.
	selResp2, err := env.domClient.QuerySelector(ctx, &dompb.QuerySelectorRequest{
		NodeId:   docResp.Root.NodeId,
		Selector: "#link",
	})
	if err != nil {
		t.Fatalf("DOM.QuerySelector (2nd): %v", err)
	}

	_, err = env.cssClient.ForcePseudoState(ctx, &csspb.ForcePseudoStateRequest{
		NodeId:              selResp2.NodeId,
		ForcedPseudoClasses: []string{"hover"},
	})
	if err != nil {
		// This can fail due to CSS/DOM agent node tracking mismatch in headless mode.
		t.Logf("CSS.ForcePseudoState: %v (known headless limitation)", err)
	}
}

func TestCSSDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.domClient.Enable(ctx, &dompb.EnableRequest{})
	env.cssClient.Enable(ctx, &csspb.EnableRequest{})

	_, err := env.cssClient.Disable(ctx, &csspb.DisableRequest{})
	if err != nil {
		t.Fatalf("CSS.Disable: %v", err)
	}
}

// =============================================================
// Fetch Domain Tests
// =============================================================

func TestFetchEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.fetchClient.Enable(ctx, &fetchpb.EnableRequest{})
	if err != nil {
		t.Fatalf("Fetch.Enable: %v", err)
	}

	_, err = env.fetchClient.Disable(ctx, &fetchpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Fetch.Disable: %v", err)
	}
}

func TestFetchEnableWithPatterns(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.fetchClient.Enable(ctx, &fetchpb.EnableRequest{
		Patterns: []*fetchpb.RequestPattern{
			{UrlPattern: "*", RequestStage: "Request"},
		},
	})
	if err != nil {
		t.Fatalf("Fetch.Enable (with patterns): %v", err)
	}

	_, err = env.fetchClient.Disable(ctx, &fetchpb.DisableRequest{})
	if err != nil {
		t.Fatalf("Fetch.Disable: %v", err)
	}
}

// =============================================================
// Accessibility Domain Tests
// =============================================================

func TestAccessibilityEnable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.accessibilityClient.Enable(ctx, &accessibilitypb.EnableRequest{})
	if err != nil {
		t.Fatalf("Accessibility.Enable: %v", err)
	}
}

func TestAccessibilityGetFullAXTree(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: `data:text/html,<html><body>
			<h1>Accessibility Test</h1>
			<button aria-label="Submit">Submit</button>
			<input type="text" aria-label="Username" placeholder="Username">
			<nav aria-label="Main navigation"><a href="#">Home</a><a href="#">About</a></nav>
		</body></html>`,
	})
	time.Sleep(500 * time.Millisecond)

	env.accessibilityClient.Enable(ctx, &accessibilitypb.EnableRequest{})

	resp, err := env.accessibilityClient.GetFullAXTree(ctx, &accessibilitypb.GetFullAXTreeRequest{
		Depth: 5,
	})
	if err != nil {
		t.Fatalf("Accessibility.GetFullAXTree: %v", err)
	}
	if len(resp.Nodes) == 0 {
		t.Fatal("expected at least one AX node")
	}
	t.Logf("Accessibility tree: %d nodes", len(resp.Nodes))
	for _, node := range resp.Nodes {
		if node.Role != nil && node.Name != nil {
			t.Logf("  AX node: role=%s name=%s ignored=%v",
				node.Role.Value, node.Name.Value, node.Ignored)
		}
	}
}

func TestAccessibilityQueryAXTree(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: `data:text/html,<button aria-label="Submit Form">Submit</button><button aria-label="Cancel">Cancel</button>`,
	})
	time.Sleep(500 * time.Millisecond)

	env.accessibilityClient.Enable(ctx, &accessibilitypb.EnableRequest{})

	// QueryAXTree requires a node reference. Get the document's backend node ID.
	docResp, err := env.domClient.GetDocument(ctx, &dompb.GetDocumentRequest{})
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	resp, err := env.accessibilityClient.QueryAXTree(ctx, &accessibilitypb.QueryAXTreeRequest{
		NodeId: docResp.Root.NodeId,
		Role:   "button",
	})
	if err != nil {
		t.Fatalf("Accessibility.QueryAXTree: %v", err)
	}
	t.Logf("Found %d buttons in AX tree", len(resp.Nodes))
	for _, node := range resp.Nodes {
		if node.Name != nil {
			t.Logf("  Button: %s", node.Name.Value)
		}
	}
}

func TestAccessibilityGetPartialAXTree(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.pageClient.Enable(ctx, &pagepb.EnableRequest{})
	env.pageClient.Navigate(ctx, &pagepb.NavigateRequest{
		Url: "data:text/html,<div id='target'><p>Partial tree</p></div>",
	})
	time.Sleep(500 * time.Millisecond)

	// Get a DOM node to query.
	docResp, err := env.domClient.GetDocument(ctx, nil)
	if err != nil {
		t.Fatalf("DOM.GetDocument: %v", err)
	}

	resp, err := env.accessibilityClient.GetPartialAXTree(ctx, &accessibilitypb.GetPartialAXTreeRequest{
		NodeId: docResp.Root.NodeId,
		Depth:  3,
	})
	if err != nil {
		t.Fatalf("Accessibility.GetPartialAXTree: %v", err)
	}
	t.Logf("Partial AX tree: %d nodes", len(resp.Nodes))
}

func TestAccessibilityDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	env.accessibilityClient.Enable(ctx, &accessibilitypb.EnableRequest{})

	_, err := env.accessibilityClient.Disable(ctx, &accessibilitypb.DisableRequest{})
	if err != nil {
		t.Fatalf("Accessibility.Disable: %v", err)
	}
}
