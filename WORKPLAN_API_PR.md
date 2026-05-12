# CAFE API coherency — PR plan

**Référence canonique des contrats et sémantiques :** [`WORKPLAN_API.md`](./WORKPLAN_API.md).

**Règles d’exécution (propriétaire humain) :** l’agent / les contributeurs ne font **pas** de commit, push, merge ni tags ; revue, git et publication restent manuelles. Chaque PR : branche locale, changements ciblés, tests, puis proposition de titre/message de commit et de PR (en anglais dans les sections dédiées ci‑dessous).

**Statut du document :** plan de découpe ; jalon OpenAPI **mergé** — **PR1a** [`cafe-discovery` PR #49](https://github.com/create2-labs/cafe-discovery/pull/49), **PR1b** [`cafe-crypto-policy-mgt` PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). **Aucune implémentation** des PR **2+** (handlers, edge, etc.) tant que le propriétaire n’a pas explicitement débloqué la suite au-delà des specs.

---

## Executive summary (état du dépôt au moment de la rédaction)

| Domaine | État observé dans le repo | Écart vs `WORKPLAN_API.md` |
|--------|---------------------------|----------------------------|
| **Chemins publics Discovery** | Fiber : `/discovery/*` ; wallets CAFE sous **`/discovery/v1/wallets`** uniquement (`cafe-discovery/internal/app/container.go`). L’edge envoie `/api/...` → le backend reçoit le chemin sans préfixe `/api` (`cafe-deploy/templates/nginx/nginx.conf.template`). | Cible **`/api/discovery/v1`** ⇒ backend **`/discovery/v1/...`** ; **plus** de route racine **`/wallets`** côté Discovery. |
| **Listes de scans** | `GET /discovery/scans` renvoie une pagination avec **`results`** et **`id` = adresse** (`ListScans`). TLS : `GET /discovery/tls/scans`. | **`GET …/wallets/scans`** avec **`items`**, **`scan_id`**, requêtes **`address` / `chain_id`**, tri par défaut, **`chain_id` sans `address` → 400** ; pas de liste fusionnée. |
| **POST scan** | `UnifiedScan` alloue un **`ScanID`** dans `queueWalletScan` / `queueEndpointScan` mais la réponse HTTP ne l’expose pas ; statut **`processing`**. | Réponse : **`scan_id`**, **`scan_family`**, **`status: requested`**, **`location`** ; vocabulaire de cycle de vie aligné sur le workplan. |
| **DELETE scans / 409 wallet** | Les repos Redis ont `Delete` ; pas de DELETE HTTP owner-scoped pour les scans dans `setupRoutes`. **`DELETE /discovery/v1/wallets/...`** (wallet CAFE via `CafeWalletHandler`). | Matrice DELETE + **`409 SCAN_REFERENCED_BY_POLICY`** : **CPM est autoritaire** sur l’existence de politiques persistées référençant un **`scan_id`** ; **Discovery ne lit pas** la persistance CPM : il appelle l’**endpoint interne CPM** (**PR5**) pour un verdict **`referenced`**, puis orchestre le DELETE (**PR6**). **`WALLET_REFERENCED_BY_POLICY`** : même principe (question CPM, verdict consommé par Discovery) si modélisé (**PR7**). |
| **wallet-policy-contexts** | `GET /discovery/wallet-policy-contexts` implémenté. | Candidat à la suppression **après** migration frontend vers **`wallets/scans`** + **`GET …/policies?scan_id=`** (PR tardive). |
| **CPM** | `read_api.go` : **`/api/v1/policies/...`** ; `owner_routes.go` : **`/api/v1/cpm/policies|drafts`** ; pas de **`DELETE`**, pas de **`GET ?scan_id=`**, **`scan_id` optionnel** à la persistance. | Cible **`/api/cpm/v1/...`**, **`id` + `scan_id` → 400**, **`DELETE` 204/404 uniquement**, liste par **`scan_id`**, **`scan_id` obligatoire** pour le flux Discovery → CPM. |
| **OpenAPI** | **PR1a** + **PR1b** mergées : `openapi/discovery-v1.yaml` ([`cafe-discovery` #49](https://github.com/create2-labs/cafe-discovery/pull/49)), `openapi/cpm-v1.yaml` ([`cafe-crypto-policy-mgt` #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26)). | **Deux** artefacts de contrat distincts (§0.1 + §0.2) ; l’edge se décrit via **`servers`** dans chaque spec ; détail nginx en **PR9**. |
| **Frontend** | `scanService.js` → `/discovery/scans` ; `tlsService.js` → `/discovery/tls/scans` ; CPM via **`/api` + chemin normalisé `/cpm/...`** dans `apiCpmDataSource.ts` (à rapprocher des scripts qui ciblent **`http://localhost:8082/api/v1/cpm/...`**). | Migration chemins et enveloppes ; **revérifier strip edge vs mux CPM** lors de la PR edge. |

---

## Table de suivi des PR

| PR | Branche (proposée) | Dépôt principal | PR Git | Dépend de | Risques / suites (résumé) | Objectif en une ligne |
|----|--------------------|-----------------|--------|-----------|---------------------------|------------------------|
| **1a** | `api-contract/api-coherency-openapi` | `cafe-discovery` | [#49](https://github.com/create2-labs/cafe-discovery/pull/49) | — | Dérive spec vs `WORKPLAN_API.md` ; maintenir `discovery-v1.yaml` aligné. | OpenAPI **§0.1** : **`openapi/discovery-v1.yaml`** ; pas de handler ; option validation CI. |
| **1b** | `api-contract/api-coherency-openapi` | `cafe-crypto-policy-mgt` | [#26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26) | — | Idem côté CPM ; maintenir `cpm-v1.yaml` aligné. | OpenAPI **§0.2** : **`openapi/cpm-v1.yaml`** ; pas de handler ; option validation CI. |
| **2** | `discovery/api-v1-route-skeleton` | `cafe-discovery` | — | **1a** | Clients / edge : n’utiliser que **`/api/discovery/v1/wallets`** (plus de **`/api/wallets`**). | Monter **`/discovery/v1`** ; ordre **`wallets/scans` avant `wallets/:wallet_id`** ; squelettes ; **CAFE wallets uniquement sous v1**. |
| **3** | `discovery/post-scan-contract-response` | `cafe-discovery` | — | **2** | Remplacer le stub **501** sur **`POST /discovery/v1/scan`** ; clients **`processing`** → **`requested`** (**10**). | **`POST /discovery/v1/scan`** : réponse contrat + **`requested`** + **`location`**. |
| **4** | `discovery/scan-history-lifecycle` | `cafe-discovery` | — | **2**, **3** | Remplacer les **501** sur listes / détail scans v1 ; perf listes (N+1) ; OpenAPI si écart. | Listes + détail wallet/TLS : **`items`**, pagination, tri, filtres, règles **`result`**. |
| **5** | `cpm/internal-policy-reference-by-scan` | `cafe-crypto-policy-mgt` | — | **4** | Charge à chaque DELETE scan ; timeouts, circuit, **SLO** documentés. | **CPM seul** : endpoint **interne** (token service) — « cet owner a-t-il des policies persistées qui référencent ce **`scan_id`** ? » — verdict **`referenced`** (+ **`count`** optionnel logs) ; **pas** d’IDs de policies dans la réponse. |
| **6** | `discovery/delete-semantics-cpm-reference-check` | `cafe-discovery` | — | **4**, **5** (et **7** si 409 **`wallet_id`** via CPM) | Latence DELETE ; **503** attendu si CPM indispo ; runbook exploit. | **Discovery** orchestre le DELETE scan / wallet mais **consomme seulement** le verdict CPM ; **503 fail-closed** si la vérif CPM est indisponible. |
| **7** | `cpm/policies-scan-reference-contract` | `cafe-crypto-policy-mgt` | — | **1b**, **4**, **5** | Liste **`scan_id`** en O(n) en mémoire ; plan DB ; même lookup que **PR5**. | **`/api/cpm/v1/policies`**, **`GET ?scan_id=`** (même lookup owner-scoped que **PR5**), **`DELETE ?id=`**, validations, alias rollout optionnels. |
| **8** | `cpm/decisions-explore-contract-check` | `cafe-crypto-policy-mgt` | — | **1b**, **7** | Validation trop stricte → démos ; garder voie « fixtures » (**PR7**). | **`POST …/policies/decisions/explore`** : non persistant ; DTOs alignés scan v1. |
| **9** | `deploy/api-v1-edge-alignment` | `cafe-deploy` | — | **2**, **7** | Coupure si edge avant images ; **ne pas** réintroduire de **`/api/wallets`** (hors **`/api/discovery/v1/wallets`**). | NGINX / env : **`/api/discovery/v1`**, **`/api/cpm/v1`** ; chemins §0.3 **temporaires** documentés. |
| **10** | `frontend/api-coherency-migration` | `cafe-frontend` | — | **9** (ou stack locale équivalente) | Bundles / **SW** staging ; procédure de purge ; **CRUD wallets** : base **`/api/discovery/v1/wallets`** (plus d’ancien **`/wallets`**). | Clients vers nouveaux chemins et enveloppes. |
| **11a** | `cleanup/remove-obsolete-discovery-routes` | `cafe-discovery` | — | **10** | Intégrateurs sur anciennes URLs ; com **release** ; merger avant **11b** si possible. | Retrait handlers / tests Discovery obsolètes (`GET /discovery/scans`, `GET /discovery/tls/scans`, `wallet-policy-contexts`, ancien `POST /discovery/scan`, etc.). |
| **11b** | `cleanup/remove-cpm-rollout-and-client-leftovers` | `cafe-crypto-policy-mgt`, `cafe-frontend`, `cafe-crypto-policy-mgt/scripts`, `cafe-deploy` | — | **10**, **11a** (recommandé : Discovery ne sert plus les anciennes routes avant de retirer l’edge) | Diff « fourre-tout » ; rester strictement rollout + reliquats ; tag specs si besoin. | Alias **§0.3** CPM / nginx, double enregistrement mux, scripts, références frontend résiduelles. |
| **12** | `docs/api-coherency-runbook-qa` | `cafe-documentation` (+ README optionnels) | — | **11a**, **11b** | Doc obsolète si merge tard ; exemples curl **uniquement** chemins v1 (wallets inclus). | Runbooks, exemples curl, checklist QA §8. |

**Colonne PR Git :** lien vers la pull request du dépôt concerné lorsqu’elle existe ; **—** = pas encore créée / à renseigner.

**Colonne Risques / suites (résumé) :** synthèse pour lecture rapide ; le détail (**Risks**, **Completion criteria**, dépendances, périmètres) reste dans le **chapitre de chaque PR** ci‑dessous.

**Découpe PR5 + PR6 (propriété et autorité) :** **Discovery** possède les **scans** (cycle de vie, stockage, suppression physique). **CPM** possède les **policies** et est **autoritaire** sur la question « un **`scan_id`** est-il référencé par des instances persistées pour ce **owner** ? ». **Discovery** ne connaît pas la structure interne des policies, **n’inspecte pas** la DB CPM, **ne duplique pas** d’index `scan_id → policies`, **ne décide pas** seul du 409, **ne supprime jamais** de policies en cascade. **Discovery** appelle **CPM** (PR **5**), **consomme uniquement** le verdict (`referenced: true|false`), puis applique **409** ou poursuit le **DELETE** (PR **6**). L’UI peut lister les policies à détacher via l’API publique **`GET /api/cpm/v1/policies?scan_id=...`** (**PR7**), distincte de l’endpoint interne.

---

## Jalon OpenAPI — PR Git **#49** (Discovery) + **#26** (CPM)

**Principe de dépôt :** le contrat Discovery est **possédé** par `cafe-discovery` ; le contrat CPM par `cafe-crypto-policy-mgt`. Aucun dépôt ne concentre l’OpenAPI de l’autre — évite l’ambiguïté « le CPM possède aussi Discovery ».

**Coordination :** **PR1a** et **PR1b** sont **mergées** — [`cafe-discovery` #49](https://github.com/create2-labs/cafe-discovery/pull/49), [`cafe-crypto-policy-mgt` #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). Les specs **§0.1** et **§0.2** sont sur `main` dans chaque dépôt ; **PR2** / **PR7** peuvent s’appuyer sur ces fichiers OpenAPI.

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
- **Completion criteria:** `go test` prouve l’ordre de routage ; la liste des chemins dans **`openapi/discovery-v1.yaml`** correspond aux routes montées ; **aucune** route **`/wallets`** racine côté Discovery.

---

## PR3 — Réponse `POST /discovery/v1/scan` et lifecycle initial

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
- **Completion criteria:** Tests de contrat + parité champs **`openapi/discovery-v1.yaml`** pour la réponse POST.

---

## PR4 — Historique de scans : listes, détail, pagination, tri, validation des queries

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
- **Completion criteria:** Tous les GET §0.1 sous v1 implémentés ; tests couvrant les bords de query ; comportement aligné avec **`openapi/discovery-v1.yaml`** (mettre à jour la spec si la revue révèle un écart).

---

## PR5 — CPM : vérification autoritaire des références policy → `scan_id` (interne)

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
- **Completion criteria:** Discovery peut appeler l’endpoint et interpréter uniquement **`referenced`** (PR **6**).

---

## PR6 — Discovery DELETE semantics using CPM reference verification

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
- **Completion criteria:** §2.2 / §4.2 DELETE scan du workplan respectés ; aucun accès Discovery à la persistance CPM ; documenter **503** / **`POLICY_REFERENCE_CHECK_UNAVAILABLE`** dans **`openapi/discovery-v1.yaml`** si les erreurs y sont listées.

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

- **Branch:** `cpm/decisions-explore-contract-check`
- **Repository:** `cafe-crypto-policy-mgt`
- **Objective:** Garantir que **explore** ne persiste pas d’instance finale ; accepter **`policy_context`** / filaire observation compatible avec le détail Discovery v1.
- **Scope:** `internal/api/read_api.go`, `explore_policy_context.go`, `read_api_test.go`.
- **Out of scope:** Nouvelles voies de persistance ; nginx.
- **Dependencies:** **PR1b** (**`openapi/cpm-v1.yaml`** sur `main` — [PR #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26)), **PR7** (stabilité des chemins **`/api/cpm/v1`**).
- **Tests:** Étendre les tests explore avec des payloads formés comme le DTO détail v1.
- **Validation commands:** `cd cafe-crypto-policy-mgt && go test ./...`
- **Proposed commit title:** `CPM: align decisions/explore with Discovery v1 scan context`
- **Proposed commit message:** `Verify POST …/policies/decisions/explore does not persist policies; tighten validation and tests for policy_context from v1 scan DTOs.`
- **Proposed PR title:** `CPM: decisions/explore contract vs Discovery v1`
- **Proposed PR body:** Énoncé explicite : aucune écriture policy en base sur explore ; exemples.
- **Risks:** Validation trop stricte casse des démos — garder le chemin « fixtures » documenté en PR7.
- **Completion criteria:** Tests documentent la non-persistance ; l’évaluateur s’exécute toujours.

---

## PR9 — Edge / déploiement

- **Branch:** `deploy/api-v1-edge-alignment`
- **Repository:** `cafe-deploy` (principal) ; commentaires d’env compose si nécessaire pour URLs internes PR5/6.
- **Objective:** Exposer **`/api/discovery/v1`** et **`/api/cpm/v1`** avec **`proxy_pass`** cohérent ; documenter les alias **§0.3** comme **temporaires** ; conserver le blocage **`/api/internal/`** à l’edge.
- **Scope:** `templates/nginx/nginx.conf.template`, templates env (health/blackbox) si les chemins changent.
- **Out of scope:** Suppression définitive des alias rollout (**PR11b**) ; code applicatif.
- **Dependencies:** **PR2** et **PR7** (routes existantes avant bascule clients).
- **Implementation notes:** **Résoudre l’incohérence** frontend **`/api/cpm/...`** vs mux direct **`/api/v1/cpm/...`** : une seule histoire pour navigateur et scripts après cette PR. **Ne pas** exposer d’alias edge **`/api/wallets`** (ou équivalent court) vers Discovery : les wallets CAFE passent par **`/api/discovery/v1/wallets`** après strip **`/api`**.
- **Tests:** `nginx -t` en CI ou check manuel documenté ; extraits curl dans la PR body.
- **Validation commands:** `docker compose ... config` si applicable ; `nginx -t` sur conf générée.
- **Proposed commit title:** `Deploy: edge routes for /api/discovery/v1 and /api/cpm/v1`
- **Proposed commit message:** `Update NGINX routing for WORKPLAN_API public bases; document temporary rollout paths and Discovery→CPM internal URL envs.`
- **Proposed PR title:** `Deploy: NGINX alignment for Discovery and CPM v1 APIs`
- **Proposed PR body:** Cartographie edge → upstream ; note de migration staging/prod.
- **Risks:** Coupure si chemins basculés avant images — ordonner avec **10**.
- **Completion criteria:** Stack locale : curls vers v1 OK.

---

## PR10 — Migration frontend

- **Branch:** `frontend/api-coherency-migration`
- **Repository:** `cafe-frontend`
- **Objective:** Basculer les appels Discovery vers **`/discovery/v1/...`** (via base **`/api`**), adopter **`items`**, traiter la réponse **`POST /scan`** (**`location`**, **`scan_id`**), CPM vers **`/api/cpm/v1/...`** aligné avec **9**. **CRUD wallets** : tout appeler sous **`/api/discovery/v1/wallets`** (plus d’URL **`/wallets`** ou **`/api/.../wallets`** hors ce préfixe).
- **Scope:** `src/services/scanService.js`, `tlsService.js`, services / composables **wallets** (création, liste, suppression, etc.), `cpm/apiCpmDataSource.ts`, composables / tests associés ; `.env.example` si besoin.
- **Out of scope:** Suppression backend des anciennes routes (**11a**) ; nettoyage rollout / scripts (**11b**).
- **Dependencies:** **PR9** (ou overrides locaux coordonnés).
- **Tests:** Mise à jour Vitest/Jest pour builders d’URL et parsers ; `apiCpmDataSource.spec.ts`.
- **Validation commands:** `cd cafe-frontend && npm run test` (ou script CI du projet) ; `npm run build` si applicable.
- **Proposed commit title:** `Frontend: migrate to Discovery v1 and CPM v1 APIs`
- **Proposed commit message:** `Update REST clients for new paths, pagination envelopes, and POST /scan correlation fields per WORKPLAN_API.`
- **Proposed PR title:** `Frontend: API coherency migration for Discovery and CPM`
- **Proposed PR body:** Liste des endpoints remplacés ; note feature-flag si applicable.
- **Risks:** Anciens bundles / SW en staging — procédure de purge.
- **Completion criteria:** Parcours scan + CPM principaux OK contre la stack **9**.

---

## PR11a — Suppression des routes Discovery obsolètes

- **Branch:** `cleanup/remove-obsolete-discovery-routes`
- **Repository:** `cafe-discovery` uniquement
- **Objective:** Retirer les handlers, routes Fiber et tests associés aux chemins **hors modèle cible** une fois **PR10** déployée : p.ex. **`GET /discovery/scans`**, **`GET /discovery/tls/scans`** (id = adresse/URL), **`GET /discovery/wallet-policy-contexts`** si remplacé, ancien **`POST /discovery/scan`**, etc. (**Les wallets CAFE sont déjà sous `/discovery/v1/wallets` — plus de `/wallets` racine.**)
- **Scope:** `internal/app/container.go`, handlers, tests d’intégration ; **aucun** changement `cafe-deploy` / frontend dans cette PR.
- **Out of scope:** Alias nginx, mux CPM, scripts hors repo Discovery, `cafe-frontend`.
- **Dependencies:** **PR10** (clients sur v1) ; **9** déployé sur les cibles où le cleanup s’applique.
- **Tests:** `cd cafe-discovery && go test ./...` ; grep du repo sans références aux anciens chemins enregistrés.
- **Validation commands:** Idem ; smoke `curl` direct sur backend si utile.
- **Proposed commit title:** `Remove obsolete Discovery HTTP routes after v1 migration`
- **Proposed commit message:** `Drop legacy discovery list/context/scan routes per WORKPLAN_API; v1 paths remain the only served contract.`
- **Proposed PR title:** `Cleanup: remove obsolete Discovery API routes`
- **Proposed PR body:** Liste des chemins retirés ; ordre de merge avant **11b** / edge.
- **Risks:** Intégrateurs encore sur anciennes URLs Discovery — communication release ; merger **11a** avant **11b** limite la fenêtre « edge → 404 » incohérente.
- **Completion criteria:** Aucun handler Discovery pour les routes retirées ; `go test` vert.

---

## PR11b — Retrait alias rollout CPM, edge, scripts et reliquats client

- **Branch:** `cleanup/remove-cpm-rollout-and-client-leftovers`
- **Repositories:** `cafe-crypto-policy-mgt`, `cafe-frontend`, `cafe-crypto-policy-mgt/scripts`, `cafe-deploy` (pas `cafe-discovery` — couvert par **11a**)
- **Objective:** Supprimer les chemins **§0.3** / double enregistrement sur le mux CPM, mettre à jour **nginx** / compose pour ne plus exposer les alias temporaires, aligner **scripts** et toute **référence résiduelle** frontend (grep, defaults, commentaires) une fois **11a** mergée (recommandé : plus de backend Discovery obsolète derrière le même edge).
- **Scope:** `owner_routes.go`, `read_api.go`, `auth.go` (routes retirées), templates `cafe-deploy`, `apiCpmDataSource` / env / tests, scripts shell du dépôt CPM.
- **Out of scope:** Routes Discovery (déjà **11a**) ; nouvelles fonctionnalités.
- **Dependencies:** **PR10** ; **11a** recommandée avant merge (sinon risque de filet : edge ou clients encore alignés sur anciennes hypothèses Discovery).
- **Tests:** `go test` CPM ; `npm test` frontend ; `nginx -t` ou `compose config` si touché ; scripts mis à jour exécutés manuellement ou en CI.
- **Validation commands:** Par dépôt touché.
- **Proposed commit title:** `Remove CPM rollout aliases and post-migration client/deploy leftovers`
- **Proposed commit message:** `Drop §0.3-style CPM paths from mux and edge; clean scripts and frontend references; obsolete public contract no longer served.`
- **Proposed PR title:** `Cleanup: CPM rollout removal and client/deploy leftovers`
- **Proposed PR body:** Checklist alias retirés ; lien **11a** ; note opérateur pour l’ordre de déploiement.
- **Risks:** PR « filet » si **11b** inclut trop de changements non liés — garder le diff **strictement** rollout + reliquats ; tag ou versionnement des **deux** specs OpenAPI si besoin.
- **Completion criteria:** Plus d’alias rollout documentés comme actifs ; grep sans **`/api/v1/cpm/`** ni doubles handlers CPM cibles ; edge ne route plus vers chemins supprimés.

---

## PR12 — Documentation et checklist QA

- **Branch:** `docs/api-coherency-runbook-qa`
- **Repository:** `cafe-documentation` (principal) ; README optionnels dans `cafe-discovery` / `cafe-crypto-policy-mgt`.
- **Objective:** Remplacer les exemples curl du guide développeur ; checklist QA type **§8** ; scripts alignés avec **11a** / **11b**.
- **Scope:** Runbooks et guides d’intégration ; **tous** les exemples curl côté Discovery (y compris **wallets**) sous **`/api/discovery/v1/...`** — **aucune** doc « primaire » qui cible **`/wallets`** ou **`/api/wallets`** hors ce préfixe.
- **Out of scope:** Copy marketing.
- **Dependencies:** **11a** et **11b** terminées (ou merge équivalent : cleanup complet sans fenêtre doc cassée).
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
- **Première étape d’implémentation :** jalon OpenAPI **terminé** — **PR1a** [`cafe-discovery` #49](https://github.com/create2-labs/cafe-discovery/pull/49), **PR1b** [`cafe-crypto-policy-mgt` #26](https://github.com/create2-labs/cafe-crypto-policy-mgt/pull/26). Enchaîner avec **PR2** (Discovery) et le fil CPM (**PR5** …) selon approbation propriétaire.

### Instruction type — après jalon OpenAPI (PR #49 + #26)

Lorsque vous ouvrez une PR d’implémentation (**PR2+**), lier la **PR Git** dans la colonne **PR Git** du tableau de suivi et pointer vers les specs sur `main` :

> OpenAPI jalon merged (Discovery PR #49, CPM PR #26). Proceed with PR2+ only after owner approval. Update **PR Git** and, if useful, **Risques / suites (résumé)** in the tracking table for each implementation PR; keep full detail in the PR chapter.

### Fusion optionnelle de PRs (si vous voulez réduire le nombre)

Les regroupements les moins risqués : **3 + 4** (POST scan + GET historique) ou **5 + 7** (référence interne + surface publique CPM) — à décider avant le premier merge si vous souhaitez une table raccourcie. **Ne pas** regrouper **11a** et **11b** : la séparation Discovery-only vs CPM + frontend + scripts + deploy limite le blast radius du cleanup.
