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
	// Enable page domain first.
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

	// Write to disk if output_path specified.
	if ss.OutputPath != "" {
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

	// If selector is given, resolve coordinates from it.
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

	// mousePressed then mouseReleased.
	for _, evType := range []string{"mousePressed", "mouseReleased"} {
		params := map[string]interface{}{
			"type":       evType,
			"x":         x,
			"y":         y,
			"button":    "left",
			"clickCount": 1,
		}
		if _, err := s.send(ctx, "Input.dispatchMouseEvent", params); err != nil {
			return nil, fmt.Errorf("Input.dispatchMouseEvent(%s): %w", evType, err)
		}
	}
	return &pb.StepResult{}, nil
}

func (s *Server) doTypeText(ctx context.Context, t *pb.TypeText) (*pb.StepResult, error) {
	// If selector given, focus it first.
	if t.Selector != "" {
		expr := fmt.Sprintf(`document.querySelector(%q).focus()`, t.Selector)
		params := map[string]interface{}{"expression": expr}
		if _, err := s.send(ctx, "Runtime.evaluate", params); err != nil {
			return nil, fmt.Errorf("focus selector: %w", err)
		}
	}

	// Use Input.insertText for simplicity.
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
