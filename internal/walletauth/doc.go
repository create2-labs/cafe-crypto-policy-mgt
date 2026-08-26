// Package walletauth implements CP-PERSIST canonical wallet authorization messages
// and EIP-191 / personal_sign verification (CP_PERSIST.md).
//
// Normative message binding (RD-P4+ / RD-P7): Payload SHA-256 line only
// (server-computed hash). Legacy Draft ID binding was removed in RD-P7.
package walletauth
