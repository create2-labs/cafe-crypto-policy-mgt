# Shared `payload_sha256` vectors (RD-P1 / RD-P2)

Golden inputs for the **closed hashed payload** ([ADR_20260824_remove_cp_drafts §3.2.1](https://github.com/create2-labs/cafe-adr/blob/main/ADR_20260824_remove_cp_drafts.md)).

## Normative rules

1. Object fields (closed): `schema_version`, `crypto_policy_id`, `required_posture`, `user_constraints`, `solution_profile_ref`, `accepted_provider_snapshot`, `accepted_findings`.
2. Canonicalization: **RFC 8785 JCS**, then `payload_sha256 = hex(SHA-256(jcs_bytes))` (lowercase hex).
3. Hashed subtree types: **string | boolean | object | array only** — **no** JSON `number`, **no** `null`.
4. Numeric identifiers (e.g. `chain_support_used.chain_id`) are **strings**.
5. Before JCS, `accepted_findings` is **lexicographically sorted** and **deduplicated** (server authority; clients should pre-normalize).
6. Files named `*.json` are the **already-normalized** hashed objects (findings sorted+deduped). Companion `*.sha256` holds the expected hex digest.

## Vectors

| File | Purpose |
| ---- | ------- |
| `hashed_payload_minimal.json` | Small closed-set baseline |
| `hashed_payload_realistic_nested.json` | Realistic nested `accepted_provider_snapshot` (Nicetry-shaped) **without** number/null |

Findings order/dedupe invariance is covered by Go tests in [`internal/payloadhash`](../../../payloadhash/) (RD-P2).

## Reject cases (not golden)

`payloadhash.Digest` / `DigestJSON` reject before JCS when the hashed subtree contains `null`, a JSON number, an unknown top-level hashed field, or a missing required closed field.

## Consumers

- **RD-P2:** Go [`internal/payloadhash`](../../../payloadhash/) — table-driven tests vs these vectors
- **RD-P4/P5:** challenge + persist handlers call the same Digester
- **RD-P9+:** frontend / CLI must match these digests

Hashes produced with Python `jcs` 0.2.1 (RFC 8785) for RD-P1; Go uses [`github.com/gowebpki/jcs`](https://github.com/gowebpki/jcs) (RFC 8785) and must reproduce them.
