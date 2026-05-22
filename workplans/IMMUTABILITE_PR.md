# Scan immutability & CPM — PR plan (CPM)

**Source de vérité :** [`WORKPLAN_API.md`](./WORKPLAN_API.md) — **§2.2** (couplage **W1–W7**, query **`latest=true`**), **§4.4**, **§8.6**.

**Ce fichier** = découpage PR CPM et coordination Discovery.

**Plans liés :**

| Dépôt | Document |
|-------|----------|
| Discovery | [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md) |
| Frontend | [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md) |

**Déjà livré :** policies par `scan_id` (**PR7**), référence interne scan (**PR5**), explore Option A (**PR8**), **DELETE scan → 409** (**PR6**, **W3**).

---

## Résolution du « dernier scan » (W2, après W7 CPM)

**Décision API (workplan) :** pas de route **`/latest`**. Utiliser :

```http
GET /discovery/v1/wallets/scans?address=0x…&latest=true
```

- **`items`** : au plus **1** `ScanListItem` — dernier scan **`completed`** (pas le plus récent par date si celui-ci est `failed` / en cours).
- **`total`** : **`0`** s’il n’existe **aucun** `completed` ; **`1`** possible même avec un `failed` plus récent dans l’historique.
- Implémentation Discovery : **IMM-4**.

**CPM (IMM-10)** : **W7** — refuser explore/persist si la ligne **la plus récente** (`created_at`) n’est pas **`completed`** (y compris `failed`). Puis **W2** — `scan_id` = celui de **`?latest=true`**.

**`POST …/scan` (Discovery)** : hors scope CPM ; retry autorisé si dernier `failed` (**IMM-4** / **IMM-9**).

---

## Règles WORKPLAN (responsabilité CPM)

| ID | Règle | CPM |
|----|--------|-----|
| **W7** | Pas de **nouvelle CPM** tant que le **dernier scan** (`created_at` max) n’est pas **`completed`** (`failed` ou en cours → **400**) | explore/persist → **400** `LATEST_SCAN_NOT_COMPLETED` |
| **W1** | Pas de nouveau scan si CPM pour la cible | Lookup (**IMM-9b**) — consommé par Discovery **IMM-9** |
| **W2** | CPM sur le dernier **`completed`** uniquement | Validation **`?latest=true`** (**IMM-10**) |
| **W3** | DELETE scan après suppression CPM | **`GET ?scan_id=`** + **`DELETE ?id=`** ; Discovery **409** |
| **W4** | DELETE policy sans toucher aux scans | Inchangé |
| **W5–W6** | Historique & CBOM | Discovery |

> **Ordre CPM :** **W7** (dernier = `completed`) → **W2** (`scan_id`). **Discovery POST** : en cours (**409** `SCAN_IN_PROGRESS`) → **W1**.

---

## Table de suivi (CPM)

| PR plan | GitHub issue | Dépôt | Dépend de | Objectif |
|---------|--------------|-------|-----------|----------|
| **IMM-9b** | [§ IMM-9b](#github-issue--imm-9b) | `cafe-crypto-policy-mgt` | PR7 | **W1** : lookup par `target_address` |
| **IMM-10** | [§ IMM-10](#github-issue--imm-10) | `cafe-crypto-policy-mgt` | Discovery **IMM-4** | **W7** (CPM) + **W2** : explore/persist |

---

## Création des issues GitHub

Repo : **`create2-labs/cafe-crypto-policy-mgt`**.

---

## GitHub issue — IMM-9b

### Title (copy as-is)

```
[CPM][IMM-9b] Internal lookup: persisted policies exist for wallet target_address
```

### Labels (suggested)

`cpm` · `scan-history` · `internal-api`

### Body (copy below the line)

---

**Type:** Technical task (**WORKPLAN §2.2 W1**).

**Tracking ID:** IMM-9b  
**Workplan:** [WORKPLAN_API.md §2.2 W1](./WORKPLAN_API.md)

### Summary

Owner-scoped check: persisted policy exists for wallet scans of **target_address**. Used by Discovery **IMM-9** after in-flight guard (**IMM-4**) before `POST /discovery/v1/scan`.

### Acceptance criteria

- [ ] Internal API or shared module (service token).
- [ ] `exists: true|false` (optional `count` for logs).
- [ ] Tests with bound policy; false after `DELETE …/policies?id=`.

### Tracking

| Field | Value |
|-------|--------|
| Issue | — |
| PR | — |

---

## GitHub issue — IMM-10

### Title (copy as-is)

```
[CPM][IMM-10] Policy explore/persist only when latest scan is completed (W7 + W2)
```

### Labels (suggested)

`cpm` · `scan-history` · `contract-alignment` · `option-a`

### Body (copy below the line)

---

**Type:** Technical task (**WORKPLAN §2.2 W7**, **W2**).

**Tracking ID:** IMM-10  
**Workplan:** [WORKPLAN_API.md §2.2](./WORKPLAN_API.md)

### Summary

Reject explore/persist when:

1. **W7** — **newest** scan row (`created_at`) is not **`completed`** → **400** `LATEST_SCAN_NOT_COMPLETED` (includes **`failed`** and in-flight; applies even if an older **`completed`** exists).
2. **W2** — `scan_id` ≠ latest **`completed`** from `GET …/wallets/scans?address={addr}&latest=true`.

**Not** responsible for `POST …/scan` retry policy (Discovery **IMM-4**).

### Acceptance criteria

- [ ] **W7** before **W2** on every explore/persist.
- [ ] **400** when newest is **`failed`** or in-flight.
- [ ] **`completed` A + `failed` B** (B newer): explore on A → **400** (W7).
- [ ] **400** `SCAN_ID_NOT_LATEST_FOR_TARGET` when historical `scan_id`.
- [ ] Newest resolved by **`created_at` desc** (not `latest=true` alone for W7).
- [ ] `openapi/cpm-v1.yaml` documents **W7** and **W2**.
- [ ] Contract tests updated.

### Dependencies

Discovery **IMM-4** (`latest=true` for **W2**).

### Tracking

| Field | Value |
|-------|--------|
| Issue | — |
| PR | — |

---

## Séquence CPM

```text
IMM-9b → Discovery IMM-9 (in-flight + W1)
Discovery IMM-4 (latest=true + SCAN_IN_PROGRESS) → IMM-10 (W7 CPM + W2)
```

---

## Critères d’acceptation (CPM)

- [ ] **W7** : explore/persist **400** si dernier scan ≠ **`completed`** (y compris **`failed`**).
- [ ] **W1** lookup pour Discovery IMM-9.
- [ ] **W2** : **`latest=true`** + **400** si `scan_id` non latest **`completed`**.
- [ ] **W3** / **W4** inchangés.
