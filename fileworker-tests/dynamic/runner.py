#!/usr/bin/env python3
"""Bounded runner for an agent-driven chrome-proxy test."""

from __future__ import annotations

import argparse
import datetime as dt
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

HERE = Path(__file__).resolve().parent


def loopback_url(value: str) -> str:
    parsed = urllib.parse.urlparse(value)
    if parsed.scheme not in {"http", "https"} or parsed.hostname not in {
        "127.0.0.1", "localhost", "::1"
    }:
        raise argparse.ArgumentTypeError("must be an HTTP(S) loopback URL")
    return value.rstrip("/")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--proxy", required=True, help="chrome-proxy base URL")
    parser.add_argument("--app", required=True, help="Fileworker URL")
    parser.add_argument("--scenario", type=Path, default=HERE / "scenario.md")
    parser.add_argument("--artifacts", type=Path, default=HERE / "artifacts")
    parser.add_argument("--timeout", type=int, default=480)
    parser.add_argument("--max-budget-usd", type=float, default=1.0)
    parser.add_argument("--model", default="sonnet")
    parser.add_argument("--claude", default=os.environ.get("CLAUDE_BIN", "claude"))
    parser.add_argument("--allow-non-loopback", action="store_true")
    args = parser.parse_args()
    if args.timeout < 10:
        parser.error("--timeout must be at least 10 seconds")
    if not 0 < args.max_budget_usd <= 20:
        parser.error("--max-budget-usd must be between 0 and 20")
    if not args.allow_non_loopback:
        try:
            args.proxy = loopback_url(args.proxy)
            args.app = loopback_url(args.app)
        except argparse.ArgumentTypeError as error:
            parser.error(str(error) + " (or pass --allow-non-loopback)")
    else:
        args.proxy = args.proxy.rstrip("/")
        args.app = args.app.rstrip("/")
    return args


def healthcheck(proxy: str) -> None:
    try:
        with urllib.request.urlopen(proxy + "/health", timeout=5) as response:
            if response.status != 200:
                raise RuntimeError(f"HTTP {response.status}")
    except (urllib.error.URLError, TimeoutError, RuntimeError) as error:
        raise RuntimeError(f"chrome-proxy health check failed: {error}") from error


def extract_result(envelope: object) -> object | None:
    if not isinstance(envelope, dict):
        return None
    for key in ("structured_output", "structuredOutput"):
        if isinstance(envelope.get(key), dict):
            return envelope[key]
    result = envelope.get("result")
    if isinstance(result, dict):
        return result
    if isinstance(result, str):
        try:
            parsed = json.loads(result)
            return parsed if isinstance(parsed, dict) else None
        except json.JSONDecodeError:
            return None
    return None


def main() -> int:
    args = parse_args()
    healthcheck(args.proxy)
    stamp = dt.datetime.now(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = args.artifacts.resolve() / f"{stamp}-{os.getpid()}"
    run_dir.mkdir(parents=True)

    prompt = args.scenario.read_text().replace("{{PROXY_URL}}", args.proxy).replace(
        "{{APP_URL}}", args.app + "/"
    )
    (run_dir / "prompt.txt").write_text(prompt)
    schema = json.dumps(json.loads((HERE / "result.schema.json").read_text()))
    command = [
        args.claude, "-p", prompt,
        "--model", args.model,
        "--effort", "high",
        "--output-format", "json",
        "--json-schema", schema,
        "--max-budget-usd", str(args.max_budget_usd),
        "--no-session-persistence",
        "--permission-mode", "dontAsk",
        "--allowedTools", "Bash(curl *)", "Read",
        "--disallowedTools", "Edit", "Write", "WebFetch", "WebSearch", "NotebookEdit",
    ]
    started = time.monotonic()
    timed_out = False
    stdout = ""
    stderr = ""
    exit_code = 1
    process = subprocess.Popen(
        command,
        cwd=HERE.parent.parent,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        start_new_session=True,
    )
    try:
        stdout, stderr = process.communicate(timeout=args.timeout)
        exit_code = process.returncode
    except subprocess.TimeoutExpired:
        timed_out = True
        os.killpg(process.pid, signal.SIGTERM)
        try:
            stdout, stderr = process.communicate(timeout=5)
        except subprocess.TimeoutExpired:
            os.killpg(process.pid, signal.SIGKILL)
            stdout, stderr = process.communicate()
        exit_code = 124

    (run_dir / "claude.stdout.json").write_text(stdout)
    (run_dir / "claude.stderr.log").write_text(stderr)
    parsed = None
    result = None
    try:
        parsed = json.loads(stdout)
        result = extract_result(parsed)
    except json.JSONDecodeError:
        pass
    if result is not None:
        (run_dir / "result.json").write_text(json.dumps(result, indent=2) + "\n")
    metadata = {
        "app_url": args.app + "/",
        "proxy_url": args.proxy,
        "model": args.model,
        "timeout_seconds": args.timeout,
        "max_budget_usd": args.max_budget_usd,
        "elapsed_seconds": round(time.monotonic() - started, 3),
        "timed_out": timed_out,
        "claude_exit_code": exit_code,
        "structured_result": result is not None,
        "verdict": result.get("verdict") if isinstance(result, dict) else None,
    }
    (run_dir / "run.json").write_text(json.dumps(metadata, indent=2) + "\n")
    print(run_dir)
    if timed_out:
        print("FAIL: Claude test timed out", file=sys.stderr)
        return 124
    if exit_code != 0 or result is None:
        print("FAIL: Claude did not produce a structured result", file=sys.stderr)
        return 1
    if result.get("verdict") != "PASS":
        print("FAIL: dynamic browser test reported defects", file=sys.stderr)
        return 2
    print("PASS: dynamic browser test completed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
