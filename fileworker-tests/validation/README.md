# Fileworker validation

This suite catches cross-cutting regressions that ordinary happy-path upload and
peer-transfer tests miss. It launches an isolated HTTP origin, a fresh Chrome
profile, and three real Fileworker clients. It never relies on demo query
parameters or an already-running server.

It validates:

- root production URLs and service-worker readiness;
- three-client peer uniqueness and stale-client eviction;
- automatic cross-tab OPFS index invalidation;
- uncaught JavaScript exceptions and console errors;
- desktop, compact desktop, and mobile layout bounds;
- the `More` button and inspector visibility at every viewport;
- page-level horizontal overflow, offscreen controls, accessible names, and
  pointer cursors;
- deterministic browser and HTTP-server teardown.

Run from the `chromerpc` repository:

```sh
python3 fileworker-tests/validation/run_validation.py
```

Use a different checkout or Chromium binary when needed:

```sh
python3 fileworker-tests/validation/run_validation.py \
  --app-root ../fileworker \
  --chrome /path/to/chromium
```

## Recorded walkthrough

The walkthrough follows TyG's rerunnable demo-video approach: it starts a real
server, records a fresh browser profile against the plain production root URL,
performs meaningful interactions, encodes a fixed-frame-rate WebM with ffmpeg,
and probes the artifact before reporting success. It demonstrates cross-client
OPFS updates, filtering, the real `More` inspector, peer discovery, and receive
policy selection.

```sh
python3 fileworker-tests/validation/record_walkthrough.py
```

The default artifact is
`fileworker-tests/validation/artifacts/fileworker-walkthrough.webm`. Override it
with `--output`; no query parameters or demo-only application branches are used.
