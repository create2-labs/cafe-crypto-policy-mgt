# Canonical Discovery API path constants (WORKPLAN_API_PR PR11c).
# Source from scripts: . "$(dirname "$0")/lib/discovery-route-paths.sh"

# In-process Fiber prefix (direct backend :8080).
DISCOVERY_V1_BASE="/discovery/v1"
DISCOVERY_V1_SCAN="${DISCOVERY_V1_BASE}/scan"
DISCOVERY_V1_WALLET_SCANS="${DISCOVERY_V1_BASE}/wallets/scans"
DISCOVERY_V1_TLS_SCANS="${DISCOVERY_V1_BASE}/tls/scans"

# Edge (browser / nginx HTTPS); upstream strips /api.
DISCOVERY_EDGE_V1_BASE="/api/discovery/v1"
DISCOVERY_EDGE_V1_SCAN="${DISCOVERY_EDGE_V1_BASE}/scan"
DISCOVERY_EDGE_V1_WALLET_SCANS="${DISCOVERY_EDGE_V1_BASE}/wallets/scans"
DISCOVERY_EDGE_V1_TLS_SCANS="${DISCOVERY_EDGE_V1_BASE}/tls/scans"
