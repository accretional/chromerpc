# chromerpc automation recipes

Reusable, version-controlled automation playbooks for the high-level
`HeadlessBrowserService.RunAutomation` RPC. Each `.textproto` is an
`AutomationSequence` (see `proto/cdp/headlessbrowser/headlessbrowser.proto`).

Each `RunAutomation` call runs in its own **isolated incognito browser context**
(fresh cookies/storage), so a fresh call always starts from a clean slate — which
is why anti-bot/consent walls show up (see notes per recipe).

## Recipes

| File | Pattern |
|---|---|
| `screenshot_after_load.textproto` | Navigate and wait for the page to fully load (`wait_until: networkidle`) before screenshotting. |
| `search_and_screenshot.textproto` | Type a query into a site's search box, submit, wait for results, screenshot. |
| `dismiss_consent_and_screenshot.textproto` | Click a cookie/consent banner via JS, then screenshot. |
| `scroll_to_load_lazy.textproto` | Scroll down to trigger lazy-loaded content/images, then screenshot. |
| `record_av.textproto` | Record an **audio+visual** capture of a page to a video file (screencast video + ffmpeg-muxed source audio). |

## Key building blocks

- **`navigate { url, wait_until, timeout_ms }`** — `wait_until` is `commit`
  (default, returns immediately), `load` (DOM load event), or `networkidle`
  (load + ~500ms with no in-flight requests). Prefer `networkidle` for
  content-heavy pages; it removes the need for hand-tuned `wait` steps.
- **`wait_for_selector { selector, timeout_ms }`** — block until an element
  appears (more reliable than a fixed `wait`).
- **`evaluate_script { expression }`** — run arbitrary JS (dismiss banners,
  scroll, read values). The script's return value comes back in
  `step_results[].script_result`.
- **`type_text { selector, text }`** / **`press_key { key }`** — fill inputs and
  submit (`key: "enter"`).
- **`screenshot { format, full_page, output_path }`** — omit `output_path` to get
  the PNG bytes back in `step_results[].screenshot_data` (the right choice for the
  stateless Cloud Run deployment; `output_path` writes to the container's
  ephemeral disk).
- **`record { output_path, audio_path, pre_delay_ms, max_duration_ms,
  stop_condition, start_script, output_fps }`** — record an audio+visual clip to
  a video file on the server. Video is captured with `Page.startScreencast`
  (real frames, any page); audio is muxed from `audio_path` with ffmpeg (headless
  Chrome exposes no tab-audio capture over CDP). Stops on `stop_condition` (a
  polled JS expression), `max_duration_ms`, or context cancellation. Needs
  **ffmpeg** on the server and the server run with **`--autoplay`** to script
  playback. Returns a JSON summary in `step_results[].script_result`. Best on the
  local/bidi path, not stateless Cloud Run (writes to the server's disk).

## Running a recipe

### Remote (Cloud Run, TLS + IAM token)

`RunAutomation` takes JSON via grpcurl. The `scripts/recipe-run.sh` helper
converts a textproto recipe to JSON, calls the service, and saves/open any
screenshot bytes:

```bash
HOST=<service-host> ./scripts/recipe-run.sh recipes/screenshot_after_load.textproto
```

Or hand-write the JSON (field names are the proto's snake_case):

```bash
grpcurl -H "authorization: Bearer $(gcloud auth print-identity-token)" -d '{
  "steps": [
    { "navigate": { "url": "https://example.com", "wait_until": "networkidle" } },
    { "screenshot": { "format": "png" } }
  ]
}' <service-host>:443 cdp.headlessbrowser.HeadlessBrowserService/RunAutomation
```

### Local (insecure, against a local server)

```bash
make run &                                   # starts chromerpc on :50051
go run ./cmd/automate -input recipes/screenshot_after_load.textproto
```
