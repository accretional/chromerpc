#!/usr/bin/env python3
"""Record a deterministic, production-URL Fileworker walkthrough as WebM."""
from __future__ import annotations

import argparse
import base64
import json
import shutil
import subprocess
import tempfile
import time
from pathlib import Path

from run_validation import CHROME_CANDIDATES, Harness, READY, ROOT, wait


def shot(page, directory, number):
    data = page.call("Page.captureScreenshot", {
        "format": "jpeg", "quality": 90, "fromSurface": True,
        "captureBeyondViewport": False,
    })["data"]
    (directory / f"{number:05d}.jpg").write_bytes(base64.b64decode(data))
    return number + 1


def hold(page, directory, number, seconds, fps):
    deadline = time.monotonic() + seconds
    while time.monotonic() < deadline:
        started = time.monotonic()
        number = shot(page, directory, number)
        time.sleep(max(0, 1 / fps - (time.monotonic() - started)))
    return number


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--app-root", default=str(ROOT))
    parser.add_argument("--chrome")
    parser.add_argument("--output", default=str(Path(__file__).with_name("artifacts") / "fileworker-walkthrough.webm"))
    parser.add_argument("--fps", type=int, default=10)
    args = parser.parse_args()
    ffmpeg = shutil.which("ffmpeg")
    if not ffmpeg:
        raise SystemExit("ffmpeg is required")
    chrome = args.chrome or next((p for p in CHROME_CANDIDATES if p and Path(p).exists()), None)
    if not chrome:
        raise SystemExit("Chrome/Chromium not found; pass --chrome")
    output = Path(args.output).resolve()
    output.parent.mkdir(parents=True, exist_ok=True)

    with tempfile.TemporaryDirectory(prefix="fileworker-video-") as temp:
        frames = Path(temp)
        with Harness(chrome, Path(args.app_root).resolve()) as harness:
            page, peer = harness.page(), harness.page()
            for client in (page, peer):
                wait(client, READY)
                client.call("Emulation.setDeviceMetricsOverride", {
                    "width": 1440, "height": 900, "deviceScaleFactor": 1, "mobile": False,
                })
            assert page.evaluate("location.search") == ""
            number = hold(page, frames, 0, 1.5, args.fps)

            # Import two real OPFS objects and visibly let cross-client invalidation settle.
            peer.evaluate("""Promise.all([
              FileworkerAPI.call("write",{path:"field-notes.txt",
                data:new Blob(["Peer-synchronized field notes\\nIntegrity: verified\\n"]),
                type:"text/plain"}),
              FileworkerAPI.call("write",{path:"telemetry.json",
                data:new Blob(['{"status":"nominal","packets":4096}']),
                type:"application/json"})
            ])""")
            wait(page, "document.querySelectorAll('.file-row').length >= 2")
            number = hold(page, frames, number, 2, args.fps)

            # Demonstrate filter, clear it, then open the actual More/inspector flow.
            page.evaluate("""(() => {
              const input=document.querySelector("#search-input");
              input.focus(); input.value="telemetry"; input.dispatchEvent(new Event("input",{bubbles:true}));
            })()""")
            number = hold(page, frames, number, 1.5, args.fps)
            page.evaluate("""(() => {
              const input=document.querySelector("#search-input");
              input.value=""; input.dispatchEvent(new Event("input",{bubbles:true}));
              [...document.querySelectorAll(".file-row")]
                .find(x=>x.dataset.path==="field-notes.txt").querySelector(".more-button").click();
            })()""")
            number = hold(page, frames, number, 2.5, args.fps)
            page.evaluate("document.querySelector('#close-inspector').click()")

            # Show the live encrypted peer connection and policy controls.
            page.evaluate("document.querySelector('[data-view=peers]').click(); FileworkerPeerAPI.announce()")
            peer.evaluate("FileworkerPeerAPI.announce()")
            wait(page, "FileworkerPeerAPI.connectedPeerIds().length === 1")
            number = hold(page, frames, number, 2, args.fps)
            page.evaluate("document.querySelector('[data-policy=lazy_check]').click()")
            number = hold(page, frames, number, 1.5, args.fps)

            # Finish on the transfer ledger after a real API write has been observed.
            page.evaluate("document.querySelector('[data-view=files]').click()")
            number = hold(page, frames, number, 1.5, args.fps)

        subprocess.run([
            ffmpeg, "-y", "-v", "error", "-framerate", str(args.fps),
            "-i", str(frames / "%05d.jpg"), "-c:v", "libvpx-vp9",
            "-pix_fmt", "yuv420p", "-crf", "30", "-b:v", "0",
            "-metadata", "title=Fileworker production walkthrough", str(output),
        ], check=True)
    probe = subprocess.run([
        shutil.which("ffprobe") or "ffprobe", "-v", "error", "-show_entries",
        "format=duration,size", "-of", "json", str(output),
    ], check=True, capture_output=True, text=True)
    print(json.dumps({"ok": True, "output": str(output), "probe": json.loads(probe.stdout)}))


if __name__ == "__main__":
    main()
