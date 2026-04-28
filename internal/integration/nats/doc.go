// Package nats contains NATS integration entry points for CPM.
//
// Inbound consumption of policy.assessment.requested.v0.1 is duplicate-safe:
// repeated event_id values are treated as no-op.
//
// PR16 adds outbound producer wiring for:
// - policy.assessment.completed.v0.1
// - policy.remediation.requested.v0.1
//
// Producers in this package are intentionally thin projections from CPM models
// to shared wire contracts from cafe-contracts/cafenatsv01.
package nats
