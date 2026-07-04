# Accretional ecosystem repos relevant to the refactor

- Date: 2026-07-03
- Author: Claude (Opus 4.8) — initial session
- Status: active (proto-linker IDENTIFIED via code search — user to give final nod)
- Related: [[0001-kickoff]], ../TODO.md (Phase 3/4/10)

The "proto linker/build project" wasn't `accretional/proto-linker` (404). Code-search
across the org (`gh search code` for `ServerConfig` / `BuildDescriptorProto` /
`Linker`, authed as `fredxfred`) pinned down the lineage. gluon is **not** the linker.

## proto-linker = `accretional/proto-go` (Phase 4 — single CDP server)  ✅ identified

- **`accretional/proto-go`** (private, updated 2026-06-17) — *"A Go toolchain that
  **links, builds, loads, and launches** gRPC services from exported functions"*:
  `Launch(Load(Build(Linker(...))))`. **`src/linker/`** holds `linker.go`,
  **`linker.proto`**, `compile_service.go`, `compile_memfd.go` — `Link(LinkDescriptor)
  → SourceDescriptor` (a skeltal `main` whose `Run()` serves a target's register func),
  `Build = Compile∘Link`. **This is the "gRPC server setup" the linker provides** and
  the mechanism for Phase 4: replace chromerpc's 55 hand-wired
  `Register…ServiceServer` calls + hand-listed `protoc` with a linked single server.
- **`accretional/gluon`** (public) — *not* the linker. It's the **`astkit`** AST-codegen
  substrate + a "self-modifying API service / go coding agent". proto-go's README says
  the Go **compiler service was migrated *out of* gluon into proto-go**, and all
  generated Go is rendered via `astkit.RenderFile` (never string concat). This
  explains the user's memory: they didn't set up the gRPC server bits *in gluon* —
  those live in proto-go. (gluon's one-liner "gRPC Service Linker" is aspirational/old.)
- **`accretional/go2proto`** (private, 2026-06-16) — turns a regular Go package into
  proto + server + client + converters (built with gluon's astkit). proto-go generates
  each pipeline stage as a gRPC service via go2proto.
- **`accretional/proto-type`** (public) — defines `message BuildDescriptorProto` +
  `builder_service` (`Parse(Text)→BuildDescriptorProto`, `Precompile(...)→FileDescriptorProto`).
  Foundational descriptor-building types.
- **`accretional/katarche`** (private, 2026-03-12) — the **older** prototyping
  monorepo ancestor (`server/`, `service/`, `proto-bundle/`, `proto-split/`,
  `proto-download/`, `packages/`, `tools/`). Has both `ServerConfig` (13) and
  `BuildDescriptorProto`. This is the "old prototyping repo from a month or 2 ago".
- **"protosh"** — NOT a repo and NOT the linker. It's the `Protosh` scripting service
  (`Protosh.Run` runtime) inside **`accretional/proto-expr`** — a separate thread the
  user half-remembered.
- `accretional/rpcfun` (public) — the current chromerpc README's old linker reference
  ("common linker in .../rpcfun"); superseded by the proto-go/gluon/go2proto stack.
- `accretional/proto-projection` (private) — "gluon-style descriptor-to-descriptor
  transform"; related descriptor tooling.

→ **Phase-4 plan:** use **proto-go's `src/linker`** (`linker.proto` + `Link`/`Build`)
  to link the CDP domain services into one server. Substrate: go2proto + gluon/astkit.
  **User to confirm** proto-go is the intended entrypoint (vs. driving it via gluon).

## Proto codegen / PDL tooling candidates (Phase 3 — proto sweep + auto-gen)

- **`accretional/proto-merge`** (public) — "Protobuf source code processor — scan,
  download, bundle, split, and transform .proto files." Directly relevant to the
  "pull in / bundle / generate protos" goal.
- **`accretional/protostar`** (public) — "Enhanced protobuf builds, testing, and
  codegen." Candidate to replace the hand-listed `protoc` Makefile target.
- `accretional/go2proto` (private), `accretional/proto-builder` (private) — possible
  build/codegen helpers.

## Agents-in-Chrome candidates (Phase 10)

- **`accretional/cdp-agent`** (private) — *"Adding an agent layer for chromerpc."*
  Almost certainly the home (or precedent) for the "integrate agents directly into
  Chrome" follow-on projects.
- `accretional/click-chromerpc` (private), `accretional/loopnet` (private,
  "commercial real estate automation using chromerpc") — existing consumers of
  chromerpc; useful as real-world usage references / regression targets.

## Other possibly-relevant

- `accretional/grpc-server-config` (public) — interceptors for rate limiting,
  logging, metrics, quotas + protobuf configs. Could inform Phase 7 tooling.
- `accretional/runrpc` (public) — referenced in the README "dream" (`./runrpc …`).

> All access via `gh` as `fredxfred`; private repos need that account's grant.
