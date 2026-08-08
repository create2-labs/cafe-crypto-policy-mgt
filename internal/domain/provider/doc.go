// Package provider defines the declarative Capability Provider manifest
// (ProviderManifest v0.1) used by CPM to describe external solution profiles
// such as Nicetry.
//
// Explore loads manifests via CPM_PROVIDER_MANIFEST_PATHS and applies ADR §7
// hard compatibility against solution_profile_ref, including required_posture vs
// resulting_posture (CPM-P4 / CPM-P4b). Soft findings and persist snapshot gates
// arrive in later PRs. account_validation_posture is not part of the schema.
package provider
