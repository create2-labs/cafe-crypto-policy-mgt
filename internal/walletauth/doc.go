// Package walletauth implements CP-PERSIST canonical wallet authorization messages
// and EIP-191 / personal_sign verification (CP_PERSIST.md).
//
// Normative message binding (RD-P4+): Payload SHA-256 line (server-computed hash).
// Legacy Draft ID line remains parseable until RD-P5 removes draft persist.
package walletauth
