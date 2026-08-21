# Cafe CPM — backlog

Items deferred; not blocking current IMM work unless noted.

---

## Capability Providers (ADR 2026-08-03)

See [ADR](https://github.com/create2-labs/cafe-adr/blob/main/ADR_20260803_cp_provider_abstraction.md) / [PR plan](https://github.com/create2-labs/cafe-adr/blob/main/ADR_20260803_cp_provider_abstraction_PR_PLAN.md). **CPM-P1–P11b** delivered through runtime signals. **CPM-P7** (this train): pin Nicetry manifest refs for normative persist. Next: FE amendement train (FE-P6+).

---
## Open — Retirer entièrement le mode CPM mock (frontend)

**Décision produit :** supprimer **tout** le mock CPM côté `**cafe-frontend`** — pas seulement le switch runtime `VITE_CPM_DATA_SOURCE=mock`, mais aussi `**mockCpmDataSource.ts**`, le placeholder `mock-discovery-scan-placeholder`, et l’UI / composables dédiés au parcours démo sans stack.

**Hors scope backend :** `cafe-crypto-policy-mgt` n’expose que l’API HTTP réelle ; aucun changement Go requis pour ce chantier. Les « mocks » dans les tests Go (`httptest`, etc.) restent inchangés.

**Prérequis (déjà en place) :** Option A mergé (F1–F5) ; parcours CPM = scan Discovery v1 + `createApiCpmDataSource` ; dev local `VITE_CPM_DATA_SOURCE=api` dans `cafe-deploy/env/dev.local.env` ; images release frontend déjà en `api`.

### Périmètre `cafe-frontend`


| Zone                              | Action                                                                                                                                     |
| --------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| `src/cpm/mockCpmDataSource.ts`    | **Supprimer** (+ `mockCpmDataSource.spec.ts`)                                                                                              |
| `src/cpm/cpmDataSourceFactory.ts` | Une seule implémentation : `**createApiCpmDataSource`** ; retirer `CpmDataSourceMode`, `normalizeMode`, branche `mock`                     |
| `VITE_CPM_DATA_SOURCE`            | Retirer de `vite-env.d.ts`, Dockerfiles, docs ; build toujours API                                                                         |
| `CryptoPolicyManagement.vue`      | Retirer UI mock-only (wallet type/address éditables, `confirm-mock-signature`, copy « Preparing mock selection », badge mode `mock`, etc.) |
| `useCpmPolicySelection.ts`        | Retirer `CPM_SELECTION_CONTEXT_SCAN_ID` / adresse placeholder ; explore **uniquement** après scan v1 valide                                |
| `useCpmScanContext.ts`            | Retirer option `cpmMode` liée au mock ; garder rejet placeholder si besoin défensif                                                        |
| `walletChallengeEligibility.ts`   | Retirer `mock_pr10` ; EOA = règles API uniquement                                                                                          |
| ESLint / imports fixtures         | Vérifier que les fixtures JSON ne sont importées que par les tests (pattern PR 1)                                                          |


### Tests unitaires / intégration (effet de bord principal)

Les specs qui appellent `**createMockCpmDataSource()`** ou `**mode: 'mock'**` doivent migrer vers :

- `**vi.mock('@/api')**` + réponses axios (modèle `**cpmOptionAFlow.e2e.spec.ts**` / `**apiCpmDataSource.spec.ts**`), ou
- fakes légers implémentant l’interface `**CpmDataSource**` inline dans le fichier de test.

**Fichiers à repasser (liste non exhaustive) :** `cpmDataSourceFactory.spec.ts`, `useCpmPolicySelection.spec.ts`, `usePolicyValidation.spec.ts`, `useLocalPolicyDraftStorage.spec.ts`, `useBackendDraftSave.spec.ts`, `usePolicyPersistence.spec.ts`, `useWalletChallengeGate.spec.ts`, `CryptoPolicyManagement.spec.ts`, composants CPM `*.spec.ts` qui injectent le mock.

**Ne pas confondre :** les mocks Vitest (`vi.mock`) et HTTP stubs ≠ l’ancien « mode mock CPM » produit — on **garde** les mocks de transport pour des tests rapides sans Docker.

**Acceptance tests :** `npm run test` + `npm run typecheck` verts ; `cpmOptionAFlow.e2e.spec.ts` inchangé en intention (déjà API + axios mock).

### `cafe-deploy` + docs


| Fichier                     | Action                                                                                   |
| --------------------------- | ---------------------------------------------------------------------------------------- |
| `env/dev.env.template`      | Retirer `VITE_CPM_DATA_SOURCE` ou documenter valeur fixe `api` puis suppression variable |
| `scripts/redeployalldev.sh` | Retirer `--build-arg VITE_CPM_DATA_SOURCE`                                               |
| `README.md` (cafe-deploy)   | Section « mock | api » → CPM toujours API                                                |


### Docs CPM (ce dépôt)

Mettre à jour les passages « preserve mock mode » / placeholder V1 :

- `[workplans/CPM_post_v_1_option_a_scan_context.md](./workplans/CPM_post_v_1_option_a_scan_context.md)` §13.4
- `[workplans/CPM_FRONTEND_PR_PLAN_V1.md](./workplans/CPM_FRONTEND_PR_PLAN_V1.md)` (historique V1 : noter mock retiré post-Option A)
- `[workplans/CPM_OPTION_A_PR_PLAN.md](./workplans/CPM_OPTION_A_PR_PLAN.md)` — mentions `mock-discovery-scan-placeholder` limitées au passé
- `[docs/CPM_OPTION_A_INTEGRATED.md](./docs/CPM_OPTION_A_INTEGRATED.md)`

### Acceptance (produit)

- Page `**/crypto-policy-management`** : auth + scan wallet Discovery requis pour explore ; aucun chemin UI sans scan réel.
- Aucune occurrence runtime de `**mock-discovery-scan-placeholder**` dans les payloads HTTP CPM.
- Stack locale : CPM utilisable uniquement avec Discovery + **cafe-cpm** (comportement attendu).

### Suggested PR


| Repo                     | Branche (suggestion)         | Titre (suggestion)                                                  |
| ------------------------ | ---------------------------- | ------------------------------------------------------------------- |
| `cafe-frontend`          | `cpm/remove-mock-datasource` | `refactor(cpm): remove mock CpmDataSource and VITE_CPM_DATA_SOURCE` |
| `cafe-deploy`            | (même PR ou follow-up)       | `chore(deploy): drop VITE_CPM_DATA_SOURCE mock default`             |
| `cafe-crypto-policy-mgt` | doc only                     | `docs(cpm): remove mock-mode references from workplans`             |


**Priorité :** après FE-IMM courant / quand la stack `api` est le seul chemin validé en dev.

---

## Open — Catalogue CP : ne plus lister les templates/instances dans `config.go`

**Contexte :** aujourd’hui le contenu du catalogue vit dans les JSON (`internal/domain/policy/testdata/` → `/app/policy/` dans l’image), mais **quels fichiers charger** est une liste explicite :

- défauts `defaultPolicyTemplatePaths` / `defaultPolicyInstancePaths` dans `internal/config/config.go` ;
- surcharge possible via `CPM_POLICY_TEMPLATE_PATHS` / `CPM_POLICY_INSTANCE_PATHS` (env), mais `**cafe-deploy` ne les pose pas** — chaque nouvelle CP oblige à toucher les `const` + rebuild image.

**Problème :** la liste des CP actives est de la **config d’exploitation**, pas de la logique Go. Ajouter une 2ᵉ CP (ex. `crypto_policy_template_pq_account_validation_v1.json`) ne devrait pas impliquer de modifier `config.go`.

**Pistes (à trancher) :**


| Option | Idée                                                                                                                                                   |
| ------ | ------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **A**  | Définir `CPM_POLICY_*_PATHS` dans `cafe-deploy` (`compose/25-cpm.yml` / `env/*.env`) ; laisser des défauts minimaux (une CP) ou vides dans `config.go` |
| **B**  | Fichier manifeste `policy-index.json` à la racine de `/app/policy/` (catalog + liste templates + instances)                                            |
| **C**  | Convention par répertoire : charger tous les `crypto_policy_template_*.json` / `crypto_policy_instance_*.json` valides (exclure `invalid_*`)           |


**Acceptance :** ajouter une CP = nouveaux JSON + rebuild image (ou volume, si un jour adopté) **sans** changement dans `internal/config/config.go` ; tests `LoadReadStore` / explore inchangés en intention ; doc admin (`cafe-documentation/04-cafe-admin-guide.md`) alignée.

**Repos :** `cafe-crypto-policy-mgt` (loader + tests) ; optionnel `cafe-deploy` (env) ; doc `README.md` § Read APIs.

**Priorité :** après validation UI catalogue multi-CP (2ᵉ template de test) ; confort ops / éviter dérive défauts ↔ fichiers embarqués.

---

## IMM-9b follow-up — Factor internal policy-reference handlers

**Context:** `internal/app/policy_reference_internal.go` (`POST /internal/policies/references/scan`) and `internal/app/policy_reference_wallet_target_internal.go` (`POST …/references/wallet-target`) share the same boilerplate: read/limit body, JSON decode, `user_id` + `tenant_id` → `authz.Principal`, validate principal, map persistence errors.

**Improvement:** Extract shared helpers (e.g. `readInternalServiceJSON`, `principalFromInternalUserTenant`) without changing HTTP contracts or auth route inventory.

**Acceptance:** `policy_reference_internal_test.go` and `policy_reference_wallet_target_internal_test.go` unchanged in behaviour; less duplication for future internal lookups.

**Repos:** `cafe-crypto-policy-mgt` only.

**Priorité:** plus tard (refactor confort).

---

## W1 / IMM-9b — Lookup by `target_address` in payload only (not `scan_id`)

**Context:** Internal lookup `POST /internal/policies/references/wallet-target` and `CountActiveWalletCPMContext` match owner policies/drafts by extracting a wallet address from **stored payload** (`policy_context`, `selected_wallet_policy_context`, `wallet_address`, etc.). This is intentional: Discovery must not resolve address via `policy.scan_id → Discovery detail` before allocating a new `scan_id` (**IMM-9b**).

**Implication:** Records with payload that has **no** denormalized address (e.g. PR5-style `{ "x": 1 }` with only `PolicyRecord.ScanID` set) do **not** participate in W1 for any address — rescan is not blocked. Real flows (explore/persist, UI, deploy scripts) must persist `target_address` (or equivalent) in the payload.

**Not a bug:** Do not “fix” by joining on `scan_id` and calling Discovery from CPM for W1. If product requires blocking in legacy shapes, migrate payloads or enforce address fields on `POST …/policies` / `POST …/drafts` at write time.

**Repos:** Document in `workplans/IMMUTABILITE_PR.md` / optional persist validation if product wants stricter guarantees.

**Priorité:** doc / validation produit optionnelle — **pas** une nouvelle implémentation lookup.

---

## IMM-10 follow-up — Centraliser les chemins Discovery (upstream vs public)

**Context:** IMM-10 (W7/W2) et PR13g appellent Discovery avec des littéraux dupliqués, p.ex. :

- `internal/app/auth.go` — `GET …/discovery/v1/wallets/scans` (liste `limit=1`, `latest=true`)
- `internal/app/assessment_request.go` — `GET …/discovery/v1/wallets/scans/{scan_id}` (détail)
- `internal/app/authz_scan_test.go` — mêmes paths dans les mocks HTTP

Le merge `main` ↔ `imm10` a divergé sur `**/discovery/v1/...`** (appel serveur → backend `:8080`) vs `**/api/discovery/v1/...**` (contrat public / edge WORKPLAN). Pour `CAFE_DISCOVERY_HTTP_BASE` pointant sur le backend Discovery, le couple correct est `**/discovery/v1**` (aligné Fiber / compose), pas `/api/discovery/v1` sur `:8080` direct.

**Amélioration:** Introduire des constantes ou helpers CPM (p.ex. package `discoverypaths` / extension de `cpmroutes`, ou petit module partagé) :

- `DiscoveryUpstreamWalletScans` = `/discovery/v1/wallets/scans`
- `DiscoveryUpstreamWalletScanByID(scanID)` = `/discovery/v1/wallets/scans/{scan_id}`
- Optionnel, nommées à part : chemins **public** `/api/discovery/v1/...` si un jour CPM cible l’edge — ne pas les confondre avec l’upstream.

Réutiliser ces symboles dans `auth.go`, `assessment_request.go` et les tests (mocks) pour éviter une prochaine régression au merge.

**Acceptance:** Un seul endroit définit les paths upstream ; `go test ./internal/app` inchangé ; smoke `cafe-deploy/scripts/test-cpm-imm10-wallet-scan-w7-w2-guards.sh` OK avec `CAFE_DISCOVERY_HTTP_BASE` → backend.

**Repos:** `cafe-crypto-policy-mgt` ; aligner doc `internal/config` et éventuellement `scripts/lib/discovery-route-paths.sh` (bash) sur le même vocabulaire.

**Priorité:** plus tard.