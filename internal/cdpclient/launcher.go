package cdpclient

import (
	"bufio"
	"context"
	crand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// randomID returns an unguessable 128-bit hex id used to name per-process Chrome
// directories so concurrent processes can't locate each other's files.
func randomID() string {
	b := make([]byte, 16)
	if _, err := crand.Read(b); err != nil {
		return fmt.Sprintf("ts%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// wsURLPattern matches "DevTools listening on ws://..." from Chrome's stderr.
var wsURLPattern = regexp.MustCompile(`DevTools listening on (ws://\S+)`)

// containerFlags disable background services that have no place in a headless
// automation container and that otherwise spin and time out (wasting CPU and
// slowing startup): GPU, GCM/background networking, sync, component updates,
// the on-device model, default apps/extensions, etc. Observed on Cloud Run as
// GCM "PHONE_REGISTRATION_ERROR", on_device_model "Error loading backend", GPU
// CreateCommandBuffer failures, and multi-second compositor stalls.
var containerFlags = []string{
	"--disable-gpu",
	"--disable-background-networking",
	"--disable-background-timer-throttling",
	"--disable-backgrounding-occluded-windows",
	"--disable-renderer-backgrounding",
	"--disable-sync",
	"--disable-component-update",
	"--disable-default-apps",
	"--disable-extensions",
	"--disable-client-side-phishing-detection",
	"--disable-hang-monitor",
	"--mute-audio",
	"--metrics-recording-only",
	"--no-pings",
	"--disable-features=Translate,OptimizationHints,OptimizationGuideModelDownloading,MediaRouter,BackForwardCache,InterestFeedContentSuggestions",
}

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

	// NoTmpCleanup, when true, leaves the per-process temp directory on disk
	// after the Chrome process exits (for local debugging). On servers this must
	// stay false: the dirs live in tmpfs and would otherwise grow instance
	// memory over its lifetime.
	NoTmpCleanup bool
}

// LaunchResult contains the result of launching Chrome.
type LaunchResult struct {
	// WebSocketURL is the CDP WebSocket URL.
	WebSocketURL string

	// Process is the Chrome process, for lifecycle management.
	Process *os.Process

	// Cmd is the underlying exec.Cmd for full control.
	Cmd *exec.Cmd

	// TempDir is the per-process temp directory tree created, if any.
	// Use Cleanup() (or call the caller's lifecycle teardown) to remove it.
	TempDir string

	// keepTempDir mirrors LaunchConfig.NoTmpCleanup so Cleanup() knows whether
	// to delete TempDir.
	keepTempDir bool
}

// Cleanup kills the Chrome process, waits for it, and removes its temp directory
// tree (unless NoTmpCleanup was set). Safe to call on a nil result and to call
// more than once. This is the single place process+disk lifecycle is released.
func (r *LaunchResult) Cleanup() {
	if r == nil {
		return
	}
	if r.Process != nil {
		// Kill the whole process group (Chrome + all helpers), not just the
		// main pid; the group was established via setProcessGroup at launch.
		killProcessGroup(r.Process.Pid)
	}
	if r.Cmd != nil {
		r.Cmd.Wait()
	}
	if r.TempDir != "" && !r.keepTempDir {
		os.RemoveAll(r.TempDir)
		r.TempDir = ""
	}
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

	// Reap any Chrome instances orphaned by a previously-killed launcher before
	// adding another, so a crash loop can't accumulate unbounded browsers.
	sweepOrphansOnce()

	port := cfg.Port // 0 = auto

	args := []string{
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--no-first-run",
		"--no-default-browser-check",
	}

	if cfg.Headless {
		args = append(args, "--headless=new")
	}

	var tempDir, scratchDir string
	if cfg.UserDataDir != "" {
		args = append(args, "--user-data-dir="+cfg.UserDataDir)
	} else {
		// Isolated, unguessable per-process directory tree so concurrent Chrome
		// processes can't find, enumerate, or read each other's profile, cache,
		// or temp files: a 0700 base named with a random 128-bit id, with the
		// profile, disk cache, and scratch space all confined inside it.
		base := filepath.Join(os.TempDir(), "chromerpc-"+randomID())
		userDataDir := filepath.Join(base, "user-data")
		cacheDir := filepath.Join(base, "cache")
		scratchDir = filepath.Join(base, "tmp")
		for _, d := range []string{base, userDataDir, cacheDir, scratchDir} {
			if err := os.MkdirAll(d, 0700); err != nil {
				os.RemoveAll(base)
				return nil, fmt.Errorf("launcher: create profile dir: %w", err)
			}
		}
		tempDir = base
		args = append(args, "--user-data-dir="+userDataDir, "--disk-cache-dir="+cacheDir)
	}

	args = append(args, cfg.ExtraArgs...)

	// Start with about:blank so Chrome doesn't open any default page.
	args = append(args, "about:blank")

	cmd := exec.CommandContext(ctx, chromePath, args...)
	// Put Chrome in its own process group so its entire tree can be killed at
	// once, and make context cancellation kill that whole group rather than
	// only the main pid (which would orphan the renderer/GPU helpers).
	setProcessGroup(cmd)
	cmd.Cancel = func() error {
		if cmd.Process != nil {
			killProcessGroup(cmd.Process.Pid)
		}
		return nil
	}
	if scratchDir != "" {
		// Confine Chrome's scratch/temp files to its isolated tree too.
		cmd.Env = append(os.Environ(), "TMPDIR="+scratchDir, "TMP="+scratchDir, "TEMP="+scratchDir)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		if tempDir != "" && !cfg.NoTmpCleanup {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("launcher: stderr pipe: %w", err)
	}

	launchStart := time.Now()
	if err := cmd.Start(); err != nil {
		if tempDir != "" && !cfg.NoTmpCleanup {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("launcher: start chrome: %w", err)
	}

	// Parse WebSocket URL from stderr.
	wsURL, err := parseWSURL(stderrPipe, cfg.Stderr, timeout)
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		if tempDir != "" && !cfg.NoTmpCleanup {
			os.RemoveAll(tempDir)
		}
		return nil, fmt.Errorf("launcher: %w", err)
	}
	// Time from exec to DevTools WebSocket URL — Chrome's own startup cost, with
	// no image-pull/container-start confound.
	log.Printf("launcher: Chrome ready in %s (exec -> WebSocket URL)", time.Since(launchStart).Round(time.Millisecond))

	return &LaunchResult{
		WebSocketURL: wsURL,
		Process:      cmd.Process,
		Cmd:          cmd,
		TempDir:      tempDir,
		keepTempDir:  cfg.NoTmpCleanup,
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
		result.Cleanup()
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
