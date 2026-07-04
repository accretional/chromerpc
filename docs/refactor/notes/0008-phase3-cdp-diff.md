# Phase 3 — CDP proto sweep: upstream diff + generation plan

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: active
- Related: [[0001-kickoff]], BUG-002, ../TODO.md (Phase 3)

Diff of our CDP surface vs current upstream (`ChromeDevTools/devtools-protocol@master`,
`browser_protocol.json` + `js_protocol.json` union).

## Headline
| | upstream | ours |
|---|---|---|
| domains | 58 | 54 |
| commands | 670 | 438 implemented |
| events subscribed (matched) | — | 129 |

## Domains missing from us (all EXPERIMENTAL → skip for now)
`Ads`(1), `CrashReportContext`(1), `DigitalCredentials`(1), `SmartCardEmulation`(12/14),
`WebMCP`(4/4). No **stable** domain is missing. Adding these is optional/low-priority.

## Dead surface
- `headlessbrowser` — our **custom** automation API, not a CDP domain. Removed in
  **Phase 4/5** (not a Phase-3 CDP removal). Leave for now.
- (`Database` already removed — BUG-002.)

## Stale calls upstream REMOVED/RENAMED — FIX NOW (real bugs, will 404 on Chrome 150)
| our string | file:line | fix |
|---|---|---|
| `Audits.checkContrast` | internal/server/audits/audits.go:86 | remove (removed upstream) |
| `ServiceWorker.inspectWorker` | internal/server/serviceworker/serviceworker.go:82 | remove (removed upstream) |
| `HeadlessExperimental.needsBeginFramesChanged` | internal/server/headlessexperimental/headlessexperimental.go:125,167 | remove event (domain has 0 events now) |
| `Media.playersCreated` | internal/server/media/media.go:150 | rename → `Media.playerCreated` |

## Stable command gaps worth filling later (additive, optional)
Network `getRequestPostData`/`setCookies`/`setBypassServiceWorker`; DOM
`setFileInputFiles`/`scrollIntoViewIfNeeded`/`highlightRect`; Runtime
`queryObjects`/`setAsyncCallStackDepth`; Emulation `setIdleOverride`/`clearIdleOverride`/
`setScriptExecutionDisabled`; Debugger `continueToLocation`/`restartFrame`/`setVariableValue`/
`setSkipAllPauses`; CSS `setMediaText`/`setEffectivePropertyValueForNode`. (Best generated,
not hand-written — see cdpgen.)

## Generator (`tools/cdpgen`) — conventions to encode (from existing protos)
- one `.proto` per domain: `proto/cdp/<lower>/<lower>.proto`, `package cdp.<lower>`,
  `option go_package`.
- `service <Domain>Service`; each command → `rpc <Pascal>(<Cmd>Request) returns (<Cmd>Response)`
  (even when empty). camelCase→PascalCase (special: `printToPDF`→`PrintToPDF`).
- fields: camelCase→snake_case. **`string session_id = 99;`** appended to every Request
  (adapter convention, not CDP).
- types: string→string, integer→int32, number→double, boolean→bool, base64/binary→bytes,
  array→repeated, object/$ref→message, enum→proto enum. Domain `types` → messages/enums.
- events: NOT 1 rpc each. One `rpc SubscribeEvents(Subscribe<Domain>EventsRequest) returns
  (stream <Domain>Event)`. `<Domain>Event` = `oneof event { <EventName>Event <snake> = N; }`;
  `Subscribe…Request { string session_id = 1; }`.
- experimental/deprecated → trailing `// experimental` comments (not enforced).
- generator should also emit the Makefile `proto:` file list (or we switch to buf).

**Approach:** generator emits `.proto`; existing `protoc` step makes `.pb.go`. Build
incrementally; validate generated output against the current hand-written protos
(diff) to measure fidelity before switching over.

## Built this session — tracking + generator both WORK

### Ground truth from the running Chrome (not master)
`scripts/cdp-pull.sh` launches the Chrome we ship, dumps its exact `/json/protocol`,
and diffs vs the repo. Against **Chrome/150.0.7871.47**: 57 domains, and it confirmed
all **4 stale calls are genuinely absent from 150** (not just master ahead) — so they
are real drift. 4 domains missing (all experimental); 0 dead domains.
Vendored: `proto/cdp/_upstream/chrome-protocol.json` + `VERSION` (committed);
master `browser_/js_protocol.json` gitignored (regenerable). `make cdp-pull`.

### Generator `tools/cdpgen` — 57/57 domains protoc-valid
Reads the pinned protocol JSON → emits `.proto` per the conventions above.
`make cdp-gen DOMAIN=all OUT=/tmp/gen` produces **all 57 domains, every one
protoc-valid** (`libprotoc 35.1`). RPC surface matches the hand-written protos
(spot-checked Log). Two collision classes found + fixed in the generator:
- commands that already carry a `sessionId` param (Target/Page) → don't append the
  adapter `session_id = 99`.
- domains with a type named `<Domain>Event` (BackgroundService) → name the event
  oneof wrapper `<Domain>Events` to avoid collision.

### Applied fix
- `Media.playersCreated` → `Media.playerCreated` (server subscription string;
  confirmed against the vendored 150 protocol). No proto regen needed.

### Deliberately deferred (tracked, not hand-hacked)
- Removing the 3 dead gRPC methods/events (`Audits.checkContrast`,
  `ServiceWorker.inspectWorker`, `HeadlessExperimental.needsBeginFramesChanged`)
  needs proto regen + server changes → do it via the cdpgen migration, not by hand.
- **Generator scope (prototype):** cross-domain `$ref`s and inline object/enum
  *properties* are emitted as scalar placeholders with `// cdpgen:` comments; full
  cross-domain imports + byte-fidelity to the hand-written protos + wiring
  regenerated `.pb.go` into the build is the next iteration (Phase 3→4).
