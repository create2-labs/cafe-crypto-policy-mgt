#!/usr/bin/env bash
# RD-P7 regression: no draft store / route / Draft-ID binding symbols in Go runtime packages.
# Usage (from cafe-crypto-policy-mgt root): ./scripts/check-no-draft-runtime.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

fail=0

check_absent() {
  local pattern="$1"
  local scope="$2"
  # shellcheck disable=SC2086
  if rg -n --glob '*.go' "$pattern" $scope >/tmp/cpm-rd-p7-draft-hits.txt 2>/dev/null; then
    echo "FAIL: forbidden pattern still present: $pattern"
    cat /tmp/cpm-rd-p7-draft-hits.txt
    fail=1
  fi
}

# Runtime packages (exclude archaeology docs / workplans).
SCOPE="internal/app internal/persistence internal/cpmroutes internal/walletauth internal/domain cmd"

check_absent 'func \(.*\) SaveDraft\(' "$SCOPE"
check_absent 'func \(.*\) GetDraft\(' "$SCOPE"
check_absent 'func \(.*\) DeleteDraft\(' "$SCOPE"
check_absent 'func \(.*\) PersistDraftOnce\(' "$SCOPE"
check_absent 'func \(.*\) DraftPersistStatus\(' "$SCOPE"
check_absent 'ErrDraftNotFound' "$SCOPE"
check_absent 'ErrDraftAlreadyPersisted' "$SCOPE"
check_absent 'type DraftRecord struct' "$SCOPE"
check_absent 'DraftPersistPath' "$SCOPE"
check_absent 'cpmroutes\.Drafts\b' "$SCOPE"
check_absent 'DraftCount' "$SCOPE"
check_absent 'PlatformDraftID' "$SCOPE"
check_absent 'CodeWalletAuthorizationDraftMismatch' "$SCOPE"
check_absent 'DraftID\s+string' "$SCOPE"
check_absent 'ValidateDraftPayloadForPersist' "$SCOPE"

# Authenticated route inventory must not advertise /drafts.
if rg -n '/drafts' internal/cpmroutes/routes.go >/dev/null 2>&1; then
  echo "FAIL: /drafts still referenced in cpmroutes/routes.go"
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  echo "check-no-draft-runtime: FAILED"
  exit 1
fi

echo "check-no-draft-runtime: OK"
