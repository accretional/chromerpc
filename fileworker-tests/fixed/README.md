# Fileworker fixed UI tests

Deterministic end-to-end checks for Fileworker using an isolated real Chrome
profile controlled directly through chromerpc's CDP client.

The tests serve the unmodified Fileworker checkout at a plain localhost URL. No
test-only query parameters or application hooks are used.

Covered regressions:

- drag/drop writes a real `File` through the UI and renders it in the index;
- a second, already-open client refreshes automatically after that OPFS write;
- the `···` control opens the inspector;
- the inspector starts at scroll position zero and is entirely inside the
  viewport;
- interactive `···` styling has a pointer cursor and accessible description;
- screenshots are captured into `artifacts/` for inspection.

Run:

```sh
go test ./fileworker-tests/fixed -v -count=1
```

To test a different checkout:

```sh
FILEWORKER_DIR=/path/to/fileworker go test ./fileworker-tests/fixed -v -count=1
```
