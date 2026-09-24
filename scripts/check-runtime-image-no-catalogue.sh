#!/usr/bin/env bash
# CCD-P4: the runtime image must not bake catalogue JSON, and a container
# started without a catalogue mount must exit at catalogue load.
# Usage (from cafe-crypto-policy-mgt root): ./scripts/check-runtime-image-no-catalogue.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

IMAGE="${CPM_IMAGE:-cafe-cpm:no-catalogue}"

if grep -E 'COPY .+ /app/policy/?' Dockerfile >/dev/null; then
  echo "FAIL: Dockerfile still copies into /app/policy" >&2
  exit 1
fi

docker build -f Dockerfile -t "$IMAGE" .

cid="$(docker create "$IMAGE")"
cleanup() {
  docker rm -f "$cid" >/dev/null 2>&1 || true
}
trap cleanup EXIT

if docker export "$cid" | tar -t | grep -E '(^|/)app/policy/.*\.json$'; then
  echo "FAIL: runtime image contains *.json under /app/policy" >&2
  exit 1
fi
docker rm -f "$cid" >/dev/null
trap - EXIT

set +e
logs="$(docker run --rm "$IMAGE" 2>&1)"
code=$?
set -e
if [[ "$code" -eq 0 ]]; then
  echo "FAIL: container exited 0 without a catalogue mount" >&2
  printf '%s\n' "$logs" >&2
  exit 1
fi
if ! grep -q 'catalogue' <<<"$logs"; then
  echo "FAIL: exit ${code} but logs do not mention catalogue load" >&2
  printf '%s\n' "$logs" >&2
  exit 1
fi

echo "check-runtime-image-no-catalogue: OK (exit ${code})"
