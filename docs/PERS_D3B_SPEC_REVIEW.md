# PERS-D3b-spec — revue sémantique CP (`internal/cp/v1`)

> **Statut :** revue CPM pour jalon **PERS-D3b-spec** (spec OpenAPI uniquement — pas d’impl HTTP ni DDL).
> **Contrat normatif :** [`cafe-persistence/openapi/internal/cp/v1.yaml`](../../cafe-persistence/openapi/internal/cp/v1.yaml)
> **ADR :** [ADR_20260622_persistence.md](../../cafe-discovery/docs/ADR/ADR_20260622_persistence.md) §8.2, §9.2–§9.3, §5.5

## Décision

**Approuvé** — le contrat `internal/cp/v1` reflète la sémantique métier CPM actuelle (`OwnerScopedStore`, routes publiques `cpm-v1.yaml`, guards W1/W3) et est prêt pour **PERS-D4** (DDL Postgres) puis **PERS-D4b** (handlers).

## Ownership (rappel ADR §8.2)

| Dimension | Owner |
|-----------|--------|
| Colonnes, statuts, payload JSON, invariants persist-once | **CPM** (ce document) |
| DDL, migrations, writers, SLO | **cafe-persistence** (D4) |
| Handlers HTTP `internal/cp/v1` | **cafe-persistence** (D4b) |
| Wallet EIP-191 / `personal_sign` | **CPM** public API uniquement |
| Codes HTTP produit W1/W3 | **Discovery** (facts seulement côté persistence) |

## Mapping opérations → store actuel

| `internal/cp/v1` | Store / route CPM aujourd’hui | Notes |
|------------------|-------------------------------|-------|
| `PUT /drafts/{draft_id}` | `OwnerScopedStore.SaveDraft` | Upsert par `draft_id` client ; `status=server_draft` |
| `GET /drafts/{draft_id}` | `GetDraft` | 404 owner-scoped |
| `DELETE /drafts/{draft_id}` | `DeleteDraft` | Soft delete cible D4 |
| `POST /drafts/{draft_id}/persist` | `PersistDraftOnce` | **Sans** signature — CPM vérifie wallet avant l’appel S2S |
| `GET /policies/{policy_id}` | `GetPolicy` | |
| `DELETE /policies/{policy_id}` | `DeletePolicy` | Soft delete utilisateur |
| `GET /policies?scan_id=` | `ListPersistedPoliciesForScan` | Drafts exclus ; ordre `persisted_at` desc |
| `GET /references/wallet` | `CountActiveWalletCPMContext` | W1 — `exists`, `policy_count`, `draft_count` |
| `GET /references/scan` | `ListPersistedPoliciesForScan` → count | W3 — `referenced`, `count` (policies persistées) |

## Idempotence `draft_id` (§5.5 ADR)

Aligné sur `PersistDraftOnce` et `draft_persist_state` (ADR §8.4.3) :

1. Premier succès : alloue `policy_id`, écrit `crypto_policies`, marque `completed=true`, supprime le draft.
2. Replay après succès : **409** `DRAFT_ALREADY_PERSISTED` (CPM mappe vers le code public homonyme).
3. Échec avant `completed` : retry avec le **même** `draft_id` réutilise le `policy_id` réservé.
4. Remplacement scan : anciennes lignes `status=superseded` (une seule `persisted` active par scan/owner).

**Clé d’idempotence :** `(user_id, tenant_id, draft_id)`.

## Schéma Postgres cible (D4)

Tables conformes ADR §8.4 — le contrat HTTP projette :

- `crypto_policy_drafts` → `DraftRow`
- `crypto_policies` → `PolicyRow` (`persisted` / `superseded`)
- `draft_persist_state` → état interne ; non exposé en lecture HTTP

Champs audit persist (`wallet_address`, `chain_id`, `ownership_status`, `wallet_control_method`, `wallet_control_verified_at`) : colonnes dédiées **et** miroir dans `payload` normalisé (comme `policyPayloadFromDraft` aujourd’hui). **Jamais** `signed_message` / `signature` en base.

## Frontière wallet-auth

| Couche | Responsabilité |
|--------|----------------|
| Public `POST /api/cpm/v1/drafts/{draft_id}/persist` | Vérifie `signed_message` + `signature`, binding draft/scan/wallet |
| Internal `POST /internal/cp/v1/drafts/{draft_id}/persist` | Reçoit `wallet_control_verified_at` + métadonnées déjà validées |

## W1 / W3 (existence only)

Discovery **ne doit pas** lire `PolicyRow.payload` pour les guards. Les endpoints `/references/*` retournent des **faits** ; Discovery produit les codes `CPM_EXISTS_FOR_WALLET_TARGET` / `SCAN_REFERENCED_BY_POLICY`.

Extraction adresse wallet pour W1 : même règles que `wallet_target.go` (IMM-9b) — payload `policy_context`, `wallet_address`, etc.

## Non-objectifs D3b-spec

- Pas de client HTTP CPM (→ D5a)
- Pas de `CPM_STORE=persistence` en prod dans D5a (→ **D5b** ; voir [`docs/PERS_D5B_ROLLOUT.md`](./PERS_D5B_ROLLOUT.md))
- Pas de changement `/api/cpm/v1` ni routes internes CPM existantes (proxy jusqu’à D6b)

## Sign-off

| Rôle | Statut | Date |
|------|--------|------|
| CPM métier (§8.2) | Approuvé pour merge spec | 2026-06-25 |
| cafe-persistence (contrat) | Publié `openapi/internal/cp/v1.yaml` | 2026-06-25 |
