// Package provider defines the declarative Capability Provider manifest
// (ProviderManifest v0.1) used by CPM to describe external solution profiles
// such as Nicetry.
//
// Explore loads manifests via CPM_PROVIDER_MANIFEST_PATHS and applies ADR §7
// hard compatibility against solution_profile_ref, including required_posture vs
// resulting_posture (CPM-P4 / CPM-P4b). Ranked candidates also expose soft
// findings requires_bundler / requires_local_signer_state (CPM-P5). Persist
// gates (accepted_provider_snapshot + pinned refs + user_constraints couche B)
// live in package policy (CPM-P6 / CPM-P10). Catalogue load marks contradictory
// suggested_user_constraints as Erroneous (CPM-P11a). account_validation_posture
// is not part of the schema.
package provider
