#!/usr/bin/env bash
# build_test.sh — full build + unit test + integration test for all contracts
# Usage: ./build_test.sh
# Requirements: near-go CLI v0.1.1, Rust + cargo, NEAR sandbox

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Colors ────────────────────────────────────────────────────────
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

pass()   { echo -e "  ${GREEN}✓ $1${NC}"; }
fail()   { echo -e "  ${RED}✗ $1${NC}"; exit 1; }
step()   { echo -e "\n  ${YELLOW}▶ $1${NC}"; }
header() {
    echo -e "\n${CYAN}${BOLD}══════════════════════════════════════════${NC}"
    echo -e "${CYAN}${BOLD}  $1${NC}"
    echo -e "${CYAN}${BOLD}══════════════════════════════════════════${NC}"
}

# ── Checks ────────────────────────────────────────────────────────
echo -e "\n${BOLD}Checking dependencies...${NC}"
command -v near-go >/dev/null 2>&1 || { echo -e "${RED}near-go not found${NC}"; exit 1; }
command -v cargo   >/dev/null 2>&1 || { echo -e "${RED}cargo not found${NC}"; exit 1; }
NEAR_GO_VERSION=$(near-go version 2>&1 | grep -o 'v[0-9.]*' || echo "unknown")
echo -e "  near-go: ${GREEN}${NEAR_GO_VERSION}${NC}"
echo -e "  cargo:   ${GREEN}$(cargo --version)${NC}"

# ── Helper: run one contract ──────────────────────────────────────
# $1 = contract dir name (e.g. "01-basic-auction")
run_contract() {
    local contract="$1"
    local dir="$REPO_ROOT/$contract"

    header "$contract"
    cd "$dir"

    # 1. Build
    step "Build"
    near-go build
    pass "Build OK → main.wasm $(wc -c < main.wasm) bytes"

    # 2. Unit tests
    step "Unit tests  (near-go test package)"
    near-go test package
    pass "Unit tests OK"

    # 3. Integration tests
    step "Integration tests  (cargo run)"
    cd "$dir/integration_tests"
    cargo run
    pass "Integration tests OK"

    cd "$REPO_ROOT"
}

# ── 01-basic-auction ──────────────────────────────────────────────
run_contract "01-basic-auction"

# ── 02-nft-auction ────────────────────────────────────────────────
run_contract "02-nft-auction"

# ── 03-ft-auction ─────────────────────────────────────────────────
run_contract "03-ft-auction"

# ── 04-factory ────────────────────────────────────────────────────
# Factory embeds auction.wasm at compile time.
# We use the freshly built 03-ft-auction WASM as the embedded contract.
header "04-factory  (pre-build: copy auction.wasm)"
step "Copy 03-ft-auction/main.wasm → 04-factory/auction.wasm"
cp "$REPO_ROOT/03-ft-auction/main.wasm" "$REPO_ROOT/04-factory/auction.wasm"
pass "auction.wasm copied ($(wc -c < "$REPO_ROOT/04-factory/auction.wasm") bytes)"

run_contract "04-factory"

# ── Done ──────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}══════════════════════════════════════════${NC}"
echo -e "${GREEN}${BOLD}  All 4 contracts: build + tests PASSED ✓${NC}"
echo -e "${GREEN}${BOLD}══════════════════════════════════════════${NC}"
echo ""
