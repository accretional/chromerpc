// Package headlessbrowser implements the HeadlessBrowserService gRPC service,
// executing automation sequences by orchestrating lower-level CDP commands.
package headlessbrowser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/accretional/chromerpc/internal/cdpclient"
	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

// Server implements the HeadlessBrowserService gRPC service.
type Server struct {
	pb.UnimplementedHeadlessBrowserServiceServer
	client *cdpclient.Client
}

// New creates a new HeadlessBrowser gRPC server backed by the given CDP client.
func New(client *cdpclient.Client) *Server {
	return &Server{client: client}
}

func (s *Server) send(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.Send(ctx, method, params)
}

func (s *Server) sendSession(ctx context.Context, method string, params interface{}, sessionID string) (json.RawMessage, error) {
	if sessionID != "" {
		return s.client.SendWithSession(ctx, method, params, sessionID)
	}
	return s.client.Send(ctx, method, params)
}

// sendBrowser sends a command at the browser level (no session).
func (s *Server) sendBrowser(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	return s.client.SendWithSession(ctx, method, params, "")
}

// RunAutomation executes a sequence of automation steps in order.
func (s *Server) RunAutomation(ctx context.Context, req *pb.AutomationSequence) (*pb.AutomationResult, error) {
	log.Printf("Running automation: %s (%d steps)", req.Name, len(req.Steps))

	result := &pb.AutomationResult{Success: true}

	for i, step := range req.Steps {
		label := step.Label
		if label == "" {
			label = fmt.Sprintf("step_%d", i+1)
		}

		stepResult, err := s.executeStep(ctx, step)
		if err != nil {
			stepResult = &pb.StepResult{
				Label:   label,
				Success: false,
				Error:   err.Error(),
			}
			result.StepResults = append(result.StepResults, stepResult)
			result.Success = false
			result.Error = fmt.Sprintf("step %q failed: %v", label, err)
			log.Printf("  [%d] %s: FAILED: %v", i+1, label, err)
			return result, nil
		}

		stepResult.Label = label
		stepResult.Success = true
		result.StepResults = append(result.StepResults, stepResult)
		log.Printf("  [%d] %s: OK", i+1, label)
	}

	return result, nil
}

// ExecuteStep runs a single automation step.
func (s *Server) ExecuteStep(ctx context.Context, req *pb.AutomationStep) (*pb.StepResult, error) {
	label := req.Label
	if label == "" {
		label = "step"
	}

	result, err := s.executeStep(ctx, req)
	if err != nil {
		return &pb.StepResult{
			Label:   label,
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	result.Label = label
	result.Success = true
	return result, nil
}

func (s *Server) executeStep(ctx context.Context, step *pb.AutomationStep) (*pb.StepResult, error) {
	switch a := step.Action.(type) {
	case *pb.AutomationStep_SetViewport:
		return s.doSetViewport(ctx, a.SetViewport)
	case *pb.AutomationStep_Navigate:
		return s.doNavigate(ctx, a.Navigate)
	case *pb.AutomationStep_Wait:
		return s.doWait(ctx, a.Wait)
	case *pb.AutomationStep_Screenshot:
		return s.doScreenshot(ctx, a.Screenshot)
	case *pb.AutomationStep_EvaluateScript:
		return s.doEvaluateScript(ctx, a.EvaluateScript)
	case *pb.AutomationStep_Click:
		return s.doClick(ctx, a.Click)
	case *pb.AutomationStep_TypeText:
		return s.doTypeText(ctx, a.TypeText)
	case *pb.AutomationStep_WaitForSelector:
		return s.doWaitForSelector(ctx, a.WaitForSelector)
	case *pb.AutomationStep_Reload:
		return s.doReload(ctx, a.Reload)
	case *pb.AutomationStep_ScrollTo:
		return s.doScrollTo(ctx, a.ScrollTo)
	case *pb.AutomationStep_TypeKeyByKey:
		return s.doTypeKeyByKey(ctx, a.TypeKeyByKey)
	case *pb.AutomationStep_PressKey:
		return s.doPressKey(ctx, a.PressKey)
	case *pb.AutomationStep_FullPageScreenshot:
		return s.doFullPageScreenshot(ctx, a.FullPageScreenshot)
	case *pb.AutomationStep_OpenTab:
		return s.doOpenTab(ctx, a.OpenTab)
	case *pb.AutomationStep_CloseTab:
		return s.doCloseTab(ctx, a.CloseTab)
	case *pb.AutomationStep_ListTabs:
		return s.doListTabs(ctx)
	case *pb.AutomationStep_SwitchTab:
		return s.doSwitchTab(ctx, a.SwitchTab)
	case *pb.AutomationStep_DownloadFile:
		return s.doDownloadFile(ctx, a.DownloadFile)
	case *pb.AutomationStep_Hover:
		return s.doHover(ctx, a.Hover)
	case *pb.AutomationStep_Drag:
		return s.doDrag(ctx, a.Drag)
	case *pb.AutomationStep_WaitForStable:
		return s.doWaitForStable(ctx, a.WaitForStable)
	default:
		return nil, fmt.Errorf("unknown action type")
	}
}

func (s *Server) doSetViewport(ctx context.Context, v *pb.SetViewport) (*pb.StepResult, error) {
	scaleFactor := v.DeviceScaleFactor
	if scaleFactor == 0 {
		scaleFactor = 1
	}
	params := map[string]interface{}{
		"width":             v.Width,
		"height":            v.Height,
		"deviceScaleFactor": scaleFactor,
		"mobile":            v.Mobile,
	}
	if _, err := s.send(ctx, "Emulation.setDeviceMetricsOverride", params); err != nil {
		return nil, fmt.Errorf("Emulation.setDeviceMetricsOverride: %w", err)
	}
	return &pb.StepResult{}, nil
}

func (s *Server) doNavigate(ctx context.Context, n *pb.Navigate) (*pb.StepResult, error) {
	if _, err := s.send(ctx, "Page.enable", nil); err != nil {
		return nil, fmt.Errorf("Page.enable: %w", err)
	}

	params := map[string]interface{}{"url": n.Url}
	result, err := s.send(ctx, "Page.navigate", params)
	if err != nil {
		return nil, fmt.Errorf("Page.navigate: %w", err)
	}

	var resp struct {
		ErrorText string `json:"errorText"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Page.navigate unmarshal: %w", err)
	}
	if resp.ErrorText != "" {
		return nil, fmt.Errorf("navigation error: %s", resp.ErrorText)
	}
	return &pb.StepResult{}, nil
}

func (s *Server) doWait(_ context.Context, w *pb.Wait) (*pb.StepResult, error) {
	time.Sleep(time.Duration(w.Milliseconds) * time.Millisecond)
	return &pb.StepResult{}, nil
}

func (s *Server) doScreenshot(ctx context.Context, ss *pb.Screenshot) (*pb.StepResult, error) {
	params := map[string]interface{}{}

	format := strings.ToLower(ss.Format)
	if format == "" {
		format = "png"
	}
	params["format"] = format

	if ss.Quality > 0 {
		params["quality"] = ss.Quality
	}
	if ss.FullPage {
		params["captureBeyondViewport"] = true
	}

	result, err := s.send(ctx, "Page.captureScreenshot", params)
	if err != nil {
		return nil, fmt.Errorf("Page.captureScreenshot: %w", err)
	}

	var resp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("captureScreenshot unmarshal: %w", err)
	}

	imageData, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("captureScreenshot decode: %w", err)
	}

	if ss.OutputPath != "" {
		if err := os.MkdirAll(filepath.Dir(ss.OutputPath), 0755); err != nil {
			return nil, fmt.Errorf("mkdir for screenshot: %w", err)
		}
		if err := os.WriteFile(ss.OutputPath, imageData, 0644); err != nil {
			return nil, fmt.Errorf("write screenshot %s: %w", ss.OutputPath, err)
		}
		log.Printf("    Screenshot saved: %s (%d bytes)", ss.OutputPath, len(imageData))
	}

	return &pb.StepResult{ScreenshotData: imageData}, nil
}

func (s *Server) doEvaluateScript(ctx context.Context, es *pb.EvaluateScript) (*pb.StepResult, error) {
	params := map[string]interface{}{
		"expression":    es.Expression,
		"returnByValue": true,
	}
	result, err := s.send(ctx, "Runtime.evaluate", params)
	if err != nil {
		return nil, fmt.Errorf("Runtime.evaluate: %w", err)
	}

	var resp struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails *struct {
			Text string `json:"text"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("Runtime.evaluate unmarshal: %w", err)
	}
	if resp.ExceptionDetails != nil {
		return nil, fmt.Errorf("script exception: %s", resp.ExceptionDetails.Text)
	}

	return &pb.StepResult{ScriptResult: string(resp.Result.Value)}, nil
}

func (s *Server) doClick(ctx context.Context, c *pb.Click) (*pb.StepResult, error) {
	x, y := c.X, c.Y

	if c.Selector != "" {
		expr := fmt.Sprintf(`(function() {
			var el = document.querySelector(%q);
			if (!el) return null;
			var r = el.getBoundingClientRect();
			return {x: r.x + r.width/2, y: r.y + r.height/2};
		})()`, c.Selector)
		params := map[string]interface{}{"expression": expr, "returnByValue": true}
		result, err := s.send(ctx, "Runtime.evaluate", params)
		if err != nil {
			return nil, fmt.Errorf("resolve selector: %w", err)
		}
		var resp struct {
			Result struct {
				Value *struct {
					X float64 `json:"x"`
					Y float64 `json:"y"`
				} `json:"value"`
			} `json:"result"`
		}
		if err := json.Unmarshal(result, &resp); err != nil || resp.Result.Value == nil {
			return nil, fmt.Errorf("selector %q not found", c.Selector)
		}
		x, y = resp.Result.Value.X, resp.Result.Value.Y
	}

	button := c.Button
	if button == "" {
		button = "left"
	}

	for _, evType := range []string{"mousePressed", "mouseReleased"} {
		params := map[string]interface{}{
			"type":       evType,
			"x":          x,
			"y":          y,
			"button":     button,
			"clickCount": 1,
		}
		if _, err := s.send(ctx, "Input.dispatchMouseEvent", params); err != nil {
			return nil, fmt.Errorf("Input.dispatchMouseEvent(%s): %w", evType, err)
		}
	}
	return &pb.StepResult{}, nil
}

func (s *Server) doTypeText(ctx context.Context, t *pb.TypeText) (*pb.StepResult, error) {
	if t.Selector != "" {
		expr := fmt.Sprintf(`document.querySelector(%q).focus()`, t.Selector)
		params := map[string]interface{}{"expression": expr}
		if _, err := s.send(ctx, "Runtime.evaluate", params); err != nil {
			return nil, fmt.Errorf("focus selector: %w", err)
		}
	}

	params := map[string]interface{}{"text": t.Text}
	if _, err := s.send(ctx, "Input.insertText", params); err != nil {
		return nil, fmt.Errorf("Input.insertText: %w", err)
	}
	return &pb.StepResult{}, nil
}

func (s *Server) doWaitForSelector(ctx context.Context, ws *pb.WaitForSelector) (*pb.StepResult, error) {
	timeout := time.Duration(ws.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	deadline := time.Now().Add(timeout)

	expr := fmt.Sprintf(`document.querySelector(%q) !== null`, ws.Selector)
	for time.Now().Before(deadline) {
		params := map[string]interface{}{"expression": expr, "returnByValue": true}
		result, err := s.send(ctx, "Runtime.evaluate", params)
		if err == nil {
			var resp struct {
				Result struct {
					Value bool `json:"value"`
				} `json:"result"`
			}
			if json.Unmarshal(result, &resp) == nil && resp.Result.Value {
				return &pb.StepResult{}, nil
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("selector %q not found within %v", ws.Selector, timeout)
}

func (s *Server) doReload(ctx context.Context, r *pb.Reload) (*pb.StepResult, error) {
	params := map[string]interface{}{}
	if r.IgnoreCache {
		params["ignoreCache"] = true
	}
	if _, err := s.send(ctx, "Page.reload", params); err != nil {
		return nil, fmt.Errorf("Page.reload: %w", err)
	}
	return &pb.StepResult{}, nil
}

func (s *Server) doScrollTo(ctx context.Context, st *pb.ScrollTo) (*pb.StepResult, error) {
	expr := fmt.Sprintf("window.scrollTo(%f, %f)", st.X, st.Y)
	params := map[string]interface{}{"expression": expr}
	if _, err := s.send(ctx, "Runtime.evaluate", params); err != nil {
		return nil, fmt.Errorf("scrollTo: %w", err)
	}
	return &pb.StepResult{}, nil
}

// doTypeKeyByKey types each character individually with delays.
func (s *Server) doTypeKeyByKey(ctx context.Context, t *pb.TypeKeyByKey) (*pb.StepResult, error) {
	if t.Selector != "" {
		expr := fmt.Sprintf(`document.querySelector(%q).focus()`, t.Selector)
		params := map[string]interface{}{"expression": expr}
		if _, err := s.send(ctx, "Runtime.evaluate", params); err != nil {
			return nil, fmt.Errorf("focus selector: %w", err)
		}
	}

	delay := time.Duration(t.DelayMs) * time.Millisecond
	if delay == 0 {
		delay = 50 * time.Millisecond
	}

	for _, ch := range t.Text {
		text := string(ch)
		// keyDown
		params := map[string]interface{}{
			"type":           "keyDown",
			"text":           text,
			"unmodifiedText": text,
			"key":            text,
		}
		if _, err := s.send(ctx, "Input.dispatchKeyEvent", params); err != nil {
			return nil, fmt.Errorf("keyDown %q: %w", text, err)
		}
		// keyUp
		params = map[string]interface{}{
			"type": "keyUp",
			"key":  text,
		}
		if _, err := s.send(ctx, "Input.dispatchKeyEvent", params); err != nil {
			return nil, fmt.Errorf("keyUp %q: %w", text, err)
		}
		time.Sleep(delay)
	}
	return &pb.StepResult{}, nil
}

// doPressKey dispatches a special key event (Enter, Tab, Escape, etc.).
func (s *Server) doPressKey(ctx context.Context, k *pb.PressKey) (*pb.StepResult, error) {
	key := k.Key

	// Map key names to key codes and text representations.
	keyCode := 0
	text := ""
	switch strings.ToLower(key) {
	case "enter", "return":
		key = "Enter"
		keyCode = 13
		text = "\r"
	case "tab":
		key = "Tab"
		keyCode = 9
		text = "\t"
	case "escape", "esc":
		key = "Escape"
		keyCode = 27
	case "backspace":
		key = "Backspace"
		keyCode = 8
	case "arrowdown":
		key = "ArrowDown"
		keyCode = 40
	case "arrowup":
		key = "ArrowUp"
		keyCode = 38
	case "arrowleft":
		key = "ArrowLeft"
		keyCode = 37
	case "arrowright":
		key = "ArrowRight"
		keyCode = 39
	case "space":
		key = " "
		keyCode = 32
		text = " "
	}

	// keyDown
	params := map[string]interface{}{
		"type":                  "rawKeyDown",
		"key":                   key,
		"windowsVirtualKeyCode": keyCode,
		"nativeVirtualKeyCode":  keyCode,
	}
	if text != "" {
		params["text"] = text
		params["unmodifiedText"] = text
	}
	if _, err := s.send(ctx, "Input.dispatchKeyEvent", params); err != nil {
		return nil, fmt.Errorf("keyDown %q: %w", key, err)
	}

	// keyUp
	params = map[string]interface{}{
		"type":                  "keyUp",
		"key":                   key,
		"windowsVirtualKeyCode": keyCode,
		"nativeVirtualKeyCode":  keyCode,
	}
	if _, err := s.send(ctx, "Input.dispatchKeyEvent", params); err != nil {
		return nil, fmt.Errorf("keyUp %q: %w", key, err)
	}

	return &pb.StepResult{}, nil
}

// doFullPageScreenshot captures the entire scrollable page.
func (s *Server) doFullPageScreenshot(ctx context.Context, fps *pb.FullPageScreenshot) (*pb.StepResult, error) {
	// Get the full page dimensions via layout metrics.
	metricsResult, err := s.send(ctx, "Page.getLayoutMetrics", nil)
	if err != nil {
		return nil, fmt.Errorf("Page.getLayoutMetrics: %w", err)
	}

	var metrics struct {
		CSSContentSize struct {
			Width  float64 `json:"width"`
			Height float64 `json:"height"`
		} `json:"cssContentSize"`
	}
	if err := json.Unmarshal(metricsResult, &metrics); err != nil {
		return nil, fmt.Errorf("unmarshal metrics: %w", err)
	}

	width := metrics.CSSContentSize.Width
	height := metrics.CSSContentSize.Height
	if width == 0 {
		width = 1280
	}
	if height == 0 {
		height = 800
	}

	// Cap height to avoid enormous screenshots.
	maxHeight := 16384.0
	if height > maxHeight {
		height = maxHeight
	}

	format := strings.ToLower(fps.Format)
	if format == "" {
		format = "png"
	}

	// Use clip to capture the full content area.
	params := map[string]interface{}{
		"format": format,
		"clip": map[string]interface{}{
			"x":      0,
			"y":      0,
			"width":  width,
			"height": height,
			"scale":  1,
		},
		"captureBeyondViewport": true,
	}
	if fps.Quality > 0 {
		params["quality"] = fps.Quality
	}

	result, err := s.send(ctx, "Page.captureScreenshot", params)
	if err != nil {
		return nil, fmt.Errorf("Page.captureScreenshot (full page): %w", err)
	}

	var resp struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("captureScreenshot unmarshal: %w", err)
	}

	imageData, err := base64.StdEncoding.DecodeString(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("captureScreenshot decode: %w", err)
	}

	if fps.OutputPath != "" {
		if err := os.MkdirAll(filepath.Dir(fps.OutputPath), 0755); err != nil {
			return nil, fmt.Errorf("mkdir for screenshot: %w", err)
		}
		if err := os.WriteFile(fps.OutputPath, imageData, 0644); err != nil {
			return nil, fmt.Errorf("write screenshot %s: %w", fps.OutputPath, err)
		}
		log.Printf("    Full-page screenshot saved: %s (%d bytes)", fps.OutputPath, len(imageData))
	}

	return &pb.StepResult{ScreenshotData: imageData}, nil
}

// doOpenTab creates a new browser tab and returns the target ID.
func (s *Server) doOpenTab(ctx context.Context, ot *pb.OpenTab) (*pb.StepResult, error) {
	params := map[string]interface{}{
		"url": ot.Url,
	}
	result, err := s.sendBrowser(ctx, "Target.createTarget", params)
	if err != nil {
		return nil, fmt.Errorf("Target.createTarget: %w", err)
	}

	var resp struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal createTarget: %w", err)
	}

	log.Printf("    Opened tab: %s", resp.TargetID)
	return &pb.StepResult{ScriptResult: resp.TargetID}, nil
}

// doCloseTab closes a tab by target ID.
func (s *Server) doCloseTab(ctx context.Context, ct *pb.CloseTab) (*pb.StepResult, error) {
	targetID := ct.TargetId
	if targetID == "" {
		return nil, fmt.Errorf("target_id required for close_tab")
	}
	params := map[string]interface{}{
		"targetId": targetID,
	}
	if _, err := s.sendBrowser(ctx, "Target.closeTarget", params); err != nil {
		return nil, fmt.Errorf("Target.closeTarget: %w", err)
	}
	log.Printf("    Closed tab: %s", targetID)
	return &pb.StepResult{}, nil
}

// doListTabs returns a JSON array of all open browser tab target IDs.
func (s *Server) doListTabs(ctx context.Context) (*pb.StepResult, error) {
	result, err := s.sendBrowser(ctx, "Target.getTargets", nil)
	if err != nil {
		return nil, fmt.Errorf("Target.getTargets: %w", err)
	}
	var resp struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("list_tabs: unmarshal: %w", err)
	}
	var ids []string
	for _, t := range resp.TargetInfos {
		if t.Type == "page" {
			ids = append(ids, `"`+t.TargetID+`"`)
		}
	}
	out := "[" + strings.Join(ids, ",") + "]"
	return &pb.StepResult{ScriptResult: out}, nil
}

// doSwitchTab attaches to a target and switches the default session.
func (s *Server) doSwitchTab(ctx context.Context, st *pb.SwitchTab) (*pb.StepResult, error) {
	params := map[string]interface{}{
		"targetId": st.TargetId,
		"flatten":  true,
	}
	result, err := s.sendBrowser(ctx, "Target.attachToTarget", params)
	if err != nil {
		return nil, fmt.Errorf("Target.attachToTarget: %w", err)
	}

	var resp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(result, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal attach: %w", err)
	}

	s.client.SetSessionID(resp.SessionID)
	log.Printf("    Switched to tab %s (session %s)", st.TargetId, resp.SessionID)
	return &pb.StepResult{ScriptResult: resp.SessionID}, nil
}

// doDownloadFile opens a URL in a new tab (like a user clicking a link),
// sets the browser download directory, finds and clicks the download button,
// then waits for the file to appear on disk.
func (s *Server) doDownloadFile(ctx context.Context, df *pb.DownloadFile) (*pb.StepResult, error) {
	outputDir := filepath.Dir(df.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("mkdir: %w", err)
	}

	// Open the URL in a new tab.
	createResult, err := s.sendBrowser(ctx, "Target.createTarget", map[string]interface{}{
		"url": df.Url,
	})
	if err != nil {
		return nil, fmt.Errorf("create tab: %w", err)
	}
	var createResp struct {
		TargetID string `json:"targetId"`
	}
	if err := json.Unmarshal(createResult, &createResp); err != nil {
		return nil, fmt.Errorf("unmarshal createTarget: %w", err)
	}
	targetID := createResp.TargetID
	defer func() {
		s.sendBrowser(ctx, "Target.closeTarget", map[string]interface{}{"targetId": targetID})
	}()

	// Attach to the new tab.
	attachResult, err := s.sendBrowser(ctx, "Target.attachToTarget", map[string]interface{}{
		"targetId": targetID,
		"flatten":  true,
	})
	if err != nil {
		return nil, fmt.Errorf("attach: %w", err)
	}
	var attachResp struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(attachResult, &attachResp); err != nil {
		return nil, fmt.Errorf("unmarshal attach: %w", err)
	}
	dlSession := attachResp.SessionID

	// Set download behavior at browser level to auto-save to our directory.
	s.sendBrowser(ctx, "Browser.setDownloadBehavior", map[string]interface{}{
		"behavior":      "allowAndName",
		"downloadPath":  outputDir,
		"eventsEnabled": true,
	})
	// Also set on the page session.
	s.sendSession(ctx, "Page.setDownloadBehavior", map[string]interface{}{
		"behavior":     "allow",
		"downloadPath": outputDir,
	}, dlSession)

	// Wait for the page to load.
	time.Sleep(4 * time.Second)

	// Try to find and click a download button.
	clickExpr := `(function(){
		// pdf.js download button
		var btn = document.getElementById('download');
		if(btn) { btn.click(); return 'clicked #download'; }
		// secondaryDownload in pdf.js
		btn = document.getElementById('secondaryDownload');
		if(btn) { btn.click(); return 'clicked #secondaryDownload'; }
		// Button with download in id/class
		btn = document.querySelector('button[id*="download"], a[id*="download"], [class*="download"] button, [class*="download"] a');
		if(btn) { btn.click(); return 'clicked download button'; }
		// Any anchor with download attribute
		btn = document.querySelector('a[download]');
		if(btn) { btn.click(); return 'clicked a[download]'; }
		// Chrome's built-in PDF viewer uses shadow DOM
		var viewer = document.querySelector('embed[type="application/pdf"]');
		if(viewer) return 'chrome-pdf-viewer';
		return 'no-download-button';
	})()`

	clickResult, err := s.sendSession(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression":    clickExpr,
		"returnByValue": true,
	}, dlSession)
	if err != nil {
		return nil, fmt.Errorf("find download button: %w", err)
	}

	var clickResp struct {
		Result struct {
			Value string `json:"value"`
		} `json:"result"`
	}
	json.Unmarshal(clickResult, &clickResp)
	clickStatus := clickResp.Result.Value
	log.Printf("    Download click: %s", clickStatus)

	if clickStatus == "chrome-pdf-viewer" {
		// For Chrome's built-in PDF viewer, use Ctrl+S / Cmd+S to save.
		s.sendSession(ctx, "Input.dispatchKeyEvent", map[string]interface{}{
			"type":                  "rawKeyDown",
			"key":                   "s",
			"code":                  "KeyS",
			"windowsVirtualKeyCode": 83,
			"nativeVirtualKeyCode":  83,
			"modifiers":             4, // Meta (Cmd on Mac)
		}, dlSession)
		s.sendSession(ctx, "Input.dispatchKeyEvent", map[string]interface{}{
			"type":                  "keyUp",
			"key":                   "s",
			"code":                  "KeyS",
			"windowsVirtualKeyCode": 83,
			"nativeVirtualKeyCode":  83,
		}, dlSession)
	} else if clickStatus == "no-download-button" {
		return nil, fmt.Errorf("no download button found on page")
	}

	// Wait for the download to complete by watching for the file.
	deadline := time.Now().Add(15 * time.Second)
	var downloadedPath string
	for time.Now().Before(deadline) {
		time.Sleep(500 * time.Millisecond)
		entries, _ := os.ReadDir(outputDir)
		for _, e := range entries {
			name := e.Name()
			// Skip partial downloads.
			if strings.HasSuffix(name, ".crdownload") || strings.HasSuffix(name, ".tmp") {
				continue
			}
			if strings.HasSuffix(strings.ToLower(name), ".pdf") {
				info, _ := e.Info()
				if info != nil && info.Size() > 0 {
					downloadedPath = filepath.Join(outputDir, name)
					break
				}
			}
		}
		if downloadedPath != "" {
			break
		}
	}

	if downloadedPath == "" {
		return nil, fmt.Errorf("download timed out - no PDF appeared in %s", outputDir)
	}

	// Rename to the desired output path if different.
	if downloadedPath != df.OutputPath {
		os.Rename(downloadedPath, df.OutputPath)
		downloadedPath = df.OutputPath
	}

	info, _ := os.Stat(downloadedPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}

	log.Printf("    Downloaded: %s (%d bytes)", downloadedPath, size)
	return &pb.StepResult{ScriptResult: fmt.Sprintf("%d bytes", size)}, nil
}

// doWaitForStable injects a MutationObserver that tracks the last DOM change
// timestamp, then polls until the DOM has been quiet for quiet_period_ms or
// timeout_ms elapses.
func (s *Server) doWaitForStable(ctx context.Context, w *pb.WaitForStable) (*pb.StepResult, error) {
	quietPeriod := time.Duration(w.QuietPeriodMs) * time.Millisecond
	if quietPeriod == 0 {
		quietPeriod = 500 * time.Millisecond
	}
	timeout := time.Duration(w.TimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	// Install a MutationObserver that resets a timestamp on every DOM change.
	// The guard flag makes re-injection on subsequent calls a no-op.
	installJS := `(function(){
		if(window.__cdp_stable_observer) return 'already installed';
		window.__cdp_stable_ts = Date.now();
		window.__cdp_stable_observer = new MutationObserver(function(){
			window.__cdp_stable_ts = Date.now();
		});
		window.__cdp_stable_observer.observe(document.documentElement, {
			childList: true, subtree: true, attributes: true
		});
		return 'installed';
	})()`

	if _, err := s.send(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression": installJS, "returnByValue": true,
	}); err != nil {
		return nil, fmt.Errorf("WaitForStable: install observer: %w", err)
	}

	// Poll until quiet period elapses or timeout is reached.
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
		result, err := s.send(ctx, "Runtime.evaluate", map[string]interface{}{
			"expression": "Date.now() - window.__cdp_stable_ts", "returnByValue": true,
		})
		if err != nil {
			continue
		}
		var resp struct {
			Result struct {
				Value float64 `json:"value"`
			} `json:"result"`
		}
		if json.Unmarshal(result, &resp) == nil {
			if time.Duration(resp.Result.Value)*time.Millisecond >= quietPeriod {
				log.Printf("    WaitForStable: DOM quiet for %.0fms", resp.Result.Value)
				return &pb.StepResult{}, nil
			}
		}
	}
	return nil, fmt.Errorf("WaitForStable: timed out after %v waiting for %v of DOM quiet", timeout, quietPeriod)
}

// doHover moves the mouse cursor to the given coordinates.
func (s *Server) doHover(ctx context.Context, h *pb.Hover) (*pb.StepResult, error) {
	params := map[string]interface{}{
		"type": "mouseMoved",
		"x":    h.X,
		"y":    h.Y,
	}
	if _, err := s.send(ctx, "Input.dispatchMouseEvent", params); err != nil {
		return nil, fmt.Errorf("Input.dispatchMouseEvent(mouseMoved): %w", err)
	}
	return &pb.StepResult{}, nil
}

// doDrag performs a drag from (start_x, start_y) to (end_x, end_y) using CDP
// mouse events: mousePressed → mouseMoved → mouseReleased.
func (s *Server) doDrag(ctx context.Context, d *pb.Drag) (*pb.StepResult, error) {
	steps := []map[string]interface{}{
		{"type": "mousePressed", "x": d.StartX, "y": d.StartY, "button": "left", "clickCount": 1},
		{"type": "mouseMoved", "x": d.EndX, "y": d.EndY, "button": "left", "buttons": 1},
		{"type": "mouseReleased", "x": d.EndX, "y": d.EndY, "button": "left", "clickCount": 1},
	}
	for _, params := range steps {
		if _, err := s.send(ctx, "Input.dispatchMouseEvent", params); err != nil {
			return nil, fmt.Errorf("Input.dispatchMouseEvent(%s): %w", params["type"], err)
		}
	}
	return &pb.StepResult{}, nil
}
