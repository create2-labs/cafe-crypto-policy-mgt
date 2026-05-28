# Scan immutability & CPM — PR plan (CPM)

> **English product summary:** [CAFE functional specifications](https://github.com/create2-labs/cafe-documentation/blob/main/functional-specifications.md) and [technical specifications](https://github.com/create2-labs/cafe-documentation/blob/main/technical-specifications.md).

**Source de vérité :** [`WORKPLAN_API.md`](./WORKPLAN_API.md) — **§0**, **§2.2** (couplage **W1–W8**, query **`latest=true`**), **§4.4**, **§8.6**.

**All API paths in this document refer to the canonical public prefixes defined in WORKPLAN_API.md: `/api/discovery/v1` and `/api/cpm/v1`.**

**Ce fichier** = découpage PR CPM et coordination Discovery.

**Plans liés :**

| Dépôt | Document |
|-------|----------|
| Discovery | [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md) |
| Frontend | [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md) |

**Déjà livré :** policies par `scan_id` (**PR7**), référence interne scan (**PR5**), explore Option A (**PR8**), **DELETE scan → 409** (**PR6**, **W3**).

**Périmètre CPM produit actuel :** assessment/remediation **wallet-only** (scans **wallet / EVM**). Les scans **TLS** restent dans **Discovery** (historique, CBOM, observation) — **pas** de cible CPM assessment/remediation pour cette release. Voir **WORKPLAN §5.4.6**.

---

## Principe W1 — un contexte CPM actif par adresse

**Décision produit :** tant qu’une **policy persistée** **ou** un **draft** existe pour une **`target_address`** (wallet), **aucun** nouveau **`POST /api/discovery/v1/scan`**. L’utilisateur doit **finaliser** (`POST /api/cpm/v1/policies`) ou **supprimer** le brouillon (`DELETE /api/cpm/v1/drafts?id=…`) et/ou la policy (`DELETE /api/cpm/v1/policies?id=…`) avant tout rescan — y compris après un scan **`failed`**.

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

> **Discovery POST** : **W8** (`SCAN_IN_PROGRESS` si en cours) ; puis **W1** (policy **ou** draft). **W7** ne s’applique **pas** à POST.

---

## Table de suivi (CPM)

| PR plan | GitHub  | Dépôt | Dépend de | Objectif |
|---------|--------------|-------|-----------|----------|
| **IMM-9b** | [#38](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/38) | `cafe-crypto-policy-mgt` | PR7 | **W1** : lookup policy **+** draft par `target_address` normalisée |
| **IMM-10** | [#40](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/40) | `cafe-crypto-policy-mgt` | Discovery **IMM-4** | **W7** (newest row) + **W2** (`latest=true`) |

---

##  IMM-9b


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

- Policies and drafts from wallet scans must be resolvable by:
  - **`scan_id`**
  - **`scan_family = wallet`**
  - **normalized `target_address`**
  - optionally **`wallet_type`** (for UX reload — not required for **409** path)

### Public draft deletion (W1 unblock)

Platform draft removal for **W1** uses the canonical public API:

- **`DELETE /api/cpm/v1/drafts?id=…`** → **`204`** \| **`404`** (**WORKPLAN §2.4**, **§4.4**).

**Client workflow (not IMM-9b):** user may **export draft locally**, **`DELETE`** platform draft, run new scan, then **reload local save** only if latest **`completed`** scan matches same **`target_address`** and **`wallet_type`** — see [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md).

### Acceptance criteria

- [ ] Internal API or shared module (service token); callable **before** new scan creation.
- [ ] Lookup by **normalized `target_address`** only — no Discovery round-trip for address resolution.
- [ ] Response: `{ exists, policy_count, draft_count }`.
- [ ] **409** path when `exists: true` (policy and/or draft).
- [ ] Wallet-only scope: TLS scans **out of** W1 lookup for current product flow.
- [ ] Tests: policy → true; draft → true; false when both removed.


---

##  IMM-10


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

2. If `newest.status != completed` → reject with **`400`** **`LATEST_SCAN_NOT_COMPLETED`** (includes `failed`, `requested`, `started` — even when an older **`completed`** exists).

   Example: **`completed` A** + **`failed` B** (B newer) → **400** for explore/persist on **any** `scan_id`, including A.

### Step 2 — W2 (`latest=true`, not `limit=1`)

3. Fetch the latest **`completed`** scan:

   ```http
   GET /api/discovery/v1/wallets/scans?address=0x…&latest=true
   ```

4. If requested `scan_id` ≠ `scan_id` returned by **`latest=true`** → **`400`** **`SCAN_ID_NOT_LATEST_FOR_TARGET`**.

### Acceptance criteria

- [ ] **W7** runs **before** **W2** on every explore/persist.
- [ ] **W7** uses newest row (`limit=1` + default sort) — **never** `latest=true`.
- [ ] **W2** uses **`latest=true`** — **never** `limit=1` alone.
- [ ] **400** `LATEST_SCAN_NOT_COMPLETED` when newest is `failed` or in-flight.
- [ ] **400** `SCAN_ID_NOT_LATEST_FOR_TARGET` for historical `scan_id`.
- [ ] Wallet-only: no TLS assessment/remediation path in this PR.
- [ ] Contract tests: in-flight, failed, `completed`+`failed` newer, historical `scan_id`.

### Dependencies

Discovery **IMM-4** (`latest=true`, default sort, lifecycle enum).

### Tracking

| Field | Value |
|-------|--------|
| Issue | — |
| PR | — |

---

## Critères d’acceptation (CPM)

- [ ] **W1** : lookup policy **+** draft par **`target_address`** normalisée pour **IMM-9**.
- [ ] **`DELETE /api/cpm/v1/drafts?id=…`** contractualisé (**WORKPLAN §0.2**).
- [ ] **W7** : newest row testé — **pas** `latest=true`.
- [ ] **W2** : `latest=true` testé — **pas** `limit=1` seul.
- [ ] **W3** / **W4** inchangés.
- [ ] TLS : hors scope assessment/remediation CPM produit actuel.

---

## Ordre recommandé (extrait chaîne IMM)

Voir [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md) pour la séquence complète. Côté CPM :

0. Patch **WORKPLAN_API.md** si nécessaire (draft **DELETE**, TLS wallet-only, chemins publics).
4. **IMM-9b** — lookup `target_address` + draft/policy.
6. **IMM-10** — W7 puis W2.

Frontend **FE-IMM-*** : après stabilisation backend (**IMM-4**, **IMM-9**, **IMM-10**).
