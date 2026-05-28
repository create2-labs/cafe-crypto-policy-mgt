# CAFE API coherency — PR plan

**Référence canonique des contrats et sémantiques :** [`WORKPLAN_API.md`](./WORKPLAN_API.md).

**Règles d’exécution (propriétaire humain) :** l’agent / les contributeurs ne font **pas** de commit, push, merge ni tags ; revue, git et publication restent manuelles. Chaque PR : branche locale, changements ciblés, tests, puis proposition de titre/message de commit et de PR (en anglais dans les sections dédiées ci‑dessous).

**Statut du document :** plan de découpe ; jalon OpenAPI **mergé** — **PR1a** [`cafe-discovery` PR #49](https://github.com/create2-labs/cafe-discovery/pull/49), **PR1b** [`cafe-crypto-policy-mgt` PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). **Chaîne livrée sur `main` (Discovery + CPM, prérequis PR6)** — **PR2** absorbé dans **`cafe-discovery` [#51](https://github.com/create2-labs/cafe-discovery/pull/51)** (PR dédiée **[#50](https://github.com/create2-labs/cafe-discovery/pull/50)** fermée sans merge) ; **PR3** [#51](https://github.com/create2-labs/cafe-discovery/pull/51) ; **PR4** [#52](https://github.com/create2-labs/cafe-discovery/pull/52) ; **PR5** [`cafe-crypto-policy-mgt` #27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27) ; **PR6** [#53](https://github.com/create2-labs/cafe-discovery/pull/53) ; **PR7** (contrat public CPM policies) — [`cafe-crypto-policy-mgt` #28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28). **PR8** [`cafe-crypto-policy-mgt` #29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR9** (edge) — [`cafe-deploy` #11](https://github.com/create2-labs/cafe-deploy/pull/11). **PR10** (frontend) — [`cafe-frontend` #52](https://github.com/create2-labs/cafe-frontend/pull/52). **PR11a** (cleanup routes Discovery legacy) — [`cafe-discovery` #54](https://github.com/create2-labs/cafe-discovery/pull/54). **PR11b** (cleanup rollout CPM / edge) — CPM [#30](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30), deploy [#12](https://github.com/create2-labs/cafe-deploy/pull/12). **PR11c** (factorisation chemins API par dépôt) — CPM [#31](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/31), Discovery [#55](https://github.com/create2-labs/cafe-discovery/pull/55), deploy [#13](https://github.com/create2-labs/cafe-deploy/pull/13), frontend [#54](https://github.com/create2-labs/cafe-frontend/pull/54). **PR13a** (parité UI `result` v1 + defaults TLS) — [`cafe-discovery` #56](https://github.com/create2-labs/cafe-discovery/pull/56). **PR13c** (retrait CBOM serveur) — [`cafe-discovery` #57](https://github.com/create2-labs/cafe-discovery/pull/57). **PR13d** (utilities v1 + retrait assessment Discovery) — [`cafe-discovery` #58](https://github.com/create2-labs/cafe-discovery/pull/58). **PR13b** (hydratation cartes v1, sans CBOM dans `src/`) — réalisé sur `cafe-frontend` `main`. **PR13e** (client utilities v1 : rpcs/scanners) — [`cafe-frontend` #56](https://github.com/create2-labs/cafe-frontend/pull/56). **PR13f** (e2e sans assessment Discovery) — [`cafe-deploy` #15](https://github.com/create2-labs/cafe-deploy/pull/15). **PR13g** (assessment HTTP CPM, wallet-only) — [`cafe-crypto-policy-mgt` #33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33). **PR13h** (e2e assessment CPM) — [`cafe-deploy` #16](https://github.com/create2-labs/cafe-deploy/pull/16). Suite : doc (**PR12**) ; reliquats **`cafe.sh`/`README`** CBOM (suivi doc/CLI).

---

## Executive summary (état du dépôt au moment de la rédaction)

| Domaine | État observé dans le repo | Écart vs `WORKPLAN_API.md` |
|--------|---------------------------|----------------------------|
| **Chemins publics Discovery** | **`/discovery/v1`** sur `main` ; **`GET …/v1/rpcs`**, **`GET …/v1/scanners`** (public, sans JWT) — **PR13d** [#58](https://github.com/create2-labs/cafe-discovery/pull/58). **PR13e** — **`cafe-frontend` [#56](https://github.com/create2-labs/cafe-frontend/pull/56)** : edge **`/api/discovery/v1/rpcs`**, **`/api/discovery/v1/scanners`**. | **`/plans`** hors v1 (exemption **PR13d**) ; voir chapitre **PR13d**. |
| **Hydratation UI (CBOM)** | **PR13a** [#56](https://github.com/create2-labs/cafe-discovery/pull/56) : détail v1 **`result`** enrichi. **PR13b** : l’app (`cafe-frontend` `src/`) hydrate les cartes via **`GET …/wallets|tls/scans/{scan_id}`** — plus d’appel runtime **CBOM historique** en parcours nominal. **PR13c** [#57](https://github.com/create2-labs/cafe-discovery/pull/57) : route CBOM retirée côté serveur. | Reliquats **scripts** : `cafe-frontend/scripts/cafe.sh` (`--cboms`) et **README** obsolète — suivi doc/CLI (non bloquant UI ; **bloquant** si on utilise `cafe.sh --cboms`). |
| **Assessment policy (HTTP)** | Discovery **ne publie plus** assessment HTTP (**PR13d** [#58](https://github.com/create2-labs/cafe-discovery/pull/58)). CPM expose le déclencheur HTTP wallet-only (**PR13g** [`cafe-crypto-policy-mgt` #33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33)) et consomme **`policy.assessment.requested.v0.1`** (`assessment_consumer.go`). **PR13h** [`cafe-deploy` #16](https://github.com/create2-labs/cafe-deploy/pull/16) ajoute le smoke e2e via CPM. | **Décision livrée :** déclencheur HTTP = **CPM** : **`POST /api/cpm/v1/policies/assessment/request`**. TLS = Discovery uniquement (pas de policy assessment CPM). |
| **Listes de scans** | **PR4** — listes owner wallet/TLS. **`GET /discovery/v1/tls/scans/defaults`** + détail default par **`scan_id`** — **PR13a** [#56](https://github.com/create2-labs/cafe-discovery/pull/56). OpenAPI aligné (`discovery-v1.yaml`). | — |
| **POST scan** | **PR3** — **`POST /discovery/v1/scan`** : **`scan_id`**, **`requested`**, **`location`**. Ancien **`POST /discovery/v1/scan`** retiré — **PR11a** [#54](https://github.com/create2-labs/cafe-discovery/pull/54). | — |
| **DELETE scans / 409 wallet** | **PR6** — DELETE scans v1 + vérif CPM (**PR5** / **`SCAN_REFERENCED_BY_POLICY`**). | **409 `WALLET_REFERENCED_BY_POLICY`** sur DELETE **wallet** (Discovery — hors périmètre CPM **PR7**). |
| **façade policy-context (retirée)** | Retiré — **PR11a** [#54](https://github.com/create2-labs/cafe-discovery/pull/54) (remplacé par **`wallets/scans`** + CPM **`GET …/policies?scan_id=`** — **PR7** [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28)). | — |
| **CPM** | **PR5** [#27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27) interne ; **PR7** [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28) policies ; **PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29) explore ; **PR11b**/**PR11c**. **PR13g** [#33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33) : **`POST …/policies/assessment/request`** wallet-only → **202** + NATS. | — |
| **OpenAPI** | **PR1a** + **PR1b** mergées : `openapi/discovery-v1.yaml` ([`cafe-discovery` #49](https://github.com/create2-labs/cafe-discovery/pull/49)), `openapi/cpm-v1.yaml` ([`cafe-crypto-policy-mgt` #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26)). Edge documentée en **PR9** — **`cafe-deploy` [#11](https://github.com/create2-labs/cafe-deploy/pull/11)**. | **Deux** artefacts de contrat distincts (§0.1 + §0.2) ; l’edge se décrit via **`servers`** dans chaque spec. |
| **Frontend** | **PR10** [#52](https://github.com/create2-labs/cafe-frontend/pull/52) ; **PR11c** [#54](https://github.com/create2-labs/cafe-frontend/pull/54) — `src/api/routePaths.ts`, services Discovery/CPM. **PR13b** : hydratation cartes par détail v1 **`scan_id`** (`scanService.js`, `tlsService.js`) — **pas** d’appel **CBOM historique** dans `src/`. **PR13e** [#56](https://github.com/create2-labs/cafe-frontend/pull/56) : utilities **`/api/discovery/v1/rpcs`**, **`/scanners`**. | Mettre à jour **`scripts/cafe.sh`** et **README** (encore CBOM) ; voir ligne Hydratation UI. |

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
| **7** | `cpm/policies-scan-reference-contract` | `cafe-crypto-policy-mgt` | [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28) | **1b**, **4**, **5** | Liste **`scan_id`** en O(n) en mémoire ; plan DB ; même lookup que **PR5**. | **`/api/cpm/v1/policies`**, **`GET ?scan_id=`** (même lookup owner-scoped que **PR5**), **`DELETE ?id=`**, validations. **Réalisé.** |
| **8** | `cpm/decisions-explore-contract-check` | `cafe-crypto-policy-mgt` | [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29) | **1b**, **7** | Validation trop stricte → démos ; garder voie « fixtures » (**PR7**). | **`POST …/policies/decisions/explore`** : non persistant ; DTOs alignés scan v1. |
| **9** | `deploy/api-v1-edge-alignment` | `cafe-deploy` | [#11](https://github.com/create2-labs/cafe-deploy/pull/11) | **2**, **7** | Coupure si edge avant images ; **ne pas** réintroduire de **`/api/wallets`** (hors **`/api/discovery/v1/wallets`**). | NGINX / env : **`/api/discovery/v1`**, **`/api/cpm/v1`** ; chemins §0.3 **temporaires** documentés. |
| **10** | `frontend/api-coherency-migration` | `cafe-frontend` | [#52](https://github.com/create2-labs/cafe-frontend/pull/52) | **9** (ou stack locale équivalente) | Bundles / **SW** staging ; procédure de purge ; **CRUD wallets** : base **`/api/discovery/v1/wallets`**. | Clients vers nouveaux chemins et enveloppes. |
| **11a** | `cleanup/remove-obsolete-discovery-routes` | `cafe-discovery` | [#54](https://github.com/create2-labs/cafe-discovery/pull/54) | **10** | Intégrateurs sur anciennes URLs — note release ; **11b** peut suivre. | Retrait listes/scan/context legacy ; **conserver** routes hors v1 (CBOM, utilities, assessments). **Réalisé.** |
| **11b** | `cleanup/remove-cpm-rollout-and-client-leftovers` | `cafe-crypto-policy-mgt`, `cafe-crypto-policy-mgt/scripts`, `cafe-deploy` *(pas de PR Git `cafe-frontend` dédiée — client déjà sur v1 ; **11c** [#54](https://github.com/create2-labs/cafe-frontend/pull/54) pour les chemins)* | CPM [#30](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30), deploy [#12](https://github.com/create2-labs/cafe-deploy/pull/12) | **10**, **11a**, **7** | Déployer CPM + nginx ensemble. | Alias **§0.3** CPM / nginx retirés ; scripts CPM + e2e sur **`/api/cpm/v1`**. **Réalisé** (CPM + deploy). |
| **11c** | `refactor/centralize-api-route-paths` | `cafe-crypto-policy-mgt`, `cafe-discovery`, `cafe-deploy`, `cafe-frontend` *(PR Git **par dépôt**, pas de lib partagée inter-repo)* | CPM [#31](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/31), Discovery [#55](https://github.com/create2-labs/cafe-discovery/pull/55), deploy [#13](https://github.com/create2-labs/cafe-deploy/pull/13), frontend [#54](https://github.com/create2-labs/cafe-frontend/pull/54) | **11b** (chemins canoniques figés) | Refactor pure ; risque dérive auth ↔ mux si test d’alignement absent. | Constantes / helpers **locaux** : mux, auth, tests, scripts, clients — **sans** mutualisation inter-dépôt. **Réalisé.** |
| **13a** | `discovery/v1-scan-result-ui-parity` | `cafe-discovery` | [#56](https://github.com/create2-labs/cafe-discovery/pull/56) | **11a** | `result` plus large que le minimal spec — revue OpenAPI ; defaults TLS lecture par `scan_id`. | Enrichir **`result`** v1 + **`GET …/tls/scans/defaults`** + détail default TLS ; OpenAPI aligné. **Réalisé.** |
| **13b** | `frontend/v1-scan-detail-hydration` | `cafe-frontend` | *(sur `main` ; commit `feat(frontend): hydrate scan cards…`)* | **13a** | Régression cartes ; N× détail. | Hydrater cartes via détail v1 ; retirer CBOM client **`src/`**. **Réalisé** (distinct de **13e**). |
| **13c** | `discovery/remove-cbom-route` | `cafe-discovery`, `cafe-deploy` (scripts) | [#57](https://github.com/create2-labs/cafe-discovery/pull/57) | **13b** (`src/` sans CBOM) | Note release ; **`cafe.sh --cboms`** cassé tant que scripts non migrés. | Retirer **CBOM historique**. **Réalisé** (Discovery). |
| **13d** | `discovery/utilities-and-assessments-v1` | `cafe-discovery` | [#58](https://github.com/create2-labs/cafe-discovery/pull/58) | **11b**, **13c** | Assessment = CPM (**13g**) ; ne pas regrouper **13d**+**13g** ; prioriser **13f** si CI/smoke. | Finaliser utilities v1 (**rpcs**, **scanners**) ; retirer assessment Discovery. **Réalisé.** |
| **13e** | `frontend/discovery-v1-utility-routes` | `cafe-frontend` | [#56](https://github.com/create2-labs/cafe-frontend/pull/56) | **13d** | Edge **`/api/discovery/v1/rpcs`** et **`/scanners`** (pas chemins backend nus). | Client utilities : **`scanService.js`**, **`cafe.sh`**, **`routePaths`**. **Réalisé.** |
| **13f** | `deploy/e2e-remove-discovery-assessment` | `cafe-deploy` | [#15](https://github.com/create2-labs/cafe-deploy/pull/15) | **13d** | Gap e2e assessment fermé par **13h** [#16](https://github.com/create2-labs/cafe-deploy/pull/16). | Retirer **`publish_assessment_request`** Discovery de **`e2e-dev-stack.sh`**. **Réalisé.** |
| **13g** | `cpm/policies-assessment-request` | `cafe-crypto-policy-mgt` | [#33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33) | **13d**, **4** | Latence Discovery synchrone avant **202**. | **`POST …/policies/assessment/request`** → **202** ; wallet-only ; OpenAPI + tests. **Réalisé.** |
| **13h** | `deploy/e2e-cpm-assessment-request` | `cafe-deploy` | [#16](https://github.com/create2-labs/cafe-deploy/pull/16) | **13g**, **13f** | — | Smoke e2e assessment via endpoint CPM. **Réalisé.** |
| **12** | `docs/api-coherency-runbook-qa` | `cafe-documentation` (+ README optionnels) | — | **13d**, **13e**, **13f** ; **13g** si doc assessment | Doc obsolète si merge avant **13d**. | Runbooks **v1** ; assessment = CPM ; plus de **`assessment HTTP historique Discovery`**. |

**Colonne PR Git :** lien vers la pull request du dépôt concerné lorsqu’elle existe ; **—** = pas encore créée / à renseigner. *Remarque : une PR plan peut être livrée dans une PR Git unique (ex. **PR2** + **PR3** → **`cafe-discovery` #51**) ; une PR Git peut être **fermée sans merge** si le périmètre a été réintégré ailleurs (ex. **#50** → **#51**).*

**Colonne Risques / suites (résumé) :** synthèse pour lecture rapide ; le détail (**Risks**, **Completion criteria**, dépendances, périmètres) reste dans le **chapitre de chaque PR** ci‑dessous.

**Découpe PR5 + PR6 (propriété et autorité) :** **Discovery** possède les **scans** (cycle de vie, stockage, suppression physique). **CPM** possède les **policies** et est **autoritaire** sur la question « un **`scan_id`** est-il référencé par des instances persistées pour ce **owner** ? ». **Discovery** ne connaît pas la structure interne des policies, **n’inspecte pas** la DB CPM, **ne duplique pas** d’index `scan_id → policies`, **ne décide pas** seul du 409, **ne supprime jamais** de policies en cascade. **Discovery** appelle **CPM** (PR **5**), **consomme uniquement** le verdict (`referenced: true|false`), puis applique **409** ou poursuit le **DELETE** (PR **6**). L’UI peut lister les policies à détacher via l’API publique **`GET /api/cpm/v1/policies?scan_id=...`** (**PR7**), distincte de l’endpoint interne.

---

## Jalon OpenAPI — PR Git **#49** (Discovery) + **#26** (CPM)

**Principe de dépôt :** le contrat Discovery est **possédé** par `cafe-discovery` ; le contrat CPM par `cafe-crypto-policy-mgt`. Aucun dépôt ne concentre l’OpenAPI de l’autre — évite l’ambiguïté « le CPM possède aussi Discovery ».

**Coordination :** **PR1a** et **PR1b** sont **mergées** — [`cafe-discovery` #49](https://github.com/create2-labs/cafe-discovery/pull/49), [`cafe-crypto-policy-mgt` #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). **Chaîne implémentation Discovery/CPM (PR plan 2→6) sur `main` :** [#51](https://github.com/create2-labs/cafe-discovery/pull/51) (PR2+PR3), [#52](https://github.com/create2-labs/cafe-discovery/pull/52) (PR4), [`cafe-crypto-policy-mgt` #27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27) (PR5), [#53](https://github.com/create2-labs/cafe-discovery/pull/53) (PR6). **PR7** livrée [`cafe-crypto-policy-mgt` #28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28). **PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR11b** CPM [#30](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30), deploy [#12](https://github.com/create2-labs/cafe-deploy/pull/12). **PR11c** — voir chapitre **PR11c**.

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
- **Out of scope:** Logique complète liste/détail, DELETE, modification des chemins CBOM, suppression de l’ancien la liste wallet historique.
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
- **Out of scope:** DTOs liste/détail (**PR4**), retrait de l’ancien `POST /discovery/v1/scan` (**PR11a**).
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

**Merge :** livré dans **`cafe-crypto-policy-mgt`** via [**PR #28**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28) (`feat(cpm): add v1 policies API and scan_id contract`).

- **Branch:** `cpm/policies-scan-reference-contract`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Implémenter **§0.2** sur le mux (alias **§0.3** optionnels marqués dépréciés) ; **`GET /policies?id=`** vs **`GET /policies?scan_id=`** mutuellement exclusifs → **400** ; **`DELETE /policies?id=`** **204/404 uniquement** ; **`POST /policies`** rejette **`scan_id`** manquant pour le flux Discovery → CPM (exceptions documentées : brouillons, fixtures, tests hors produit).
- **Scope:** `internal/app/owner_routes.go`, `internal/persistence/owner_scoped_store.go` (index / scan par **`scan_id`**), `internal/app/auth.go`, enregistrement **`read_api.go`** pour catalog sous **`/api/cpm/v1/policies/...`** (enregistrement double alias de transition jusqu’à **11b** si besoin de coupure).
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
- **Completion criteria:** Contrat public aligné avec **`openapi/cpm-v1.yaml`** (mises à jour si besoin) ; table d’auth à jour. **Réalisé** — [PR #28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28) ; alias rollout retirés ensuite en **PR11b** [#30](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30).

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
- **Implementation notes:** **Résoudre l’incohérence** frontend **`/api/cpm/...`** vs mux direct **les alias CPM de transition** : une seule histoire pour navigateur et scripts après cette PR. **Ne pas** exposer d’alias edge **`/api/wallets`** (ou équivalent court) vers Discovery : les wallets CAFE passent par **`/api/discovery/v1/wallets`** après strip **`/api`**.
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
- **Out of scope:** Suppression backend des routes hors contrat §0 — **réalisée** en **PR11a** ([#54](https://github.com/create2-labs/cafe-discovery/pull/54)) ; suppression **CBOM historique** (**PR13c**) ; enrichissement **`result`** v1 — **réalisé** **PR13a** ([#56](https://github.com/create2-labs/cafe-discovery/pull/56)) ; hydratation détail (**PR13b**) ; nettoyage rollout CPM (**11b**).
- **Dependencies:** **PR9** (ou overrides locaux coordonnés).
- **Tests:** Mise à jour Vitest/Jest pour builders d’URL et parsers ; `apiCpmDataSource.spec.ts`.
- **Validation commands:** `cd cafe-frontend && npm run test` (ou script CI du projet) ; `npm run build` si applicable.
- **Proposed commit title:** `Frontend: migrate to Discovery v1 and CPM v1 APIs`
- **Proposed commit message:** `Update REST clients for new paths, pagination envelopes, and POST /scan correlation fields per WORKPLAN_API.`
- **Proposed PR title:** `Frontend: API coherency migration for Discovery and CPM`
- **Proposed PR body:** Liste des endpoints remplacés ; note feature-flag si applicable.
- **Risks:** Anciens bundles / SW en staging — procédure de purge.
- **Completion criteria:** Parcours scan + listes v1 + appels CPM catalogue/explore via edge OK contre la stack **9**. **Réalisé** — [`cafe-frontend` PR #52](https://github.com/create2-labs/cafe-frontend/pull/52). *(Hydratation cartes par détail v1 : **PR13b** sur `main` ; defaults TLS : **PR13a** + **PR13b**.)*

---

## Routes Discovery hors v1 après PR11a (conservées volontairement)

**PR11a** ([#54](https://github.com/create2-labs/cafe-discovery/pull/54), mergée) a retiré les routes **remplacées par v1** (listes par adresse/URL, ancien POST scan, façade policy-context (retirée)). Les chemins ci‑dessous **restent montés** — ce n’est pas un oubli ; la fermeture est planifiée par phase.

| Route | Auth | Rôle actuel | Sortie planifiée |
|-------|------|-------------|------------------|
| **`/discovery/v1/*`** | Bearer owner | Contrat canonique (listes, détail, POST scan, wallets CAFE, DELETE) | **Garder** |
| **CBOM historique** | Bearer owner | *(retiré)* | **PR13c** [#57](https://github.com/create2-labs/cafe-discovery/pull/57) — **Réalisé** |
| **l'ancien déclencheur assessment Discovery** | Bearer owner | *(retiré)* | Remplacement **PR13g** : **`POST /api/cpm/v1/policies/assessment/request`** |
| **`GET /discovery/v1/rpcs`**, **`GET /discovery/v1/scanners`** | Public | *(retirés)* | **PR13d** [#58](https://github.com/create2-labs/cafe-discovery/pull/58) → **`GET …/v1/rpcs`**, **`GET …/v1/scanners`** |

**Détail scan (post-PR13a) :** `GET …/wallets|tls/scans/{scan_id}` renvoie un **`result` enrichi** (parité champs UI / CBOM) — livré **PR13a** [#56](https://github.com/create2-labs/cafe-discovery/pull/56). **PR13b** : l’app frontend n’appelle plus CBOM en parcours nominal (**`src/`** vérifié). **PR13c** : route serveur retirée. Reliquats **`cafe-frontend/scripts/cafe.sh`** et README — suivi hors périmètre **13e**.

**Defaults TLS :** `GET /discovery/v1/tls/scans` liste **uniquement** les scans **owner** (`default=false`). Catalogue partagé : **`GET /discovery/v1/tls/scans/defaults`** ; détail catalogue par **`scan_id`** (y compris `is_default`) — **PR13a** [#56](https://github.com/create2-labs/cafe-discovery/pull/56). Le frontend ne doit plus déduire les defaults depuis la liste owner + flag `default` (comportement **PR10** obsolète ; **PR13b** pour `listDefaultScans()` côté client).

---

## Chaîne PR13 — Fermeture dette CBOM

```text
PR13a (Discovery: result enrichi + defaults TLS) — mergé [#56]
  → PR13b (Frontend: hydratation par scan_id, plus de CBOM dans src/) — réalisé sur main
    → PR13c (Discovery: retirer CBOM historique) — mergé [#57]
      → PR13d (Discovery: rpcs/scanners v1, retirer assessment Discovery) — mergé [#58]
        → PR13e (Frontend: utilities rpcs/scanners v1) — mergé [#56 cafe-frontend]
        → PR13f (Deploy: e2e sans l'ancien déclencheur assessment Discovery) — mergé [#15]
          → PR13g (CPM: POST /api/cpm/v1/policies/assessment/request) — mergé [#33]
            → PR13h (Deploy: e2e smoke assessment CPM) — mergé [#16]
      → PR12 (Doc: runbooks v1, assessment CPM)
```

**Règles :** ne pas regrouper **13a+13b** ni **13b+13c** (app **`src/`** avant retrait CBOM serveur — respecté) ; **13d** et **13g** (Discovery cleanup vs CPM assessment). **13e** ≠ **13b** (utilities vs hydratation cartes).

**Reliquats CBOM (hors app) :** `cafe-frontend/scripts/cafe.sh` (`--cboms`, parcours obsolète) ; **README** frontend — à traiter en suivi doc/CLI (**PR12** ou PR scripts dédiée), pas comme dette **PR13b** runtime UI.

---

## PR11a — Suppression des routes Discovery obsolètes

**Merge :** livré dans **`cafe-discovery`** via [**PR #54**](https://github.com/create2-labs/cafe-discovery/pull/54) (branche `cleanup/remove-obsolete-discovery-routes`).

- **Branch:** `cleanup/remove-obsolete-discovery-routes`
- **Repository:** `cafe-discovery` uniquement
- **Objective:** Retirer les handlers, routes Fiber et tests associés aux chemins **remplacés par v1** une fois **PR10** déployée : **la liste wallet historique**, **la liste TLS historique**, **la façade policy-context historique**, **`POST /discovery/v1/scan`**, et code mort associé.
- **Scope:** `internal/app/container.go`, `discovery.go`, `tls.go`, tests ; **ne pas** retirer **CBOM historique**, **l'ancien déclencheur assessment Discovery**, **`GET /discovery/v1/rpcs`**, **`GET /discovery/v1/scanners`** (voir tableau ci‑dessus).
- **Out of scope:** **CBOM historique** (**PR13c**) ; enrichissement **`result`** v1 — **réalisé** **PR13a** ([#56](https://github.com/create2-labs/cafe-discovery/pull/56)) ; frontend ; nginx ; CPM.
- **Dependencies:** **PR10** (clients sur v1) ; **9** déployé sur les cibles où le cleanup s’applique.
- **Tests:** `cd cafe-discovery && go test ./...` ; grep du repo sans références aux anciens chemins enregistrés.
- **Validation commands:** Idem ; smoke `curl` direct sur backend si utile.
- **Proposed commit title:** `Remove obsolete Discovery HTTP routes after v1 migration`
- **Proposed commit message:** `Drop legacy discovery list/context/scan routes per WORKPLAN_API. Keep CBOM historique assessments, rpcs, and scanners until PR13. v1 remains the product contract for lists and scan trigger.`
- **Proposed PR title:** `Cleanup: remove obsolete Discovery API routes`
- **Proposed PR body:** Tableau chemins retirés vs conservés ; lien **PR13** pour CBOM ; merger avant **11b**.
- **Risks:** Intégrateurs sur anciennes URLs — communication release (post-merge **#54**).
- **Completion criteria:** Aucun handler pour les routes retirées ; routes **conservées** toujours servies ; `go test` vert. **Réalisé** — [`cafe-discovery` PR #54](https://github.com/create2-labs/cafe-discovery/pull/54).

---

## PR11b — Retrait alias rollout CPM, edge, scripts et reliquats client

**Merge :** CPM [#30](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30), deploy [#12](https://github.com/create2-labs/cafe-deploy/pull/12). Pas de PR Git **`cafe-frontend`** dédiée (client déjà sur **`/api/cpm/v1`** en **PR10** ; chemins centralisés en **PR11c** [#54](https://github.com/create2-labs/cafe-frontend/pull/54)).

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
- **Completion criteria:** Plus d’alias rollout documentés comme actifs ; grep sans **`alias CPM de transition `** ni doubles handlers CPM cibles ; edge ne route plus vers chemins supprimés. **Réalisé** — CPM [#30](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30), deploy [#12](https://github.com/create2-labs/cafe-deploy/pull/12).

---

## PR11c — Factorisation des chemins API (par dépôt, sans lib inter-repo)

**Merge :** livré **CPM** via [**PR #31**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/31) (`internal/cpmroutes`), **Discovery** via [**PR #55**](https://github.com/create2-labs/cafe-discovery/pull/55) (`internal/discoveryroutes`), **deploy** via [**PR #13**](https://github.com/create2-labs/cafe-deploy/pull/13) (`scripts/lib/api-route-paths.sh`), **frontend** via [**PR #54**](https://github.com/create2-labs/cafe-frontend/pull/54) (`src/api/routePaths.ts`).

**Principe :** après **PR11b**, les préfixes publics sont stables (**`/api/cpm/v1`**, **`/discovery/v1`**, **`/api/discovery/v1`** à l’edge, etc.). Les littéraux étaient **éparpillés** (mux, inventaire auth, tests, scripts shell, clients). **PR11c** centralise ces chaînes **dans chaque dépôt concerné** — **pas** de package ni de module partagé entre repos (contrats OpenAPI et déploiements restent découplés).

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
- **Completion criteria:** Chemins canoniques définis **une fois par dépôt** ; mux et auth CPM alignés sur les mêmes constantes ; tests importent ces constantes ; **pas** de nouveau préfixe rollout ; contrat edge inchangé vs **11b**. **Réalisé** — CPM [#31](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/31), Discovery [#55](https://github.com/create2-labs/cafe-discovery/pull/55), deploy [#13](https://github.com/create2-labs/cafe-deploy/pull/13), frontend [#54](https://github.com/create2-labs/cafe-frontend/pull/54).

---

## PR13a — Parité UI du champ `result` v1 (Discovery)

**Merge :** livré dans **`cafe-discovery`** via [**PR #56**](https://github.com/create2-labs/cafe-discovery/pull/56) (branche `discovery/v1-scan-result-ui-parity`).

- **Branch:** `discovery/v1-scan-result-ui-parity`
- **Repository:** `cafe-discovery`
- **Objective:** Enrichir **`result`** sur **`GET /discovery/v1/wallets/scans/{scan_id}`** et **`GET /discovery/v1/tls/scans/{scan_id}`** pour couvrir les champs consommés par l’UI aujourd’hui via CBOM (`risk_score`, `nist_level`, `algorithm`, `cipher_suites[]`, certificat, etc.) ; exposer le catalogue TLS partagé ; permettre la lecture d’un scan **default** par **`scan_id`**.
- **Scope:** Handlers v1 détail scan, mappers CBOM → DTO `result` ; **`GET /discovery/v1/tls/scans/defaults`** (enregistrer **avant** `/tls/scans/:scan_id`) ; `FindOwnedUserTLSScanByID` / lookup default pour détail ; `openapi/discovery-v1.yaml`.
- **Out of scope:** Retrait **CBOM historique** (**PR13c**) ; changements frontend (**PR13b**).
- **Dependencies:** **PR11a** ([#54](https://github.com/create2-labs/cafe-discovery/pull/54)) ; **PR4** (détail v1 existant).
- **Tests:** `go test ./...` ; tests contrat OpenAPI si présents ; curls détail wallet/TLS vs champs UI attendus.
- **Validation commands:** `curl` détail par `scan_id` (owner + default TLS) ; liste defaults vs liste owner disjointes.
- **Proposed commit title:** `feat(discovery): enrich v1 scan result for UI parity and TLS defaults`
- **Proposed commit message:** `Expand WalletScanResult/TlsScanResult per WORKPLAN_API. Add GET /discovery/v1/tls/scans/defaults. Allow default TLS scan detail by scan_id. Update discovery-v1.yaml.`
- **Proposed PR title:** `feat(discovery): v1 scan result UI parity and TLS defaults catalog`
- **Proposed PR body:** Tableau champs CBOM → `result` ; lien **PR13b** ; note OpenAPI.
- **Risks:** `result` plus large que le minimal spec — revue propriétaire ; versioning documenté si besoin.
- **Completion criteria:** Détail v1 suffisant pour rendre CBOM optionnel côté UI ; defaults TLS listables et lisibles par `scan_id` ; OpenAPI à jour. **Réalisé** — [`cafe-discovery` PR #56](https://github.com/create2-labs/cafe-discovery/pull/56).

---

## PR13b — Hydratation frontend par `scan_id` (sans CBOM)

**Statut :** **Réalisé** sur `cafe-frontend` `main` (commit `feat(frontend): hydrate scan cards from Discovery v1 detail by scan_id`). **Non absorbé par PR13e** — **13e** ne couvre que **rpcs** / **scanners** ([#56](https://github.com/create2-labs/cafe-frontend/pull/56)).

**Vérification (`grep` sur `src/`, `scripts/`, `README.md`) :**

- **`src/`** : aucune référence **`CBOM historique`** ; hydratation via **`scanService.js`**, **`tlsService.js`**, **`discoveryScanMappers`** → **`GET …/wallets|tls/scans/{scan_id}`**.
- **Reliquats hors app (suivi)** : **`scripts/cafe.sh`** (`--cboms`, parcours obsolète post-**PR13c**) ; **README.md** — mettre à jour en **PR12** ou PR scripts.

- **Branch:** `frontend/v1-scan-detail-hydration`
- **Repository:** `cafe-frontend`
- **Objective:** Hydrater les cartes via **`GET /discovery/v1/wallets|tls/scans/{scan_id}`** (champ **`result`**) ; conserver listes synopsis v1 ; utiliser **`listDefaultScans()`** / defaults endpoint (**PR13a**).
- **Scope:** Stores/composables scan wallet et TLS, services, mappers synopsis → carte ; tests Vitest/Jest.
- **Out of scope:** Suppression route CBOM serveur (**PR13c**) ; enrichissement backend (**PR13a** fait en amont) ; utilities **PR13e** ; migration **`cafe.sh`** (suivi doc/CLI).
- **Dependencies:** **PR13a** mergée ([#56](https://github.com/create2-labs/cafe-discovery/pull/56)).
- **Tests:** `npm run test` ; smoke manuel listes + ouverture carte + defaults TLS.
- **Validation commands:** Réseau navigateur : plus de requêtes vers `CBOM historique` en parcours nominal ; `grep -R '/discovery/v1/wallets/scans/' src` → vide.
- **Proposed commit title:** `Frontend: hydrate scan cards from Discovery v1 detail by scan_id`
- **Proposed commit message:** `Remove CBOM client calls; map v1 scan detail result to UI card model. Use tls/scans/defaults for shared endpoints.`
- **Proposed PR title:** `Frontend: v1 scan detail hydration (CBOM removal prep)`
- **Proposed PR body:** Endpoints retirés vs nouveaux ; dépendance **PR13a** ; checklist régression cartes.
- **Risks:** Régression affichage si mapping incomplet ; N+1 détail — garder limite de concurrence existante.
- **Completion criteria:** Parcours wallet/TLS OK sans CBOM dans **`src/`** ; grep **`src/`** sans `CBOM historique`. **Réalisé.**

---

## PR13c — Retrait de CBOM historique

- **Branch:** `discovery/remove-cbom-route`
- **Repositories:** `cafe-discovery` ; `cafe-deploy` (scripts e2e/smoke si références CBOM)
- **Objective:** Supprimer handlers, routes et tests **CBOM historique** une fois **PR13b** en production.
- **Scope:** `container.go`, handlers CBOM, tests ; `e2e-dev-stack.sh` / smokes : attendre détail v1 ou statut scan au lieu de `wait_for_cbom`.
- **Out of scope:** Utilities assessments/rpcs (**PR13d**) ; app frontend (**13b** réalisée sur `src/`).
- **Dependencies:** **PR13b** — app **`src/`** sans CBOM (réalisé avant merge **13c**).
- **Tests:** `go test ./...` ; e2e deploy vert ; `curl` CBOM → **404**.
- **Validation commands:** Stack complète ; scripts deploy sans variable CBOM.
- **Proposed commit title:** `Remove Discovery CBOM HTTP routes after v1 detail migration`
- **Proposed commit message:** `Drop GET CBOM historique per WORKPLAN_API. Update deploy e2e to use v1 scan detail. Clients must use scan_id hydration only.`
- **Proposed PR title:** `Cleanup: remove Discovery CBOM routes`
- **Proposed PR body:** Note release intégrateurs ; lien **PR13a**/**PR13b** ; diff scripts.
- **Risks:** Intégrateurs externes encore sur CBOM — communication release.
- **Completion criteria:** Plus de route CBOM ; e2e vert ; OpenAPI sans CBOM (ou marqué removed).

---

## PR13d — Finaliser la surface utilities Discovery v1 *(Discovery seul)*

**Merge :** livré dans **`cafe-discovery`** via [**PR #58**](https://github.com/create2-labs/cafe-discovery/pull/58) (branche `discovery/utilities-and-assessments-v1`).

**Périmètre confirmé**

- Migrer **`GET /discovery/v1/rpcs`** → **`GET /discovery/v1/rpcs`** (backend).
- Migrer **`GET /discovery/v1/scanners`** → **`GET /discovery/v1/scanners`** (backend).
- Retirer **l'ancien déclencheur assessment Discovery**.
- **Pas** d’alias legacy.
- **Aucun** changement CPM.

- **Branch:** `discovery/utilities-and-assessments-v1`
- **Repository:** `cafe-discovery` uniquement
- **Objective:** Clore les dernières routes Discovery **métier** hors **`/discovery/v1`** ; assessment HTTP = **CPM** (**PR13g**).
- **Scope:**
  - **Backend :** **`GET /discovery/v1/rpcs`**, **`GET /discovery/v1/scanners`** (même payload ; **public**, sans JWT).
  - Retirer **l'ancien déclencheur assessment Discovery** et code associé (**`RequestAssessment`**, **`AssessmentRequest`**, **`buildPolicyAssessmentRequestedEvent`**, **`discovery_assessment_test.go`**, mux).
  - Retirer **`/discovery/rpcs`**, **`/discovery/scanners`**, **`assessment HTTP historique Discovery`** — **pas** d’alias.
  - **`internal/discoveryroutes`** : **`RPCs`**, **`Scanners`** (+ edge optionnel **`EdgeRPCs`**, **`EdgeScanners`**).
  - **`openapi/discovery-v1.yaml`** : **`/rpcs`**, **`/scanners`** (paths relatifs v1 ; edge **`/api/discovery/v1/...`** via **`servers`**) ; plus d’assessment Discovery.
  - **`README.md`** : distinguer exemples **backend** vs **edge** ; assessment = CPM-owned.
- **Implementation notes:** Groupe v1 **public** pour utilities (sans **`JWTMiddleware`**), distinct du groupe v1 JWT (scans, wallets). Supprimer le groupe JWT **`/discovery`** s’il ne sert plus qu’à assessment.
- **Out of scope:** **PR13g** (CPM) ; **PR13e** (frontend) ; **PR13f** (deploy) ; CBOM ; rollout.
- **Dependencies:** **PR11b** ; **PR13c** mergée ([#57](https://github.com/create2-labs/cafe-discovery/pull/57)).
- **Tests:** `go test ./...` ; tests v1 utilities ; **404** sur anciens chemins (pattern **`cbom_route_removed_test`**).
- **Validation commands:** `cd cafe-discovery && go test ./...` ; curl backend **`GET /discovery/v1/rpcs`**, **`GET /discovery/v1/scanners`** ; anciens chemins → **404**.
- **Proposed commit title:** `refactor(discovery): finalize v1 utility route surface`
- **Proposed commit message:** `Move rpcs and scanners to /discovery/v1 (public). Remove l'ancien déclencheur assessment Discovery; policy assessment is CPM-owned (PR13g). Update discovery-v1.yaml and README. No legacy aliases.`
- **Proposed PR title:** `Discovery: finalize non-v1 utility route surface`
- **Proposed PR body:** Tableau backend vs edge ; lien **PR13f**/**PR13e**/**PR13g** ; exemption **`/plans`** documentée.
- **Risks:** CI/smoke rouge entre **13d** et **13f** — prioriser merge **13f** ; UI locale entre **13d** et **13e** — même sprint.
- **Completion criteria:** Plus aucune route Discovery **métier** publique hors **`/discovery/v1`**, sauf endpoints techniques/internes (**`/health`**, **`/metrics`**, **`/internal/*`**). OpenAPI v1 inclut **`/rpcs`** et **`/scanners`** ; tests verts. **Réalisé** — [#58](https://github.com/create2-labs/cafe-discovery/pull/58).

**Exemption `/plans` (hors PR13d) :** **`GET /plans`**, **`GET /plans/current`**, **`GET /plans/usage`** restent sous **`/plans`** (quota / abonnement), pas sous **`/discovery/v1`**. Hors périmètre **`WORKPLAN_API.md` §0.1** (contrat scan/wallet/TLS).

**Suivi optionnel (non numéroté) :** migrer vers **`/discovery/v1/plans/...`** ou documenter comme API « compte » distincte dans une PR dédiée — **ne pas** bloquer **PR13d**.

---

## PR13e — Client utilities Discovery v1

**Merge :** livré dans **`cafe-frontend`** via [**PR #56**](https://github.com/create2-labs/cafe-frontend/pull/56).

- **Branch:** `frontend/discovery-v1-utility-routes`
- **Repository:** `cafe-frontend`
- **Objective:** Aligner client et scripts sur les chemins **edge** **`GET /api/discovery/v1/rpcs`** et **`GET /api/discovery/v1/scanners`** (pas les chemins backend nus **`/discovery/v1/...`** sauf appel direct au pod Discovery).
- **Scope:** **`src/services/scanService.js`** (appel RPCs) ; **`src/api/routePaths.ts`** si centralisation ; **`scripts/cafe.sh`** (commandes **`rpcs`**, **`scanners`**) ; **`README.md`** frontend.
- **Out of scope:** Assessment (**PR13g**) ; CPM ; Discovery backend.
- **Dependencies:** **PR13d** mergée ou stack locale avec routes v1 utilities.
- **Tests:** `npm run test` ; smoke **`cafe.sh rpcs`**, **`cafe.sh scanners`**.
- **Proposed commit title:** `Frontend: use Discovery v1 utility routes for rpcs and scanners`
- **Proposed PR title:** `Frontend: Discovery v1 rpcs and scanners paths`
- **Completion criteria:** Grep **`src/`** sans **`/discovery/rpcs`** ni **`/discovery/scanners`** (hors historique commenté). **Réalisé** — [#56](https://github.com/create2-labs/cafe-frontend/pull/56).

---

## PR13f — Deploy e2e : retirer assessment Discovery

**Merge :** livré dans **`cafe-deploy`** via [**PR #15**](https://github.com/create2-labs/cafe-deploy/pull/15) (branche `deploy/e2e-remove-discovery-assessment`).

- **Branch:** `deploy/e2e-remove-discovery-assessment`
- **Repository:** `cafe-deploy`
- **Objective:** **`e2e-dev-stack.sh`** ne doit plus appeler **l'ancien déclencheur assessment Discovery** une fois **PR13d** mergée. **Merger en priorité** après **13d** lorsque ce script est sur le chemin CI/smoke.
- **Scope:** Retirer ou désactiver **`publish_assessment_request`** et son appel dans le mode **`legacy`** ; commentaire pointant vers **PR13g** / **PR13h** pour réintroduire le smoke assessment via CPM.
- **Out of scope:** Implémenter endpoint CPM (**PR13g**) ; Discovery ; frontend.
- **Dependencies:** **PR13d** mergée.
- **Tests:** Exécuter e2e en mode concerné ; pas de référence **`assessment HTTP historique Discovery`**.
- **Proposed commit title:** `Deploy e2e: drop Discovery assessment request after PR13d`
- **Proposed PR title:** `Deploy: remove e2e Discovery assessment trigger`
- **Risks:** Gap de couverture e2e assessment fermé par **PR13h** [`cafe-deploy` #16](https://github.com/create2-labs/cafe-deploy/pull/16).
- **Completion criteria:** E2e vert sans endpoint Discovery assessment. **Réalisé** — [#15](https://github.com/create2-labs/cafe-deploy/pull/15).

---

## PR13g — CPM : `POST /api/cpm/v1/policies/assessment/request` *(wallet scans only)*

**Merge :** livré dans **`cafe-crypto-policy-mgt`** via [**PR #33**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33) (branche `cpm/policies-assessment-request`).

- **Branch:** `cpm/policies-assessment-request`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Déclencheur HTTP **owner** pour assessment policy **async**, **wallet scan only** ; remplace **l'ancien déclencheur assessment Discovery** (retiré en **PR13d**).

**Périmètre produit (figé)**

> CPM policy assessment requests are **wallet-only**. TLS endpoints are discovered and inventoried by Discovery, but CAFE does **not** define TLS migration policies in CPM.

- **Pas** de support assessment pour scans TLS.
- **`scan_id`** doit référencer un **wallet scan** Discovery (owner-scoped).
- Si **`scan_id`** est inconnu, non lisible par l’owner, ou référence un scan **TLS** → **rejeter** (voir codes HTTP ci‑dessous).
- OpenAPI **`cpm-v1.yaml`** et tests : libellé explicite **wallet-scan only**.

**Contrat HTTP (cible)**

| Élément | Valeur |
|---------|--------|
| **Méthode / chemin (edge)** | **`POST /api/cpm/v1/policies/assessment/request`** |
| **Auth** | Bearer JWT owner |
| **Réponse succès** | **`202 Accepted`** (traitement async accepté) |
| **Corps requête** | **`scan_id`** (UUID, **wallet scan** Discovery), **`selection_request`**, **`client_request_id`** (optionnel) |
| **Champ interdit / rejeté** | **`policy_context`** client — rejet **400** si présent ; **pas** d’ignore silencieux |

**`policy_context` :** le contrat async repose sur le wallet scan Discovery (**`GET …/wallets/scans/{scan_id}`**). CPM reconstruit l’observation côté serveur ; un snapshot client est **rejeté** (**400**), jamais ignoré.

**Codes HTTP (figés — validation, authz, scan)**

| Situation | Code | Note |
|-----------|------|------|
| JWT absent / invalide | **401** | Auth CPM existante |
| **`policy_context`** présent (clé non vide / objet fourni) | **400** | Rejet explicite — pas d’ignore silencieux |
| **`scan_id`** mal formé, **`selection_request`** invalide, champs inconnus interdits | **400** | Validation entrée |
| **`scan_id`** inconnu ou non lisible par l’owner (authz deny) | **404** | Convention owner-scoped |
| **`scan_id`** TLS ou scan non-wallet | **404** | Pas de migration policy TLS en CPM |
| Discovery authz / détail indisponible (timeout, 5xx) | **503** | Fail-closed ; ne pas émettre l’événement |
| Wallet scan valide, assessment accepté | **202** | |

**Ne pas** utiliser **403** pour « scan non lisible » sur cette route : figer **404** dans OpenAPI, handler et tests.

**Flux CPM (obligatoire)**

1. Authentifier le JWT owner.
2. Parser le corps : si **`policy_context`** est présent → **400** ; valider **`scan_id`** (UUID) et **`selection_request`**.
3. Vérifier l’accès lecture via Discovery **internal authz** (**`POST /internal/authz/scans/{scanId}/can-read`**) — si deny → **404**.
4. Récupérer le détail autoritaire **wallet** : **`GET /discovery/v1/wallets/scans/{scan_id}`** (backend upstream ; identité owner) — **uniquement** ce chemin, **pas** **`…/tls/scans/…`**.
5. Si la réponse indique un scan non-wallet (ex. tentative TLS, **`scan_family`** ≠ wallet, ou détail TLS) → **404**.
6. Construire l’observation depuis le détail Discovery (pas de snapshot client).
7. Émettre **`policy.assessment.requested.v0.1`** (NATS) **ou** job équivalent — **une** voie, idempotency existante.
8. Répondre **202** avec corrélation (**`event_id`**, **`correlation_id`**, etc. — OpenAPI).

- **Scope:** Handler ; client Discovery (authz + **wallet** détail only) ; **`openapi/cpm-v1.yaml`** (description wallet-only) ; **`internal/cpmroutes`** ; tests : wallet OK → **202** ; TLS **`scan_id`** → **404** ; authz deny → **404** ; pas de **`policy_context`** client dans l’événement NATS.
- **Out of scope:** Assessment TLS ; handler Discovery (**PR13d**) ; **`decisions/explore`** ; persistance policy.
- **Dependencies:** **PR13d** ; **`GET …/wallets/scans/{scan_id}`** stable (**PR4** / **PR13a**) ; Discovery internal authz.
- **Distinction produit:**

| Surface | Sync/async | Éligibilité | Corps client |
|---------|------------|-------------|--------------|
| **`POST …/policies/decisions/explore`** | Sync | Preview (wallet context) | Peut inclure **`policy_context`** |
| **`POST …/policies/assessment/request`** | Async (**202**) | **Wallet scan only** | **`scan_id`** + **`selection_request`** uniquement ; **`policy_context`** → **400** |

**Matrice de tests obligatoire (handler + OpenAPI)**

| Cas | Réponse attendue |
|-----|------------------|
| Corps avec **`policy_context`** | **400** |
| **`scan_id`** wallet valide, Discovery OK | **202** |
| **`scan_id`** inconnu | **404** |
| Authz deny (scan non lisible par l’owner) | **404** |
| **`scan_id`** TLS / non-wallet | **404** |
| Discovery indisponible (authz ou détail) | **503** |

- **Tests:** `go test ./...` ; OpenAPI tag/description wallet-only ; wallet **`scan_id`** + mock Discovery → **202** ; TLS **`scan_id`** → **404** ; authz deny → **404** (pas **403**) ; Discovery down → **503**.
- **Env:** `CAFE_SCAN_AUTHORIZATION_URL`, `CAFE_DISCOVERY_HTTP_BASE`, `CPM_NATS_URL`.
- **Proposed commit title:** `CPM v1: wallet-only POST policies/assessment/request (202)`
- **Proposed commit message:** `Add owner assessment trigger for Discovery wallet scans only. Fetch authoritative wallet scan detail; reject TLS scan_id. Authz deny returns 404. No client policy_context. Emit policy.assessment.requested.v0.1. 202 Accepted.`
- **Proposed PR title:** `CPM v1: wallet-only policy assessment request`
- **Risks:** Latence synchrone Discovery avant **202** ; timeouts authz/détail wallet.
- **Completion criteria:** OpenAPI et tests indiquent **wallet-scan only** ; TLS **`scan_id`** rejeté ; authz deny → **404** ; curl CPM remplace l’ancien Discovery assessment ; événement NATS valide **sans** **`policy_context`** client. **Réalisé** — [#33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33).

---

## PR13h — Deploy e2e : assessment via CPM

**Merge :** livré dans **`cafe-deploy`** via [**PR #16**](https://github.com/create2-labs/cafe-deploy/pull/16) (branche `deploy/e2e-cpm-assessment-request`).

- **Branch:** `deploy/e2e-cpm-assessment-request`
- **Repository:** `cafe-deploy`
- **Objective:** Réintroduire un smoke e2e **`policy.assessment.requested`** via **`POST /api/cpm/v1/policies/assessment/request`** (**PR13g**).
- **Scope:** `e2e-dev-stack.sh` ; éventuellement script CPM **`test-wallet-scan-and-cpm-policy.sh`** si parcours documenté.
- **Dependencies:** **PR13g** mergée ; **PR13f** mergée.
- **Proposed commit title:** `Deploy e2e: CPM policy assessment request smoke`
- **Proposed PR title:** `Deploy: add e2e CPM policy assessment request smoke`
- **Completion criteria:** E2e **`legacy`** (ou mode dédié) vérifie publication assessment après scan v1. **Réalisé** — [`cafe-deploy` PR #16](https://github.com/create2-labs/cafe-deploy/pull/16).

---

## PR12 — Documentation et checklist QA

- **Branch:** `docs/api-coherency-runbook-qa`
- **Repository:** `cafe-documentation` (principal) ; README optionnels dans `cafe-discovery` / `cafe-crypto-policy-mgt`.
- **Objective:** Runbooks et guide développeur alignés **v1** ; assessment policy **CPM-owned** ; plus de CBOM ni routes Discovery supprimées.
- **Scope:** Runbooks et guides d’intégration ; curls Discovery **`/api/discovery/v1/...`** uniquement ; **`GET …/v1/rpcs`**, **`GET …/v1/scanners`** ; section trigger assessment : **`POST /api/cpm/v1/policies/assessment/request`** (**PR13g**) — **plus** **l'ancien déclencheur assessment Discovery** ; **`decisions/explore`** vs assessment async ; checklist QA **§8**.
- **Out of scope:** Copy marketing ; implémentation handlers (**13d**/**13g**).
- **Dependencies:** **13d**, **13e**, **13f**, **13g** mergées.
- **Tests:** N/A (markdown) ; lien HTTP optionnel en CI si déjà présent.
- **Validation commands:** Revue manuelle ; `markdownlint` seulement si déjà dans le repo.
- **Proposed commit title:** `Docs: CAFE API v1 coherency runbook and QA checklist`
- **Proposed commit message:** `Update developer guide and runbooks for /api/discovery/v1 and /api/cpm/v1; add sign-off checklist referencing WORKPLAN_API §8.`
- **Proposed PR title:** `Documentation: API coherency runbook and QA`
- **Proposed PR body:** Liens vers **`cafe-discovery/openapi/discovery-v1.yaml`** et **`cafe-crypto-policy-mgt/openapi/cpm-v1.yaml`**, smokes curls, ordre release **13d→13g→12**.
- **Risks:** Doc obsolète si merge tardif après **11a**/**11b** — merger rapidement.
- **Completion criteria:** Plus de références « primaires » aux chemins supprimés.

---

## Rappel global (git et implémentation)

- **Pas de commit, push, merge ou tags** depuis le plan automatisé : le propriétaire humain garde la main sur git et la publication.
- **Jalon OpenAPI** — **PR1a** [#49](https://github.com/create2-labs/cafe-discovery/pull/49), **PR1b** [#26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). Chaîne **PR2→PR6** : [#51](https://github.com/create2-labs/cafe-discovery/pull/51), [#52](https://github.com/create2-labs/cafe-discovery/pull/52), CPM [#27](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/27), [#53](https://github.com/create2-labs/cafe-discovery/pull/53). **PR7** CPM [#28](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28). **PR8** [#29](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR9** [`cafe-deploy` #11](https://github.com/create2-labs/cafe-deploy/pull/11). **PR10** [`cafe-frontend` #52](https://github.com/create2-labs/cafe-frontend/pull/52). **PR11a** [`cafe-discovery` #54](https://github.com/create2-labs/cafe-discovery/pull/54). **PR11b** CPM [#30](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30), deploy [#12](https://github.com/create2-labs/cafe-deploy/pull/12). **PR11c** CPM [#31](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/31), Discovery [#55](https://github.com/create2-labs/cafe-discovery/pull/55), deploy [#13](https://github.com/create2-labs/cafe-deploy/pull/13), frontend [#54](https://github.com/create2-labs/cafe-frontend/pull/54). **PR13a** Discovery [#56](https://github.com/create2-labs/cafe-discovery/pull/56). **PR13c** [#57](https://github.com/create2-labs/cafe-discovery/pull/57). **PR13d** [#58](https://github.com/create2-labs/cafe-discovery/pull/58). **PR13e** frontend [#56](https://github.com/create2-labs/cafe-frontend/pull/56). **PR13f** deploy [#15](https://github.com/create2-labs/cafe-deploy/pull/15). **PR13g** CPM [#33](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33). **PR13h** deploy [#16](https://github.com/create2-labs/cafe-deploy/pull/16). **Suite :** **PR12**. Détail par chapitre ci‑dessus.

### Instruction type — après jalon OpenAPI (PR #49 + #26)

Lorsque vous ouvrez une PR d’implémentation (**PR2+**), lier la **PR Git** dans la colonne **PR Git** du tableau de suivi et pointer vers les specs sur `main` :

> OpenAPI jalon merged (Discovery PR #49, CPM PR #26). Chaîne **PR2→PR6** documentée avec liens Git (ex. **PR6** → Discovery **#53** ; **PR2**/**PR3** → **#51** ; **PR4** → **#52** ; **PR5** → CPM **#27** ; **#50** fermée sans merge). **PR7** → CPM [**#28**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/28). **PR8** → [**#29**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/29). **PR9** → **`cafe-deploy` [#11](https://github.com/create2-labs/cafe-deploy/pull/11)**. **PR10** → **`cafe-frontend` [#52](https://github.com/create2-labs/cafe-frontend/pull/52)**. **PR11a** → **`cafe-discovery` [#54](https://github.com/create2-labs/cafe-discovery/pull/54)**. **PR11b** → CPM [**#30**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/30), deploy [**#12**](https://github.com/create2-labs/cafe-deploy/pull/12). **PR11c** → CPM [**#31**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/31), Discovery [**#55**](https://github.com/create2-labs/cafe-discovery/pull/55), deploy [**#13**](https://github.com/create2-labs/cafe-deploy/pull/13), frontend [**#54**](https://github.com/create2-labs/cafe-frontend/pull/54). **PR13a** → Discovery [**#56**](https://github.com/create2-labs/cafe-discovery/pull/56). **PR13c** → [#57](https://github.com/create2-labs/cafe-discovery/pull/57). **PR13d** → [#58](https://github.com/create2-labs/cafe-discovery/pull/58). **PR13e** → frontend [**#56**](https://github.com/create2-labs/cafe-frontend/pull/56). **PR13f** → deploy [**#15**](https://github.com/create2-labs/cafe-deploy/pull/15). **PR13g** → CPM [**#33**](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/33). **PR13h** → deploy [**#16**](https://github.com/create2-labs/cafe-deploy/pull/16). Mettre à jour le **résumé exécutif** lorsque le comportement livré diverge.

### Fusion optionnelle de PRs (si vous voulez réduire le nombre)

Les regroupements les moins risqués : **3 + 4** (POST scan + GET historique) ou **5 + 7** (référence interne + surface publique CPM) — à décider avant le premier merge si vous souhaitez une table raccourcie. **Ne pas** regrouper : **11a** et **11b** ; **11b** et **11c** ; **13a** et **13b** ; **13b** et **13c** (le client doit basculer avant de retirer CBOM côté serveur) ; **13d** et **13g** (Discovery cleanup vs CPM assessment).
