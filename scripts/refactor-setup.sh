#!/usr/bin/env bash
#
# refactor-setup.sh — install & verify the prerequisites for the chromerpc refactor
# testing harness (see docs/refactor/notes/0003-testing-strategy.md).
#
# Idempotent and safe to re-run: it checks before installing and never fails just
# because something is already present. Designed for macOS + Homebrew (the dev
# machine); on Linux it verifies tools and tells you what to install.
#
# Usage:
#   scripts/refactor-setup.sh [--project <GCP_PROJECT_ID>] [--region <REGION>]
#                             [--with-protoc] [--with-buf] [--no-install]
#
#   --project     Set the active gcloud project (for the Cloud Run deploy leg).
#                 May also be passed via the PROJECT env var.
#   --region      Default GCP region (default: us-central1).
#   --with-protoc Also install protobuf compiler + Go plugins (for `make proto`).
#   --with-buf    Also install buf (future proto tooling, Phase 3/4).
#   --no-install  Verify only; don't install anything (report gaps and exit).
#
# What the harness needs (and why):
#   Go 1.25+        build/test everything            (required)
#   Chrome          local CDP tests (Phase A)        (required)
#   claude CLI      agentic dynamic-nav suite        (required for Phase C5)
#   grpcurl         smoke-test.sh / recipe-run.sh    (required for Phase C)
#   gcloud SDK      Cloud Run build+deploy (Phase B) (required for Phase B/C)
#   docker          NOT required — we use the Cloud Build path (build in-cloud)
#   protoc/buf      only when regenerating protos     (optional)
#
# After this script: authenticate interactively with `gcloud auth login`
# (this script cannot do that for you) and confirm the project.

set -uo pipefail

# ---- args ---------------------------------------------------------------------
PROJECT="${PROJECT:-}"
REGION="${REGION:-us-central1}"
WITH_PROTOC=0
WITH_BUF=0
NO_INSTALL=0
while [ $# -gt 0 ]; do
  case "$1" in
    --project) PROJECT="${2:-}"; shift 2 ;;
    --region)  REGION="${2:-}"; shift 2 ;;
    --with-protoc) WITH_PROTOC=1; shift ;;
    --with-buf) WITH_BUF=1; shift ;;
    --no-install) NO_INSTALL=1; shift ;;
    -h|--help) sed -n '2,40p' "$0"; exit 0 ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

# ---- pretty output ------------------------------------------------------------
GREEN=$'\e[32m'; YELLOW=$'\e[33m'; RED=$'\e[31m'; BOLD=$'\e[1m'; RST=$'\e[0m'
ok()   { printf '  %sOK%s   %s\n' "$GREEN" "$RST" "$*"; }
warn() { printf '  %sWARN%s %s\n' "$YELLOW" "$RST" "$*"; }
miss() { printf '  %sMISS%s %s\n' "$RED" "$RST" "$*"; }
hdr()  { printf '\n%s== %s ==%s\n' "$BOLD" "$*" "$RST"; }
have() { command -v "$1" >/dev/null 2>&1; }

REMAINING=()   # manual follow-ups
FAILED=0       # required tool still missing after install attempts

OS="$(uname -s)"
BREW=0
if have brew; then BREW=1; fi

brew_install() {  # brew_install <formula> [--cask]
  local pkg="$1"; shift || true
  if [ "$NO_INSTALL" = 1 ]; then return 1; fi
  if [ "$BREW" != 1 ]; then return 1; fi
  echo "  installing $pkg via Homebrew..."
  brew install "$@" "$pkg"
}

# ---- Homebrew -----------------------------------------------------------------
hdr "Homebrew"
if [ "$BREW" = 1 ]; then
  ok "brew $(brew --version 2>/dev/null | head -1 | awk '{print $2}')"
else
  if [ "$OS" = "Darwin" ]; then
    miss "Homebrew not found — required to auto-install tools on macOS."
    REMAINING+=("Install Homebrew: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"")
  else
    warn "Not macOS; will verify tools only and report install hints."
  fi
fi

# ---- Go -----------------------------------------------------------------------
hdr "Go toolchain"
if have go; then
  ok "$(go version)"
else
  miss "go not found (required)."
  brew_install go && have go && ok "$(go version)" || { FAILED=1; REMAINING+=("Install Go 1.25+: https://go.dev/dl/ or 'brew install go'"); }
fi

# ---- Chrome -------------------------------------------------------------------
hdr "Chrome / Chromium (local CDP tests)"
CHROME_MAC="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
if have google-chrome || have google-chrome-stable || have chromium || have chromium-browser || [ -x "$CHROME_MAC" ]; then
  if [ -x "$CHROME_MAC" ]; then ok "Google Chrome ($("$CHROME_MAC" --version 2>/dev/null))"; else ok "chrome/chromium on PATH"; fi
else
  miss "No Chrome/Chromium found (required for Phase A local CDP tests)."
  brew_install google-chrome --cask && [ -x "$CHROME_MAC" ] && ok "installed Chrome" || { FAILED=1; REMAINING+=("Install Chrome: 'brew install --cask google-chrome'"); }
fi

# ---- claude CLI ---------------------------------------------------------------
hdr "claude CLI (agentic suite)"
if have claude; then
  ok "claude at $(command -v claude)"
else
  warn "claude CLI not found — needed for the multi-agent dynamic-nav suite (Phase C5)."
  REMAINING+=("Install the claude CLI (Claude Code) — see https://claude.com/claude-code")
fi

# ---- grpcurl ------------------------------------------------------------------
hdr "grpcurl (deployed-service tests)"
if have grpcurl; then
  ok "$(grpcurl --version 2>&1 | head -1)"
else
  miss "grpcurl not found (required for smoke-test.sh / recipe-run.sh)."
  brew_install grpcurl && have grpcurl && ok "installed $(grpcurl --version 2>&1 | head -1)" || { FAILED=1; REMAINING+=("Install grpcurl: 'brew install grpcurl'"); }
fi

# ---- gcloud SDK ---------------------------------------------------------------
hdr "Google Cloud SDK (Cloud Run build + deploy)"
# The cask installs gcloud under Homebrew's share dir; make it usable this session.
GCLOUD_INC=""
if [ "$BREW" = 1 ]; then
  GCLOUD_INC="$(brew --prefix 2>/dev/null)/share/google-cloud-sdk/path.bash.inc"
fi
if ! have gcloud && [ -n "$GCLOUD_INC" ] && [ -f "$GCLOUD_INC" ]; then
  # shellcheck disable=SC1090
  source "$GCLOUD_INC" || true
fi
if have gcloud; then
  ok "$(gcloud --version 2>/dev/null | head -1)"
else
  miss "gcloud not found (required for Phase B deploy + Phase C remote tests)."
  brew_install google-cloud-sdk --cask
  if [ -n "$GCLOUD_INC" ] && [ -f "$GCLOUD_INC" ]; then source "$GCLOUD_INC" || true; fi
  if have gcloud; then
    ok "installed $(gcloud --version 2>/dev/null | head -1)"
    REMAINING+=("Add gcloud to your shell profile: echo 'source \"$GCLOUD_INC\"' >> ~/.zshrc")
  else
    FAILED=1
    REMAINING+=("Install gcloud: 'brew install --cask google-cloud-sdk', then source its path.bash.inc")
  fi
fi

# ---- gcloud auth + project ----------------------------------------------------
if have gcloud; then
  hdr "gcloud auth & project"
  ACCT="$(gcloud auth list --filter=status:ACTIVE --format='value(account)' 2>/dev/null | head -1)"
  if [ -n "$ACCT" ]; then
    ok "active account: $ACCT"
  else
    warn "no active gcloud account — auth is interactive, run it yourself:"
    REMAINING+=("Authenticate: 'gcloud auth login' (in this session use: ! gcloud auth login)")
  fi
  if [ -n "$PROJECT" ]; then
    if gcloud config set project "$PROJECT" >/dev/null 2>&1; then ok "project set: $PROJECT"; else warn "could not set project '$PROJECT' (auth first?)"; REMAINING+=("Set project after auth: 'gcloud config set project $PROJECT'"); fi
  else
    CURPROJ="$(gcloud config get-value project 2>/dev/null)"
    if [ -n "$CURPROJ" ] && [ "$CURPROJ" != "(unset)" ]; then
      ok "current project: $CURPROJ"
    else
      warn "no project set — pass --project <ID> or run 'gcloud config set project <ID>'."
      REMAINING+=("Choose an existing GCP project: 'gcloud config set project <PROJECT_ID>' (needs billing + Run/Build/Artifact Registry)")
    fi
  fi
  echo "  region for deploy scripts: $REGION (export REGION=$REGION to override defaults)"
fi

# ---- docker (informational) ---------------------------------------------------
hdr "docker (optional)"
if have docker; then ok "docker present ($(docker --version 2>/dev/null))"; else
  warn "docker not installed — NOT required. The harness uses the Cloud Build path (deploy-cloudrun.sh) which builds the image in-cloud. Only needed for the deploy-local-image.sh path.";
fi

# ---- optional: protoc + go plugins --------------------------------------------
if [ "$WITH_PROTOC" = 1 ]; then
  hdr "protoc + Go plugins (proto codegen)"
  have protoc && ok "$(protoc --version)" || { brew_install protobuf && have protoc && ok "installed $(protoc --version)" || REMAINING+=("Install protoc: 'brew install protobuf'"); }
  if have go; then
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest 2>/dev/null && ok "protoc-gen-go" || warn "protoc-gen-go install failed"
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest 2>/dev/null && ok "protoc-gen-go-grpc" || warn "protoc-gen-go-grpc install failed"
    REMAINING+=("Ensure \$(go env GOPATH)/bin is on PATH for protoc plugins")
  fi
fi

# ---- optional: buf ------------------------------------------------------------
if [ "$WITH_BUF" = 1 ]; then
  hdr "buf (future proto tooling)"
  have buf && ok "$(buf --version)" || { brew_install buf && have buf && ok "installed $(buf --version)" || REMAINING+=("Install buf: 'brew install bufbuild/buf/buf'"); }
fi

# ---- summary ------------------------------------------------------------------
hdr "Summary"
printf '  Required for Phase A (local): '; { have go && { [ -x "$CHROME_MAC" ] || have chromium || have google-chrome || have google-chrome-stable; }; } && printf '%sREADY%s\n' "$GREEN" "$RST" || printf '%sNOT READY%s\n' "$RED" "$RST"
printf '  Required for Phase B/C (deploy+remote): '; { have gcloud && have grpcurl; } && printf '%sTOOLS READY%s (still need auth+project below)\n' "$GREEN" "$RST" || printf '%sMISSING TOOLS%s\n' "$RED" "$RST"
printf '  Required for Phase C5 (agentic): '; have claude && printf '%sREADY%s\n' "$GREEN" "$RST" || printf '%sMISSING%s\n' "$RED" "$RST"

if [ "${#REMAINING[@]}" -gt 0 ]; then
  hdr "Manual follow-ups"
  for r in "${REMAINING[@]}"; do printf '  • %s\n' "$r"; done
fi

echo
if [ "$FAILED" = 1 ]; then
  echo "${RED}Setup incomplete: one or more required tools are still missing.${RST}"
  exit 1
fi
echo "${GREEN}Tooling install/verify complete.${RST} Finish any manual follow-ups above (esp. gcloud auth + project), then run the harness."
exit 0
