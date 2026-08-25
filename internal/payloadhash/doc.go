// Package payloadhash computes the server-authoritative payload_sha256 for
// CP persist (ADR_20260824_remove_cp_drafts §3.2.1 / RD-P2).
//
// Closed hashed fields are canonicalized with RFC 8785 JCS, then hashed with
// SHA-256 (lowercase hex). Top-level accepted_findings are lexicographically
// sorted and deduplicated before JCS. The hashed subtree may only contain
// JSON string, boolean, object, and array — never number or null.
//
// HTTP routes are intentionally untouched here; RD-P4/P5 wire this package.
package payloadhash
