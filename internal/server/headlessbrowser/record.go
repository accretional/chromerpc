package headlessbrowser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pb "github.com/accretional/chromerpc/proto/cdp/headlessbrowser"
)

// recFrame is one captured screencast frame: its wall-clock offset (seconds)
// from the moment recording started, plus the raw JPEG bytes.
type recFrame struct {
	t    float64
	data []byte
}

// doRecord captures an audio+visual recording of the current tab to a video
// file on the server.
//
// VIDEO is captured with Page.startScreencast — real rendered frames, so it
// works on ANY page (DOM, SVG, canvas) in headless Chrome, unlike an in-page
// canvas.captureStream (canvas-only) or getDisplayMedia (needs a GUI/gesture).
//
// AUDIO cannot be pulled from the tab over CDP in headless Chrome (there is no
// tab-audio-capture command; screencast is video-only; getDisplayMedia/
// tabCapture need an extension or a real audio device). So audio is muxed in
// from a caller-supplied source file (audio_path) with ffmpeg. This is exact
// (the pristine source, not a re-recording) and is the right model for demos
// that already own their source audio.
//
// Stop conditions (any of): max_duration_ms elapses; stop_condition JS returns
// truthy; or ctx is cancelled (the bidi stream disconnected).
func (s *Server) doRecord(ctx context.Context, r *pb.Record) (*pb.StepResult, error) {
	if strings.TrimSpace(r.OutputPath) == "" {
		return nil, fmt.Errorf("record: output_path required")
	}

	// Resolve defaults.
	maxDur := time.Duration(r.MaxDurationMs) * time.Millisecond
	if maxDur <= 0 {
		maxDur = 15 * time.Second
	}
	pollEvery := time.Duration(r.StopPollMs) * time.Millisecond
	if pollEvery <= 0 {
		pollEvery = 250 * time.Millisecond
	}
	quality := int(r.Quality)
	if quality <= 0 || quality > 100 {
		quality = 90
	}
	outFps := int(r.OutputFps)
	if outFps <= 0 {
		outFps = 30
	}

	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, fmt.Errorf("record: ffmpeg not found on PATH (required to encode the video): %w", err)
	}

	// Page domain must be enabled to receive screencast frame events.
	if _, err := s.send(ctx, "Page.enable", nil); err != nil {
		return nil, fmt.Errorf("Page.enable: %w", err)
	}

	// The CDP (flat-mode) session whose screencast frames we want. With a
	// runContext it's that isolated session; otherwise the client default
	// (the interactive streaming path).
	sid := s.client.SessionID()
	if rc := runContextFrom(ctx); rc != nil {
		sid = rc.sessionID
	}

	// Pre-record delay (honor cancellation).
	if r.PreDelayMs > 0 {
		select {
		case <-time.After(time.Duration(r.PreDelayMs) * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Frame buffer + an ack pump. Chrome sends the next screencast frame only
	// after the previous one is acked, so every frame must be acked to keep the
	// stream flowing. The frame event fires on the cdpclient read-loop
	// goroutine, so its handler MUST NOT block on a CDP round-trip (that would
	// deadlock the read-loop) — it only appends the frame and hands the ack id
	// to a dedicated pump goroutine.
	var (
		mu     sync.Mutex
		frames []recFrame
	)
	ackCh := make(chan int, 1024)
	var ackWG sync.WaitGroup
	ackWG.Add(1)
	go func() {
		defer ackWG.Done()
		for frameSess := range ackCh {
			actx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, _ = s.sendSession(actx, "Page.screencastFrameAck",
				map[string]interface{}{"sessionId": frameSess}, sid)
			cancel()
		}
	}()

	var start time.Time
	unsub := s.client.On("Page.screencastFrame", func(_ string, params json.RawMessage, evSid string) {
		if evSid != sid {
			return
		}
		var ev struct {
			Data      string `json:"data"`
			SessionId int    `json:"sessionId"`
		}
		if json.Unmarshal(params, &ev) != nil {
			return
		}
		raw, err := base64.StdEncoding.DecodeString(ev.Data)
		if err != nil {
			return
		}
		mu.Lock()
		frames = append(frames, recFrame{t: time.Since(start).Seconds(), data: raw})
		mu.Unlock()
		ackCh <- ev.SessionId // keep the stream alive (see note above)
	})

	// Arm the screencast.
	scParams := map[string]interface{}{"format": "jpeg", "quality": quality}
	if r.EveryNthFrame > 1 {
		scParams["everyNthFrame"] = r.EveryNthFrame
	}
	if r.MaxWidth > 0 {
		scParams["maxWidth"] = r.MaxWidth
	}
	if r.MaxHeight > 0 {
		scParams["maxHeight"] = r.MaxHeight
	}
	start = time.Now()
	if _, err := s.send(ctx, "Page.startScreencast", scParams); err != nil {
		unsub()
		close(ackCh)
		ackWG.Wait()
		return nil, fmt.Errorf("Page.startScreencast: %w", err)
	}

	// Optionally kick off playback / animation at the moment recording begins.
	if strings.TrimSpace(r.StartScript) != "" {
		if _, err := s.send(ctx, "Runtime.evaluate", map[string]interface{}{
			"expression": r.StartScript, "returnByValue": true, "awaitPromise": true,
		}); err != nil {
			log.Printf("    record: start_script error (continuing): %v", err)
		}
	}

	// Wait for a stop condition.
	stopReason := s.recordWait(ctx, sid, maxDur, pollEvery, r.StopCondition)

	// Stop capture. Use a detached context so this still runs if ctx was the
	// thing that cancelled us (bidi disconnect).
	dctx, dcancel := context.WithTimeout(context.Background(), 5*time.Second)
	_, _ = s.sendSession(dctx, "Page.stopScreencast", nil, sid)
	dcancel()
	unsub()
	// Give any in-flight frame a moment to land, then close the ack pump.
	time.Sleep(120 * time.Millisecond)
	close(ackCh)
	ackWG.Wait()

	mu.Lock()
	captured := frames
	frames = nil
	mu.Unlock()
	wall := time.Since(start).Seconds()

	if len(captured) == 0 {
		return nil, fmt.Errorf("record: no frames captured (screencast produced nothing in %.1fs)", wall)
	}

	videoSecs, err := s.encodeRecording(captured, wall, outFps, r.OutputPath, r.AudioPath)
	if err != nil {
		return nil, err
	}

	info, _ := os.Stat(r.OutputPath)
	size := int64(0)
	if info != nil {
		size = info.Size()
	}
	hasAudio := strings.TrimSpace(r.AudioPath) != ""
	summary := map[string]interface{}{
		"output":        r.OutputPath,
		"bytes":         size,
		"frames":        len(captured),
		"video_seconds": round2(videoSecs),
		"wall_seconds":  round2(wall),
		"has_audio":     hasAudio,
		"fps":           outFps,
		"stop":          stopReason,
	}
	js, _ := json.Marshal(summary)
	log.Printf("    Recording saved: %s (%d bytes, %d frames, %.2fs, audio=%v, stop=%s)",
		r.OutputPath, size, len(captured), videoSecs, hasAudio, stopReason)
	return &pb.StepResult{ScriptResult: string(js)}, nil
}

// recordWait blocks until a stop condition fires and returns the reason:
// "max_duration", "stop_condition", or "disconnect".
func (s *Server) recordWait(ctx context.Context, sid string, maxDur, pollEvery time.Duration, cond string) string {
	deadline := time.NewTimer(maxDur)
	defer deadline.Stop()

	var pollC <-chan time.Time
	if strings.TrimSpace(cond) != "" {
		tk := time.NewTicker(pollEvery)
		defer tk.Stop()
		pollC = tk.C
	}

	for {
		select {
		case <-ctx.Done():
			return "disconnect"
		case <-deadline.C:
			return "max_duration"
		case <-pollC:
			if s.evalTruthy(ctx, sid, cond) {
				return "stop_condition"
			}
		}
	}
}

// evalTruthy evaluates a JS expression in the page and reports whether the
// result is truthy. Errors are treated as false (keep recording).
func (s *Server) evalTruthy(ctx context.Context, sid, expr string) bool {
	res, err := s.sendSession(ctx, "Runtime.evaluate", map[string]interface{}{
		"expression": expr, "returnByValue": true,
	}, sid)
	if err != nil {
		return false
	}
	var resp struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if json.Unmarshal(res, &resp) != nil {
		return false
	}
	switch v := strings.TrimSpace(string(resp.Result.Value)); v {
	case "true":
		return true
	case "", "false", "null", "0", "\"\"", "undefined":
		return false
	default:
		// Any other non-empty JSON value (non-zero number, non-empty string,
		// object) counts as truthy.
		return true
	}
}

// encodeRecording writes the captured JPEG frames to a temp dir, builds an
// ffmpeg concat manifest whose per-frame durations reproduce the real capture
// pacing, and encodes a constant-frame-rate video — muxing audioPath if given.
// Returns the encoded video's duration in seconds.
func (s *Server) encodeRecording(frames []recFrame, wall float64, outFps int, outPath, audioPath string) (float64, error) {
	tmp, err := os.MkdirTemp("", "chromerpc-rec-")
	if err != nil {
		return 0, fmt.Errorf("record: temp dir: %w", err)
	}
	defer os.RemoveAll(tmp)

	// Per-frame display durations from arrival offsets. The last frame is held
	// for the tail between its arrival and the recording stop (>= 1 output frame).
	minDur := 1.0 / 240.0
	maxFrameDur := 3.0
	clampDur := func(d float64) float64 {
		if d < minDur {
			return minDur
		}
		if d > maxFrameDur {
			return maxFrameDur
		}
		return d
	}

	var b strings.Builder
	b.WriteString("ffconcat version 1.0\n")
	for i, f := range frames {
		name := fmt.Sprintf("f-%06d.jpg", i)
		if err := os.WriteFile(filepath.Join(tmp, name), f.data, 0644); err != nil {
			return 0, fmt.Errorf("record: write frame: %w", err)
		}
		var dur float64
		if i+1 < len(frames) {
			dur = clampDur(frames[i+1].t - f.t)
		} else {
			tail := wall - f.t
			if tail < 1.0/float64(outFps) {
				tail = 1.0 / float64(outFps)
			}
			dur = clampDur(tail)
		}
		fmt.Fprintf(&b, "file '%s'\nduration %.4f\n", name, dur)
	}
	// concat demuxer holds the last-listed file for its duration only if it is
	// repeated; append the final frame once more without a duration.
	fmt.Fprintf(&b, "file '%s'\n", fmt.Sprintf("f-%06d.jpg", len(frames)-1))

	manifest := filepath.Join(tmp, "frames.txt")
	if err := os.WriteFile(manifest, []byte(b.String()), 0644); err != nil {
		return 0, fmt.Errorf("record: write manifest: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return 0, fmt.Errorf("record: mkdir out: %w", err)
	}

	webm := strings.EqualFold(filepath.Ext(outPath), ".webm")
	haveAudio := strings.TrimSpace(audioPath) != ""
	if haveAudio {
		if _, statErr := os.Stat(audioPath); statErr != nil {
			return 0, fmt.Errorf("record: audio_path %q: %w", audioPath, statErr)
		}
	}

	args := []string{"-y", "-f", "concat", "-safe", "0", "-i", manifest}
	if haveAudio {
		args = append(args, "-i", audioPath)
	}
	// Normalize to CFR, even dimensions (yuv420p needs both), from the VFR input.
	vf := fmt.Sprintf("fps=%d,scale=trunc(iw/2)*2:trunc(ih/2)*2,format=yuv420p", outFps)
	args = append(args, "-vf", vf)
	if webm {
		args = append(args, "-c:v", "libvpx-vp9", "-b:v", "0", "-crf", "32", "-row-mt", "1")
		if haveAudio {
			args = append(args, "-c:a", "libopus", "-b:a", "160k")
		}
	} else {
		args = append(args, "-c:v", "libx264", "-preset", "veryfast", "-crf", "20",
			"-movflags", "+faststart")
		if haveAudio {
			args = append(args, "-c:a", "aac", "-b:a", "160k")
		}
	}
	if haveAudio {
		// Explicit mapping + stop at the shorter stream so a long source audio
		// is trimmed to the recorded video length.
		args = append(args, "-map", "0:v:0", "-map", "1:a:0", "-shortest")
	}
	args = append(args, outPath)

	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("record: ffmpeg failed: %w\n%s", err, tailStr(string(out), 1500))
	}

	return probeDuration(outPath), nil
}

// probeDuration returns the container duration (seconds) via ffprobe, or 0.
func probeDuration(path string) float64 {
	out, err := exec.Command("ffprobe", "-v", "error", "-show_entries",
		"format=duration", "-of", "default=nw=1:nk=1", path).Output()
	if err != nil {
		return 0
	}
	var d float64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &d)
	return d
}

func round2(f float64) float64 { return float64(int(f*100+0.5)) / 100 }

func tailStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}
