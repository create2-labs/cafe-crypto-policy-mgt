# Scan immutability & CPM — PR plan (CPM)

> **English product summary:** [CAFE functional specifications](https://github.com/create2-labs/cafe-documentation/blob/main/functional-specifications.md) and [technical specifications](https://github.com/create2-labs/cafe-documentation/blob/main/technical-specifications.md).

**Source de vérité :** [`WORKPLAN_API.md`](./WORKPLAN_API.md) — **§0**, **§2.2** (couplage **W1–W8**, query **`latest=true`**), **§2.2.1 P1**, **§4.4**, **§8.6**.

**All API paths in this document refer to the canonical public prefixes defined in WORKPLAN_API.md: `/api/discovery/v1` and `/api/cpm/v1`.**

**Ce fichier** = découpage PR CPM et coordination Discovery.

**Plans liés :**

| Dépôt | Document |
|-------|----------|
| Discovery | [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md) |
| Frontend | [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md) |

**Déjà livré :** policies par `scan_id` (**PR7**), référence interne scan (**PR5**), explore Option A (**PR8**), **DELETE scan → 409** (**PR6**, **W3**), **IMM-9b** lookup W1 ([#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38)), **IMM-10** W7/W2 ([#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40)), **IMM-W1-1…3** **`DELETE /api/cpm/v1/drafts?id=…`** ([#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41)–[#44](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/44)), **normalisation adresse wallet** ([#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45) NATS, [#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46) assessment HTTP) — smokes `test-cpm-imm-w1-delete-draft.sh`, `test-discovery-imm9-wallet-scan-w1-cpm-block.sh` step 5.

**Suite CPM (backlog) :** voir [`TODO.md`](../TODO.md) — refactor handlers internes, chemins Discovery upstream, doc lookup payload-only.

**Périmètre CPM produit actuel :** assessment/remediation **wallet-only** (scans **wallet / EVM**). Les scans **TLS** restent dans **Discovery** (historique, CBOM, observation) — **pas** de cible CPM assessment/remediation pour cette release. Voir **WORKPLAN §5.4.6**.

---

## Principe W1 — un contexte CPM actif par adresse

**Décision produit :** tant qu’une **policy persistée** **ou** un **draft** existe pour une **`target_address`** (wallet), **aucun** nouveau **`POST /api/discovery/v1/scan`**. L’utilisateur doit **finaliser** (`POST /api/cpm/v1/policies`) ou **supprimer** le brouillon (`DELETE /api/cpm/v1/drafts?id=…`) et/ou la policy (`DELETE /api/cpm/v1/policies?id=…`) avant tout rescan — y compris après un scan **`failed`**.

**Parcours W1 :** Discovery **409** si policy/draft (**IMM-9**) ; déblocage via **`DELETE /api/cpm/v1/drafts?id=…`** (**IMM-W1-2**, livré). Normalisation **`target_address`** / sujets **`wallet:0x…`** : [#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45)–[#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46). Voir [`TODO.md`](../TODO.md) pour le backlog technique restant.

---

## W7 vs W2 — ne pas confondre les résolutions Discovery

| Règle | Résolution Discovery | Exemple |
|-------|----------------------|---------|
| **W2** (CPM) | **Dernier scan `completed`** uniquement | `GET /api/discovery/v1/wallets/scans?address=0x…&latest=true` |
| **W7** (CPM) | **Newest row** par `created_at` desc, `scan_id` desc — **pas** `latest=true` | `GET /api/discovery/v1/wallets/scans?address=0x…&limit=1` (tri par défaut) |

- **Ne jamais** utiliser `latest=true` pour **W7**.
- **Ne jamais** utiliser `limit=1` seul comme substitut de **W2** (la ligne la plus récente peut être `requested`, `started` ou `failed`).

---

## Règles WORKPLAN (responsabilité CPM)

| ID | Règle | CPM |
|----|--------|-----|
| **W1** | **Un contexte CPM actif max** : pas de scan si **policy** ou **draft** sur la cible (après **W8**) | Lookup policies **+** drafts (**IMM-9b**) → Discovery **IMM-9** |
| **W2** | CPM sur le dernier **`completed`** uniquement | **`GET …/wallets/scans?address=&latest=true`** (**IMM-10**) |
| **W3–W6** | DELETE / historique / CBOM | Voir WORKPLAN |
| **W7** | Pas d’explore/persist tant que le **newest row** n’est pas **`completed`** | explore/persist → **400** `LATEST_SCAN_NOT_COMPLETED` |
| **W8** | **Rescan** : `POST …/scan` refusé seulement si scan **en cours** ; **OK** si newest **`failed`** (sous **W1**) — **indépendant de W7** | Implémenté côté **Discovery** (**IMM-4**, **IMM-9**) |
| **P1** | Quota plan (success-only, monotonic) | **Hors CPM** — Discovery **IMM-6b** ; **`used`** = completed success ; DELETE policy/draft/scan **ne modifie pas** **`used`** ; voir WORKPLAN **§2.2.1** |

> **Discovery POST** : **W8** (`SCAN_IN_PROGRESS` si en cours) ; puis **W1** (policy **ou** draft). **W7** ne s’applique **pas** à POST.

---

## Table de suivi (CPM)

| PR plan | GitHub | Dépôt | Dépend de | Objectif |
|---------|--------|-------|-----------|----------|
| **IMM-9b** | [#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38) | `cafe-crypto-policy-mgt` | PR7 | **W1** : lookup policy **+** draft par `target_address` normalisée |
| **IMM-10** | [#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40) | `cafe-crypto-policy-mgt` | Discovery **IMM-4** | **W7** (newest row) + **W2** (`latest=true`) |
| **IMM-W1-1** | [#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41)  | `cafe-crypto-policy-mgt` | IMM-9b | **`DeleteDraft`** sur `OwnerScopedStore` |
| **IMM-W1-2** | [#42](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/42) | `cafe-crypto-policy-mgt` | **IMM-W1-1** | Handler **`DELETE /api/cpm/v1/drafts?id=…`** + OpenAPI |
| **IMM-W1-2** | [#43](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/43) | `cafe-crypto-policy-mgt` | **IMM-W1-1** | Handler **`DELETE /api/cpm/v1/drafts?id=…`** + OpenAPI |
| **IMM-W1-3** | [#44](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/44) | `cafe-crypto-policy-mgt` | **IMM-W1-2** | Tests routes + smoke deploy step 5 (rescan after delete draft) |
| **IMM-9b-norm-1** | [#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45) | `cafe-crypto-policy-mgt` | IMM-9b | `NormalizeWalletSubjectID` — consumer NATS assessment |
| **IMM-9b-norm-2** | [#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46) | `cafe-crypto-policy-mgt` | **#45** | `WalletSubjectIDFromAddress` — `POST …/policies/assessment/request` |

---

## IMM-W1-1 — `DeleteDraft` (store)

- **Status:** mergee dans [#41](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/41) 
- **Branch:** `cpm/imm-w1-delete-draft-store`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Suppression owner-scoped d’un brouillon plateforme — miroir **`DeletePolicy`**.
- **Scope:**
  - `OwnerScopedStore.DeleteDraft(principal, id)` → **`ErrDraftNotFound`**, **`ErrForbidden`**
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
- **Objective:** **`DELETE /api/cpm/v1/drafts?id=…`** → **204** \| **404** (**WORKPLAN §2.4**, **§4.4**).
- **Scope:**
  - `registerOwnerScopedRoutesForPrefix` — handler **`DELETE`** sur **`/drafts`**
  - Même auth / owner scope que **GET** / **POST** drafts
  - **`openapi/cpm-v1.yaml`**
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
  - Compléter **`cafe-deploy/scripts/test-discovery-imm9-wallet-scan-w1-cpm-block.sh`** step 5 (rescan **200** après DELETE draft)
- **Out of scope:** Frontend FE-IMM-1.
- **Dependencies:** **IMM-W1-2**, Discovery IMM-9 (mergé)
- **Proposed commit title:** `test: W1 unblock via DELETE platform draft`
- **Completion criteria:** Smoke step 5 vert ; **409** → DELETE draft → **POST scan** accepté.

---

## IMM-9b

**Type:** Technical task (**WORKPLAN §2.2 W1**).

**Status** : mergée dans [#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38)

### Summary

Owner-scoped internal lookup **before** Discovery allocates a new `scan_id` on **`POST /api/discovery/v1/scan`**. Discovery knows the **normalized target address** from the request body — not a new `scan_id`. **Do not** require Discovery to resolve address via `policy.scan_id → Discovery detail → address`.

**Product rule:** at most **one active CPM context per normalized wallet address** — user must finalize policy or delete platform draft (and policy if needed) before rescan.

### Contract (internal, service token)

- **Input:** normalized **`target_address`** (same normalization as Discovery wallet scans), owner scope from service auth.
- **Output (minimal):**

```json
{ "exists": true, "policy_count": 1, "draft_count": 0 }
```

or `{ "exists": false, "policy_count": 0, "draft_count": 0 }`.

### Public draft deletion (W1 unblock)

Platform draft removal for **W1** uses the canonical public API:

- **`DELETE /api/cpm/v1/drafts?id=…`** → **`204`** \| **404** (**WORKPLAN §2.4**, **§4.4**) — **IMM-W1-2**.

**Client workflow (not IMM-9b):** user may **export draft locally**, **`DELETE`** platform draft, run new scan, then **reload local save** only if latest **`completed`** scan matches same **`target_address`** and **`wallet_type`** — see [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md).

### Acceptance criteria

- [x] Internal API or shared module (service token); callable **before** new scan creation.
- [x] Lookup by **normalized `target_address`** only — no Discovery round-trip for address resolution.
- [x] Response: `{ exists, policy_count, draft_count }`.
- [x] **409** path when `exists: true` (policy and/or draft).
- [x] Wallet-only scope: TLS scans **out of** W1 lookup for current product flow.
- [x] Tests: policy → true; draft → true; false when both removed.

### Normalisation adresse (follow-up, livré)

- **Status :** [#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45) (NATS), [#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46) (assessment HTTP).
- **Règle canonique :** `NormalizeWalletTargetAddress` dans `internal/persistence/wallet_target.go` (même normalisation que Discovery).
- **Helpers :** `NormalizeWalletSubjectID` (NATS) ; `WalletSubjectIDFromAddress` (HTTP assessment depuis `target_address` Discovery).
- [x] Plus de `normalizeHexAddress` / `normalizeWalletSubjectForAssessment` locaux.

---

## IMM-10

**Type:** Technical task (**WORKPLAN §2.2 W7**, **W2**).

**Status** : mergée dans [#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40)

### Summary

On **`POST /api/cpm/v1/policies/decisions/explore`** and **`POST /api/cpm/v1/policies`** for **wallet** targets (**Option A**, `binding=discovery`), enforce guards in order:

### Step 1 — W7 (newest row, not `latest=true`)

1. Fetch the **newest** wallet scan row for the target address (default list sort: `created_at` desc, `scan_id` desc).

   Example:

   ```http
   GET /api/discovery/v1/wallets/scans?address=0x…&limit=1
   ```

2. If `newest.status != completed` → reject with **`400`** **`LATEST_SCAN_NOT_COMPLETED`**.

### Step 2 — W2 (`latest=true`, not `limit=1`)

3. Fetch the latest **`completed`** scan:

   ```http
   GET /api/discovery/v1/wallets/scans?address=0x…&latest=true
   ```

4. If requested `scan_id` ≠ `scan_id` returned by **`latest=true`** → **`400`** **`SCAN_ID_NOT_LATEST_FOR_TARGET`**.

### Acceptance criteria

- [x] **W7** runs **before** **W2** on every explore/persist.
- [x] **W7** uses newest row (`limit=1` + default sort) — **never** `latest=true`.
- [x] **W2** uses **`latest=true`** — **never** `limit=1` alone.
- [x] Wallet-only: no TLS assessment/remediation path in this PR.

### Dependencies

Discovery **IMM-4** (`latest=true`, default sort, lifecycle enum).

---

## Critères d’acceptation (CPM)

- [x] **W1** : lookup policy **+** draft par **`target_address`** normalisée pour **IMM-9**.
- [x] **`DELETE /api/cpm/v1/drafts?id=…`** livré (**IMM-W1-2**, [#42](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/42)) — **WORKPLAN §0.2**.
- [x] **W7** : newest row testé — **pas** `latest=true`.
- [x] **W2** : `latest=true` testé — **pas** `limit=1` seul.
- [x] **W3** / **W4** inchangés (**DELETE policy** déjà livré).
- [x] TLS : hors scope assessment/remediation CPM produit actuel.

---

## Ordre recommandé (extrait chaîne IMM)

Voir [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md) pour la séquence complète. Côté CPM :

1. **IMM-W1-1** → **IMM-W1-3**, **IMM-9b**, **IMM-10**, **IMM-9b-norm-1…2** ([#45](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/45)–[#46](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/46)) — mergés.
2. Backlog CPM : [`TODO.md`](../TODO.md) (refactor handlers internes, chemins Discovery upstream).

**Parallèle Discovery :** **IMM-6b-1…8** (quota **P1/G1–G4** — success-only, garde POST, commit atomique, CBOM) — voir [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md).

Frontend **FE-IMM-*** : [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md) — après **IMM-W1-3** et **IMM-6b-5** minimum.
