# Scan immutability & CPM — PR plan (CPM)

> **English product summary:** [CAFE functional specifications](https://github.com/create2-labs/cafe-documentation/blob/main/functional-specifications.md) and [technical specifications](https://github.com/create2-labs/cafe-documentation/blob/main/technical-specifications.md).

**Source de vérité :** `[WORKPLAN_API.md](./WORKPLAN_API.md)` — **§0**, **§2.2** (couplage **W1–W8**, query `**latest=true`**), **§2.2.1 P1**, **§4.4**, **§8.6**.

**All API paths in this document refer to the canonical public prefixes defined in WORKPLAN_API.md: `/api/discovery/v1` and `/api/cpm/v1`.**

**Ce fichier** = découpage PR CPM et coordination Discovery.

**Plans liés :**


| Dépôt     | Document                                                                       |
| --------- | ------------------------------------------------------------------------------ |
| Discovery | `[cafe-discovery/IMMUTABILITE_PR.md](../../cafe-discovery/IMMUTABILITE_PR.md)` |
| Frontend  | `[cafe-frontend/IMMUTABILITE.md](../../cafe-frontend/IMMUTABILITE.md)`         |


**Déjà livré :** policies par `scan_id` (**PR7**), référence interne scan (**PR5**), explore Option A (**PR8**), **DELETE scan → 409** (**PR6**, **W3**), **IMM-9b** lookup W1 ([#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38)), **IMM-10** W7/W2 ([#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40)), **IMM-W1-1…3** `**DELETE /api/cpm/v1/drafts?id=…`** ([#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41)–[#44](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/44)), **normalisation adresse wallet** ([#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45) NATS, [#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46) assessment HTTP), **CPM-DRAFT-1A** gel contrat brouillon plateforme (docs — `[CPM_DRAFT_1_PR.md](./CPM_DRAFT_1_PR.md)`) — smokes `test-cpm-imm-w1-delete-draft.sh`, `test-discovery-imm9-wallet-scan-w1-cpm-block.sh` step 5.

**Suite CPM (backlog) :** voir `[TODO.md](../TODO.md)` — refactor handlers internes, chemins Discovery upstream, doc lookup payload-only.

**Périmètre CPM produit actuel :** assessment/remediation **wallet-only** (scans **wallet / EVM**). Les scans **TLS** restent dans **Discovery** (historique, CBOM, observation) — **pas** de cible CPM assessment/remediation pour cette release. Voir **WORKPLAN §5.4.6**.

---

## Principe W1 — policy bloque le rescan ; draft orphelin + rebind CPM (tranché 2026-06)

**Décision produit :**

- **Policy persistée** sur une `**target_address`** (wallet) → **aucun** nouveau `**POST /api/discovery/v1/scan`** (**409**).
- **Draft plateforme seul** → **n’empêche pas** le rescan (**IMM-W1-4** / révision **IMM-9**).
- Après rescan, le draft peut rester sur un **ancien** `**scan_id`** (**orphan**) → **explore / validate / persist** CPM **bloqués** jusqu’à **rebind manuel** sur le dernier scan `**completed`** (**W2**) — bouton **« Rebind to last scan for this address »** ; `**wallet_type`** inchangé requis.

**Parcours W1 (policy) :** Discovery **409** si policy (**IMM-9**) ; CTAs finalize / `**DELETE /api/cpm/v1/policies?id=…`**. `**DELETE /api/cpm/v1/drafts?id=…`** (**IMM-W1-2…3** ✅) = abandon draft — **pas** prérequis rescan.

**Hors scope :** sauvegarde locale client (export / `localStorage` / reload) — **retiré du périmètre produit (2026-06)**. Voir `[cafe-frontend/IMMUTABILITE_PR.md](../../cafe-frontend/IMMUTABILITE_PR.md)` (**FE-IMM-4** = rebind orphan uniquement).

---

## W7 vs W2 — ne pas confondre les résolutions Discovery


| Règle        | Résolution Discovery                                                         | Exemple                                                                    |
| ------------ | ---------------------------------------------------------------------------- | -------------------------------------------------------------------------- |
| **W2** (CPM) | **Dernier scan `completed`** uniquement                                      | `GET /api/discovery/v1/wallets/scans?address=0x…&latest=true`              |
| **W7** (CPM) | **Newest row** par `created_at` desc, `scan_id` desc — **pas** `latest=true` | `GET /api/discovery/v1/wallets/scans?address=0x…&limit=1` (tri par défaut) |


- **Ne jamais** utiliser `latest=true` pour **W7**.
- **Ne jamais** utiliser `limit=1` seul comme substitut de **W2** (la ligne la plus récente peut être `requested`, `started` ou `failed`).

---

## Règles WORKPLAN (responsabilité CPM)


| ID        | Règle                                                                                                                                  | CPM                                                                                                                                                      |
| --------- | -------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **W1**    | **Policy persistée** bloque le rescan ; **draft seul** non (**IMM-W1-4**) ; **orphan draft** → rebind **W2** avant explore/persist     | Lookup policies pour guard POST (**IMM-9b**) ; drafts pour orphan CPM                                                                                    |
| **W2**    | CPM sur le dernier `**completed`** uniquement                                                                                          | `**GET …/wallets/scans?address=&latest=true`** (**IMM-10**)                                                                                              |
| **W3–W6** | DELETE / historique / CBOM                                                                                                             | Voir WORKPLAN                                                                                                                                            |
| **W7**    | Pas d’explore/persist tant que le **newest row** n’est pas `**completed`**                                                             | explore/persist → **400** `LATEST_SCAN_NOT_COMPLETED`                                                                                                    |
| **W8**    | **Rescan** : `POST …/scan` refusé seulement si scan **en cours** ; **OK** si newest `**failed`** (sous **W1**) — **indépendant de W7** | Implémenté côté **Discovery** (**IMM-4**, **IMM-9**)                                                                                                     |
| **P1**    | Quota plan (success-only, monotonic)                                                                                                   | **Hors CPM** — Discovery **IMM-6b** ; `**used`** = completed success ; DELETE policy/draft/scan **ne modifie pas** `**used`** ; voir WORKPLAN **§2.2.1** |


> **Discovery POST** : **W8** (`SCAN_IN_PROGRESS` si en cours) ; puis **W1** (**policy persistée** uniquement — **IMM-W1-4**). **W7** ne s’applique **pas** à POST.

---

## Table de suivi (CPM)


| PR plan           | GitHub                                                                | Dépôt                    | Dépend de           | Objectif                                                            |
| ----------------- | --------------------------------------------------------------------- | ------------------------ | ------------------- | ------------------------------------------------------------------- |
| **IMM-9b**        | [#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38) | `cafe-crypto-policy-mgt` | PR7                 | **W1** : lookup policy **+** draft par `target_address` normalisée  |
| **IMM-10**        | [#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40) | `cafe-crypto-policy-mgt` | Discovery **IMM-4** | **W7** (newest row) + **W2** (`latest=true`)                        |
| **IMM-W1-1**      | [#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41) | `cafe-crypto-policy-mgt` | IMM-9b              | `**DeleteDraft`** sur `OwnerScopedStore`                            |
| **IMM-W1-2**      | [#42](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/42) | `cafe-crypto-policy-mgt` | **IMM-W1-1**        | Handler `**DELETE /api/cpm/v1/drafts?id=…`** + OpenAPI              |
| **IMM-W1-2**      | [#43](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/43) | `cafe-crypto-policy-mgt` | **IMM-W1-1**        | Handler `**DELETE /api/cpm/v1/drafts?id=…`** + OpenAPI              |
| **IMM-W1-3**      | [#44](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/44) | `cafe-crypto-policy-mgt` | **IMM-W1-2**        | Tests routes + smoke deploy step 5 (rescan after delete draft)      |
| **IMM-9b-norm-1** | [#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45) | `cafe-crypto-policy-mgt` | IMM-9b              | `NormalizeWalletSubjectID` — consumer NATS assessment               |
| **IMM-9b-norm-2** | [#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46) | `cafe-crypto-policy-mgt` | **#45**             | `WalletSubjectIDFromAddress` — `POST …/policies/assessment/request` |
| **IMM-DOC-1**     | ✅ done                                                                | `cafe-crypto-policy-mgt` | —                   | WORKPLAN **§5.1.1** + **§5.4.10** ; cross-links IMM-D1…D5, FE-IMM-10…14, IMM-DEP-1 |
| **IMM-OPS-1**     | [#50](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/50)                                                                     | `cafe-crypto-policy-mgt` | IMM-10 ✅            | Log structuré + métrique Prometheus explore sans candidat deployable (**REQ9**) |
| **IMM-OPS-2**     | —                                                                     | `cafe-deploy`            | **IMM-OPS-1**       | Dashboard Grafana + alerte hausse soutenue rejection codes          |
| **IMM-OPS-3**     | —                                                                     | TBD                      | **IMM-OPS-1**, **IMM-OPS-2** | Future visibilité produit/admin sur gaps couverture CP — **non bloquant** |


---

## Tableau de suivi des PR (pilotage)

Convention statuts: `✅ done` | `🟡 in progress` | `⚪ planned`.


| PR                | Statut    | Branche                                       | GitHub                                                                                          | Dépendances     | Résultat clé                                                                              |
| ----------------- | --------- | --------------------------------------------- | ----------------------------------------------------------------------------------------------- | --------------- | ----------------------------------------------------------------------------------------- |
| **IMM-9b**        | ✅ done    | `cpm/imm-9b-owner-lookup`                     | [#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38)                           | PR7             | Lookup W1 policy + draft par `target_address` normalisée                                  |
| **IMM-10**        | ✅ done    | `cpm/imm-10-w7-w2-gates`                      | [#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40)                           | Discovery IMM-4 | Garde W7 (`limit=1`) + W2 (`latest=true`)                                                 |
| **IMM-W1-1**      | ✅ done    | `cpm/imm-w1-delete-draft-store`               | [#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41)                           | IMM-9b          | `OwnerScopedStore.DeleteDraft`                                                            |
| **IMM-W1-2**      | ✅ done    | `cpm/imm-w1-delete-draft-route`               | [#42](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/42)                           | IMM-W1-1        | Route `DELETE /api/cpm/v1/drafts?id=…`                                                    |
| **IMM-W1-2**      | ✅ done    | `cpm/imm-w1-delete-draft-route`               | [#43](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/43)                           | IMM-W1-1        | Ajustements route + OpenAPI                                                               |
| **IMM-W1-3**      | ✅ done    | `cpm/imm-w1-delete-draft-smoke`               | [#44](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/44)                           | IMM-W1-2        | Tests intégration + smoke step 5 (legacy : rescan after delete draft)                     |
| **IMM-W1-4**      | ✅ done    | `discovery/w1-policy-only-post-guard`         | livré in-tree + smoke step 5                                                                    | IMM-9 ✅         | Révision **W1** : POST scan **409** si **policy** seule ; draft seul OK ; `blocking_kind` |
| **IMM-9b-norm-1** | ✅ done    | `cpm/imm-9b-wallet-normalization-nats`        | [#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45)                           | IMM-9b          | Normalisation NATS `NormalizeWalletSubjectID`                                             |
| **IMM-9b-norm-2** | ✅ done    | `cpm/imm-9b-wallet-normalization-http`        | [#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46)                           | #45             | Normalisation HTTP `WalletSubjectIDFromAddress`                                           |
| **CPM-DRAFT-1A**  | ✅ done    | `cpm/cpm-draft-1a-contract-docs`              | [#67](https://github.com/create2-labs/cafe-frontend/pull/67) (merge en meme temps que FE-IMM-1) | —               | OpenAPI + WORKPLAN §4.4.1 + README + `[CPM_DRAFT_1_PR.md](./CPM_DRAFT_1_PR.md)`           |
| **CPM-DRAFT-1B**  | ✅ done    | `cpm/cpm-draft-1b-backend-dto`                | [#47](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/47)                           | 1A              | `DraftUpsertRequest`/`Response`, validation, erreurs structurées                          |
| **CPM-DRAFT-1C**  | ✅ done    | `cpm/cpm-draft-1c-contract-tests`             | [#48](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/48)                           | 1B              | Tests contrat POST/GET/DELETE + cross-owner                                               |
| **IMM-DOC-1**     | ✅ done    | `cpm/doc-chain-all-or-nothing-data-integrity` | —                                                                                               | —               | WORKPLAN **§5.1.1** (chains tout-ou-rien) + **§5.4.10** (scanners-only `result`) ; cross-links IMM-D1…D5, FE-IMM-10…14, IMM-DEP-1 |
| **IMM-OPS-1**     | ⚪ planned | `cpm/explore-no-candidate-observability`      | —                                                                                               | IMM-10 ✅        | Hook explore post-Evaluate ; log `cpm.explore.no_deployable_candidate` ; compteur Prometheus ; `GET /metrics` minimal (**REQ9**) |
| **IMM-OPS-2**     | ⚪ planned | `deploy/cpm-explore-rejection-dashboard`      | `cafe-deploy`                                                                                   | **IMM-OPS-1**   | Dashboard Grafana + alerte warning hausse soutenue `incompatible.chain_scope`             |


> **CPM-DRAFT-1** : gel contrat brouillon plateforme — **1A/B/C ✅** ([#47](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/47)–[#48](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/48)). Frontend **CPM-DRAFT-2** ✅ ([#68](https://github.com/create2-labs/cafe-frontend/pull/68)) — voir `[cafe-frontend/IMMUTABILITE_PR.md](../../cafe-frontend/IMMUTABILITE_PR.md)`.

---

## Tableau de suivi FE-IMM (cafe-frontend)

Source de référence frontend: `[cafe-frontend/IMMUTABILITE_PR.md](../../cafe-frontend/IMMUTABILITE_PR.md)`.

> **Renumerotation 2026-06 :** ancien **FE-IMM-4** (CBOM) → **FE-IMM-8** ; **FE-IMM-1b** (export local) hors périmètre ; **FE-IMM-4** actuel = gate draft orphelin + rebind W2.


| PR FE-IMM       | Statut    | Branche                                    | GitHub                                                                                                                     | Dépendances                                                  | Objectif                                                                                             |
| --------------- | --------- | ------------------------------------------ | -------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------- |
| **FE-IMM-0**    | ✅ done    | n/a                                        | [#66](https://github.com/create2-labs/cafe-frontend/pull/66)                                                               | IMM-6b-5                                                     | P1 quota breakdown dashboard                                                                         |
| **FE-IMM-1**    | ✅ done    | `fe-imm/pr-1-api-data-layer`               | [#67](https://github.com/create2-labs/cafe-frontend/pull/67) (merge en meme temps que CPM-DRAFT-1A)                        | —                                                            | API/data layer (wallet scans, POST scan, draft/policy delete, CBOM client)                           |
| **FE-IMM-2**    | ✅ done    | `fe-imm/pr-2-scan-eligibility-model`       | [#69](https://github.com/create2-labs/cafe-frontend/pull/69)                                                               | FE-IMM-1                                                     | Modèle d'éligibilité scan (W8, G1, G2)                                                               |
| **FE-IMM-3**    | ✅ done    | `fe-imm/pr-3-scan-trigger-ui`              | [#70](https://github.com/create2-labs/cafe-frontend/pull/70), [#72](https://github.com/create2-labs/cafe-frontend/pull/72) | FE-IMM-1, FE-IMM-2, **IMM-W1-4** ✅                           | UI trigger scan ; **policy** W1 on POST ; draft-only informational — **no** Save locally             |
| **FE-IMM-4**    | ✅ done    | `fe-imm/pr-4-draft-rebind-orphan-gate`     | [#82](https://github.com/create2-labs/cafe-frontend/pull/82)                                                               | FE-IMM-1, FE-IMM-5, FE-IMM-11, CPM-DRAFT-2 ✅, **IMM-W1-4** ✅ | Orphan draft gate ; **Rebind to last scan** ; explore/persist blocked until W2 ; **no** local export |
| **FE-IMM-5**    | ✅ done    | `fe-imm/pr-5-cpm-scan-selector-w2-w7`      | [#73](https://github.com/create2-labs/cafe-frontend/pull/73)                                                               | FE-IMM-1                                                     | Séparation W2 (`latest=true`) / W7 (`limit=1`)                                                       |
| **FE-IMM-6**    | ✅ done    | `fe-imm/pr-6-delete-policy-ux`             | [#74](https://github.com/create2-labs/cafe-frontend/pull/74)                                                               | FE-IMM-1                                                     | UX suppression policy (W3/W4)                                                                        |
| **FE-IMM-7**    | ✅ done    | `fe-imm/pr-7-delete-scan-ux`               | [#76](https://github.com/create2-labs/cafe-frontend/pull/76)                                                               | FE-IMM-1, FE-IMM-6                                           | UX suppression scan + copy P1                                                                        |
| **FE-IMM-8**    | ✅ done    | `fe-imm/pr-8-cbom-row-ux`                  | [#75](https://github.com/create2-labs/cafe-frontend/pull/75)                                                               | FE-IMM-1                                                     | UX lien CBOM uniquement sur scan success completed *(ex-ancien FE-IMM-4)*                            |
| **FE-IMM-9**    | ⚪ planned | `fe-imm/pr-9-imm-regression-tests`         | —                                                                                                                          | FE-IMM-1 à FE-IMM-8                                          | Couverture tests/régressions IMM                                                                     |
| **FE-IMM-10**   | ✅ done    | `fe-imm/pr-10-scan-mappers-no-defaults`    | [#77](https://github.com/create2-labs/cafe-frontend/pull/77)                                                               | FE-IMM-1, **IMM-D4**                                         | Supprimer defaults EOA dans mappers scan                                                             |
| **FE-IMM-11**   | ✅ done    | `fe-imm/pr-11-cpm-wallet-type-from-detail` | [#78](https://github.com/create2-labs/cafe-frontend/pull/78)                                                               | FE-IMM-10, **IMM-D5** ✅                                      | CPM : wallet type depuis detail only                                                                 |
| **FE-IMM-12**   | ✅ done    | `fe-imm/pr-12-scans-wallet-type-v1`        | [#79](https://github.com/create2-labs/cafe-frontend/pull/79)                                                               | FE-IMM-10                                                    | UI : `wallet_type` canonique v1                                                                      |
| **FE-IMM-13**   | ✅ done    | `fe-imm/pr-13-cpm-explore-rejection-ux`    | [#80](https://github.com/create2-labs/cafe-frontend/pull/80)                                                               | FE-IMM-1, FE-IMM-5 (**REQ8**)                                | Bannière explore : pourquoi aucune policy deployable                                                 |
| **FE-IMM-14**   | ✅ done    | `fe-imm/pr-14-data-integrity-tests`        | [#81](https://github.com/create2-labs/cafe-frontend/pull/81)                                                               | FE-IMM-10…13                                                 | Tests data integrity + explore rejection                                                             |
| **CPM-DRAFT-2** | ✅ done    | `fe-imm/cpm-draft-2-adoption`              | [#68](https://github.com/create2-labs/cafe-frontend/pull/68)                                                               | CPM-DRAFT-1 ✅                                                | `savePolicyDraft` → contrat gelé ; erreurs structurées UX                                            |


---

## IMM-W1-1 — `DeleteDraft` (store)

- **Status:** mergee dans [#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41) 
- **Branch:** `cpm/imm-w1-delete-draft-store`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Suppression owner-scoped d’un brouillon plateforme — miroir `**DeletePolicy`**.
- **Scope:**
  - `OwnerScopedStore.DeleteDraft(principal, id)` → `**ErrDraftNotFound`**, `**ErrForbidden`**
  - Tests unitaires `owner_scoped_store_test.go`
- **Out of scope:** Route HTTP.
- **Dependencies:** IMM-9b (mergé)
- **Proposed commit title:** `cpm: add OwnerScopedStore.DeleteDraft for W1 unblock`
- **Completion criteria:** Tests verts ; sémantique identique à **DELETE policy** (404 idempotent côté handler à venir).

---

## IMM-W1-2 — Route HTTP `DELETE …/drafts?id=`

- **Status:** mergee dans [#42](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/42)
- **Branch:** `cpm/imm-w1-delete-draft-route`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** `**DELETE /api/cpm/v1/drafts?id=…`** → **204**  **404** (**WORKPLAN §2.4**, **§4.4**).
- **Scope:**
  - `registerOwnerScopedRoutesForPrefix` — handler `**DELETE`** sur `**/drafts`**
  - Même auth / owner scope que **GET** / **POST** drafts
  - `**openapi/cpm-v1.yaml`**
- **Out of scope:** Discovery ; frontend.
- **Dependencies:** **IMM-W1-1**
- **Proposed commit title:** `cpm: add DELETE /api/cpm/v1/drafts?id= for W1 rescan unblock`
- **Completion criteria:** OpenAPI aligné ; **204** si existait ; **404** si déjà absent.

---

## IMM-W1-3 — Tests intégration + smoke deploy

- **Status:** mergee dans [#44](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/44)
- **Branch:** `cpm/imm-w1-delete-draft-smoke`
- **Repository:** `cafe-crypto-policy-mgt` (+ script `cafe-deploy` si nécessaire)
- **Objective:** Verrouiller le parcours **draft → DELETE draft → POST scan OK**.
- **Scope:**
  - `owner_routes_test.go` — DELETE draft 204/404, cross-owner
  - Compléter `**cafe-deploy/scripts/test-discovery-imm9-wallet-scan-w1-cpm-block.sh`** step 5 (rescan **200** après DELETE draft)
- **Out of scope:** Frontend FE-IMM-1.
- **Dependencies:** **IMM-W1-2**, Discovery IMM-9 (mergé)
- **Proposed commit title:** `test: W1 unblock via DELETE platform draft`
- **Completion criteria (IMM-W1-3 legacy smoke):** step 5 documented — **IMM-W1-4** replaces « rescan after DELETE draft » as W1 unblock path ; DELETE draft remains valid for **discard** only.

---

## IMM-9b

**Type:** Technical task (**WORKPLAN §2.2 W1**).

**Status** : mergée dans [#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38)

### Summary

Owner-scoped internal lookup **before** Discovery allocates a new `scan_id` on `**POST /api/discovery/v1/scan`**. Discovery knows the **normalized target address** from the request body — not a new `scan_id`. **Do not** require Discovery to resolve address via `policy.scan_id → Discovery detail → address`.

**Product rule:** at most **one active CPM context per normalized wallet address** — user must finalize policy or delete platform draft (and policy if needed) before rescan.

### Contract (internal, service token)

- **Input:** normalized `**target_address`** (same normalization as Discovery wallet scans), owner scope from service auth.
- **Output (minimal):**

```json
{ "exists": true, "policy_count": 1, "draft_count": 0 }
```

or `{ "exists": false, "policy_count": 0, "draft_count": 0 }`.

### Public draft deletion (discard)

Platform draft removal uses the canonical public API:

- `**DELETE /api/cpm/v1/drafts?id=…`** → `**204`** / **404** (**WORKPLAN §2.4**, **§4.4**) — **IMM-W1-2**.

Under **IMM-W1-4**, DELETE draft is **discard** only — it is **not** required to unblock rescan (draft alone no longer blocks POST). Rescan unblock requires removing **persisted policy** (**409** when `policy_count > 0`).

**Client workflow (not IMM-9b):** after rescan, an **orphan platform draft** (stale `scan_id`) blocks CPM explore/persist until **manual rebind** to the latest `**completed`** scan (**W2**) — **FE-IMM-4** in `[cafe-frontend/IMMUTABILITE_PR.md](../../cafe-frontend/IMMUTABILITE_PR.md)`. **No** client-side local draft export / reload.

### Acceptance criteria

- Internal API or shared module (service token); callable **before** new scan creation.
- Lookup by **normalized `target_address`** only — no Discovery round-trip for address resolution.
- Response: `{ exists, policy_count, draft_count, platform_draft_id? }`.
- **POST scan guard (Discovery IMM-W1-4):** **409** when `**policy_count > 0`** — draft alone does **not** block.
- `**wallet-target-context`** may still expose draft for informational UI / orphan detection.
- Wallet-only scope: TLS scans **out of** W1 lookup for current product flow.
- Tests: policy → true; draft → true; false when both removed.

### Normalisation adresse (follow-up, livré)

- **Status :** [#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45) (NATS), [#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46) (assessment HTTP).
- **Règle canonique :** `NormalizeWalletTargetAddress` dans `internal/persistence/wallet_target.go` (même normalisation que Discovery).
- **Helpers :** `NormalizeWalletSubjectID` (NATS) ; `WalletSubjectIDFromAddress` (HTTP assessment depuis `target_address` Discovery).
- Plus de `normalizeHexAddress` / `normalizeWalletSubjectForAssessment` locaux.

---

## IMM-10

**Type:** Technical task (**WORKPLAN §2.2 W7**, **W2**).

**Status** : mergée dans [#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40)

### Summary

On `**POST /api/cpm/v1/policies/decisions/explore`** and `**POST /api/cpm/v1/policies`** for **wallet** targets (**Option A**, `binding=discovery`), enforce guards in order:

### Step 1 — W7 (newest row, not `latest=true`)

1. Fetch the **newest** wallet scan row for the target address (default list sort: `created_at` desc, `scan_id` desc).
  Example:
2. If `newest.status != completed` → reject with `**400`** `**LATEST_SCAN_NOT_COMPLETED`**.

### Step 2 — W2 (`latest=true`, not `limit=1`)

1. Fetch the latest `**completed`** scan:
  ```http
   GET /api/discovery/v1/wallets/scans?address=0x…&latest=true
  ```
2. If requested `scan_id` ≠ `scan_id` returned by `**latest=true**` → `**400**` `**SCAN_ID_NOT_LATEST_FOR_TARGET**`.

### Acceptance criteria

- **W7** runs **before** **W2** on every explore/persist.
- **W7** uses newest row (`limit=1` + default sort) — **never** `latest=true`.
- **W2** uses `**latest=true`** — **never** `limit=1` alone.
- Wallet-only: no TLS assessment/remediation path in this PR.

### Dependencies

Discovery **IMM-4** (`latest=true`, default sort, lifecycle enum).

---

## Critères d’acceptation (CPM)

- **W1** : lookup policy **+** draft par `**target_address`** normalisée pour **IMM-9**.
- `**DELETE /api/cpm/v1/drafts?id=…`** livré (**IMM-W1-2**, [#42](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/42)) — **WORKPLAN §0.2**.
- **W7** : newest row testé — **pas** `latest=true`.
- **W2** : `latest=true` testé — **pas** `limit=1` seul.
- **W3** / **W4** inchangés (**DELETE policy** déjà livré).
- TLS : hors scope assessment/remediation CPM produit actuel.

---

## Ordre recommandé (extrait chaîne IMM)

Voir `[cafe-discovery/IMMUTABILITE_PR.md](../../cafe-discovery/IMMUTABILITE_PR.md)` pour la séquence complète. Côté CPM :

1. **IMM-W1-1** → **IMM-W1-3**, **IMM-9b**, **IMM-10**, **IMM-9b-norm-1…2** ([#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45)–[#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46)) — mergés.
2. Backlog CPM : `[TODO.md](../TODO.md)` (refactor handlers internes, chemins Discovery upstream).
3. **IMM-DOC-1** ✅ — WORKPLAN **§5.1.1** + **§5.4.10** (liens Discovery **IMM-D1…D5**, frontend **FE-IMM-10…14**, deploy **IMM-DEP-1**).
4. **IMM-OPS-1…2** — observabilité plateforme explore sans candidat deployable (**REQ9**) ; **FE-IMM-13** / **REQ8** ✅ livré en amont.
5. **IMM-OPS-3** — cadrage futur visibilité admin produit (deferred, non bloquant pour **IMM-OPS-1…2**).

**Parallèle Discovery :** **IMM-6b-1…8** (quota **P1/G1–G4** — success-only, garde POST, commit atomique, CBOM) — voir `[cafe-discovery/IMMUTABILITE_PR.md](../../cafe-discovery/IMMUTABILITE_PR.md)`.

**Parallèle Discovery (data integrity) :** **IMM-D1…D5**, **IMM-DEP-1** — voir `[cafe-discovery/IMMUTABILITE_PR.md § Scan data integrity](../../cafe-discovery/IMMUTABILITE_PR.md#scan-data-integrity-principe--scanners-only)`.

Frontend **FE-IMM-*** : `[cafe-frontend/IMMUTABILITE.md](../../cafe-frontend/IMMUTABILITE.md)` — après **IMM-W1-3** et **IMM-6b-5** minimum.

---

## CPM explore — chains (W1 inchangé)

**Décision (2026-06) :** une policy persistée par adresse (**W1**). `selection_request.target_chain_ids` est un contrat **tout-ou-rien** : chaque chain demandée doit entrer dans le scope instance/catalog pour un candidat **deployable**. Pas de couverture partielle silencieuse.

**Exemple :** discovered/requested chains `[1, 2, 5]` ; candidate scope `[1, 3, 5]` → rejet attendu `incompatible.chain_scope` (chaîne `2` manquante). Le moteur de compatibilité reste inchangé.


| Rôle                            | PR / artefact                                                              |
| ------------------------------- | -------------------------------------------------------------------------- |
| Spec WORKPLAN                   | **IMM-DOC-1** ✅ (**§5.1.1**, **§5.4.10**)                                  |
| UX user : expliquer le rejet    | **FE-IMM-13** ✅ (**REQ8**)                                                |
| Ops : instrumenter CPM          | **IMM-OPS-1** (**REQ9**) — hook post-Evaluate ; log + compteur dominant-code ; `GET /metrics` |
| Ops : visualiser / alerter      | **IMM-OPS-2** (**REQ9**) — Grafana dashboard + alerte hausse soutenue     |
| Admin produit (futur)           | **IMM-OPS-3** — deferred ; synthèse gaps couverture CP, non bloquant        |
| Moteur compatibilité            | Inchangé (`compatibility_result.go` — blocking `incompatible.chain_scope`) |


---

## IMM-DOC-1 — WORKPLAN: chains & data integrity

- **Status:** ✅ done (in-tree)
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Document explicit product rules in `WORKPLAN_API.md` (and cross-link Discovery/frontend plans).
- **Scope (livré) :**
  - **[`WORKPLAN_API.md` §5.1.1](./WORKPLAN_API.md#511-explore--périmètre-chaînes-target_chain_ids-tout-ou-rien)** : `target_chain_ids` tout-ou-rien vs `scope.chain_ids` ; rejet `incompatible.chain_scope`.
  - **[`WORKPLAN_API.md` §5.4.10](./WORKPLAN_API.md#5410-intégrité-des-données-scan-scanners-only)** : scanners only écrivent `result` ; `OnStarted` lifecycle-only.
  - **§8.6** / **§8.8** : critères explore chains + data integrity.
  - Cross-links : Discovery **IMM-D1…D5**, **FE-IMM-10…14**, **IMM-DEP-1**.
- **Out of scope:** Changing compatibility evaluator semantics ; **`cafe-documentation`** (specs EN publiques — backlog séparé).
- **Proposed commit title:** `docs(cpm): document chain scope and scan data integrity rules`

---

## IMM-OPS-1 — Explore: no deployable candidate observability

- **Status:** ⚪ planned
- **Branch:** `cpm/explore-no-candidate-observability`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Donner de la visibilité plateforme quand `POST /api/cpm/v1/policies/decisions/explore` retourne HTTP **200**, `selected_policy_id` vide, et `rejected_candidates` non vide (**REQ9**).
- **Rationale:** Ce cas n’est **pas** une erreur technique. C’est un signal produit / ops indiquant que Discovery a trouvé un contexte wallet exploitable, mais que CPM ne peut pas proposer de Crypto Policy deployable. Cela peut révéler :
  - un gap du catalogue de policies ;
  - un mismatch de chain scope ;
  - une couverture multi-chain incomplète ;
  - un cas utilisateur fréquent à traiter côté produit.

### Point d’accroche (handler explore)

Instrumenter **uniquement** dans `internal/api/read_api.go`, **après** `PolicyDecisionEvaluator.Evaluate`, **avant** `respondJSON(200, …)` :

| Condition | Instrumenter ? |
| --------- | -------------- |
| `len(decision.RankedCandidates) == 0` **et** `len(decision.RejectedCandidates) > 0` | **Oui** — log + **un** increment métrique |
| Candidat sélectionné (`len(RankedCandidates) > 0`) | **Non** |
| `rejected_candidates` vide | **Non** |
| Erreurs HTTP (400 guards W7/W2, decode, evaluate error, auth) | **Non** |

Le hook ne modifie **pas** le corps JSON de la réponse explore.

### Prometheus — compteur

**Nom :** `cpm_explore_no_deployable_candidate_total`

**Labels autorisés (faible cardinalité uniquement) :**

| Label | Sémantique |
| ----- | ---------- |
| `rejection_code` | Code **dominant** de l’événement — **un seul** increment par événement, **pas** un increment par code. Priorité : (1) `incompatible.chain_scope` si présent parmi les rejets ; (2) sinon premier autre code bloquant stable si disponible ; (3) sinon `unknown`. |
| `wallet_type` | Valeur canonique du `policy_context` si disponible ; sinon `unknown`. |
| `binding` | `discovery` si `scan_id` présent (corps ou `policy_context`) ou si le `policy_context` indique un contexte Discovery ; sinon `unknown`. **Ne pas** inventer `fixture`, `catalog` ou `none` si le handler ne peut pas les dériver clairement. |
| `missing_chain_count` | Bucket string : `0`, `1`, `2`, `3`, `4_plus`, ou `unknown` — voir § missing_chain_count ci-dessous. |

**Interdits comme labels Prometheus :** `scan_id`, wallet address, wallet address hash, `chain_ids`, `policy_instance_id`, catalog ids, `request_id`, `tenant_id`, `owner_id`.

**Dépendance :** ajouter `prometheus/client_golang` ; package `internal/metrics` ; exposer `GET /metrics` via `promhttp.Handler()` dans `internal/app/app.go`. **Pas** de configuration scrape `cafe-deploy` dans cette PR — **IMM-OPS-2**.

### missing_chain_count (label bucket)

Pour les rejets `incompatible.chain_scope` :

1. Calculer, par candidat rejeté avec ce code, les chaînes de `selection_request.target_chain_ids` absentes du `scope.chain_ids` du candidat (quand les données sont disponibles).
2. S’il y a **plusieurs** candidats `incompatible.chain_scope`, prendre le **minimum** de chaînes manquantes — le candidat le plus proche d’être deployable.
3. Bucketiser ce minimum : `0` → `0`, `1` → `1`, `2` → `2`, `3` → `3`, `≥4` → `4_plus`.
4. Si les données ne sont pas disponibles de manière fiable → `unknown`.

Les vrais `missing_chain_ids` vont dans le **log structuré**, jamais en label.

### Logs structurés

**Event name :** `cpm.explore.no_deployable_candidate`

**Champs autorisés dans le log (investigation) :**

- `event` (nom ci-dessus)
- `scan_id` si disponible
- `wallet_address_hash` — adresse **normalisée** puis hashée ; **jamais** l’adresse brute
- `wallet_type`
- `requested_chain_ids` (`selection_request.target_chain_ids`)
- `observed_chain_ids` (chains du `policy_context` / observation)
- `candidate_chain_ids` si disponibles (scope des candidats rejetés)
- `missing_chain_ids` si disponibles
- `rejected_candidates_count`
- `rejection_codes` (liste agrégée)
- `dominant_rejection_code` (même règle de priorité que le label Prometheus)
- candidate policy `instance_id` / `template_id` / catalog ids si disponibles
- `request_id` si le pattern `X-Request-Id` existe déjà sur la requête

**Hash wallet :** SHA-256 tronqué (ex. 16 hex chars) si aucun pattern existant dans le repo ; normaliser via `NormalizeWalletTargetAddress` avant hash.

### Implémentation ciblée (fichiers attendus)

| Fichier / zone | Rôle |
| -------------- | ---- |
| `internal/metrics/` | Compteur `cpm_explore_no_deployable_candidate_total` |
| `internal/api/explore_observability.go` (ou équivalent) | Helper log + labels + `missing_chain_count` |
| `internal/api/read_api.go` | Hook post-Evaluate |
| `internal/app/app.go` | `GET /metrics` |
| `*_test.go` | Voir critères ci-dessous |

### Tests minimum

- Test moteur ou helper : produire / reconnaître `incompatible.chain_scope` (ex. `target_chain_ids` `[1,2,5]` vs scope candidat `[1,3,5]`).
- Test helper observabilité :
  - ranked vide + rejected non vide → increment métrique + log émis ;
  - candidat sélectionné → **pas** d’increment ;
  - rejected vide → **pas** d’increment ;
  - labels basse cardinalité (pas de `scan_id` / adresse en labels) ;
  - `missing_chain_count` bucketisé correctement ;
  - log ne contient **pas** l’adresse brute, seulement le hash (si testable sans fragilité excessive).

### Out of scope (strict)

- Sémantique du moteur de compatibilité (`compatibility_result.go`) ;
- **FE-IMM-13** / UX user ;
- corps JSON explore, OpenAPI explore ;
- Grafana / alerting ;
- `cafe-deploy` (scrape Prometheus, dashboards, edge public) ;
- **IMM-OPS-3**.

### Dependencies

- Explore handler ✅ (`internal/api/read_api.go`)
- Guards W7/W2 ✅ (`internal/app/auth.go` — hors hook observabilité)

### Acceptance criteria

- Hook uniquement quand `len(ranked)==0` et `len(rejected)>0`, avant `respondJSON(200)`.
- Un log structuré `cpm.explore.no_deployable_candidate` par événement.
- Un increment `cpm_explore_no_deployable_candidate_total` par événement (code dominant, pas multi-increment).
- `GET /metrics` expose le compteur.
- Cas `incompatible.chain_scope` couvert par test ; helper observabilité testé.
- Aucun label haute cardinalité ; adresse wallet jamais en clair (log ou métrique).
- Réponse API explore inchangée.

### Limites connues / reste IMM-OPS-2

- Pas de job scrape Prometheus dans `cafe-deploy` — métrique exposée mais pas encore consommée par la stack ops.
- Pas de métrique totale `explore` pour calculer un ratio no-candidate / total — alerte ratio conditionnelle reportée à **IMM-OPS-2** si métrique ajoutée plus tard.
- Pas de dashboard Grafana ni alerte hausse soutenue.

- **Proposed commit title:** `feat(cpm): add observability for no deployable explore candidate`

---

## IMM-OPS-2 — Deploy: Grafana dashboard and alert for CPM explore no-candidate events

- **Status:** ⚪ planned
- **Branch:** `deploy/cpm-explore-rejection-dashboard`
- **Repository:** `cafe-deploy`
- **Objective:** Ajouter une visibilité ops/admin dans la stack observability quand CPM ne peut pas sélectionner de Crypto Policy deployable pendant explore.
- **Scope:**
  - Dashboard ou panel Grafana montrant :
    - volume de no-deployable-candidate par `rejection_code` ;
    - focus sur `incompatible.chain_scope` ;
    - évolution temporelle ;
    - taux de no-candidate explores si une métrique totale d’explore existe.
  - Alerte basée sur une **hausse soutenue**, pas sur un événement unitaire.
  - L’alerte doit détecter :
    - hausse soutenue de `incompatible.chain_scope` ;
    - ou ratio anormal de no-candidate explores par rapport au total des explores, si disponible.
- **Out of scope:**
  - notification par wallet ;
  - stockage événementiel détaillé ;
  - UI admin produit ;
  - modification backend CPM ;
  - labels Prometheus haute cardinalité.
- **Prérequis IMM-OPS-1 (livré par la PR CPM) :** compteur `cpm_explore_no_deployable_candidate_total` + endpoint `GET /metrics` sur `cafe-cpm`. **Cette PR ajoute** la configuration scrape Prometheus / edge dans `cafe-deploy`.
- **Dependencies:** **IMM-OPS-1**
- **Acceptance criteria:**
  - Un panel Grafana permet de visualiser les rejets CPM explore par code.
  - Une alerte warning existe sur hausse soutenue de `incompatible.chain_scope`.
  - Le dashboard fonctionne uniquement avec les métriques exposées par **IMM-OPS-1**.
  - Aucun label haute cardinalité n’est introduit.
- **Proposed commit title:** `feat(deploy): add CPM explore rejection dashboard`

---

## Future / deferred — IMM-OPS-3

> **Hors chemin critique release.** Ne bloque ni **IMM-OPS-1** ni **IMM-OPS-2**.

## IMM-OPS-3 — Admin visibility for CPM policy coverage gaps

- **Status:** 🧊 deferred
- **Repository:** TBD — dépend d’une décision produit sur l’interface admin cible (pas de repo `cafe-admin` à ce jour).
- **Objective:** Cadrer une future visibilité produit/admin sur les gaps de couverture CP détectés pendant explore, en particulier `incompatible.chain_scope`.
- **Positionnement:**
  - **IMM-OPS-1** = instrumentation backend CPM ;
  - **IMM-OPS-2** = dashboard / alert ops ;
  - **IMM-OPS-3** = future vue produit/admin actionnable ;
  - **IMM-OPS-3** ne bloque ni **IMM-OPS-1** ni **IMM-OPS-2**.
- **Scope potentiel:**
  - Synthèse des cas récents où aucune CP deployable n’a été trouvée ;
  - Groupement par : rejection code, missing chains, wallet type, policy template / catalog family si disponible ;
  - Aide à identifier : CP absente pour certaines chaînes, CP trop restrictive, mismatch Discovery / CPM, besoin de création ou ajustement de catalog entry.
- **Privacy / data handling:**
  - Utiliser des données anonymisées ou hashées pour les wallets ;
  - Ne pas exposer de wallet address brute sauf décision explicite produit/sécurité ;
  - Ne pas persister de données sensibles sans design préalable.
- **Out of scope:**
  - implémentation immédiate ;
  - workflow automatique de création de CP ;
  - notification temps réel ;
  - Slack / email ;
  - stockage détaillé d’événements wallet sans décision privacy.
- **Dependencies:** **IMM-OPS-1**, **IMM-OPS-2**, décision produit sur l’interface admin cible.
- **Acceptance criteria (cadrage doc) :**
  - Le ticket distingue clairement observability technique, dashboard ops et visibilité admin produit.
  - Le ticket précise les contraintes privacy / cardinality.
  - Le ticket est non bloquant pour **IMM-OPS-1** et **IMM-OPS-2**.
  - Les cross-links **REQ9** / **IMM-OPS-1…3** sont cohérents.
- **Proposed commit title:** `docs(product): specify admin visibility for CPM policy coverage gaps`

