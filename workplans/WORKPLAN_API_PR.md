# CAFE API coherency — PR plan

**Référence canonique des contrats et sémantiques :** [`WORKPLAN_API.md`](./WORKPLAN_API.md).

**Règles d’exécution (propriétaire humain) :** l’agent / les contributeurs ne font **pas** de commit, push, merge ni tags ; revue, git et publication restent manuelles. Chaque PR : branche locale, changements ciblés, tests, puis proposition de titre/message de commit et de PR (en anglais dans les sections dédiées ci‑dessous).

**Statut du document :** plan de découpe ; jalon OpenAPI **mergé** — **PR1a** [`cafe-discovery` PR #49](https://github.com/create2-labs/cafe-discovery/pull/49), **PR1b** [`cafe-crypto-policy-mgt` PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). **Chaîne livrée sur `main` (Discovery + CPM, prérequis PR6)** — **PR2** absorbé dans **`cafe-discovery` [#51](https://github.com/create2-labs/cafe-discovery/pull/51)** (PR dédiée **[#50](https://github.com/create2-labs/cafe-discovery/pull/50)** fermée sans merge) ; **PR3** [#51](https://github.com/create2-labs/cafe-discovery/pull/51) ; **PR4** [#52](https://github.com/create2-labs/cafe-discovery/pull/52) ; **PR5** [`cafe-crypto-policy-mgt` #27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27) ; **PR6** [#53](https://github.com/create2-labs/cafe-discovery/pull/53) ; **PR8** [`cafe-crypto-policy-mgt` #29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR9** (edge) — [`cafe-deploy` #11](https://github.com/create2-labs/cafe-deploy/pull/11). **PR10** (frontend) — [`cafe-frontend` #52](https://github.com/create2-labs/cafe-frontend/pull/52). **PR11a** (cleanup routes Discovery legacy) — [`cafe-discovery` #54](https://github.com/create2-labs/cafe-discovery/pull/54). Suite : **PR7**+, **PR11b**, **PR11c** (factorisation chemins API par dépôt), **fermeture dette CBOM** (**PR13a**→**PR13c**), doc (**PR12**), selon **dépendances** et **revue propriétaire** ; ce document reste la découpe de référence.

---

## Executive summary (état du dépôt au moment de la rédaction)

| Domaine | État observé dans le repo | Écart vs `WORKPLAN_API.md` |
|--------|---------------------------|----------------------------|
| **Chemins publics Discovery** | **`/discovery/v1`** sur `main` (**[#51](https://github.com/create2-labs/cafe-discovery/pull/51)** …) ; wallets CAFE sous **`/discovery/v1/wallets`**. Edge **PR9** — **`cafe-deploy` [#11](https://github.com/create2-labs/cafe-deploy/pull/11)**. Frontend **PR10** — **`cafe-frontend` [#52](https://github.com/create2-labs/cafe-frontend/pull/52)**. **PR11a** — **`cafe-discovery` [#54](https://github.com/create2-labs/cafe-discovery/pull/54)** : listes/scan/context legacy retirés. | **Plus** d’anciennes listes/POST scan ; routes **hors v1** encore servies (voir **§ Routes Discovery hors v1 après PR11a**). |
| **Hydratation UI (CBOM)** | **PR10** : listes **v1** + **`GET /discovery/cbom/{address\|url}`** pour cartes (wallet + TLS). Détail v1 **`GET …/scans/{scan_id}`** existe mais **`result`** est **minimal** vs champs UI (`risk_score`, `cipher_suites[]`, etc.). | Cible : hydrater par **`scan_id`** via **`result`** enrichi — **PR13a**→**PR13c** ; puis retirer **`/discovery/cbom/*`**. |
| **Listes de scans** | **PR4** — **`GET /discovery/v1/wallets/scans`**, **`GET /discovery/v1/tls/scans`** (owner-only ; **pas** les defaults TLS dans la liste owner). | **`GET /discovery/v1/tls/scans/defaults`** (catalogue partagé) — **PR13a** ; aligner OpenAPI. |
| **POST scan** | **PR3** — **`POST /discovery/v1/scan`** : **`scan_id`**, **`requested`**, **`location`**. Ancien **`POST /discovery/scan`** retiré — **PR11a** [#54](https://github.com/create2-labs/cafe-discovery/pull/54). | — |
| **DELETE scans / 409 wallet** | **PR6** — **`DELETE /discovery/v1/wallets/scans/{scan_id}`**, **`DELETE /discovery/v1/tls/scans/{scan_id}`** + vérif CPM (**PR5**). | **409 `WALLET_REFERENCED_BY_POLICY`** (**PR7**). |
| **wallet-policy-contexts** | Retiré — **PR11a** [#54](https://github.com/create2-labs/cafe-discovery/pull/54) (remplacé par **`wallets/scans`** + CPM **`GET …/policies?scan_id=`** — **PR7**). | — |
| **CPM** | **PR5** sur `main` — **[#27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27)** : **`POST /internal/policies/references/scan`** (token service, verdict **`referenced`**). **PR8** sur `main` — **[#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29)** : **`POST …/policies/decisions/explore`** non persistant, **`policy_context`** compatible détail scan Discovery v1. Toujours **`read_api.go`** / **`owner_routes.go`** sur chemins historiques **`/api/v1/...`** ; pas encore **`GET ?scan_id=`** public, **`DELETE` policies** cible v1, **`/api/cpm/v1`** mux complet (**PR7** / **PR9** / **11b**) selon état du dépôt. | Cible **`/api/cpm/v1/...`**, **`id` + `scan_id` → 400**, **`DELETE` 204/404 uniquement**, liste par **`scan_id`**, **`scan_id` obligatoire** pour le flux Discovery → CPM côté API publique. |
| **OpenAPI** | **PR1a** + **PR1b** mergées : `openapi/discovery-v1.yaml` ([`cafe-discovery` #49](https://github.com/create2-labs/cafe-discovery/pull/49)), `openapi/cpm-v1.yaml` ([`cafe-crypto-policy-mgt` #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26)). Edge documentée en **PR9** — **`cafe-deploy` [#11](https://github.com/create2-labs/cafe-deploy/pull/11)**. | **Deux** artefacts de contrat distincts (§0.1 + §0.2) ; l’edge se décrit via **`servers`** dans chaque spec. |
| **Frontend** | **PR10** mergée — **`cafe-frontend` [#52](https://github.com/create2-labs/cafe-frontend/pull/52)** : v1 listes/scan ; **CBOM** pour hydratation ; CPM **`/api/cpm/v1`**. Parcours CPM « spec UX » **hors** PR10 — **Option A**. | **PR13b** migration hydratation ; **PR7** CPM public ; **PR11b** cleanup rollout CPM ; **PR11c** factorisation chemins client. |

---

## Table de suivi des PR

| PR | Branche (proposée) | Dépôt principal | PR Git | Dépend de | Risques / suites (résumé) | Objectif en une ligne |
|----|--------------------|-----------------|--------|-----------|---------------------------|------------------------|
| **1a** | `api-contract/api-coherency-openapi` | `cafe-discovery` | [#49](https://github.com/create2-labs/cafe-discovery/pull/49) | — | Dérive spec vs `WORKPLAN_API.md` ; maintenir `discovery-v1.yaml` aligné. | OpenAPI **§0.1** : **`openapi/discovery-v1.yaml`** ; pas de handler ; option validation CI. |
| **1b** | `api-contract/api-coherency-openapi` | `cafe-crypto-policy-mgt` | [#26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26) | — | Idem côté CPM ; maintenir `cpm-v1.yaml` aligné. | OpenAPI **§0.2** : **`openapi/cpm-v1.yaml`** ; pas de handler ; option validation CI. |
| **2** | `discovery/api-v1-route-skeleton` | `cafe-discovery` | [#51](https://github.com/create2-labs/cafe-discovery/pull/51) *(inclut PR2 ; [#50](https://github.com/create2-labs/cafe-discovery/pull/50) fermée sans merge)* | **1a** | Clients / edge : n’utiliser que **`/api/discovery/v1/wallets`** (plus de **`/api/wallets`**). | Monter **`/discovery/v1`** ; ordre **`wallets/scans` avant `wallets/:wallet_id`** ; squelettes ; **CAFE wallets uniquement sous v1**. |
| **3** | `discovery/post-scan-contract-response` | `cafe-discovery` | [#51](https://github.com/create2-labs/cafe-discovery/pull/51) | **2** | Remplacer le stub **501** sur **`POST /discovery/v1/scan`** ; clients **`processing`** → **`requested`** (**10**). | **`POST /discovery/v1/scan`** : réponse contrat + **`requested`** + **`location`**. |
| **4** | `discovery/scan-history-lifecycle` | `cafe-discovery` | [#52](https://github.com/create2-labs/cafe-discovery/pull/52) | **2**, **3** | Remplacer les **501** sur listes / détail scans v1 ; perf listes (N+1) ; OpenAPI si écart. | Listes + détail wallet/TLS : **`items`**, pagination, tri, filtres, règles **`result`**. |
| **5** | `cpm/internal-policy-reference-by-scan` | `cafe-crypto-policy-mgt` | [#27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27) | **4** | Charge à chaque DELETE scan ; timeouts, circuit, **SLO** documentés. | **CPM seul** : endpoint **interne** (token service) — « cet owner a-t-il des policies persistées qui référencent ce **`scan_id`** ? » — verdict **`referenced`** (+ **`count`** optionnel logs) ; **pas** d’IDs de policies dans la réponse. |
| **6** | `discovery/delete-semantics-cpm-reference-check` | `cafe-discovery` | [#53](https://github.com/create2-labs/cafe-discovery/pull/53) | **4**, **5** (et **7** si 409 **`wallet_id`** via CPM) | Latence DELETE ; **503** attendu si CPM indispo ; runbook exploit. | **Discovery** orchestre le DELETE scan / wallet mais **consomme seulement** le verdict CPM ; **503 fail-closed** si la vérif CPM est indisponible. |
| **7** | `cpm/policies-scan-reference-contract` | `cafe-crypto-policy-mgt` | — | **1b**, **4**, **5** | Liste **`scan_id`** en O(n) en mémoire ; plan DB ; même lookup que **PR5**. | **`/api/cpm/v1/policies`**, **`GET ?scan_id=`** (même lookup owner-scoped que **PR5**), **`DELETE ?id=`**, validations, alias rollout optionnels. |
| **8** | `cpm/decisions-explore-contract-check` | `cafe-crypto-policy-mgt` | [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29) | **1b**, **7** | Validation trop stricte → démos ; garder voie « fixtures » (**PR7**). | **`POST …/policies/decisions/explore`** : non persistant ; DTOs alignés scan v1. |
| **9** | `deploy/api-v1-edge-alignment` | `cafe-deploy` | [#11](https://github.com/create2-labs/cafe-deploy/pull/11) | **2**, **7** | Coupure si edge avant images ; **ne pas** réintroduire de **`/api/wallets`** (hors **`/api/discovery/v1/wallets`**). | NGINX / env : **`/api/discovery/v1`**, **`/api/cpm/v1`** ; chemins §0.3 **temporaires** documentés. |
| **10** | `frontend/api-coherency-migration` | `cafe-frontend` | [#52](https://github.com/create2-labs/cafe-frontend/pull/52) | **9** (ou stack locale équivalente) | Bundles / **SW** staging ; procédure de purge ; **CRUD wallets** : base **`/api/discovery/v1/wallets`**. | Clients vers nouveaux chemins et enveloppes. |
| **11a** | `cleanup/remove-obsolete-discovery-routes` | `cafe-discovery` | [#54](https://github.com/create2-labs/cafe-discovery/pull/54) | **10** | Intégrateurs sur anciennes URLs — note release ; **11b** peut suivre. | Retrait listes/scan/context legacy ; **conserver** routes hors v1 (CBOM, utilities, assessments). **Réalisé.** |
| **11b** | `cleanup/remove-cpm-rollout-and-client-leftovers` | `cafe-crypto-policy-mgt`, `cafe-frontend`, `cafe-crypto-policy-mgt/scripts`, `cafe-deploy` | — | **10**, **11a** | Diff « fourre-tout » ; **ne pas** y mélanger CBOM (**PR13**) ni factorisation chemins (**11c**). | Alias **§0.3** CPM / nginx, scripts, reliquats client CPM. |
| **11c** | `refactor/centralize-api-route-paths` | `cafe-crypto-policy-mgt`, `cafe-discovery`, `cafe-deploy`, `cafe-frontend` *(PR Git **par dépôt**, pas de lib partagée inter-repo)* | — | **11b** (chemins canoniques figés) | Refactor pure ; risque dérive auth ↔ mux si test d’alignement absent. | Constantes / helpers **locaux** : mux, auth, tests, scripts, clients — **sans** mutualisation inter-dépôt. |
| **13a** | `discovery/v1-scan-result-ui-parity` | `cafe-discovery` | — | **11a** | `result` trop large vs spec minimale — revue OpenAPI ; defaults TLS lecture par `scan_id`. | Enrichir **`result`** v1 + **`GET …/tls/scans/defaults`** + détail default TLS ; OpenAPI aligné. |
| **13b** | `frontend/v1-scan-detail-hydration` | `cafe-frontend` | — | **13a** | Régression cartes si mapping incomplet ; charge N× détail (garder concurrency). | Hydrater listes via **`GET …/scans/{scan_id}`** ; retirer appels **`/discovery/cbom/*`**. |
| **13c** | `discovery/remove-cbom-route` | `cafe-discovery`, `cafe-deploy` (scripts) | — | **13b** | Intégrateurs sur CBOM — note release. | Supprimer **`GET /discovery/cbom/*`** ; e2e sans CBOM. |
| **13d** | `discovery/utilities-and-assessments-v1` *(optionnel)* | `cafe-discovery` | — | **11b**, **13c** | Périmètre flou si mélangé avec **13**. | Migrer ou documenter **`assessments/request`**, **`rpcs`**, **`scanners`**. |
| **12** | `docs/api-coherency-runbook-qa` | `cafe-documentation` (+ README optionnels) | — | **11a**, **11b**, **13c** | Doc obsolète si merge tard. | Runbooks **sans** CBOM ; curl **v1** uniquement ; checklist QA §8. |

**Colonne PR Git :** lien vers la pull request du dépôt concerné lorsqu’elle existe ; **—** = pas encore créée / à renseigner. *Remarque : une PR plan peut être livrée dans une PR Git unique (ex. **PR2** + **PR3** → **`cafe-discovery` #51**) ; une PR Git peut être **fermée sans merge** si le périmètre a été réintégré ailleurs (ex. **#50** → **#51**).*

**Colonne Risques / suites (résumé) :** synthèse pour lecture rapide ; le détail (**Risks**, **Completion criteria**, dépendances, périmètres) reste dans le **chapitre de chaque PR** ci‑dessous.

**Découpe PR5 + PR6 (propriété et autorité) :** **Discovery** possède les **scans** (cycle de vie, stockage, suppression physique). **CPM** possède les **policies** et est **autoritaire** sur la question « un **`scan_id`** est-il référencé par des instances persistées pour ce **owner** ? ». **Discovery** ne connaît pas la structure interne des policies, **n’inspecte pas** la DB CPM, **ne duplique pas** d’index `scan_id → policies`, **ne décide pas** seul du 409, **ne supprime jamais** de policies en cascade. **Discovery** appelle **CPM** (PR **5**), **consomme uniquement** le verdict (`referenced: true|false`), puis applique **409** ou poursuit le **DELETE** (PR **6**). L’UI peut lister les policies à détacher via l’API publique **`GET /api/cpm/v1/policies?scan_id=...`** (**PR7**), distincte de l’endpoint interne.

---

## Jalon OpenAPI — PR Git **#49** (Discovery) + **#26** (CPM)

**Principe de dépôt :** le contrat Discovery est **possédé** par `cafe-discovery` ; le contrat CPM par `cafe-crypto-policy-mgt`. Aucun dépôt ne concentre l’OpenAPI de l’autre — évite l’ambiguïté « le CPM possède aussi Discovery ».

**Coordination :** **PR1a** et **PR1b** sont **mergées** — [`cafe-discovery` #49](https://github.com/create2-labs/cafe-discovery/pull/49), [`cafe-crypto-policy-mgt` #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). **Chaîne implémentation Discovery/CPM (PR plan 2→6) sur `main` :** [#51](https://github.com/create2-labs/cafe-discovery/pull/51) (PR2+PR3), [#52](https://github.com/create2-labs/cafe-discovery/pull/52) (PR4), [`cafe-crypto-policy-mgt` #27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27) (PR5), [#53](https://github.com/create2-labs/cafe-discovery/pull/53) (PR6) — voir **suivi PR Git** dans le tableau et les chapitres **PR2** … **PR6** ci‑dessous. **PR8** livrée [`cafe-crypto-policy-mgt` #29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR7** peut s’appuyer sur **`cpm-v1.yaml`** et le lookup **PR5**.

---

## PR1a — OpenAPI Discovery (§0.1)

**Merge :** livrée dans **`cafe-discovery`** via [**PR #49**](https://github.com/create2-labs/cafe-discovery/pull/49) (spec **`openapi/discovery-v1.yaml`** sur `main`).

| Fichier | Dépôt | Contenu (réf. `WORKPLAN_API.md`) |
|---------|--------|----------------------------------|
| **`openapi/discovery-v1.yaml`** | `cafe-discovery` | §0.1 : base publique **`/api/discovery/v1`**, wallets, **`wallets/scans`**, **`tls/scans`**, **`POST …/scan`**, DELETE, erreurs côté Discovery (**`SCAN_REFERENCED_BY_POLICY`**, **`WALLET_REFERENCED_BY_POLICY`**), lifecycle / listes / DTOs détail wallet & TLS. Annexe **§0.3** uniquement pour chemins **Discovery** de transition si utile. |

**Edge (Discovery) :** chemins publics **`https://host/api/discovery/v1/...`** documentés via **`servers`** et, si besoin, note sur le strip **`/api`** côté upstream (détail nginx = **PR9**).

- **Branch:** `api-contract/api-coherency-openapi`
- **Repository:** `cafe-discovery` uniquement
- **Objective:** Publier le contrat machine-readable **§0.1** comme artefact reviewable **avant** **PR2** (OpenAPI + éventuelle validation CI ; **aucun** handler).
- **Scope:** OpenAPI 3.x ; schémas liste vs détail (`ScanListItem`, `TLSListItem`, DTOs détail, réponse **`POST /scan`**, tri par défaut **`created_at` desc, `scan_id` desc`**).
- **Out of scope:** Spec CPM (**PR1b**), handlers, nginx, frontend, migrations DB.
- **Dependencies:** Aucune (référence `WORKPLAN_API.md` uniquement).
- **Implementation notes:** **Greenfield** pour ce fichier. Références croisées **discursives** vers CPM autorisées dans les `description` (lien repo), sans **import OpenAPI** depuis le dépôt CPM.
- **Tests:** `npx @redocly/cli lint openapi/discovery-v1.yaml` ; CI `cafe-discovery` si présente.
- **Validation commands:** Lint ci-dessus ; `cd cafe-discovery && go test ./...` (inchangé si spec seule).
- **Proposed commit title:** `Add OpenAPI spec for Discovery API v1`
- **Proposed commit message:** `Add openapi/discovery-v1.yaml describing /api/discovery/v1 per WORKPLAN_API.md (routes, scans, wallets, errors, lifecycle). No runtime change.`
- **Proposed PR title:** `OpenAPI: Discovery /api/discovery/v1 contract`
- **Proposed PR body:** Lien `WORKPLAN_API.md` ; moitié CPM sur `main` — [`cafe-crypto-policy-mgt` PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26) ; non-goals (pas nginx, pas handlers) ; enchaînement **PR2** … **PR12**.
- **Risks:** (Historique) désalignement spec si le workplan évolue sans mettre à jour **`discovery-v1.yaml`** / **`cpm-v1.yaml`** sur `main` — suivre les PR d’implémentation et `WORKPLAN_API.md`.
- **Completion criteria:** Chemins Discovery reviewables dans **`cafe-discovery/openapi/discovery-v1.yaml`** sur `main` ; fichier validé (CI ou commande locale documentée). **Réalisé** — [PR #49](https://github.com/create2-labs/cafe-discovery/pull/49).

---

## PR1b — OpenAPI CPM (§0.2)

**Merge :** livrée dans **`cafe-crypto-policy-mgt`** via [**PR #26**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26) (spec **`openapi/cpm-v1.yaml`** sur `main`).

| Fichier | Dépôt | Contenu (réf. `WORKPLAN_API.md`) |
|---------|--------|----------------------------------|
| **`openapi/cpm-v1.yaml`** | `cafe-crypto-policy-mgt` | §0.2 : base **`/api/cpm/v1`**, catalogue, templates, instances, **`decisions/explore`**, policies, drafts, règles **`id` vs `scan_id`**, codes d’erreur côté CPM. Annexe **§0.3** pour alias CPM / catalogue **`/api/v1/...`** si utile. |

**Edge (CPM) :** chemins publics **`https://host/api/cpm/v1/...`** documentés via **`servers`** (détail nginx = **PR9**).

- **Branch:** `api-contract/api-coherency-openapi`
- **Repository:** `cafe-crypto-policy-mgt` uniquement
- **Objective:** Publier le contrat machine-readable **§0.2** comme artefact reviewable **avant** **PR7** / **PR8** (OpenAPI + éventuelle validation CI ; **aucun** handler).
- **Scope:** OpenAPI 3.x ; schémas et chemins CPM sous **`/api/cpm/v1`**.
- **Out of scope:** Spec Discovery (**PR1a**), handlers, nginx, frontend, migrations DB ; pas de fichier unique fusionnant les deux services.
- **Dependencies:** Aucune (référence `WORKPLAN_API.md` uniquement).
- **Implementation notes:** **Greenfield** pour ce fichier. Références croisées **discursives** vers Discovery autorisées dans les `description` (lien repo), sans **import OpenAPI** depuis le dépôt Discovery.
- **Tests:** `npx @redocly/cli lint openapi/cpm-v1.yaml` ; CI `cafe-crypto-policy-mgt` si présente.
- **Validation commands:** Lint ci-dessus ; `cd cafe-crypto-policy-mgt && go test ./...` (inchangé si spec seule).
- **Proposed commit title:** `Add OpenAPI spec for CPM API v1`
- **Proposed commit message:** `Add openapi/cpm-v1.yaml describing /api/cpm/v1 per WORKPLAN_API.md (policies, drafts, catalog, explore). No runtime change.`
- **Proposed PR title:** `OpenAPI: CPM /api/cpm/v1 contract`
- **Proposed PR body:** Lien `WORKPLAN_API.md` ; Discovery **PR1a** sur `main` — [`cafe-discovery` PR #49](https://github.com/create2-labs/cafe-discovery/pull/49) ; non-goals (pas nginx, pas handlers) ; enchaînement **PR7** … **PR12** (selon fil CPM).
- **Risks:** (Historique) désalignement spec si le workplan évolue sans mettre à jour **`cpm-v1.yaml`** / **`discovery-v1.yaml`** sur `main` — suivre les PR d’implémentation et `WORKPLAN_API.md`.
- **Completion criteria:** Chemins CPM reviewables dans **`cafe-crypto-policy-mgt/openapi/cpm-v1.yaml`** sur `main` ; fichier validé (CI ou commande locale documentée). **Réalisé** — [PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26).

---

## PR2 — Squelette d’routes Discovery `/discovery/v1` et ordre d’enregistrement

**Merge :** livré dans **`cafe-discovery`** via [**PR #51**](https://github.com/create2-labs/cafe-discovery/pull/51) (avec **PR3** sur la même PR Git). La PR dédiée au seul squelette — [**#50**](https://github.com/create2-labs/cafe-discovery/pull/50) — a été **fermée sans merge** (périmètre PR2 réintégré dans **#51**).

- **Branch:** `discovery/api-v1-route-skeleton`
- **Repository:** `cafe-discovery`
- **Objective:** Introduire le groupe **`/discovery/v1`** reflétant la surface cible avec **ordre Fiber correct** (`/wallets/scans`, `/wallets/scans/:scan_id` avant **`/wallets/:wallet_id`** / `:pubKeyHash` côté impl).
- **Scope:** `internal/app/container.go` (et découpe handler si nécessaire) ; placeholders **501** ou **404** explicites pour sous-routes non encore implémentées jusqu’aux PR **3** / **4** ; **CAFE wallets** exclusivement sous **`/discovery/v1/wallets`** (**plus** de groupe racine **`/wallets`**).
- **Out of scope:** Logique complète liste/détail, DELETE, modification des chemins CBOM, suppression de l’ancien `GET /discovery/scans`.
- **Dependencies:** **PR1a** (référence : **`cafe-discovery/openapi/discovery-v1.yaml`** sur `main` — mergée [PR #49](https://github.com/create2-labs/cafe-discovery/pull/49)).
- **Implementation notes:** **`POST`** et **`PUT`** wallets restent exposés sous **`/discovery/v1/wallets`** (création / mise à jour) pour parité produit ; le tableau **§0.1** du workplan liste surtout **GET**/**DELETE** sur la ressource wallet — aligner la spec OpenAPI si le produit fige l’inventaire exact des verbes.
- **Tests:** Tests Fiber : **`GET /discovery/v1/wallets/scans`** ne matche pas **`/discovery/v1/wallets/:pubKeyHash`** ; tests d’ordre de routes.
- **Validation commands:** `cd cafe-discovery && go test ./...`
- **Proposed commit title:** `Add Discovery /v1 route tree with wallets/scans ordering`
- **Proposed commit message:** `Register /discovery/v1 routes with wallets/scans before wallets/:wallet_id per WORKPLAN_API. Stub handlers where implementation follows in subsequent PRs.`
- **Proposed PR title:** `Discovery: v1 API route skeleton and registration order`
- **Proposed PR body:** Table des routes, lien vers **`openapi/discovery-v1.yaml`**, suites POST scan / listes / DELETE.
- **Risks:** Clients ou scripts encore sur **`/wallets`** ou **`/api/.../wallets`** hors **`/api/discovery/v1/...`** — migration **10** / doc ; nginx (**PR9**) doit router uniquement la cible v1.
- **Completion criteria:** `go test` prouve l’ordre de routage ; la liste des chemins dans **`openapi/discovery-v1.yaml`** correspond aux routes montées ; **aucune** route **`/wallets`** racine côté Discovery. **Réalisé** — périmètre livré dans [`cafe-discovery` PR #51](https://github.com/create2-labs/cafe-discovery/pull/51) (PR **[#50](https://github.com/create2-labs/cafe-discovery/pull/50)** fermée sans merge).

---

## PR3 — Réponse `POST /discovery/v1/scan` et lifecycle initial

**Merge :** livré dans **`cafe-discovery`** via [**PR #51**](https://github.com/create2-labs/cafe-discovery/pull/51) (même PR Git que **PR2**).

- **Branch:** `discovery/post-scan-contract-response`
- **Repository:** `cafe-discovery`
- **Objective:** Aligner **`POST …/scan`** sur le workplan : corps (address **XOR** url), **`scan_id`** à l’acceptation, réponse avec **`scan_family`**, **`status: requested`**, **`location`**.
- **Scope:** `internal/handler/discovery.go` (`UnifiedScan` / helpers de queue) ; réutiliser l’UUID déjà créé pour NATS.
- **Out of scope:** DTOs liste/détail (**PR4**), retrait de l’ancien `POST /discovery/scan` (**PR11a**).
- **Dependencies:** **PR2** (route v1 montée).
- **Implementation notes:** Implémenter **uniquement** sous **`/discovery/v1/scan`** (plus de chemin parallèle hors v1). Persistance **`requested`** avant publish NATS si ce n’est pas déjà garanti — possible toucher `internal/service` / workers au minimum.
- **Tests:** Handler : exclusion mutuelle, forme JSON, **`scan_id`** présent et stable à l’acceptation ; comportement 503 scanner absent inchangé si applicable.
- **Validation commands:** `cd cafe-discovery && go test ./...`
- **Proposed commit title:** `Discovery v1: POST scan returns scan_id, family, and location`
- **Proposed commit message:** `Align POST /discovery/v1/scan with WORKPLAN_API: allocate scan_id at acceptance, status requested, and return location for wallet or TLS scan detail URL.`
- **Proposed PR title:** `Discovery v1: POST /scan contract and requested status`
- **Proposed PR body:** Exemple avant/après ; corrélation NATS inchangée ; dépendance PR2.
- **Risks:** Clients supposant l’ancien libellé **`processing`** — migration **10** / **11a**/**11b** ; noter en risque release.
- **Completion criteria:** Tests de contrat + parité champs **`openapi/discovery-v1.yaml`** pour la réponse POST. **Réalisé** — [`cafe-discovery` PR #51](https://github.com/create2-labs/cafe-discovery/pull/51).

---

## PR4 — Historique de scans : listes, détail, pagination, tri, validation des queries

**Merge :** livré dans **`cafe-discovery`** via [**PR #52**](https://github.com/create2-labs/cafe-discovery/pull/52).

- **Branch:** `discovery/scan-history-lifecycle`
- **Repository:** `cafe-discovery`
- **Objective:** Implémenter les **GET** cibles : listes wallet + TLS et détails par **`scan_id`** ; règles **`result`** / immutabilité sur le détail ; lifecycle aligné workplan.
- **Scope:** `DiscoveryHandler`, `TLSHandler`, `userScanCache` / dépôts ; remplacer **`results`** par **`items`** ; tri par défaut **`created_at` desc, `scan_id` desc** ; **`GET …/wallets/scans?chain_id=` sans `address` → 400** ; liste TLS : ne pas supporter **`address` / `chain_id`** (400 si query interdite fournie, interprétation stricte).
- **Out of scope:** DELETE et **409** (**PR6**) ; suppression des anciennes listes (**PR11a**).
- **Dependencies:** **PR2**, **PR3** (URLs de détail et stabilité **`scan_id`**).
- **Implementation notes:** Toutes les routes cibles sont sous **`/discovery/v1/...`** (listes scans wallet/TLS déjà préfixées **`…/wallets/scans`**, **`…/tls/scans`** — **pas** de **`/wallets`** racine). **`GET …/wallets/scans/{scan_id}`** et **`GET …/tls/scans/{scan_id}`** ; aligner le service `wallet_policy_context` si encore utilisé en interne.
- **Tests:** Table-driven : enveloppe pagination, ordre de tri, **400** sur queries invalides, plusieurs lignes d’historique pour une même adresse.
- **Validation commands:** `cd cafe-discovery && go test ./...`
- **Proposed commit title:** `Discovery v1: wallet and TLS scan lists and detail DTOs`
- **Proposed commit message:** `Implement GET wallets/scans, GET tls/scans, and GET …/scans/{scan_id} with items/total/limit/offset, default sort, address/chain_id validation, and result lifecycle rules on detail.`
- **Proposed PR title:** `Discovery v1: scan history lists and scan_id detail`
- **Proposed PR body:** Matrice des filtres ; dépendance PR3 ; DELETE en PR6.
- **Risks:** Perf si la liste impose des jointures Redis/DB — éviter N+1 ; synopsis minimal en liste comme au workplan.
- **Completion criteria:** Tous les GET §0.1 sous v1 implémentés ; tests couvrant les bords de query ; comportement aligné avec **`openapi/discovery-v1.yaml`** (mettre à jour la spec si la revue révèle un écart). **Réalisé** — [`cafe-discovery` PR #52](https://github.com/create2-labs/cafe-discovery/pull/52).

---

## PR5 — CPM : vérification autoritaire des références policy → `scan_id` (interne)

**Merge :** livré dans **`cafe-crypto-policy-mgt`** via [**PR #27**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27).

**Séparation des responsabilités**

| Service | Rôle |
|---------|------|
| **Discovery** | Possède les **scans** ; exécute le **DELETE** scan une fois autorisé ; **ne** déduit **pas** les références policy lui-même. |
| **CPM** | Possède les **policies** ; **seul** capable de répondre correctement à : « ce **`scan_id`** est-il référencé par au moins une instance persistée pour ce **`user_id` / `tenant_id`** ? » |

Discovery **ne doit pas** : lire la persistance CPM directement, maintenir un index local `scan_id → policies`, inférer les références à partir de payloads, ni supprimer des policies en cascade lors d’un DELETE scan.

- **Branch:** `cpm/internal-policy-reference-by-scan`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Exposer un **verdict** booléen (et optionnellement un **`count`** pour logs / diagnostic uniquement) : Discovery n’a **pas besoin** des IDs de policies pour retourner **409** ; le frontend utilisera **`GET /api/cpm/v1/policies?scan_id=...`** (**PR7**) pour afficher quelles policies délier avant un nouveau DELETE scan.
- **Scope:** Endpoint **interne** (même modèle de confiance que les routes **`/internal/...`** existantes : token service, **non** exposé à l’edge). La résolution « policies persistées de cet **owner** qui référencent ce **`scan_id`** » doit passer par une **fonction / service interne partagé** (lookup owner-scoped) réutilisé tel quel par **`GET /api/cpm/v1/policies?scan_id=...`** en **PR7** (voir ci‑dessous).
- **Out of scope:** Handler DELETE côté Discovery (**PR6**) ; l’enregistrement HTTP public **`GET ?scan_id=`** (**PR7**) — mais le **module de lookup** introduit ici est la **seule** source de vérité pour les deux surfaces.
- **Dependencies:** **PR4** (identité produit **`scan_id`** stable).
- **Implementation notes:** Variables d’URL CPM interne pour Discovery en **PR9** (compose) ou **6** (config Discovery) ; option : documenter la route dans **`openapi/cpm-v1.yaml`** (tag `internal` ou fichier annexe) pour versionner le contrat sans l’exposer au SDK public. Extraire le lookup dans un paquet interne (ex. `internal/persistence` ou `internal/policyrefs`) pour que **PR7** n’ait **aucune** seconde implémentation « parallèle ».

**Contrat interne proposé (à figer à l’implémentation)**

`POST /internal/policies/references/scan` (chemin exact aligné sur les conventions CPM existantes ; **hors** `/api/cpm/v1` public).

**Requête (exemple)**

```json
{
  "scan_id": "550e8400-e29b-41d4-a716-446655440000",
  "user_id": "user-123",
  "tenant_id": "tenant-abc"
}
```

**Réponse (exemple)**

```json
{
  "referenced": true,
  "count": 2
}
```

- **`referenced`** : obligatoire pour le verdict.
- **`count`** : **optionnel** ; utile pour observabilité / logs ; Discovery **n’en a pas besoin** pour choisir **409** vs **204**.

**Deux surfaces HTTP, une seule logique métier (avec PR7)**

| Surface | Route | Réponse / usage |
|---------|--------|------------------|
| **Interne** (service token) | `POST /internal/policies/references/scan` | **`referenced`** (+ **`count`** optionnel) — garde-fou **DELETE** Discovery. |
| **Publique owner** (JWT, **PR7**) | `GET /api/cpm/v1/policies?scan_id=...` | Liste des **instances** de policy pour le frontend (détacher avant nouveau DELETE scan). |

Both endpoints must rely on the same internal owner-scoped policy lookup logic to avoid divergent semantics between the Discovery delete guard and the public frontend listing.

- **Tests:** `referenced: true` lorsqu’au moins une policy persistée du owner pointe vers le **`scan_id`** ; `false` sinon ; après **`DELETE /policies?id=...`** côté CPM, même appel → `false` ; token invalide → **403** ; corps invalide → **400** ; tests unitaires sur le **lookup partagé** pour que **`referenced`** et la liste **PR7** restent cohérents sur les mêmes jeux de données.
- **Validation commands:** `cd cafe-crypto-policy-mgt && go test ./...`
- **Proposed commit title:** `CPM: internal policy reference check by scan_id for Discovery`
- **Proposed commit message:** `Add service-authenticated POST /internal/policies/references/scan returning referenced (+ optional count) so Discovery can fail closed on scan delete without reading CPM persistence.`
- **Proposed PR title:** `CPM: authoritative internal scan_id policy reference check`
- **Proposed PR body:** Ownership table ; no policy IDs in response ; public `GET …/policies?scan_id=` is PR7 for UX ; shared owner-scoped lookup with PR7 ; security (internal token, not on edge).
- **Risks:** Charge sur CPM à chaque DELETE scan — timeouts courts + circuit ; documenter SLO.
- **Completion criteria:** Discovery peut appeler l’endpoint et interpréter uniquement **`referenced`** (PR **6**). **Réalisé** — [`cafe-crypto-policy-mgt` PR #27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27).

---

## PR6 — Discovery DELETE semantics using CPM reference verification

**Merge :** livrée dans **`cafe-discovery`** via [**PR #53**](https://github.com/create2-labs/cafe-discovery/pull/53) (branche proposée `discovery/delete-semantics-cpm-reference-check`).

**Principe :** Discovery **reste propriétaire** de l’action « supprimer ce scan » (effacement des données scan côté Discovery). Discovery **ne détermine pas** lui-même les références policy : il **demande à CPM** le verdict via l’endpoint interne de la **PR5**, puis **consomme seulement** ce verdict.

**Flux (ex. `DELETE /api/discovery/v1/wallets/scans/{scan_id}`)**

1. **Discovery** vérifie localement : le scan **existe**, appartient à l’**utilisateur** authentifié, n’est **pas** déjà supprimé → sinon **404** (ou règle workplan équivalente).
2. **Discovery** appelle **CPM** (interne) : « cet **owner** a-t-il des **policies persistées** qui référencent ce **`scan_id`** ? »
3. **Interprétation du verdict CPM**
   - **`referenced: true`** → Discovery répond **409** avec code **`SCAN_REFERENCED_BY_POLICY`** (sans avoir inspecté les policies).
   - **`referenced: false`** → Discovery **supprime** le scan et répond **204**.
   - **Erreur, timeout, réponse JSON invalide, erreur HTTP « transitoire » côté CPM (p.ex. 5xx), ou impossibilité d’obtenir un verdict fiable** → **fail closed** : Discovery **ne supprime pas** le scan ; réponse recommandée **503 Service Unavailable** avec un corps explicite, par exemple :

```json
{
  "error": "POLICY_REFERENCE_CHECK_UNAVAILABLE",
  "message": "The scan cannot be deleted because policy references could not be verified."
}
```

   - **CPM répond `401` ou `403` sur l’appel interne** : traiter comme **misconfiguration inter-service** (token Discovery→CPM absent/incorrect, rôle interne mal aligné, etc.) — **pas** comme un refus « utilisateur ». Discovery **ne propage pas** ce **401/403** au client utilisateur (ce serait **trompeur** : l’utilisateur n’est pas forcément interdit sur le scan). Même **503** fail-closed qu’au-dessus, **même code d’erreur** recommandé (**`POLICY_REFERENCE_CHECK_UNAVAILABLE`**), éventuellement avec détail **serveur** (logs / `request_id`) pour l’exploitation, **sans** exposer la nature exacte du secret manquant au navigateur.

**À ne pas faire**

- Discovery lit directement la **base CPM** (ou tout store policy).
- Discovery **duplique** un index **`scan_id` → policies**.
- Discovery **décide seul** si une policy référence un scan (sans appel CPM).
- Discovery **supprime des policies en cascade** lors d’un DELETE scan.
- Discovery **propage** au client final le **401** ou **403** renvoyé par CPM sur l’appel interne de vérification (confondrait **auth inter-service** et **auth utilisateur** sur le scan).

**Symétrie TLS :** même orchestration pour **`DELETE …/tls/scans/{scan_id}`** (même appel interne CPM par **`scan_id`**).

- **Branch:** `discovery/delete-semantics-cpm-reference-check`
- **Repository:** `cafe-discovery`
- **Objective:** Implémenter la matrice DELETE (wallets, wallet scans, TLS scans) : **204 / 404 / 409** selon `WORKPLAN_API.md`, avec **409** scan **uniquement** sur verdict CPM **`referenced: true`** ; **503** fail-closed si la vérification CPM est indisponible, **y compris** lorsque CPM renvoie **401/403** sur l’endpoint interne (traités comme erreur d’intégration, **pas** propagés au client).
- **Scope:** Handlers v1 ; client HTTP vers CPM interne (config **URL + token** alignée sur **PR9** ou variables Discovery) ; **DELETE wallet** ne supprime **pas** les scans ; idempotence **404** sur second DELETE.
- **Out of scope:** Implémentation de l’endpoint CPM (**PR5**) ; **`GET ?scan_id=`** public (**PR7**) ; sémantique **`DELETE /api/cpm/v1/policies`**.
- **Dependencies:** **PR4** (ressources scan adressables), **PR5** (endpoint interne disponible) ; **PR7** si le **409 `WALLET_REFERENCED_BY_POLICY`** repose sur une question CPM distincte (référence explicite **`wallet_id`**).
- **Implementation notes:** Même pattern pour **wallet** et **TLS** scans ; tests d’intégration optionnels derrière flag si les deux services ne sont pas toujours disponibles en CI.
- **Tests:** Mock client CPM : `referenced true` → 409 ; `false` → 204 ; timeout / 5xx / JSON invalide → **503** + **`POLICY_REFERENCE_CHECK_UNAVAILABLE`** ; **401/403** depuis CPM interne → **503** (jamais **403** utilisateur sur ce chemin) ; second DELETE → **404**.
- **Validation commands:** `cd cafe-discovery && go test ./...`
- **Proposed commit title:** `Discovery v1: DELETE scans with CPM reference verification`
- **Proposed commit message:** `Before deleting wallet or TLS scans, call CPM internal reference check; return 409 from CPM verdict; map CPM 401/403/5xx/timeouts to 503 fail-closed (no user 403 on misconfigured inter-service auth); Discovery never inspects CPM persistence.`
- **Proposed PR title:** `Discovery v1: DELETE semantics using CPM reference verification`
- **Proposed PR body:** Ownership diagram (short) ; link PR5 contract ; 503 JSON example ; **401/403 CPM interne → 503** (pas de propagation **403** utilisateur) ; anti-patterns list for reviewers.
- **Risks:** Latence DELETE ; operability si CPM down — documenter que **503** est attendu et que l’utilisateur peut réessayer après correction CPM / réseau.
- **Completion criteria:** §2.2 / §4.2 DELETE scan du workplan respectés ; aucun accès Discovery à la persistance CPM ; documenter **503** / **`POLICY_REFERENCE_CHECK_UNAVAILABLE`** dans **`openapi/discovery-v1.yaml`** si les erreurs y sont listées. **Réalisé** — [`cafe-discovery` PR #53](https://github.com/create2-labs/cafe-discovery/pull/53).

---

## PR7 — Contrat public CPM policies (`/api/cpm/v1`, GET par `scan_id`, DELETE, validation)

- **Branch:** `cpm/policies-scan-reference-contract`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Implémenter **§0.2** sur le mux (alias **§0.3** optionnels marqués dépréciés) ; **`GET /policies?id=`** vs **`GET /policies?scan_id=`** mutuellement exclusifs → **400** ; **`DELETE /policies?id=`** **204/404 uniquement** ; **`POST /policies`** rejette **`scan_id`** manquant pour le flux Discovery → CPM (exceptions documentées : brouillons, fixtures, tests hors produit).
- **Scope:** `internal/app/owner_routes.go`, `internal/persistence/owner_scoped_store.go` (index / scan par **`scan_id`**), `internal/app/auth.go`, enregistrement **`read_api.go`** pour catalog sous **`/api/cpm/v1/policies/...`** (enregistrement double **`/api/v1/policies`** jusqu’à **11b** si besoin de transition).
- **Out of scope:** Frontend (**10**), nginx (**9**), logique explore profonde (**8**).
- **Dependencies:** **PR1b** (référence : **`cafe-crypto-policy-mgt/openapi/cpm-v1.yaml`** sur `main` — mergée [PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26)) ; **PR4** recommandé pour tests d’intégration **`scan_id`** réalistes ; **PR5** (**obligatoire** : **`GET …/policies?scan_id=`** doit appeler le **même** lookup owner-scoped que `POST /internal/policies/references/scan`, sans seconde implémentation).
- **Implementation notes:** Réutiliser le **même** helper owner-scoped introduit en **PR5** (ex. **`ListPoliciesByScanID(principal, scanID)`**) : le handler **`POST /internal/.../references/scan`** agrège en **`referenced`** / **`count`** ; ce PR expose **`GET …/policies?scan_id=`** qui retourne les **items** pour l’UI — **aucune** seconde requête SQL / boucle parallèle sur le store pour la même question. Scripts `test-wallet-scan-and-cpm-policy.sh` : toucher seulement si CI l’exige — sinon reporter à **12** / **11b**.

Both endpoints must rely on the same internal owner-scoped policy lookup logic to avoid divergent semantics between the Discovery delete guard and the public frontend listing.

- **Tests:** `owner_routes_test.go` : DELETE, liste **`scan_id`**, query combinée **400**, POST sans **`scan_id`** quand la règle est activée ; tests d’accord **interne vs public** : mêmes données ⇒ **`referenced`** cohérent avec **`len(list)`** du GET.
- **Validation commands:** `cd cafe-crypto-policy-mgt && go test ./...`
- **Proposed commit title:** `CPM v1: policies CRUD, scan_id listing, and rollout aliases`
- **Proposed commit message:** `Add /api/cpm/v1 policies and drafts routes per WORKPLAN_API: GET by id or scan_id (not both), DELETE 204/404, and scan_id validation on persist for Discovery-bound flows.`
- **Proposed PR title:** `CPM v1: policies contract and scan_id correlation`
- **Proposed PR body:** Catalog sous nouveau préfixe ; doublons §0.3 temporaires ; **`GET …/policies?scan_id=`** pour l’UI — même lookup owner-scoped que **`POST /internal/policies/references/scan`** (PR **5**) ; distinct en surface HTTP seulement.
- **Risks:** Store en mémoire : liste par **`scan_id`** en O(n) — acceptable MVP ; documenter pour passage DB.
- **Completion criteria:** Contrat public aligné avec **`openapi/cpm-v1.yaml`** (mises à jour si besoin) ; table d’auth à jour.

---

## PR8 — CPM `POST …/policies/decisions/explore` : non-persistance et alignement DTO

**Merge :** livrée dans **`cafe-crypto-policy-mgt`** via [**PR #29**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29).

- **Branch:** `cpm/decisions-explore-contract-check`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Garantir que **explore** ne persiste pas d’instance finale ; accepter **`policy_context`** / filaire observation compatible avec le détail Discovery v1.
- **Scope:** `internal/api/read_api.go`, `explore_policy_context.go`, `read_api_test.go`, `openapi/cpm-v1.yaml`.
- **Out of scope:** Nouvelles voies de persistance ; nginx.
- **Dependencies:** **PR1b** (**`openapi/cpm-v1.yaml`** sur `main` — [PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26)), **PR7** (stabilité des chemins **`/api/cpm/v1`**).
- **Tests:** Étendre les tests explore avec des payloads formés comme le DTO détail v1.
- **Validation commands:** `cd cafe-crypto-policy-mgt && go test ./...`
- **Proposed commit title:** `CPM: align decisions/explore with Discovery v1 scan context`
- **Proposed commit message:** `Verify POST …/policies/decisions/explore does not persist policies; tighten validation and tests for policy_context from v1 scan DTOs.`
- **Proposed PR title:** `CPM: decisions/explore contract vs Discovery v1`
- **Proposed PR body:** Énoncé explicite : aucune écriture policy en base sur explore ; exemples.
- **Risks:** Validation trop stricte casse des démos — garder le chemin « fixtures » documenté en PR7.
- **Completion criteria:** Tests documentent la non-persistance ; l’évaluateur s’exécute toujours. **Réalisé** — [PR #29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29).

---

## PR9 — Edge / déploiement

- **Branch:** `deploy/api-v1-edge-alignment`
- **Repository:** `cafe-deploy` (principal) ; commentaires d’env compose si nécessaire pour URLs internes PR5/6.
- **Objective:** Exposer **`/api/discovery/v1`** et **`/api/cpm/v1`** avec **`proxy_pass`** cohérent ; documenter les alias **§0.3** comme **temporaires** ; **garantir** que **`/api/internal/...`** reste **bloqué** à l’edge (NGINX **`location ^~ /api/internal/`** → **403**), y compris après toute refonte des blocs **`location /api/...`** — ne jamais router ce préfixe vers Discovery ni CPM.
- **Scope:** `templates/nginx/nginx.conf.template`, templates env (health/blackbox) si les chemins changent.
- **Out of scope:** Suppression définitive des alias rollout (**PR11b**) ; code applicatif.
- **Dependencies:** **PR2** et **PR7** (routes existantes avant bascule clients).
- **Implementation notes:** **Résoudre l’incohérence** frontend **`/api/cpm/...`** vs mux direct **`/api/v1/cpm/...`** : une seule histoire pour navigateur et scripts après cette PR. **Ne pas** exposer d’alias edge **`/api/wallets`** (ou équivalent court) vers Discovery : les wallets CAFE passent par **`/api/discovery/v1/wallets`** après strip **`/api`**.
- **Sécurité interne CPM (PR5 / PR6) — staging & prod :** avec **`CPM_AUTH_REQUIRED=true`** (cible opérationnelle), **`CAFE_POLICY_REFERENCE_INTERNAL_SERVICE_TOKEN`** doit être **défini et non vide** sur **cafe-cpm**, et **identique** au secret utilisé par **Discovery** pour appeler **`POST /internal/policies/references/scan`** (client **PR6**). Sinon : token absent → **503** sur l’endpoint interne (fail-closed pour l’appelant, mais **déploiement incomplet**). **`CPM_AUTH_REQUIRED=false`** désactive **tout** le middleware d’auth CPM : l’endpoint interne **n’exige plus** de Bearer service — **inacceptable** pour un environnement réel exposant le réseau de services ; réserver **`false`** au dev local contrôlé. Vérifier les **`env/*.env.template`** et **`compose/25-cpm.yml`** (et, au **PR6**, variables Discovery) pour forcer la paire **`CPM_AUTH_REQUIRED=true` + token interne** hors dev.
- **Tests:** `nginx -t` en CI ou check manuel documenté ; extraits curl dans la PR body ; **vérification obligatoire** : depuis l’extérieur (TLS terminé à l’edge comme en prod), **`GET` ou `POST https://<host>/api/internal/<suffixe>`** (ex. chemin factice ou **`…/auth/session/validate`**) doit répondre **403** avec le corps JSON edge actuel (**`internal API is not exposed at the edge`**), **sans** joindre l’upstream Discovery. Répéter après tout changement d’ordre des `location` ou de la map **`$backend_api_uri`** pour éviter qu’un strip **`/api`** ne réexpose **`/internal/...`**.
- **Validation commands:** `docker compose ... config` si applicable ; `nginx -t` sur conf générée.
- **Proposed commit title:** `Deploy: edge routes for /api/discovery/v1 and /api/cpm/v1`
- **Proposed commit message:** `Update NGINX routing for WORKPLAN_API public bases; document temporary rollout paths and Discovery→CPM internal URL envs.`
- **Proposed PR title:** `Deploy: NGINX alignment for Discovery and CPM v1 APIs`
- **Proposed PR body:** Cartographie edge → upstream ; note de migration staging/prod ; **preuve explicite** que **`/api/internal/*`** reste **403** à l’edge (curl + extrait de conf **`location ^~ /api/internal/`**).
- **Risks:** Coupure si chemins basculés avant images — ordonner avec **10**.
- **Completion criteria:** Stack locale : curls vers v1 OK ; **checklist edge** : **`/api/internal/...` → 403** documentée (commande curl ou test smoke) et non-régression sur les chemins **`/api/discovery/v1`** / **`/api/cpm/`**.

---

## PR10 — Migration frontend

- **Branch:** `frontend/api-coherency-migration`
- **Repository:** `cafe-frontend`
- **Objective:** Basculer les appels Discovery vers **`/discovery/v1/...`** (via base **`/api`**), adopter **`items`**, traiter la réponse **`POST /scan`** (**`location`**, **`scan_id`**), CPM vers **`/api/cpm/v1/...`** aligné avec **9**. **CRUD wallets** : tout appeler sous **`/api/discovery/v1/wallets`** (plus d’URL **`/wallets`** ou **`/api/.../wallets`** hors ce préfixe).
- **Scope:** `src/services/scanService.js`, `tlsService.js`, services / composables **wallets** (création, liste, suppression, etc.), `cpm/apiCpmDataSource.ts`, composables / tests associés ; `.env.example` si besoin.
- **Out of scope:** Suppression backend des anciennes routes — **réalisée** en **PR11a** ([#54](https://github.com/create2-labs/cafe-discovery/pull/54)) ; suppression **`GET /discovery/cbom/*`** (**PR13c**) ; enrichissement **`result`** v1 (**PR13a**) ; hydratation détail (**PR13b**) ; nettoyage rollout CPM (**11b**).
- **Dependencies:** **PR9** (ou overrides locaux coordonnés).
- **Tests:** Mise à jour Vitest/Jest pour builders d’URL et parsers ; `apiCpmDataSource.spec.ts`.
- **Validation commands:** `cd cafe-frontend && npm run test` (ou script CI du projet) ; `npm run build` si applicable.
- **Proposed commit title:** `Frontend: migrate to Discovery v1 and CPM v1 APIs`
- **Proposed commit message:** `Update REST clients for new paths, pagination envelopes, and POST /scan correlation fields per WORKPLAN_API.`
- **Proposed PR title:** `Frontend: API coherency migration for Discovery and CPM`
- **Proposed PR body:** Liste des endpoints remplacés ; note feature-flag si applicable.
- **Risks:** Anciens bundles / SW en staging — procédure de purge.
- **Completion criteria:** Parcours scan + listes v1 + appels CPM catalogue/explore via edge OK contre la stack **9**. **Réalisé** — [`cafe-frontend` PR #52](https://github.com/create2-labs/cafe-frontend/pull/52). *(Hydratation CBOM legacy et defaults TLS : dette **PR13** ; page CPM complète en mode `api` — Option A.)*

---

## Routes Discovery hors v1 après PR11a (conservées volontairement)

**PR11a** ([#54](https://github.com/create2-labs/cafe-discovery/pull/54), mergée) a retiré les routes **remplacées par v1** (listes par adresse/URL, ancien POST scan, wallet-policy-contexts). Les chemins ci‑dessous **restent montés** — ce n’est pas un oubli ; la fermeture est planifiée par phase.

| Route | Auth | Rôle actuel | Sortie planifiée |
|-------|------|-------------|------------------|
| **`/discovery/v1/*`** | Bearer owner | Contrat canonique (listes, détail, POST scan, wallets CAFE, DELETE) | **Garder** |
| **`GET /discovery/cbom/*`** | Bearer owner | Hydratation UI par **adresse / URL** (champs carte : `risk_score`, `cipher_suites[]`, CycloneDX, etc.) | **PR13b** bascule client → **PR13c** retrait serveur |
| **`POST /discovery/assessments/request`** | Bearer owner | Demande d’évaluation (hors OpenAPI v1) | **PR13d** (optionnel) ou annexe |
| **`GET /discovery/rpcs`**, **`GET /discovery/scanners`** | Public | Catalogues utilitaires | **PR13d** (optionnel) ou annexe OpenAPI |

**Écart workplan (détail scan) :** `GET …/wallets|tls/scans/{scan_id}` renvoie aujourd’hui un **`result` minimal** (`WalletScanResult` / `TlsScanResult` dans `discovery-v1.yaml`). Le CBOM legacy expose davantage de champs UI. **PR13a** aligne **`result`** sur les besoins UI, puis **PR13b** / **PR13c** ferment le CBOM.

**Defaults TLS :** `GET /discovery/v1/tls/scans` liste **uniquement** les scans **owner** (`default=false`). Le catalogue partagé est **`GET /discovery/v1/tls/scans/defaults`** (**PR13a**). Le frontend ne doit plus déduire les defaults depuis la liste owner + flag `default` (comportement **PR10** obsolète après séparation listes).

---

## Chaîne PR13 — Fermeture dette CBOM

```text
PR13a (Discovery: result enrichi + defaults TLS)
  → PR13b (Frontend: hydratation par scan_id, plus de CBOM)
    → PR13c (Discovery + scripts: retirer GET /discovery/cbom/*)
      → PR12 (Doc: exemples sans CBOM)
PR13d (optionnel): utilities / assessments hors v1
```

**Règle :** ne pas regrouper **13a+13b** ni **13b+13c** — le client doit basculer avant retrait serveur.

---

## PR11a — Suppression des routes Discovery obsolètes

**Merge :** livré dans **`cafe-discovery`** via [**PR #54**](https://github.com/create2-labs/cafe-discovery/pull/54) (branche `cleanup/remove-obsolete-discovery-routes`).

- **Branch:** `cleanup/remove-obsolete-discovery-routes`
- **Repository:** `cafe-discovery` uniquement
- **Objective:** Retirer les handlers, routes Fiber et tests associés aux chemins **remplacés par v1** une fois **PR10** déployée : **`GET /discovery/scans`**, **`GET /discovery/tls/scans`**, **`GET /discovery/wallet-policy-contexts`**, **`POST /discovery/scan`**, et code mort associé.
- **Scope:** `internal/app/container.go`, `discovery.go`, `tls.go`, tests ; **ne pas** retirer **`GET /discovery/cbom/*`**, **`POST /discovery/assessments/request`**, **`GET /discovery/rpcs`**, **`GET /discovery/scanners`** (voir tableau ci‑dessus).
- **Out of scope:** **`GET /discovery/cbom/*`** (**PR13c**) ; enrichissement **`result`** v1 (**PR13a**) ; frontend ; nginx ; CPM.
- **Dependencies:** **PR10** (clients sur v1) ; **9** déployé sur les cibles où le cleanup s’applique.
- **Tests:** `cd cafe-discovery && go test ./...` ; grep du repo sans références aux anciens chemins enregistrés.
- **Validation commands:** Idem ; smoke `curl` direct sur backend si utile.
- **Proposed commit title:** `Remove obsolete Discovery HTTP routes after v1 migration`
- **Proposed commit message:** `Drop legacy discovery list/context/scan routes per WORKPLAN_API. Keep /discovery/cbom, assessments, rpcs, and scanners until PR13. v1 remains the product contract for lists and scan trigger.`
- **Proposed PR title:** `Cleanup: remove obsolete Discovery API routes`
- **Proposed PR body:** Tableau chemins retirés vs conservés ; lien **PR13** pour CBOM ; merger avant **11b**.
- **Risks:** Intégrateurs sur anciennes URLs — communication release (post-merge **#54**).
- **Completion criteria:** Aucun handler pour les routes retirées ; routes **conservées** toujours servies ; `go test` vert. **Réalisé** — [`cafe-discovery` PR #54](https://github.com/create2-labs/cafe-discovery/pull/54).

---

## PR11b — Retrait alias rollout CPM, edge, scripts et reliquats client

- **Branch:** `cleanup/remove-cpm-rollout-and-client-leftovers`
- **Repositories:** `cafe-crypto-policy-mgt`, `cafe-frontend`, `cafe-crypto-policy-mgt/scripts`, `cafe-deploy` (pas `cafe-discovery` — couvert par **11a**)
- **Objective:** Supprimer les chemins **§0.3** / double enregistrement sur le mux CPM, mettre à jour **nginx** / compose pour ne plus exposer les alias temporaires, aligner **scripts** et toute **référence résiduelle** frontend (grep, defaults, commentaires). **PR11a** mergée ([#54](https://github.com/create2-labs/cafe-discovery/pull/54)) : plus de routes Discovery legacy côté backend.
- **Scope:** `owner_routes.go`, `read_api.go`, `auth.go` (routes retirées), templates `cafe-deploy`, `apiCpmDataSource` / env / tests, scripts shell du dépôt CPM.
- **Out of scope:** Routes Discovery (déjà **11a**) ; **CBOM** et hydratation scan (**PR13**) ; factorisation des littéraux de chemins (**PR11c**) ; nouvelles fonctionnalités.
- **Dependencies:** **PR10** ; **PR11a** mergée ([#54](https://github.com/create2-labs/cafe-discovery/pull/54)).
- **Tests:** `go test` CPM ; `npm test` frontend ; `nginx -t` ou `compose config` si touché ; scripts mis à jour exécutés manuellement ou en CI.
- **Validation commands:** Par dépôt touché.
- **Proposed commit title:** `Remove CPM rollout aliases and post-migration client/deploy leftovers`
- **Proposed commit message:** `Drop §0.3-style CPM paths from mux and edge; clean scripts and frontend references; obsolete public contract no longer served.`
- **Proposed PR title:** `Cleanup: CPM rollout removal and client/deploy leftovers`
- **Proposed PR body:** Checklist alias retirés ; lien **11a** ; note opérateur pour l’ordre de déploiement.
- **Risks:** PR « filet » si **11b** inclut trop de changements non liés — garder le diff **strictement** rollout + reliquats ; tag ou versionnement des **deux** specs OpenAPI si besoin.
- **Completion criteria:** Plus d’alias rollout documentés comme actifs ; grep sans **`/api/v1/cpm/`** ni doubles handlers CPM cibles ; edge ne route plus vers chemins supprimés.

---

## PR11c — Factorisation des chemins API (par dépôt, sans lib inter-repo)

**Principe :** après **PR11b**, les préfixes publics sont stables (**`/api/cpm/v1`**, **`/discovery/v1`**, **`/api/discovery/v1`** à l’edge, etc.). Les littéraux restent **éparpillés** (mux, inventaire auth, tests, scripts shell, clients). **PR11c** centralise ces chaînes **dans chaque dépôt concerné** — **pas** de package ni de module partagé entre repos (contrats OpenAPI et déploiements restent découplés).

**Découpe Git :** une **PR Git par dépôt** (même nom de branche proposé partout pour la lisibilité) ; revue et merge **indépendants**. Aucune obligation de merger tous les dépôts le même jour : l’ordre n’affecte pas le comportement runtime tant que **11b** est déjà en place.

| Dépôt | Cibles typiques (exemples) | Package / module local (indicatif) |
|-------|---------------------------|-------------------------------------|
| **`cafe-crypto-policy-mgt`** | `read_api.go`, `owner_routes.go`, `auth.go` (`routeInventory`), `scan_id.go`, `*_test.go`, scripts `scripts/*.sh` | ex. `internal/cpmroutes` — constantes **`V1Base`**, **`PoliciesPrefix`**, routes catalogue / explore / drafts |
| **`cafe-discovery`** | `container.go`, enregistrement **`/discovery/v1`**, handlers v1, tests Fiber / HTTP | ex. `internal/discoveryroutes` — base **`/discovery/v1`**, segments wallets / tls / scan |
| **`cafe-deploy`** | `templates/nginx/nginx.conf.template`, `scripts/e2e-dev-stack.sh`, smokes | ex. variables shell ou fichier `scripts/lib/*-paths.sh` **dans ce repo uniquement** |
| **`cafe-frontend`** | `scanService.js`, `tlsService.js`, `walletService.js`, `apiCpmDataSource.ts`, `cpmDataSourceFactory.ts`, tests Vitest | ex. `src/api/routePaths.ts` (ou module CPM / Discovery séparés) — **`VITE_CPM_API_BASE_URL`** inchangé côté contrat |

- **Branch:** `refactor/centralize-api-route-paths` *(même nom dans chaque dépôt ; PR Git distincte)*
- **Repositories:** `cafe-crypto-policy-mgt`, `cafe-discovery`, `cafe-deploy`, `cafe-frontend` — **un merge par repo** ; pas de dépôt « meta »
- **Objective:** Réduire la duplication et l’écart **mux ↔ auth ↔ tests** en important des constantes locales ; **aucun** changement de contrat HTTP visible pour les clients après **11b**.
- **Scope:** Refactor Go/TS/JS/shell **dans le dépôt** ; tests existants passent avec les mêmes URLs effectives ; optionnel : test d’alignement « routes enregistrées ⊆ `routeInventory` » (CPM) ou équivalent Discovery.
- **Out of scope:** Modifier les chemins publics (déjà fixés en **11b**) ; OpenAPI (reste source de vérité documentaire — éventuel check CI **dans** le repo seulement) ; bibliothèque partagée inter-repo ; **CBOM** (**PR13**) ; fonctionnalités **PR7** / **PR13a**.
- **Dependencies:** **PR11b** mergée sur les dépôts où le cleanup rollout / chemins canoniques s’applique (au minimum **cafe-crypto-policy-mgt** + **cafe-deploy** ; **cafe-frontend** si **11b** n’y a laissé aucun reliquat, PR **11c** frontend peut être **réduite** ou **reportée**).
- **Tests:** `go test ./...` (CPM, Discovery) ; `npm test` (frontend) ; `nginx -t` / e2e deploy si **cafe-deploy** touché — **comportement identique** aux smokes **11b**.
- **Validation commands:** Par dépôt ; grep : littéraux de préfixe v1 concentrés dans le module de routes du repo (tolérer OpenAPI / README).
- **Proposed commit title:** `refactor: centralize API route path constants`
- **Proposed commit message:** `Introduce per-repo route path constants for mux, auth inventory, and tests. No HTTP contract change after PR11b. No cross-repo shared module.`
- **Proposed PR title:** `Refactor: centralize API route paths (per repository)`
- **Proposed PR body:** Liste des fichiers par dépôt ; rappel **pas de lib inter-repo** ; lien **PR11b** ; PRs sœurs dans les autres dépôts si applicable ; test d’alignement auth/mux si ajouté.
- **Risks:** Diff large mais mécanique — séparer **11b** et **11c** pour la revue ; oublier un littéral dans un script → grep de non-régression par dépôt.
- **Completion criteria:** Chemins canoniques définis **une fois par dépôt** ; mux et auth CPM alignés sur les mêmes constantes ; tests importent ces constantes ; **pas** de nouveau préfixe rollout ; contrat edge inchangé vs **11b**.

---

## PR13a — Parité UI du champ `result` v1 (Discovery)

- **Branch:** `discovery/v1-scan-result-ui-parity`
- **Repository:** `cafe-discovery`
- **Objective:** Enrichir **`result`** sur **`GET /discovery/v1/wallets/scans/{scan_id}`** et **`GET /discovery/v1/tls/scans/{scan_id}`** pour couvrir les champs consommés par l’UI aujourd’hui via CBOM (`risk_score`, `nist_level`, `algorithm`, `cipher_suites[]`, certificat, etc.) ; exposer le catalogue TLS partagé ; permettre la lecture d’un scan **default** par **`scan_id`**.
- **Scope:** Handlers v1 détail scan, mappers CBOM → DTO `result` ; **`GET /discovery/v1/tls/scans/defaults`** (enregistrer **avant** `/tls/scans/:scan_id`) ; `FindOwnedUserTLSScanByID` / lookup default pour détail ; `openapi/discovery-v1.yaml`.
- **Out of scope:** Retrait **`/discovery/cbom/*`** (**PR13c**) ; changements frontend (**PR13b**).
- **Dependencies:** **PR11a** ([#54](https://github.com/create2-labs/cafe-discovery/pull/54)) ; **PR4** (détail v1 existant).
- **Tests:** `go test ./...` ; tests contrat OpenAPI si présents ; curls détail wallet/TLS vs champs UI attendus.
- **Validation commands:** `curl` détail par `scan_id` (owner + default TLS) ; liste defaults vs liste owner disjointes.
- **Proposed commit title:** `Discovery v1: enrich scan result for UI parity and TLS defaults catalog`
- **Proposed commit message:** `Expand WalletScanResult/TlsScanResult per WORKPLAN_API. Add GET /discovery/v1/tls/scans/defaults. Allow default TLS scan detail by scan_id. Update discovery-v1.yaml.`
- **Proposed PR title:** `Discovery v1: scan result UI parity and TLS defaults endpoint`
- **Proposed PR body:** Tableau champs CBOM → `result` ; lien **PR13b** ; note OpenAPI.
- **Risks:** `result` plus large que le minimal spec — revue propriétaire ; versioning documenté si besoin.
- **Completion criteria:** Détail v1 suffisant pour rendre CBOM optionnel côté UI ; defaults TLS listables et lisibles par `scan_id` ; OpenAPI à jour.

---

## PR13b — Hydratation frontend par `scan_id` (sans CBOM)

- **Branch:** `frontend/v1-scan-detail-hydration`
- **Repository:** `cafe-frontend`
- **Objective:** Remplacer les appels **`GET /discovery/cbom/{address|url}`** par **`GET /discovery/v1/wallets|tls/scans/{scan_id}`** lors du rendu des cartes ; conserver listes synopsis v1 ; utiliser **`listDefaultScans()`** / defaults endpoint (**PR13a**).
- **Scope:** Stores/composables scan wallet et TLS, services, mappers synopsis → carte ; tests Vitest/Jest.
- **Out of scope:** Suppression route CBOM serveur (**PR13c**) ; enrichissement backend (**PR13a** fait en amont).
- **Dependencies:** **PR13a** déployée (ou stack locale avec `result` enrichi).
- **Tests:** `npm run test` ; smoke manuel listes + ouverture carte + defaults TLS.
- **Validation commands:** Réseau navigateur : plus de requêtes vers `/discovery/cbom/` en parcours nominal.
- **Proposed commit title:** `Frontend: hydrate scan cards from Discovery v1 detail by scan_id`
- **Proposed commit message:** `Remove CBOM client calls; map v1 scan detail result to UI card model. Use tls/scans/defaults for shared endpoints.`
- **Proposed PR title:** `Frontend: v1 scan detail hydration (CBOM removal prep)`
- **Proposed PR body:** Endpoints retirés vs nouveaux ; dépendance **PR13a** ; checklist régression cartes.
- **Risks:** Régression affichage si mapping incomplet ; N+1 détail — garder limite de concurrence existante.
- **Completion criteria:** Parcours wallet/TLS OK sans CBOM ; grep frontend sans `/discovery/cbom`.

---

## PR13c — Retrait de `GET /discovery/cbom/*`

- **Branch:** `discovery/remove-cbom-route`
- **Repositories:** `cafe-discovery` ; `cafe-deploy` (scripts e2e/smoke si références CBOM)
- **Objective:** Supprimer handlers, routes et tests **`GET /discovery/cbom/*`** une fois **PR13b** en production.
- **Scope:** `container.go`, handlers CBOM, tests ; `e2e-dev-stack.sh` / smokes : attendre détail v1 ou statut scan au lieu de `wait_for_cbom`.
- **Out of scope:** Utilities assessments/rpcs (**PR13d**) ; frontend (déjà sans CBOM en **13b**).
- **Dependencies:** **PR13b** mergée et déployée.
- **Tests:** `go test ./...` ; e2e deploy vert ; `curl` CBOM → **404**.
- **Validation commands:** Stack complète ; scripts deploy sans variable CBOM.
- **Proposed commit title:** `Remove Discovery CBOM HTTP routes after v1 detail migration`
- **Proposed commit message:** `Drop GET /discovery/cbom per WORKPLAN_API. Update deploy e2e to use v1 scan detail. Clients must use scan_id hydration only.`
- **Proposed PR title:** `Cleanup: remove Discovery CBOM routes`
- **Proposed PR body:** Note release intégrateurs ; lien **PR13a**/**PR13b** ; diff scripts.
- **Risks:** Intégrateurs externes encore sur CBOM — communication release.
- **Completion criteria:** Plus de route CBOM ; e2e vert ; OpenAPI sans CBOM (ou marqué removed).

---

## PR13d — Utilities et assessments hors v1 *(optionnel)*

- **Branch:** `discovery/utilities-and-assessments-v1`
- **Repository:** `cafe-discovery`
- **Objective:** Décider pour **`POST /discovery/assessments/request`**, **`GET /discovery/rpcs`**, **`GET /discovery/scanners`** : migrer sous **`/discovery/v1/...`**, documenter comme annexe hors contrat produit, ou retirer si inutilisés.
- **Scope:** Routes, handlers, OpenAPI annexe ou extension v1 ; clients/scripts si existants.
- **Out of scope:** CBOM (**PR13c**) ; CPM rollout (**11b**).
- **Dependencies:** **PR11b** ; **PR13c** recommandée (surface Discovery stabilisée).
- **Tests:** Selon périmètre retenu.
- **Proposed commit title:** `Discovery: align or document non-v1 utility routes`
- **Proposed PR title:** `Discovery: utilities and assessments route decision`
- **Risks:** Périmètre flou — garder PR petite ou reporter.
- **Completion criteria:** Chaque route hors v1 a un statut explicite (v1, annexe, ou removed).

---

## PR12 — Documentation et checklist QA

- **Branch:** `docs/api-coherency-runbook-qa`
- **Repository:** `cafe-documentation` (principal) ; README optionnels dans `cafe-discovery` / `cafe-crypto-policy-mgt`.
- **Objective:** Remplacer les exemples curl du guide développeur ; checklist QA type **§8** ; scripts alignés avec **11a**, **11b**, **13c** (plus de CBOM dans les exemples primaires).
- **Scope:** Runbooks et guides d’intégration ; **tous** les exemples curl Discovery sous **`/api/discovery/v1/...`** — **aucune** doc « primaire » sur CBOM ni anciennes listes.
- **Out of scope:** Copy marketing.
- **Dependencies:** **11a**, **11b**, **13c** terminées (ou équivalent : plus de CBOM servi, clients sur détail v1).
- **Tests:** N/A (markdown) ; lien HTTP optionnel en CI si déjà présent.
- **Validation commands:** Revue manuelle ; `markdownlint` seulement si déjà dans le repo.
- **Proposed commit title:** `Docs: CAFE API v1 coherency runbook and QA checklist`
- **Proposed commit message:** `Update developer guide and runbooks for /api/discovery/v1 and /api/cpm/v1; add sign-off checklist referencing WORKPLAN_API §8.`
- **Proposed PR title:** `Documentation: API coherency runbook and QA`
- **Proposed PR body:** Liens vers **`cafe-discovery/openapi/discovery-v1.yaml`** et **`cafe-crypto-policy-mgt/openapi/cpm-v1.yaml`**, smokes curls, ordre de release.
- **Risks:** Doc obsolète si merge tardif après **11a**/**11b** — merger rapidement.
- **Completion criteria:** Plus de références « primaires » aux chemins supprimés.

---

## Rappel global (git et implémentation)

- **Pas de commit, push, merge ou tags** depuis le plan automatisé : le propriétaire humain garde la main sur git et la publication.
- **Jalon OpenAPI** — **PR1a** [#49](https://github.com/create2-labs/cafe-discovery/pull/49), **PR1b** [#26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). Chaîne **PR2→PR6** : [#51](https://github.com/create2-labs/cafe-discovery/pull/51), [#52](https://github.com/create2-labs/cafe-discovery/pull/52), CPM [#27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27), [#53](https://github.com/create2-labs/cafe-discovery/pull/53). **PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR9** [`cafe-deploy` #11](https://github.com/create2-labs/cafe-deploy/pull/11). **PR10** [`cafe-frontend` #52](https://github.com/create2-labs/cafe-frontend/pull/52). **PR11a** [`cafe-discovery` #54](https://github.com/create2-labs/cafe-discovery/pull/54). **Suite recommandée :** **PR7** → **PR11b** → **PR11c** *(PR Git par dépôt)* → **PR13a** → **PR13b** → **PR13c** → **PR12** (option **PR13d**). Détail par chapitre ci‑dessus.

### Instruction type — après jalon OpenAPI (PR #49 + #26)

Lorsque vous ouvrez une PR d’implémentation (**PR2+**), lier la **PR Git** dans la colonne **PR Git** du tableau de suivi et pointer vers les specs sur `main` :

> OpenAPI jalon merged (Discovery PR #49, CPM PR #26). Chaîne **PR2→PR6** documentée avec liens Git (ex. **PR6** → Discovery **#53** ; **PR2**/**PR3** → **#51** ; **PR4** → **#52** ; **PR5** → CPM **#27** ; **#50** fermée sans merge). **PR8** → CPM [**#29**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR9** → **`cafe-deploy` [#11](https://github.com/create2-labs/cafe-deploy/pull/11)**. **PR10** → **`cafe-frontend` [#52](https://github.com/create2-labs/cafe-frontend/pull/52)**. **PR11a** → **`cafe-discovery` [#54](https://github.com/create2-labs/cafe-discovery/pull/54)**. Après **PR10**, la dette **CBOM** est suivie par **PR13a→PR13c** (ne pas la traiter dans **PR11b**). **PR11c** : factorisation des chemins **par dépôt** après **11b**, sans lib inter-repo. Mettre à jour le **résumé exécutif** lorsque le comportement livré diverge. Poursuivre selon **approbation propriétaire** et **dépendances**.

### Fusion optionnelle de PRs (si vous voulez réduire le nombre)

Les regroupements les moins risqués : **3 + 4** (POST scan + GET historique) ou **5 + 7** (référence interne + surface publique CPM) — à décider avant le premier merge si vous souhaitez une table raccourcie. **Ne pas** regrouper : **11a** et **11b** ; **11b** et **11c** ; **13a** et **13b** ; **13b** et **13c** (le client doit basculer avant de retirer CBOM côté serveur).
