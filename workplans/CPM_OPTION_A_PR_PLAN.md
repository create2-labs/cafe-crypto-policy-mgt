# CPM Post-V1 Option A — Multi-Repo PR Plan

Cross-repository work plan for connecting the **Crypto Policy Management (CPM)** experience to **real authenticated Discovery wallet scans** using the **canonical Discovery v1 scan APIs** and **CPM v1 public routes**, **without** a giant single PR or a frontend-first spike.

**Référence de vérité (contrats, jalons mergés, chemins HTTP) :** [`WORKPLAN_API_PR.md`](./WORKPLAN_API_PR.md) et, en amont, [`WORKPLAN_API.md`](./WORKPLAN_API.md).

**Référence de vérité (frontend CPM livré jusque périmètre V1) :** [`CPM_FRONTEND_PR_PLAN_V1.md`](./CPM_FRONTEND_PR_PLAN_V1.md) — séquence PR 1–15 **closée** (dernier merge : `cafe-frontend` [#53](https://github.com/create2-labs/cafe-frontend/pull/53), 2026-05-15). Le travail **F\*** ci-dessous est **post-V1 frontend** : il s’ajoute après `CpmDataSource` API-backed (**PR 12**), tout en gardant les invariants (pas d’fixtures dans les vues, etc.).

**Repositories:** `cafe-discovery`, `cafe-crypto-policy-mgt` (CPM), `cafe-frontend`.

**Related design notes:** `cafe-crypto-policy-mgt/cpm_post_v_1_option_a_scan_context.md`, `cafe-crypto-policy-mgt/CPM_AuthN_AuthZ_workplan.md`.

---

## Objective

Deliver **Option A** (réconciliée avec la cohérence API **v1**): les scans wallet Discovery (persistés côté Discovery aujourd’hui) sont exposés au client via **`GET /discovery/v1/wallets/scans`** et le **détail** **`GET /discovery/v1/wallets/scans/{scan_id}`** ; l’utilisateur **choisit un `scan_id`** (et le front construit ou transmet les champs nécessaires) ; CPM **`POST /api/cpm/v1/policies/decisions/explore`** s’exécute avec l’enveloppe **`scan_id` + `policy_context` + `selection_request`** où **`policy_context`** est **compatible avec le DTO détail v1** (sync, preview — voir **PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29) dans `WORKPLAN_API_PR.md`) ; la **persistance** lie les policies au **`scan_id`** via les règles **`binding`** CPM (**PR7** [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28)). L’UI peut aussi lister les policies liées à un scan via **`GET /api/cpm/v1/policies?scan_id=`** (même lookup owner-scoped que l’endpoint interne de référence scan).

> **Distinct :** **`POST /api/cpm/v1/policies/assessment/request`** (**PR13g** [#33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33)) est **wallet-scan only**, **sans** `policy_context` client — CPM recharge l’observation côté serveur depuis le détail Discovery. Ne pas confondre avec **explore** (preview) ci-dessus.

Hard rules from product/architecture (alignées `WORKPLAN_API_PR.md`):

- **Backend / contrats d’abord** — la surface canonique Discovery est **`/discovery/v1`** (edge **`/api/discovery/v1/...`** après strip **`/api`**).
- **Ne pas réintroduire** l’ancienne façade **`GET /discovery/wallet-policy-contexts`** : **retirée** avec **PR11a** [#54](https://github.com/create2-labs/cafe-discovery/pull/54).
- **Do not** claim Discovery is DB-free today; Persistence Service remains the **target** authoritative scan-data owner—Discovery backend still has DB access in the interim.
- **Respect AUTH-██** already merged (principal, scan binding, fail-closed behavior where configured).
- **Frontend:** no direct DB; no unauthenticated Persistence Service calls; **no `mock-discovery-scan-placeholder` in API mode** once Option A wiring (**F4**) lands — aujourd’hui le placeholder est **explicitement** dans le périmètre V1 mock (**`CPM_FRONTEND_PR_PLAN_V1.md`**, PR 5 / PR 12).

---

## Architecture decision

**Option A (alignée API v1 — `WORKPLAN_API_PR.md`) :**

1. Discovery expose **listes et détail** wallet sous **`/discovery/v1/wallets/scans`** et **`/discovery/v1/wallets/scans/{scan_id}`** (edge : **`/api/discovery/v1/...`**). Contrat machine-readable : **`cafe-discovery/openapi/discovery-v1.yaml`** (**PR1a** [#49](https://github.com/create2-labs/cafe-discovery/pull/49)), implémentation listes/détail **PR4** [#52](https://github.com/create2-labs/cafe-discovery/pull/52).
2. Les réponses sont les **DTOs v1** (liste + détail), pas des dumps bruts ; le détail sert de **source de vérité** pour aligner **`policy_context`** côté **explore** (**PR8**).
3. CPM **`POST /api/cpm/v1/policies/decisions/explore`** accepte **`scan_id` + `policy_context` + `selection_request`** (non persistant — **PR8**).
4. CPM **`POST /api/cpm/v1/policies`** persiste avec **`scan_id`** selon **`binding`** (défaut **`discovery`** : UUID requis — voir code/tests `owner_routes.go`) — **PR7** / règles documentées OpenAPI **PR1b** [#26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26).
5. Le frontend charge la liste + le détail via **HTTP authentifié** Discovery v1, sélectionne un scan, et alimente **`CpmDataSource`** / explore en **API mode** (**`CPM_FRONTEND_PR_PLAN_V1.md`** PR 12 + extensions F\*).

**Référence policies par scan (UI / détacher avant DELETE) :** **`GET /api/cpm/v1/policies?scan_id=`** — **PR7**. **Discovery** consomme le verdict interne CPM pour **409** sur DELETE (**PR5** [#27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27) / **PR6** [#53](https://github.com/create2-labs/cafe-discovery/pull/53)) — inchangé vs workplan API.

AUTH-05 / internal Discovery authz endpoints restent le chemin pour les **contrôles binaires** scan lorsque CPM délègue **can-read** ; **la liste et le détail** restent **Discovery-owned** (v1).

---

## Current state (aligné `WORKPLAN_API_PR.md` + `CPM_FRONTEND_PR_PLAN_V1.md` — 2026-05)

Synthèse à jour des **jalons mergés** documentés dans le workplan API et le plan frontend V1 (le détail fichier par fichier peut diverger légèrement sur un clone local non à jour de `main`).

### cafe-discovery

- **Contrat canonique :** **`openapi/discovery-v1.yaml`** (**PR1a** [#49](https://github.com/create2-labs/cafe-discovery/pull/49)).
- **Liste + détail scans wallet :** **`GET /discovery/v1/wallets/scans`**, **`GET /discovery/v1/wallets/scans/{scan_id}`** — **PR4** [#52](https://github.com/create2-labs/cafe-discovery/pull/52).
- **Doc Option A mainteneur (v1 ↔ CPM) :** référence **`docs/CPM_OPTION_A_DISCOVERY_V1_CONTRACT.md`** — **A2** [#60](https://github.com/create2-labs/cafe-discovery/pull/60) (complète **PR12b** [#59](https://github.com/create2-labs/cafe-discovery/pull/59) README).
- **Historique :** une façade **`GET /discovery/wallet-policy-contexts`** avait existé (ex. PR [#48](https://github.com/create2-labs/cafe-discovery/pull/48)) ; elle a été **retirée** avec le nettoyage routes legacy — **PR11a** [#54](https://github.com/create2-labs/cafe-discovery/pull/54). Toute nouvelle doc ou client Option A doit utiliser **v1**.
- **Services internes :** le code peut encore réutiliser ou factoriser une logique du type *wallet policy context* **en interne** tant qu’elle n’expose pas une route publique parallèle au contrat v1 — cf. note **PR4** dans `WORKPLAN_API_PR.md` (*aligner le service `wallet_policy_context` si encore utilisé*).

### cafe-crypto-policy-mgt

- **Explore (sync, `policy_context`) :** **`POST /api/cpm/v1/policies/decisions/explore`** — **PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29) ; alignement DTO détail Discovery v1 ; tests `internal/api/read_api_test.go` (ex. `TestDecisionExplore_optionA_policy_context` selon l’arbre actuel).
- **Policies owner + `scan_id` :** **`GET /api/cpm/v1/policies?scan_id=`**, persist **`POST /api/cpm/v1/policies`** avec règles **`binding`** — **PR7** [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28) ; pour **`binding=discovery`** (défaut), un **`scan_id`** UUID valide est **requis** (`owner_routes.go` / tests).
- **Assessment async :** **`POST /api/cpm/v1/policies/assessment/request`** — **PR13g** [#33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33) — **sans** `policy_context` client.
- **OpenAPI :** **`openapi/cpm-v1.yaml`** — **PR1b** [#26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26).
- **Scripts :** **`scripts/test-discovery-v1-wallet-scans-to-cpm.sh`** — smoke Option A v1 uniquement (sign-in → list/detail → explore → persist optionnel) ; **#34** mergé. Suppression des scripts legacy CBOM / `wallet-policy-contexts`. CI : **`bash -n`** sur le smoke script (`.github/workflows/ci.yml`).

### cafe-frontend

- **Plan CPM frontend V1 :** **clos** — **`CPM_FRONTEND_PR_PLAN_V1.md`**, dernier merge **PR 15** → [#53](https://github.com/create2-labs/cafe-frontend/pull/53) (2026-05-15) : **`CpmDataSource`**, Vue Flow verrouillé, validation / EOA / persist, **`createApiCpmDataSource`** (**PR 12** [#45](https://github.com/create2-labs/cafe-frontend/pull/45)), gates erreurs (**PR 13c**), etc.
- **`CPM_SELECTION_CONTEXT_SCAN_ID`** / valeur **`mock-discovery-scan-placeholder`** : **explicitement dans le périmètre livré V1** (sélection de template + explore en API mode avec contexte synthétique jusqu’à branchement scan réel). Le retrait conditionné **API mode** relève des PR **F4+**.
- **Parcours scan nominal hors CPM :** migration **Discovery v1** + hydratation cartes (**PR10** [#52](https://github.com/create2-labs/cafe-frontend/pull/52), **PR13b** sur `main`, **PR13e** [#56](https://github.com/create2-labs/cafe-frontend/pull/56)) — réutilisable pour la liste/détail wallet côté Option A.
- **E2E navigateur :** toujours un sujet pour **F5** (Vitest comme base V1 ; Playwright ou équivalent si la gouvernance repo l’accepte).

### Edge routing (`cafe-deploy`, `WORKPLAN_API_PR.md` PR9 / PR11b)

- Chemins publics **Discovery** : **`/api/discovery/v1/...`** ; **CPM** : **`/api/cpm/v1/...`**. **`/api/internal/...`** doit rester **bloqué** à l’edge (**403**).

---

## Target flow

```text
Discovery wallet scan (flux existants ; DB Discovery aujourd’hui ; PS cible long terme)
       ↓
GET /discovery/v1/wallets/scans (JWT, pagination items/total/limit/offset)
       ↓
GET /discovery/v1/wallets/scans/{scan_id} (JWT, détail — DTO source pour policy_context explore)
       ↓
Frontend: data source liste + détail v1 + useCpmScanContext (F1–F3) → CpmDataSource (PR 12)
       ↓
POST /api/cpm/v1/policies/decisions/explore
  { scan_id, policy_context, selection_request }
       ↓
CPM evaluate → graph / ranked candidates → validation / EOA / persist
       ↓
POST /api/cpm/v1/policies (binding discovery → scan_id UUID requis)
       ↓
(optionnel UI) GET /api/cpm/v1/policies?scan_id=…  — liste policies liées au scan (PR7)
```

**Parcours async distinct (hors enveloppe explore ci-dessus) :**

```text
POST /api/cpm/v1/policies/assessment/request
  { scan_id, selection_request }   // pas de policy_context client — PR13g
```

---

## Known contract mismatches (document early; align OpenAPI / types / UI copy)

Les champs exacts suivent **`openapi/discovery-v1.yaml`** (détail wallet) et **`openapi/cpm-v1.yaml`**. Les écarts de vocabulaire typiques sont figés dans **`cafe-discovery`** **A2** ([#60](https://github.com/create2-labs/cafe-discovery/pull/60), `docs/CPM_OPTION_A_DISCOVERY_V1_CONTRACT.md`) ; le mapping frontend (**F1**) doit suivre cette référence :

| Area | UX / doc souhaitable | Comportement / spec à vérifier |
|------|----------------------|--------------------------------|
| Posture PQ / champs `result` | Libellés produit unifiés | Aligner sur le DTO v1 détail wallet + règles **`result`** **PR4** / **PR13a** [#56](https://github.com/create2-labs/cafe-discovery/pull/56). |
| `wallet_type` / kind | `EOA` \| *smart account* \| … | Domaine Discovery / CPM : **`EOA`**, **`AA`**, **`Contract`**, etc. — table consommateur : **A2** [#60](https://github.com/create2-labs/cafe-discovery/pull/60) / types TS **F1**. |

Ces points **doivent** être traités ou reportés **explicitement** avant de figer les types **F1** (data source v1).

---

## PR sequence, branches, roll-out order, et suivi

**Ordre de merge suggéré** : **A → B → C → F → D** (l’ordre des lignes du tableau). Phase **C** (scripts + tests de contrat) avant **F** (frontend **post-V1**).

**Note — évolution vs ancienne rédaction :** **A1**, **A2**, **B1**, **B2** et **C1** sont **mergés** sur `main` selon **`WORKPLAN_API_PR.md`** et **#34** (scripts v1). Ce tableau marque cet état pour éviter de replanifier du travail déjà livré ; la suite (**C2** gels contrat, **F\***, **D1**) continue sous forme de petites PR.

**Référence frontend :** **`CPM_FRONTEND_PR_PLAN_V1.md`** — V1 terminé ; **F1–F5** = **suite** après **PR 12** (API data source).

Remplir **Statut**, **N° PR**, **Lien**, **Assigné**, **Notes** au fil des travaux. Valeurs suggérées pour **Statut** : `à faire` · `en cours` · `PR ouverte` · `en revue` · `mergé` · `supersedé` · `bloqué`.

| # | PR | Dépôt (repo) | Branche | Objectif (résumé) | Statut | N° PR | Lien | Assigné | Notes |
|---:|----|--------------|---------|-------------------|--------|-------|------|---------|-------|
| 1 | A1 | cafe-discovery | — | **Surface canonique liste/détail wallet v1** (remplace l’ancienne façade contexts) | mergé | 49,52,54 | `WORKPLAN_API_PR` **PR1a, PR4, PR11a** | | **#48** contexts **retiré** par **#54** ; pas de réouverture de `wallet-policy-contexts`. |
| 2 | A2 | cafe-discovery (+ liens CPM) | `option-a/a2-discovery-cpm-v1-contract-docs` | Doc **mainteneur** : v1 scans wallet vs CPM explore/assessment/policies ; URLs edge | mergé | 60 | [#60](https://github.com/create2-labs/cafe-discovery/pull/60) | | Doc : `docs/CPM_OPTION_A_DISCOVERY_V1_CONTRACT.md` ; **PR12b** [#59](https://github.com/create2-labs/cafe-discovery/pull/59) jalons README antérieurs. |
| 3 | B1 | cafe-crypto-policy-mgt | — | **Explore** + `policy_context` aligné détail v1 | mergé | 29 | [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29) | | Suite : durcissements optionnels + doc seulement si écart. |
| 4 | B2 | cafe-crypto-policy-mgt | — | **Policies** + **GET ?scan_id=** + règles persist **binding** | mergé | 28 | [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28) | | Voir aussi **PR5/6** Discovery↔CPM référence scan. |
| 5 | C1 | cafe-crypto-policy-mgt | `option-a/c1-option-a-script` | Script smoke : sign-in → **v1** list/detail → explore → persist | mergé | 34 | [#34](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/34) | | `test-discovery-v1-wallet-scans-to-cpm.sh` ; plus de scripts CBOM / contexts legacy. |
| 6 | C2 | cafe-crypto-policy-mgt (+ discovery si besoin) | `option-a/c2-contract-api-tests` | Tests contrat **v1** + CPM (complément **go test** existants) | | | | | Matrice : items liste v1, détail, explore, persist **binding=discovery**. |
| 7 | F1 | cafe-frontend | `option-a/f1-discovery-v1-wallet-scan-data-source` | Data source **liste + détail** scans wallet v1 | | | | | Post **PR 12** ; pas de couplage CPM dans le client HTTP Discovery. |
| 8 | F2 | cafe-frontend | `option-a/f2-frontend-cpm-scan-context` | Composable **`useCpmScanContext`** (sélection `scan_id`) | | | | | |
| 9 | F3 | cafe-frontend | `option-a/f3-frontend-cpm-scan-selector` | UI sélecteur de scan CPM (données v1) | | | | | |
| 10 | F4 | cafe-frontend | `option-a/f4-frontend-feed-scan-context-to-cpm` | Brancher contexte réel sur **`apiCpmDataSource` / explore** | | | | | Retirer **`mock-discovery-scan-placeholder`** en **API mode** quand un scan valide est choisi. |
| 11 | F5 | cafe-frontend | `option-a/f5-frontend-e2e` | E2E parcours Option A (**post-V1**) | | | | | Framework à trancher (Playwright vs extension Vitest/MSW) — governance repo. |
| 12 | D1 | cafe-documentation / CPM + liens | `option-a/d1-option-a-docs` | Narratif intégré **v1** + diagrams + scripts | | | | | Réf **`WORKPLAN_API_PR.md`** comme tête de chaîne merged. |

---

## PR A1 — Discovery **v1 wallet scans** (surface canonique ; historique `wallet-policy-contexts`)

**Statut : mergé** selon **`WORKPLAN_API_PR.md`** — liste + détail sous **`/discovery/v1/wallets/scans`** (**PR4** [#52](https://github.com/create2-labs/cafe-discovery/pull/52)) ; contrat OpenAPI **PR1a** [#49](https://github.com/create2-labs/cafe-discovery/pull/49).

**Historique (ne pas réimplémenter comme route publique nominale) :** une exposition **`GET /discovery/wallet-policy-contexts`** avait été livrée (**ex. PR [#48](https://github.com/create2-labs/cafe-discovery/pull/48)**) puis **retirée** au profit des APIs **v1** — **PR11a** [#54](https://github.com/create2-labs/cafe-discovery/pull/54). Tout client, script ou doc Option A doit s'appuyer sur **`openapi/discovery-v1.yaml`**, pas sur l'ancienne façade.

### 1. Goal (canonique aujourd'hui)

Fournir pagination, filtres et **détail** par **`scan_id`** conformes **§0.1** **`WORKPLAN_API.md`**, avec un DTO détail utilisable pour **`policy_context`** côté **CPM explore** (**PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29)).

### 2. Scope (référence merged)

- Handlers + services **`GET /discovery/v1/wallets/scans`**, **`GET …/wallets/scans/{scan_id}`**
- **`cafe-discovery/openapi/discovery-v1.yaml`**

### 3. Acceptance criteria (rappel)

- **401** sans / JWT invalide sur routes protégées.
- Listes : **`items`**, **`total`**, **`limit`**, **`offset`** ; **400** sur requêtes invalides (**PR4**).
- Isolement **owner** (user A ≠ user B).
- DTO détail / **`result`** conformes spec + **PR13a** [#56](https://github.com/create2-labs/cafe-discovery/pull/56) où applicable.

### 4. Tests

`cd cafe-discovery && GOWORK=off go test ./... -count=1`

### 5. Explicit non-goals

- Réintroduire **`GET …/wallet-policy-contexts`** comme surface publique nominale.

### 6. Références PR Git

Voir **`WORKPLAN_API_PR.md`** — **PR1a**, **PR4**, **PR11a**.

### 8. PR description template (doc / rappel release)

```markdown
## Summary
Option A relies on Discovery **v1** wallet scan list + detail (PR4 #52), not wallet-policy-contexts (removed PR11a #54).

## References
WORKPLAN_API_PR.md § PR4, PR11a ; openapi/discovery-v1.yaml
```

---

## PR A2 — Documentation contrat **Discovery v1 ↔ CPM** (Option A)

**Statut : mergé** — **cafe-discovery** [#60](https://github.com/create2-labs/cafe-discovery/pull/60).

**Branch :** `option-a/a2-discovery-cpm-v1-contract-docs` (merge dans `main` via PR ci-dessus)

### 1. Goal

Publier une **référence mainteneur** qui relie :

- **`GET /discovery/v1/wallets/scans`** + **`GET …/wallets/scans/{scan_id}`** (DTO détail = base du **`policy_context`** explore),
- **`POST /api/cpm/v1/policies/decisions/explore`** (sync, **avec** `policy_context` — **PR8**),
- **`POST /api/cpm/v1/policies`** + **`GET …/policies?scan_id=`** (**PR7**),
- **`POST /api/cpm/v1/policies/assessment/request`** (**sans** `policy_context` client — **PR13g**),

avec **URLs directes vs edge** (`/api/discovery/v1`, `/api/cpm/v1`). Croiser **`WORKPLAN_API.md`**, **`WORKPLAN_API_PR.md`**, **`CPM_FRONTEND_PR_PLAN_V1.md`** (suite **F\***).

### 2. Scope (expected)

- `cafe-discovery/docs/**/*.md` et/ou README (cf. aussi **PR12b** [#59](https://github.com/create2-labs/cafe-discovery/pull/59)).
- Liens vers **`openapi/discovery-v1.yaml`** et **`cafe-crypto-policy-mgt/openapi/cpm-v1.yaml`**.

### 3. Acceptance criteria

- **Aucune** mention de **`/discovery/wallet-policy-contexts`** comme chemin actif **sans** la qualifier *historique / retiré*.
- Pagination : champs **`items`** alignés OpenAPI (**pas** `{ contexts }` historique).
- **Table de correspondence normative §3.1** (obligatoire dans la livrable doc A2 — figer avant **F1/F4**) : pour **chaque** champ de l’objet **`policy_context`** accepté par CPM (**`explore`**, synchrone), une ligne indiquant : **(a)** clef/path côté DTO **`GET /discovery/v1/wallets/scans/{scan_id}`** (référence **`openapi/discovery-v1.yaml`** ou schémas générés), **(b)** clef/path côté **`policy_context`**, **(c)** règle (copie 1:1, renommage, dérivation, défaut/absence), **(d)** type + **liste fermée des valeurs / enums** où applicable.
- Révision croisée avec le code **`cafe-crypto-policy-mgt/internal/api/explore_policy_context.go`** (validation stricte) : la doc doit **coller aux validateurs**, pas inventer des alias silencieux.

#### 3.1 — Champs dont le mapping et les enums doivent être figés dans A2 (non exhaustif tant que OpenAPI/commande décide)

Les noms ci-dessous sont **indicatifs** : la livrable A2 doit reprendre les **noms exacts** du contrat Discovery v1 + de `policy_context`.

| Domaine | Exigence A2 |
|--------|--------------|
| **`wallet_type` / genre de compte** | Table **Discovery v1 → `policy_context`** : valeurs autorisées côté fil (ex. domaine **`EOA`**, **`AA`**, **`Contract`**…) vs normalisation **`normalizeWireAccountKind`** ; interdits ou synonymes (**`SMART_ACCOUNT`**…) → soit rejetées, soit mappées **explicitement**. |
| **Posture PQ** | Champ ou dérivation vers **`current_pq_posture`** dans `policy_context` : **liste fermée** alignée Discovery + CPM (**ex.** `classical_only`, `hybrid`, `full_pq`) — aucun libellé UI seul comme vérité. |
| **`result` (lifecycle)** | Sémantique du bloc **`result`** v1 (**PR13a**/OpenAPI) : quelles clés peuvent nourrir **`policy_context`** (statut scan, niveaux PQ, erreurs…) ; comportement si liste vs détail diffère. |
| **`chain_ids`** | Type tableau ; règle **stricte** « réseaux inconnus / non mappés → **`[]`** » vs interdiction d’inventer un fallback (**ex.** pas de `[1]` implicite) ; alignement liste courte vs enrichissement détail. |
| **`scanned_at` / timestamps** | Source unique (liste vs détail), format **RFC3339** ou variante documentée (**Nano**…) — doit matcher ce que **`explore`** accepte. |

### 4. Tests

Revue doc ; liens ; **checklist reviewer** « chaque ligne du §3.1 recoupée avec `discovery-v1.yaml` + `explore_policy_context.go` ». Option CI : lien vers test **C2** qui vérifie un **golden sample** corps explore construit depuis un JSON de détail v1 après la table §3.1 (sans appeler Discovery).

### 5. Explicit non-goals

- Remplacer **`WORKPLAN_API.md`** — **référencer** la vérité, ne pas fork.

### 6. Suggested commit message

`docs: Discovery v1 wallet scans ↔ CPM Option A flows`

### 7. Suggested PR title

`docs: Wallet scan v1 and CPM explore/persist contract (Option A)`

### 8. PR description template

```markdown
## Summary
Documents Option A using Discovery v1 list/detail + CPM v1 per WORKPLAN_API_PR.md.

## Acceptance
URL matrix ; §3.1 mapping v1 scan detail → `policy_context` (wallet_type, PQ, result, chain_ids, enums) ; explore vs assessment body rules.

## Non-goals
No behavior change unless OpenAPI typo fix requires it.
```

---

## PR B1 — CPM explore accepts **Option A / v1-aligned** payload

**Statut cible : mergé** — **`WORKPLAN_API_PR.md` PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29).

**Branch (suites / durcissement optionnel) :** `option-a/b1-cpm-explore-option-a-payload`

### 1. Goal

Garantir **`POST /api/cpm/v1/policies/decisions/explore`** traite **`scan_id` + `policy_context` + `selection_request`** comme contrat **preview synchrone** recommandé ; le **`policy_context`** est **aligné sur le DTO détail** **`GET /discovery/v1/wallets/scans/{scan_id}`** ; préserver la forme de sortie evaluateur (**`decision`** / **`ranked_candidates`**). Documenter **AUTH-02** et limites (CPM ne vérifie pas seul la propriété du scan sans délégation Discovery).

### 2. Scope (référence merged + écarts éventuels)

- `internal/api/read_api.go`, `explore_policy_context.go`, `read_api_test.go`, `openapi/cpm-v1.yaml`

### 3. Acceptance criteria

- Corps façon Option A → **200** lorsque l’authz permet.
- **4xx** clairs sur **`policy_context`** mal formé, `scan_id` conflictuels, champs invalides.
- Correspondance **OpenAPI** `cpm-v1.yaml` + **`WORKPLAN_API.md`**.

### 4. Tests

`cd cafe-crypto-policy-mgt && GOWORK=off go test ./... -count=1`

### 5. Explicit non-goals

- Assessment async **`POST …/policies/assessment/request`** — **pas** de `policy_context` client (**PR13g**).

### 6–8. Templates PR (référence)

Titre type : `CPM: decisions/explore vs Discovery v1 scan context` — voir **PR8** dans **`WORKPLAN_API_PR.md`**.

---

## PR B2 — CPM persist / liste par **`scan_id`**

**Statut cible : mergé** — **`WORKPLAN_API_PR.md` PR7** [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28) (liste **`GET /api/cpm/v1/policies?scan_id=`**, persist **`POST /api/cpm/v1/policies`**, **`binding`**).

**Branch (suite) :** `option-a/b2-cpm-persist-scan-binding`

### 1. Goal

Pour **`binding=discovery`** (défaut), **`scan_id`** **UUID obligatoire** à la persistence ; autres bindings (**`fixture`**, **`catalog`**, **`none`**) peuvent autoriser **`scan_id`** vide selon les règles produit — **`owner_routes.go`**. Réponse **`item`** expose le **`scan_id`** stocké.

### 2. Scope

- `internal/app/owner_routes.go`, `internal/persistence/owner_scoped_store.go`, tests `owner_routes_test.go`

### 3. Acceptance criteria

- Cohérent avec **PR7** + **internal reference** (**PR5/6**) pour DELETE scan Discovery.

### 4–8. Référence

Voir chapitres **PR5**, **PR6**, **PR7** dans **`WORKPLAN_API_PR.md`**.

---

## PR C1 — Script smoke Option A (**v1**)

**Statut : mergé** — **cafe-crypto-policy-mgt** [#34](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/34).

**Branch:** `option-a/c1-option-a-script`

### 1. Goal

**Livré :** script bash unique **`test-discovery-v1-wallet-scans-to-cpm.sh`** — sign-in Discovery → **`GET /discovery/v1/wallets/scans`** → **`GET …/wallets/scans/{scan_id}`** → **`POST /api/cpm/v1/policies/decisions/explore`** → **`POST /api/cpm/v1/policies`** (optionnel **`SKIP_PERSIST=1`**). **Ne plus** appeler **`/discovery/wallet-policy-contexts`** (retiré **PR11a**). **Pas** de parcours CBOM / scan polling legacy dans les scripts actifs.

### 2. Scope (livré)

- `scripts/test-discovery-v1-wallet-scans-to-cpm.sh` + libs **`scripts/lib/discovery-route-paths.sh`**, **`scripts/lib/cpm-route-paths.sh`**
- Suppression **`test-discovery-wallet-contexts-to-cpm.sh`**, **`test-wallet-scan-and-cpm-policy.sh`**
- CI : **`bash -n`** sur le smoke script (`.github/workflows/ci.yml`)

### 3. Acceptance criteria (rappel — satisfait sur `main`)

- **Help** documente edge **`/api/discovery/v1/wallets/scans`**, **`/api/cpm/v1/...`**
- **`CURL_REDIRECT`** default **`0`** ; pas d’impression JWT/mot de passe
- Ambiguïté multi-scan sans **`SCAN_ID`** → échec explicite documenté
- Aucune sous-chaîne route legacy **`wallet-policy-contexts`** dans **`scripts/**/*.sh`** (vérification manuelle / **C2** pour automatiser)

### 4. Tests

```bash
bash -n scripts/test-discovery-v1-wallet-scans-to-cpm.sh
# shellcheck (optional): shellcheck scripts/test-discovery-v1-wallet-scans-to-cpm.sh
```

Manual: `SKIP_PERSIST=1` against local stack.

### 5. Explicit non-goals

- Scripts CBOM ou **`LEGACY_SCAN_AND_CBOM_FLOW`** (retirés avec **#34**).
- Garde-fou shell versionné en CI — reporté à **C2** si réintroduit (`scripts/ci/…`).

### 6. Suggested commit message (historique)

`chore(scripts): replace legacy smoke scripts with Discovery v1 Option A flow`

### 7. Suggested PR title (historique)

`chore(scripts): Discovery v1 wallet scans smoke test for Option A (C1)`

### 8. PR description template (historique)

```markdown
## Summary
Option A bash smoke uses Discovery v1 list + detail and CPM explore/persist; legacy context/CBOM scripts removed.

## Test plan
bash -n scripts/test-discovery-v1-wallet-scans-to-cpm.sh; manual SKIP_PERSIST=1
```

---

## PR C2 — Tests contrat **v1**

**Branch:** `option-a/c2-contract-api-tests`

### 1. Goal

Compléter la couverture **CI-friendly** alignée **`openapi/discovery-v1.yaml`** + **`cpm-v1.yaml`** et **`WORKPLAN_API_PR.md`** (au-delà des suites existantes).

### 2. Scope

- Paquet `contract` ou extension tests existants ; pas d’obligation Docker si **httptest** suffit

### 3. Acceptance criteria matrix (minimum)

**Discovery v1 (wallet scans)**

- Pas JWT → **401** ; JWT valide → **200** sur liste autorisée
- Enveloppe liste : **`items`**, pagination
- Détail **`…/wallets/scans/{scan_id}`** : champs stables **assertés** (alignement OpenAPI)
- Isolement owner

**CPM**

- Explore Option A → **200** (store fixture si besoin)
- Corps invalide → **4xx**
- Persist **`binding=discovery`** sans **`scan_id`** valide → **4xx** ; avec UUID → **`item.scan_id`** renseigné
- (Rappel) Assessment request avec **`policy_context`** → **400** (**PR13g**)

**Régression façade retirée (obligatoire — même logique que C1)**

- Test automatisé (**`go test`** + **`TestScript`**, cible **`Makefile`**, ou job CI) invoquant la **même** condition d’échec que **C1** sur **`cafe-crypto-policy-mgt/scripts/**/*.sh`**.  
- **`*.md`** (documentation / workplans) **exclus**, y compris s’ils mentionnent encore l’historique **`wallet-policy-contexts`**. Dériver la commande dans un petit shell versionné (ex. **`scripts/ci/assert-scripts-use-discovery-wallet-scans-v1-only.sh`**) pour que **C2** + **CI** partagent une seule source — **hors périmètre #34** (CI actuelle : **`bash -n`** uniquement).

**Golden mapping (couplé A2)**

- Au moins un test (Go ou snapshot JSON) où un **`detail` Discovery v1** fixture → corps **`explore`** attendu (champs **`policy_context`**) vérifiant **wallet_type**, **posture PQ**, **`result`/dérivés**, **`chain_ids`** selon la **table §3.1** de **A2** — garantit que le mapping documenté reste présent dans le codebase.

### 4–8. Voir templates existants ; référencer **WORKPLAN_API_PR.md** PR1a/1b

---

## PR F1 — Frontend **Discovery v1 wallet scan** data source

**Branch:** `option-a/f1-discovery-v1-wallet-scan-data-source`

**Contexte :** suite à **`CPM_FRONTEND_PR_PLAN_V1.md`** (**clos** — **PR 12** API `CpmDataSource`). Ce module est **séparé** de **`CpmDataSource`** : HTTP Discovery uniquement.

### 1. Goal

Client typé **authentifié** : **`listWalletScans({ limit?, offset?, … })`** + **`getWalletScanDetail(scanId)`** sur **`/api/discovery/v1/wallets/scans`** et **`…/wallets/scans/{scan_id}`** (ou équivalent aligné **`scanService.js`** / `routePaths` — **PR10**/**PR13e**).

### 2. Scope

- Nouveau module TS (ex. `src/discovery/walletScanV1DataSource.ts`) + tests Vitest mocks

### 3. Acceptance criteria

- Types alignés **A2** / OpenAPI ; erreurs structurées ; **pas** d’appel CPM dans ce client

### 4–8. Mettre à jour titres / messages : **v1**, pas *wallet-policy-contexts*

---

## PR F2 — Frontend `useCpmScanContext`

**Branch:** `option-a/f2-frontend-cpm-scan-context`

### 1. Goal

Composable : liste + sélection **`scan_id`**, sync **`?scanId=`**, états loading/empty/error ; **ne** déclenche **pas** explore tant qu’aucun scan valide n’est choisi (sauf politique mock V1 inchangée).

### 2. Scope

- `cafe-frontend/src/cpm/**/*.ts` (ou `src/composables/…`)

### 3. Acceptance criteria

- **API mode** : pas de **`mock-discovery-scan-placeholder`** comme **scan réellement sélectionné** — cohérent avec **F4** (le mock V1 reste acceptable **tant que** l’utilisateur n’a pas choisi de scan ; documenter la transition).
- Expose **`selectedScanId`**, **`selectedScanDetail`**, chargement async

### 4–8. Voir template historique ; remplacer *Discovery contexts* par *v1 scan list/detail*

---

## PR F3 — UI sélecteur de scan CPM

**Branch:** `option-a/f3-frontend-cpm-scan-selector`

### 1. Goal

Sélecteur alimenté **uniquement** par **F1** (DTO v1 liste + résumé depuis détail ou champs liste).

### 2. Scope

- Composants Vue CPM / page Crypto Policy Management

### 3. Acceptance criteria

- Utiliser **`/api/discovery/v1/wallets/scans`** (edge) — **pas** d’ancien **`wallet-policy-contexts`**
- États : vide (**lancer un scan wallet**), erreur auth/réseau, liste

### 4–8. Templates : remplacer références *wallet-policy-contexts*

---

## PR F4 — Brancher scan réel sur **`apiCpmDataSource`**

**Branch:** `option-a/f4-frontend-feed-scan-context-to-cpm`

### 1. Goal

En **API mode**, lorsqu’un scan v1 est sélectionné : **`selectionScanId`** = UUID réel ; construire **`policy_context`** compatible **explore** depuis le **détail v1** ; appeler **`getPolicySelection`** / explore avec **`scan_id` + `policy_context` + `selection_request`**. **Ne pas** envoyer **`mock-discovery-scan-placeholder`** dans les requêtes réseau.

### 2. Scope

- `src/cpm/useCpmPolicySelection.ts`, `apiCpmDataSource.ts`, mapping éventuel

### 3. Acceptance criteria

- **Contrat fonctionnel inchangé** : en **API mode**, dès qu’un **vrai scan** est sélectionné (**`scan_id`** UUID présent depuis la sélection utilisateur — **pas** uniquement depuis un mock V1 résiduel), le client **ne doit plus** faire transiter **`mock-discovery-scan-placeholder`** dans **aucune** charge utile CPM issue de ce flux (**`scan_id`** top-level, champs imbriqués du **`policy_context`**, ni tout autre clé corrélée).
- **Garde-fou réseau obligatoire (au moins un test)** — pas seulement assert sur **`selectionScanId`** en mémoire : un **test au niveau transport** (**Vitest**) qui intercepte la requête réelle envoyée (**`fetch` mock / MSW / `createApiCpmDataSource`** hook) et **échoue** si le **corps HTTP sérialisé** (string) du **`POST …/policies/decisions/explore`** contient la sous-chaîne **`mock-discovery-scan-placeholder`**, **dans le scénario** « API mode + scan v1 sélectionné + appel explore déclenché ».  
  - **Variante acceptée** : spy sur **`Request.body`** / **`request.clone().text()`** avant envoi, avec la même assertion sur la chaîne brute.  
  - **Hors scope de ce test** : parcours **mock-only** où aucun scan n’a été sélectionné (placeholder encore permis jusqu’à choix utilisateur — cohérent **F2**).
- Changement de scan ⇒ reset validation / EOA / brouillons incohérents (aligné **PR 8–11** V1).

### 4–8. Dépend de **B1** (merged) + **A2 §3.1** (mapping canonique pour construire **`policy_context`** sans surprise)

---

## PR F5 — E2E **post-V1**

**Branch:** `option-a/f5-frontend-e2e`

### 1. Goal

Parcours critique : auth → liste scans v1 → sélection → templates / explore → validation → persist avec **`scan_id`** visible.

### 2. Scope

- Playwright ou autre si gouvernance repo ; base **Vitest** déjà en place (**`CPM_FRONTEND_PR_PLAN_V1.md`**)

### 3. Acceptance criteria

- Aligné **edge** **`/api/discovery/v1`** + **`/api/cpm/v1`**

### 4–8. Inchangé en intention

---

## PR D1 — Documentation intégrée **(v1 truth)**

**Branch:** `option-a/d1-option-a-docs`

### 1. Goal

Narratif **Option A réconcilié** : **`WORKPLAN_API_PR.md`** comme chaîne merged ; diagrammes Discovery v1 ↔ CPM v1 ↔ frontend (**post F\***) ; distinguer **explore** (avec `policy_context`) vs **assessment** (**sans** `policy_context` client, **PR13g**).

### 2. Scope

- `cafe-crypto-policy-mgt/docs/`, liens **`cafe-documentation`**, **`WORKPLAN_API.md`**, **`CPM_FRONTEND_PR_PLAN_V1.md`**

### 3. Acceptance criteria

- Schéma : scan → DB Discovery (PS cible long terme) → **v1 list/detail** → UI → **explore** → **persist**
- Table URL **direct / edge** ; scripts **C1** mis à jour
- **Pas** de présentation de `wallet-policy-contexts` comme route active sans note *historique*

### 4–8. Référencer **PR11a** suppression façade + **PR8/7/13g**

---

## Risks and open questions

- **Champs PQ / `result` v1**, **`wallet_type`**, **`chain_ids`** — déchargés dans la **table §3.1** **A2** + golden **C2** ; types TS **F1** suivent cette table.
- **AUTH-02** / latence Discovery : UX fail-closed — coordonner **F3/F4**.
- **Forme du payload persist** (`payload` JSON riche vs minimal) — documenter hors **D1**.
- **F5 tooling** : Playwright vs extension Vitest/MSW — décision governance **`cafe-frontend`** (**`CPM_FRONTEND_PR_PLAN_V1.md` § backlog** peut informer mais ne remplace pas la décision repo).

---

## Manual validation commands

```bash
# Discovery unit + integration suites
cd cafe-discovery && GOWORK=off go test ./... -count=1

# CPM unit + integration suites
cd cafe-crypto-policy-mgt && GOWORK=off go test ./... -count=1

# Option A bash smoke (C1 #34 : chemins /discovery/v1 + /api/cpm/v1)
SKIP_PERSIST=1 \
  DISCOVERY_EMAIL='user@example.com' DISCOVERY_PASSWORD='secret' \
  ./cafe-crypto-policy-mgt/scripts/test-discovery-v1-wallet-scans-to-cpm.sh

# Script syntax / optional shellcheck (C1 #34)
bash -n cafe-crypto-policy-mgt/scripts/test-discovery-v1-wallet-scans-to-cpm.sh

# Frontend — base V1 (voir CPM_FRONTEND_PR_PLAN_V1.md CI)
cd cafe-frontend && npm test
cd cafe-frontend && npm run typecheck && npm run lint
```

**Edge (rappel) :** parcours navigateur contre **`https://<host>/api/discovery/v1/...`** et **`/api/cpm/v1/...`** — **`WORKPLAN_API_PR.md` PR9**.

```bash
# Interdiction route morte dans les scripts shell actifs (identique à C1 / C2)
if grep -R "wallet-policy-contexts" cafe-crypto-policy-mgt/scripts --include='*.sh' -n; then
  exit 1
fi
```

---

## Global non-goals (entire initiative)

- Big-bang refactor of Persistence Service ingestion or ripping DB out of Discovery prematurely.
- **Réintroduire** la route **`GET /discovery/wallet-policy-contexts`** comme contrat nominal (voir **PR11a** **`WORKPLAN_API_PR.md`**).
- **Ne pas** contredire **`WORKPLAN_API_PR.md`** sur les bases publiques : **`/api/discovery/v1`**, **`/api/cpm/v1`**.
- Frontend calling Persistence Service or SQL directly.

**Note CBOM / historique :** le serveur Discovery **`GET /discovery/cbom/*`** a été **retiré** (**PR13c** [#57](https://github.com/create2-labs/cafe-discovery/pull/57)) ; l’UI **nominal** passe par détail **v1** (**PR13b** sur `main`). Reliquats scripts/README sont suivis sous **WORKPLAN** (ex. **`cafe.sh --cboms`**).

---

_Révision document : aligné sur [`WORKPLAN_API_PR.md`](./WORKPLAN_API_PR.md) et [`CPM_FRONTEND_PR_PLAN_V1.md`](./CPM_FRONTEND_PR_PLAN_V1.md) ; jalons **A1, A2, B1, B2, C1 (#34)** marqués **mergé** ; la surface `wallet-policy-contexts` est **historique** ; le contrat nominal est **Discovery v1** `wallets/scans` + **CPM** `/api/cpm/v1`. **Suite :** **C2** — grep/CI automatisé sur `scripts/**/*.sh` ; **F4** — test transport interdisant `mock-discovery-scan-placeholder` sur explore ; **A2 §3.1** — mapping normatif détail v1 → `policy_context`._

_End of PR plan._
