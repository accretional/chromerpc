package cdpclient

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// wsURLPattern matches "DevTools listening on ws://..." from Chrome's stderr.
var wsURLPattern = regexp.MustCompile(`DevTools listening on (ws://\S+)`)

// LaunchConfig configures how Chrome is launched.
type LaunchConfig struct {
	// ChromePath is the path to the Chrome/Chromium binary.
	// If empty, common locations are searched.
	ChromePath string

	// Port for remote debugging. 0 means auto-assign.
	Port int

	// Headless runs Chrome in headless mode.
	Headless bool

	// UserDataDir is the Chrome user data directory.
	// If empty, a temp directory is created.
	UserDataDir string

	// ExtraArgs are additional command-line arguments.
	ExtraArgs []string

	// StartTimeout is how long to wait for Chrome to start and print the
	// WebSocket URL. Default: 30s.
	StartTimeout time.Duration

	// Stderr is where to forward Chrome's stderr after parsing the WS URL.
	// If nil, stderr is discarded.
	Stderr io.Writer
}

// LaunchResult contains the result of launching Chrome.
type LaunchResult struct {
	// WebSocketURL is the CDP WebSocket URL.
	WebSocketURL string

	// Process is the Chrome process, for lifecycle management.
	Process *os.Process

	// Cmd is the underlying exec.Cmd for full control.
	Cmd *exec.Cmd

	// TempDir is the temp user data dir created, if any.
	// Caller should clean this up when done.
	TempDir string
}

// Launch starts a Chrome process with remote debugging enabled and returns
// the WebSocket URL parsed from stderr.
func Launch(ctx context.Context, cfg LaunchConfig) (*LaunchResult, error) {
	chromePath := cfg.ChromePath
	if chromePath == "" {
		var err error
		chromePath, err = findChrome()
		if err != nil {
			return nil, err
		}
	}

	timeout := cfg.StartTimeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	port := cfg.Port // 0 = auto

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--no-first-run",
		"--no-default-browser-check",
	}

	if cfg.Headless {
		args = append(args, "--headless=new")
	}

	var tempDir string
	if cfg.UserDataDir != "" {
		args = append(args, "--user-data-dir="+cfg.UserDataDir)
	} else {
		var err error
		tempDir, err = os.MkdirTemp("", "chromerpc-*")
		if err != nil {
			return nil, fmt.Errorf("launcher: create temp dir: %w", err)
		}
		args = append(args, "--user-data-dir="+tempDir)
	}

	args = append(args, cfg.ExtraArgs...)

	// Start with about:blank so Chrome doesn't open any default page.
	args = append(args, "about:blank")

	cmd := exec.CommandContext(ctx, chromePath, args...)

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("launcher: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("launcher: start chrome: %w", err)
	}

	// Parse WebSocket URL from stderr.
	wsURL, err := parseWSURL(stderrPipe, cfg.Stderr, timeout)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		if tempDir != "" {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("launcher: %w", err)
	}

	return &LaunchResult{
		WebSocketURL: wsURL,
		Process:      cmd.Process,
		Cmd:          cmd,
		TempDir:      tempDir,
	}, nil
}

// parseWSURL reads Chrome's stderr looking for the DevTools WebSocket URL.
func parseWSURL(stderr io.Reader, forward io.Writer, timeout time.Duration) (string, error) {
	type result struct {
		url string
		err error
	}
	ch := make(chan result, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if m := wsURLPattern.FindStringSubmatch(line); len(m) == 2 {
				ch <- result{url: m[1]}
				// Continue reading to avoid blocking Chrome, forwarding to
				// the configured writer.
				if forward != nil {
					io.Copy(forward, stderr)
				} else {
					io.Copy(io.Discard, stderr)
				}
				return
			}
			if forward != nil {
				fmt.Fprintln(forward, line)
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- result{err: fmt.Errorf("read stderr: %w", err)}
		} else {
			ch <- result{err: fmt.Errorf("chrome exited without printing WebSocket URL")}
		}
	}()

	select {
	case r := <-ch:
		return r.url, r.err
	case <-time.After(timeout):
		return "", fmt.Errorf("timed out waiting for WebSocket URL (%v)", timeout)
	}
}

// findChrome searches for Chrome/Chromium in common locations.
func findChrome() (string, error) {
	// Check PATH first.
	for _, name := range []string{
		"google-chrome",
		"google-chrome-stable",
		"chromium",
		"chromium-browser",
		"chrome",
	} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	// Platform-specific paths.
	candidates := []string{
		// macOS
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		// Linux
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		// Common container path
		"/usr/bin/google-chrome-stable",
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("chrome/chromium not found; set ChromePath or install Chrome")
}

// ConnectOrLaunch tries to connect to an existing CDP endpoint. If wsURL is
// empty, it launches Chrome with the given config and connects.
func ConnectOrLaunch(ctx context.Context, wsURL string, cfg LaunchConfig) (*Client, *LaunchResult, error) {
	if wsURL != "" {
		client, err := Dial(ctx, wsURL)
		if err != nil {
			return nil, nil, err
		}
		return client, nil, nil
	}

	result, err := Launch(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}

	client, err := Dial(ctx, result.WebSocketURL)
	if err != nil {
		result.Process.Kill()
		result.Cmd.Wait()
		if result.TempDir != "" {
			os.RemoveAll(result.TempDir)
		}
		return nil, nil, err
	}

	return client, result, nil
}

// ParseWSURLFromString extracts a WebSocket URL from a string that may
// contain the "DevTools listening on ..." message.
func ParseWSURLFromString(s string) (string, bool) {
	for _, line := range strings.Split(s, "\n") {
		if m := wsURLPattern.FindStringSubmatch(line); len(m) == 2 {
			return m[1], true
		}
	}
	return "", false
}
