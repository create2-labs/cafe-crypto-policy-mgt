package walletobserved

import v01 "github.com/create2-labs/cafe-contracts/observation/wallet/v01"

// Event is the shared cafe.discovery.wallet.observed envelope (wire contract v0.1).
type Event = v01.Event

// Subject identifies the observed wallet for this contract family.
type Subject = v01.Subject

// Payload holds the shared wire fields exported from Discovery.
type Payload = v01.Payload
