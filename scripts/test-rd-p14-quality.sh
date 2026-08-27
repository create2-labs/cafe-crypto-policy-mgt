#!/usr/bin/env bash
# RD-P14 quality — cafe-crypto-policy-mgt WORKPLAN pointers (docs-only)
set -euo pipefail
cd "$(dirname "$0")/.."
FAIL=0

if ! rg -q 'RD-P14' workplans/WORKPLAN_API.md; then
  echo "FAIL: WORKPLAN_API.md missing RD-P14 pointer" >&2
  FAIL=1
fi
if ! rg -q 'Current normative contract v1.0.0' workplans/WORKPLAN_API.md; then
  echo "FAIL: WORKPLAN_API.md missing CP_PERSIST v1.0.0 pointer" >&2
  FAIL=1
fi
if rg -q 'Next :\*\* RD-P8' workplans/WORKPLAN_API.md; then
  echo "FAIL: WORKPLAN still says Next RD-P8" >&2
  FAIL=1
fi

if [[ "$FAIL" -ne 0 ]]; then
  echo "==> RD-P14 CPM FAIL" >&2
  exit 1
fi
echo "==> RD-P14 CPM green (docs-only)"
