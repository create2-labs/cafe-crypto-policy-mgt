# Document de travail — remise à plat des API

> **English:** Normative HTTP contract (this file, French). Product behavior summary: [CAFE functional specifications](https://github.com/create2-labs/cafe-documentation/blob/main/functional-specifications.md). Integration guide: [CAFE developer guide](https://github.com/create2-labs/cafe-documentation/blob/main/03-cafe-developer-guide.md).

**Statut :** proposition — **aucune implémentation** tant que ce document n’est pas **explicitement accepté** (sign-off produit / archi / sécurité).

**Politique documentation :** seuls les chemins publics **§0** (Discovery **`/api/discovery/v1`**, CPM **`/api/cpm/v1`**) sont cités. Les surfaces HTTP hors v1 ne sont pas listées dans la doc produit ni ici sous forme de littéraux de routes.

**Périmètre du document :** définir les **surfaces HTTP** attendues, la **sémantique** des ressources, et les **arbitrages** à valider avant acceptation — **§5** consolide les **décisions finales** ; **§8** est la **checklist de sign-off** avant OpenAPI / code ; le reste du texte peut encore signaler des **points à confirmer** par section lorsqu’ils ne sont pas couverts par **§5**. Il ne modifie pas le code ni les contrats déployés.

**Dépôts concernés (à confirmer par section) :** `cafe-discovery` (observation / scans), `cafe-crypto-policy-mgt` (politiques), `cafe-deploy` / edge (chemins publics).

**Option A (définition produit) :** [`CPM_post_v_1_option_a_scan_context.md`](./CPM_post_v_1_option_a_scan_context.md). **Narratif intégré (jalons mergés) :** [`docs/CPM_OPTION_A_INTEGRATED.md`](../docs/CPM_OPTION_A_INTEGRATED.md) — plan de PR [`CPM_OPTION_A_PR_PLAN.md`](./CPM_OPTION_A_PR_PLAN.md) ; index PR mergées [`WORKPLAN_API_PR.md`](./WORKPLAN_API_PR.md).

---

## 0. Convention de chemins publics (cible pour OpenAPI / edge)

**Décision :** les URL ci‑dessous sont la **référence canonique** pour specs, clients et configuration **edge**. Dans la suite du document, la notation **`GET …/wallets`** signifie **`GET /api/discovery/v1/wallets`** (préfixe **§0.1**), sauf mention contraire. **`GET …/wallets/scans`** = **`GET /api/discovery/v1/wallets/scans`** ; **`GET …/tls/scans`** = **`GET /api/discovery/v1/tls/scans`** — **symétrie** des deux familles de listes sous **`…/wallets/…`** et **`…/tls/…`**.

### 0.1 Discovery (CAFE / scans / wallets)

**Base :** **`/api/discovery/v1`**

| Verbe(s) | Chemin |
|----------|--------|
| **GET** | `/api/discovery/v1/wallets` |
| **GET**, **DELETE** | `/api/discovery/v1/wallets/{wallet_id}` |
| **GET** | `/api/discovery/v1/wallets/scans` — scans **wallet** (cibles EVM) ; queries **`address`**, **`chain_id`**, pagination — **§2.2**, **§5.4.2** |
| **GET**, **DELETE** | `/api/discovery/v1/wallets/scans/{scan_id}` |
| **GET** | `/api/discovery/v1/tls/scans` — scans **TLS** (endpoints) ; pagination — **§2.2**, **§5.4.3** |
| **GET**, **DELETE** | `/api/discovery/v1/tls/scans/{scan_id}` |
| **POST** | `/api/discovery/v1/scan` — corps **`{ "address": "0x…" }`** **ou** **`{ "url": "https://…" }`** (mutuellement exclusif) ; même logique métier que **`POST /discovery/v1/scan`** aujourd’hui (**§2.2**) |

**Routage (impl.) :** enregistrer **`…/wallets/scans`** et **`…/wallets/scans/{scan_id}`** **avant** **`…/wallets/{wallet_id}`** pour que le segment littéral **`scans`** ne soit pas capturé comme un **`{wallet_id}`**.

### 0.2 CPM — cible **sans** redondance `…/cpm/cpm/…`

Si le service CPM est publié sous **`/api/cpm/v1`** (le segment **`cpm`** n’apparaît **qu’une fois** dans le préfixe public) :

| Verbe(s) | Chemin |
|----------|--------|
| **GET** | `/api/cpm/v1/policies/catalog` |
| **GET** | `/api/cpm/v1/policies/templates` |
| **GET** | `/api/cpm/v1/policies/instances` |
| **POST** | `/api/cpm/v1/policies/decisions/explore` |
| **POST**, **GET**, **DELETE** | `/api/cpm/v1/policies` — instances persistées (**`GET`** / **`DELETE`** avec query **`id`** ; **`GET`** liste owner-scoped avec query **`scan_id`** — **pas** **`id`** + **`scan_id`** ensemble — **§5.2**) |
| **POST**, **GET**, **DELETE** | `/api/cpm/v1/drafts` — brouillons (**`DELETE`** avec query **`id`** — **§2.4**, **§4.4**) |

**Remarque :** pas de chemin du type **`/api/cpm/v1/cpm/policies`**. Les **réponses d’erreur** référencent les **instances** persistées comme **`…/policies`** sous ce préfixe (ex. **`409`** scan — **§4.2**).

**Contrat produit CPM :** la **cible** à documenter dans **OpenAPI** et à viser côté clients est **§0.2** uniquement. La transition ingress/clients est **clôturée** (voir **§0.3**).

### 0.3 CPM — transition (clôturée)

La bascule vers **§0.2** est **terminée**. Les alias edge de transition ne sont **plus** servis ni documentés.

---

## 1. Contexte

Aujourd’hui, **la liste wallet historique** renvoie une liste paginée où chaque entrée **`id`** est une **adresse wallet**, pas un identifiant de **`scan_result`**. Conceptuellement, l’URL **`/scans`** suggère une collection d’**exécutions de scan**, ce qui entre en tension avec ce comportement.

**Constat :** lorsqu’on **redemande un scan** pour un wallet déjà scanné, le **`scan_id` (UUID)** peut **changer** au lieu d’**ajouter** une nouvelle exécution en collection. Un tel comportement **n’est pas un historique** : la collection **`/scans`** ne peut pas documenter plusieurs tentatives dans le temps si l’identifiant « glisse ».

**Cible URL (wallet)** : les scans **wallet / EVM** sont exposés sous **`…/wallets/scans`** (et **`…/wallets/scans/{scan_id}`**), en **symétrie** avec **`…/tls/scans`**, et non plus sous un segment racine isolé **`/scans`**.

Ce document pose un **modèle cible** cohérent : **collections au pluriel** uniquement, sous le **préfixe public** **§0.1** (Discovery), pour faciliter **OpenAPI** et le **frontend**.

- **Wallets CAFE** (**§2.1**) : **`GET …/wallets`** (liste d’identifiants) ; **`GET …/wallets/{wallet_id}`** (détail) ; **`DELETE …/wallets/{wallet_id}`** (effacement **owner**).
- **Scans wallet** (**§2.2**) : **`GET …/wallets/scans`** (liste + filtres **`address`** / **`chain_id`**) ; **`GET|DELETE …/wallets/scans/{scan_id}`** — collection **wallet / EVM** uniquement (distincte des wallets CAFE enregistrés sous **`…/wallets`**).
- **Scans TLS** (**§2.2**) : **`GET …/tls/scans`** (liste) ; **`GET|DELETE …/tls/scans/{scan_id}`** — collection **endpoints TLS** ; **pas** de filtres **`address`** / **`chain_id`** sur cette famille (non applicables).
- **Liste** : **`GET …/wallets/scans`** [+ **`?address`**, **`&chain_id`**] et **`GET …/tls/scans`** partagent la **même enveloppe** par collection — **`items`** + **`total`**, **`limit`**, **`offset`** — anti-**N+1** (**§4.2**). **Tri par défaut** : **`created_at`** desc, puis **`scan_id`** desc — **§2.2**.
- **Détail vs liste** : résultat métier dans **`GET …/wallets/scans/{scan_id}`** et **`GET …/tls/scans/{scan_id}`** ; en liste, **`status`** = **lifecycle** seulement (**§2.2**).
- **`DELETE …/wallets/scans/{scan_id}`** et **`DELETE …/tls/scans/{scan_id}`** : **`204` \| `404` \| `409`** ( **`409`** = référence CPM sur le **`scan_id`** concerné — **§2.2**, **§4.2**).
- **Filtres wallet** : **pas** d’adresse en **path** sous **`wallets/scans`** ; query **`address`**, **`chain_id`** — **`chain_id`** **uniquement** avec **`address`** ; **`chain_id`** seul → **`400`** (**§2.2**).
- **CAFE multi-chaînes** (**`…/wallets/scans`** seulement) : sans **`chain_id`**, la vue par adresse agrège **toutes** les chaînes ; le synopsis peut porter plusieurs **`chain_ids`**.
- **`eoa/`** : **hors** modèle cible.
- **CPM** : chemins **§0.2** (cible) ou **§0.3** (transition déploiement) ; verbes **§2.3–2.4**.

---

## 2. Objectifs (exigences métier)

### 2.1 Wallets CAFE — collection + ressource

- **`GET …/wallets`** (liste) : réponse = **liste des identifiants** (ou synopsis + **`id`**) des **wallets CAFE** du user authentifié (créés / gérés produit CAFE — **sans** assimiler une simple adresse vue au scan comme wallet CAFE **tant qu’elle n’est pas enregistrée comme telle**, sauf règle produit explicite).
- **`GET …/wallets/{wallet_id}`** : détail du wallet CAFE dont l’identifiant **`{wallet_id}`** appartient au user (**404** sinon).
- **`DELETE …/wallets/{wallet_id}`** : suppression du wallet CAFE pour **l’utilisateur propriétaire**. **`204`** si supprimé ; **`404`** si inconnu / hors scope (**idempotence** : second **`DELETE`** → **`404`** — **§5.4.8**). **`409`** si conflit métier (**p.ex.** politiques CPM persistées — **alignement** possible sur **`DELETE …/wallets/scans/{scan_id}`** — **§5.4.6**). Effets sur **scans** liés (cascade vs orphan **wallet_id**) : **§5.4.8**.

### 2.2 Scans Discovery — deux collections **`…/wallets/scans`** (wallet) et **`…/tls/scans`** (TLS)

**Principe retenu :** **deux endpoints de liste** (comme aujourd’hui **la liste wallet historique** vs **la liste TLS historique**), pour **séparer** clairement les exécutions **wallet / EVM** et les exécutions **TLS** ; la cible **uniformise** les chemins : **`…/wallets/scans`** **miroir** de **`…/tls/scans`**. **Pas** de liste unique fusionnée. Le **`POST …/scan`** avec **`address`** **ou** **`url`** reste **un seul** point de **déclenchement** (**§0.1**).

**Collection wallet — `GET …/wallets/scans`**, **`GET …/wallets/scans?address=0x…`**, **`GET …/wallets/scans?address=…&chain_id=…`**

- Réponse **paginée** : **`items`**, **`total`**, **`limit`**, **`offset`**. Chaque **`items[]`** = synopsis **minimal**, **pas** le DTO métier complet — réservé à **`GET …/wallets/scans/{scan_id}`**. Anti-**N+1** (**§4.2**).
- **`GET …/wallets/scans?address=0x…`** : filtre sur **adresse Ethereum** normalisée, **toutes chaînes** si **`chain_id`** absent ; le synopsis peut porter plusieurs **`chain_ids`** (**CAFE** multi-chaînes).
- **`GET …/wallets/scans?address=…&chain_id=N`** : **même enveloppe** ; **`chain_id`** = raffinement **mono-chaîne** pour cette adresse. **`chain_id`** **sans** **`address`** → **`400 Bad Request`** (**§2.2**).
- **`GET …/wallets/scans?address=0x…&latest=true`** (tranché — **pas** de route **`/latest`** dédiée) : retourne le **dernier scan terminé avec succès** (**lifecycle `completed`** uniquement — **pas** `failed`, **pas** `requested` / `started`) pour cette adresse. **Même enveloppe** paginée ; **`items`** ≤ 1 ; **`total`** **`0`** ou **`1`**. Tri : **`created_at`** desc, puis **`scan_id`** desc. **`latest=true`** **sans** **`address`** → **`400`**. Avec **`&chain_id=N`** : dernier **`completed`** parmi les scans matchant la chaîne. Usage : **CPM (W2)**, UI (scan de référence policy).

  > *Do not use `limit=1` alone as a substitute for `latest=true`: the newest row by `created_at` may be in progress or failed.*

  > *CPM (**W7**) : si la ligne la plus récente (`created_at`) n’est pas `completed`, bloquer explore/persist même s’il existe un `completed` plus ancien. **Rescan (**W8**)** : **`POST …/scan`** autorisé si le plus récent est `failed` (nouvelle tentative) ; refus seulement si `requested` ou `started` — **§2.2 W7–W8**.*

- **`ScanListItem`** (wallet, OpenAPI) : **`scan_id`**, **`created_at`**, **`status`** (**lifecycle** seul — **pas** posture / verdict métier). **`target_address`** (ou équivalent) et **`chain_ids`** : renseignés selon le cas ; **`null`** / **`[]`** si non encore connus (**schéma** à figer).

**Collection TLS — `GET …/tls/scans`**

- **Même enveloppe** pagination (**`items`**, **`total`**, **`limit`**, **`offset`**).
- **`TLSListItem`** (OpenAPI) : **`scan_id`**, **`endpoint`** (URL ou identifiant stable affichable), **`created_at`**, **`status`** (**lifecycle** seul). **Pas** de **`address`** / **`chain_id`** sur cette route.

**Tri par défaut des listes (tranché)** — **`GET …/wallets/scans`** (y compris **`?address`**, **`&chain_id`**) et **`GET …/tls/scans`** :

- **Ordre** : **`created_at`** **descendant**, puis **`scan_id`** **descendant** comme **départage déterministe** (tie-breaker) lorsque plusieurs lignes partagent le même **`created_at`**.
- **Objectif** : réponses **reproductibles** (tests, snapshots) et comportement **UI** prévisible — à porter **explicitement** dans **OpenAPI**.

> *Default sort order: `created_at` descending, then `scan_id` descending as deterministic tie-breaker.*

**Détail et effacement**

- **`GET …/wallets/scans/{scan_id}`** / **`GET …/tls/scans/{scan_id}`** : **DTO détail** avec **`scan_id`**, **`status`** (lifecycle), **`result`** (résultat métier — schéma **distinct** wallet vs TLS en OpenAPI). **`result`** absent ou partiel tant que non terminal ; **immutable** après état terminal (**§2.2** invariants ci‑dessous).
- **Identité, cycle de vie et immutabilité du résultat (invariants cibles — CAFE asynchrone)** :
  1. **Allocation du `scan_id` (tranché)** : le **`scan_id`** (UUID) est attribué lorsque l’**exécution est acceptée** par Discovery (**état initial `requested`**) — **avant** la publication vers le pipeline asynchrone, pour permettre le **suivi** immédiat côté client.

     > *The scan_id is allocated when the scan execution is accepted by Discovery, before publishing the scan request to the asynchronous pipeline.*

  2. **Graphes de lifecycle** (nomenclature **OpenAPI**) :
     - **`requested` → `started` → `completed`**
     - **`requested` → `started` → `failed`**
     - **`requested` → `failed`**
  3. **Métadonnées de lifecycle** (**`status`**, horodatages, progression, erreurs transitoires…) **évoluent** jusqu’à un **état terminal** (**`completed`**, **`failed`**).
  4. **Une fois** l’état **terminal** atteint, le **résultat métier** exposé (p.ex. sous **`result`** dans le DTO détail — **§4.2**) est **immutable** (**aucune réécriture** qui **substituerait** ce snapshot final).
  5. **Nouvelle exécution** (**re-scan**, nouvelle tentative) = **toujours un nouveau résultat de scan persisté**, donc **une nouvelle ligne** avec **son propre** **`scan_id`** (UUID) — **pas** de « poursuite » sous le même UUID pour une **nouvelle** exécution (**un** **`scan_id`** donné reste attaché à **la même** ligne de résultat pour toute la vie de cette ligne).

  > **Formulation de référence :**
  >
  > *Each persisted scan result row has its own `scan_id`; that id is stable for the lifetime of that row.*
  >
  > *Its lifecycle metadata may evolve until it reaches a terminal state.*
  >
  > *Once completed, the scan result payload is immutable.*
  >
  > *A new scan execution creates a new persisted scan result row, with a new `scan_id`.*
  >
  > (En **§2.2**, **`failed`** est traité comme **état terminal** : le même principe d’« immutabilité du résultat exposé après fin d’exécution » s’applique au **snapshot final** **`failed`** — schéma **OpenAPI**.)

- **Historique (listes)** : **`GET …/wallets/scans`** [+ **`?address`**, **`&chain_id`**] et **`GET …/tls/scans`** peuvent renvoyer plusieurs **`scan_id`** distincts pour une **même** cible (adresse / endpoint) si **plusieurs exécutions** ont produit **plusieurs lignes de résultat** (chacune avec son **`scan_id`**) — cf. point **5** ci‑dessus.

- **CBOM par exécution (wallet, tranché)** : **`GET …/wallets/scans/{scan_id}/cbom`** — document **CBOM** (CycloneDX ou enveloppe documentée OpenAPI) **généré à la demande** depuis la ligne de résultat de **cet** **`scan_id`** ; **jamais** stocké en blob. **`404`** si scan absent / hors scope ; règles d’accès alignées sur le détail owner. Les routes hors contrat §0 **CBOM historique** restent **retirées** (**§8.7**). Symétrie TLS optionnelle : **`GET …/tls/scans/{scan_id}/cbom`**.

- **Effacement** : **`DELETE …/wallets/scans/{scan_id}`** et **`DELETE …/tls/scans/{scan_id}`** (**owner**). **`204`** \| **`404`** \| **`409`** — **même sémantique** (**référence CPM** → **`409`** **`SCAN_REFERENCED_BY_POLICY`** — **§4.2**) pour tout **`scan_id`** référencé par une instance persistée. **Action utilisateur requise** : le client **doit d’abord** supprimer ou détacher les instances CPM (**`DELETE …/policies?id=…`**, éventuellement après **`GET …/policies?scan_id=…`** pour les lister), **puis** appeler **`DELETE …/wallets/scans/{scan_id}`** (ou TLS). **Pas** de suppression en cascade scan → policies.

- **Idempotence `DELETE` (tranché)** — **wallets CAFE**, **`…/wallets/scans/{scan_id}`**, **`…/tls/scans/{scan_id}`**, **instances CPM** : **`204`** \| **`404`** (**§5.4.8**, **§5.4.9** ; scans — **§5.4.6**) ; **`409`** possible sur les **`DELETE`** scan (**§4.2**).

- **Corrélation CPM** : le **`scan_id`** (**UUID**) est la clé de traçabilité **Option A** (CPM) pour les exécutions **wallet** (**W1–W8**, assessment/remediation produit actuel). Les scans **TLS** restent sous **Discovery** (**§5.4.6**) ; une référence policy sur un **`scan_id`** TLS ne fait **pas** du TLS une cible assessment/remediation CPM pour cette release. Tant qu’une **politique persistée** référence un **`scan_id`**, le **`DELETE`** du scan correspondant → **`409`** (**§4.4**).

**Couplage wallet ↔ CPM (tranché — familles wallet / `binding=discovery`)**

| # | Règle | Discovery | CPM |
|---|--------|-----------|-----|
| **W1** | **Au plus une policy persistée bloque le rescan par adresse** : **pas de nouveau scan** si une **policy persistée** existe pour la **`target_address`** (après **W8**). Un **brouillon (`draft`) plateforme seul** **n’empêche pas** le rescan | **`POST …/scan`** → **`409`** lorsque **policy persistée** (ex. **`CPM_POLICY_EXISTS_FOR_WALLET_TARGET`** ou **`CPM_EXISTS_FOR_WALLET_TARGET`** + **`blocking_kind: "policy"`**) — **pas** de **`409`** pour draft seul | Lookup owner-scoped : **`…/policies`** pour le guard POST ; **`…/drafts`** pour contexte CPM / orphan (**§4.4**) — draft **hors** guard POST après **IMM-W1-4** |
| **W2** | **CPM uniquement sur le dernier scan `completed`** de la cible | **`GET …/wallets/scans?address=&latest=true`** (dernier **`completed`** seulement) | **`400`** si **`scan_id`** ≠ celui de **`latest=true`** |
| **W3** | **DELETE scan** : **action utilisateur** — d’abord **supprimer la CPM**, puis **`DELETE …/wallets/scans/{scan_id}`** | **`409`** **`SCAN_REFERENCED_BY_POLICY`** tant qu’une policy référence ce **`scan_id`** ; **`204`** une fois les policies retirées | **`DELETE …/policies?id=`** **sans** effet sur les scans (**W4**) ; **`GET …/policies?scan_id=`** pour lister avant DELETE scan |
| **W4** | **Suppression CPM** sans effet sur les scans | Inchangé | **`DELETE …/policies?id=`** seul |
| **W5** | **Historique** par adresse | **`GET …/wallets/scans?address=`** | Lecture ; policies par **`scan_id`** |
| **W6** | **CBOM par scan** | **`GET …/wallets/scans/{scan_id}/cbom`** (**à la demande**) | Hors scope stockage CBOM |
| **W7** | **CPM uniquement** : pas d’explore / persist tant que la **ligne la plus récente** (plus récent par **`created_at`**) n’est pas **`completed`** (`failed` ou en cours inclus) | — (pas de garde **W7** sur **`POST …/scan`**) | **`POST …/policies`** / **explore** → **`400`** (ex. **`LATEST_SCAN_NOT_COMPLETED`**) si la newest row n’est pas **`completed`** |
| **W8** | **Rescan (`POST …/scan`)** : **indépendant de W7**. Refus **uniquement** si un scan est **en cours** (`requested` / `started`) ; si le plus récent est **`failed`**, **nouveau scan autorisé** (nouvelle ligne) — sous réserve **W1** | **`POST …/scan`** → **`409`** **`SCAN_IN_PROGRESS`** si newest = **`requested`** ou **`started`** ; **pas** de **`409`** si newest = **`failed`**. **Ne pas** bloquer POST parce que la CPM est en attente d’un **`completed`** | — |

> **Ordre des gardes (wallet) :** **`POST …/scan`** : **W8** (pas de scan **en cours**) puis **W1** (**policy persistée** uniquement — **pas** draft seul après **IMM-W1-4**). **CPM** : **W7** (newest row = **`completed`**) puis **W2** sur **`scan_id`** ; **draft orphelin** (draft sur ancien **`scan_id`**) bloque explore/validate/persist jusqu’à **rebind manuel** sur **W2** (**§2.2.1 W1b**).

> **W7 vs W8 :** **W7** bloque la **CPM** tant que newest ≠ **`completed`** ; **W8** autorise le **rescan** si newest = **`failed`** (sous **W1** policy OK), même si la CPM reste bloquée. Ex. : **`completed` A** + **`failed` B** (B plus récent) → CPM **400**, **`POST …/scan`** **OK** si **W1** OK (pas de policy).

> **Re-scan / retry :** si le plus récent est **`failed`**, un nouveau **`POST …/scan`** (**W8** OK) est possible si **W1** est satisfait (**pas de policy persistée** sur la cible) — **sans** supprimer un **draft** plateforme. Pas d’obligation de **`DELETE`** le scan **`failed`** lorsque **W1** OK. Pendant un scan **en cours** : **W8** → **`POST …/scan`** refusé ; **W7** → CPM refusée.

> **W1b — draft orphelin + rebind (CPM, tranché 2026-06) :** après un rescan autorisé avec un **draft** plateforme existant, le draft peut rester lié à un **ancien** **`scan_id`** jusqu’à action explicite de l’utilisateur. **Explore / validate / persist** CPM sont **refusés** tant que le draft n’est pas **rebindé** sur le dernier scan **`completed`** de l’adresse (**W2**, `latest=true`). Rebind via **`POST /api/cpm/v1/drafts`** (même **`id`**, nouveau **`scan_id`**, payload conservé) — bouton UI explicite **« Rebind to last scan for this address »**. **`wallet_type`** du draft / contexte vs détail **W2** : **doit** matcher — sinon refus UI (pas de migration silencieuse). **Pas** de sauvegarde locale client. Détail UX : [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md).

> **Plans d’implémentation :** [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md), [`workplans/IMMUTABILITE_PR.md`](./IMMUTABILITE_PR.md) (CPM), [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md) — **découpage PR** ; **ce document** reste la **source de vérité** contrat.

### 2.2.1 Plans — usage, garde-fous et CBOM (**P1**, **G1–G4**)

**Décision produit :** les limites de plan (**`WalletScanLimit`**, **`EndpointScanLimit`**) comparent l’usage au nombre de scans **`completed` success** comptabilisés pour l’utilisateur. **Discovery** est propriétaire (**`GET /plans/usage`**, gardes **`POST …/scan`**, persistance **`scan.completed`**).

#### P1 — Comptage et monotonicité

| Règle | Détail |
|--------|--------|
| **Consommation** | **`wallet_scans_used`** / **`endpoint_scans_used`** = succès persistés (ledger **`scan_usage_events`**, une entrée par **`scan_id`**) |
| **Monotonicité** | **Ne diminuent jamais** via **`DELETE …/scans/{scan_id}`**, **`DELETE …/policies`**, **`DELETE …/drafts`** |
| **Visibilité UX** | **`GET /plans/usage`** expose aussi **`_*_visible`** (succès non effacés) et **`_*_deleted_by_user`** (`used - visible`) |
| **Non comptés** | **`failed`**, **`requested`/`started`**, rejet **`PLAN_LIMIT_EXCEEDED`** à la complétion |

#### G1 — Garde POST : places plausibles

Autoriser **`POST …/scan`** (wallet ou TLS) seulement si :

```text
successful + in_flight < limit
```

(où **`limit`** = limite plan ; plan illimité → pas de check G1).

- **`successful`** : ledger / succès terminal **`completed`**
- **`in_flight`** : exécutions owner **`requested` \| `started`** non terminales

Sinon → **`403`** (forme d’erreur plan existante).

#### G2 — Cap parallélisme global

Même en dessous du plafond G1 :

```text
in_flight < min(limit, 3)
```

Si **`limit = 0`** (illimité) → **`in_flight < 3`**.

Sinon → **`403`**.

#### G3 — Commit atomique à la complétion (anti-bypass)

À réception **`scan.completed`**, **avant** d’écrire un résultat riche :

1. Transaction : **`try_acquire_success_slot(user, kind)`** (atomique : `successful < limit`).
2. **Si slot acquis** : persister **`completed` success** + payload **`result`** complet + event ledger.
3. **Sinon** : persister **`failed`** avec **`PLAN_LIMIT_EXCEEDED`** :
   - **Conserver** **`address`** / **`url`** (métadonnées cible) ;
   - **Ne pas** persister de **`result`** exploitable (pas de réseaux, posture, exposition clé, etc.) — empêche le contournement du quota par lecture du détail scan.

Les échecs **`failed`** « métier » (scanner) et les rejets quota **ne consomment pas** le plan.

#### G4 — CBOM (**W6**)

**`GET …/wallets/scans/{scan_id}/cbom`** (et TLS) → **404** si le scan n’est pas **`completed` success** owner. Pas de CBOM pour in-flight, **`failed`**, stub quota.

#### Implémentation

Série **IMM-6b-1…8** — [`cafe-discovery/IMMUTABILITE_PR.md`](../../cafe-discovery/IMMUTABILITE_PR.md). **Ne pas** dériver **`_*_used`** du nombre de lignes visibles (**soft delete**).

**UX frontend :** [`cafe-frontend/IMMUTABILITE.md`](../../cafe-frontend/IMMUTABILITE.md) — **P1**, **W6**, **FE-IMM-0**.

**Exception (hors parcours courant) :** suppression de **compte** / RGPD — peut purger ledger et scans.

### 2.3 Lecture du catalogue des crypto policies

- **Chemins canoniques** : **§0.2** (cible **`/api/cpm/v1/policies/catalog|templates|instances`**) ou **§0.3** (rollout **alias policies de transition**). JWT requis (**`RouteClassAuthenticated`**). Implémentation actuelle : `internal/api/read_api.go`.

### 2.4 Instances de crypto policy persistées 

- **Chemins canoniques** : **`POST` \| `GET` \| `DELETE`** sur la ressource **instances** — **§0.2** (`/api/cpm/v1/policies`) ou **§0.3** (`alias CPM de transition policies`) ; **brouillons** **`…/drafts`** (**§0.2** / **§0.3**). JWT + scope propriétaire (`cafe-crypto-policy-mgt`, `internal/app/owner_routes.go`).
- **`DELETE …/policies?id=…`** (suffixe après préfixe **§0**) : **`204`** si l’instance existait et est supprimée ; **`404`** si inconnue / hors scope / **déjà supprimée** (**idempotence** — **§5.4.9**). **Même query **`id`** que **`GET`**. Pas de **`409`** sur cette route dans ce plan.
- **`DELETE …/drafts?id=…`** : **`204`** si le brouillon existait et est supprimé ; **`404`** si inconnu / hors scope / **déjà supprimé** (**idempotence** — **§5.4.9**). **Même query **`id`** que **`GET …/drafts?id=…`**. Supprime le brouillon plateforme pour satisfaire **W1** et débloquer **`POST …/scan`** (parcours **§2.2**). Ne supprime **pas** le scan Discovery référencé.
- Le corps d’écriture **`POST`** inclut **`id`**, **`scan_id`** (liaison **`scan_result`** Discovery / **Option A** CPM lorsque applicable), **`payload`** — affiner uniquement les **règles métier** (ex. **`scan_id` obligatoire** pour certains flux) et l’AUTH scan (AUTH-02).

---

## 3. Modèle conceptuel cible

```text
User (JWT)
  ├── wallets/              → GET liste des wallet_id CAFE           (2.1)
  ├── wallets/{wallet_id}   → GET détail | DELETE effacement owner   (2.1, 4.1)
  ├── wallets/scans/        → GET liste wallet (synopsis) + ?address [& chain_id]  (2.2)
  ├── wallets/scans/{scan_id} → GET détail | GET cbom | DELETE effacement wallet  (2.2, 4.2)
  ├── tls/scans/            → GET liste TLS (synopsis)                     (2.2)
  └── tls/scans/{scan_id}   → GET détail | DELETE effacement TLS           (2.2, 4.2)

  scan_id (UUID) ← alloué dès requested (acceptation Discovery) ; lifecycle jusqu’état terminal (§2.2) ;
                  payload résultat immutable après terminal ; AUTH / CPM §2.4 ;
                  effacement = action owner uniquement

Catalogue fichier (CPM)           ← GET …/policies/catalog|templates|instances (§0.2 ou §0.3, §2.3)

Politiques persistées + drafts    ← POST/GET/DELETE …/policies ; POST/GET …/drafts (§0.2 ou §0.3, §2.4)
```

**Principes :** le **`scan_id`** est la charnière entre **Discovery** (où le scan est observé / persisté) et **CPM** (où la politique est évaluée / matérialisée en instance). **Traçabilité** : **`scan_id` stable pour une ligne de résultat donnée** (identifiant de **cette** persistance ; une **nouvelle** exécution → **nouvelle** ligne → **nouveau** `scan_id`), **lifecycle** observable (**asynchrone CAFE**) puis **gel du payload de résultat** après **état terminal** (**§2.2**) — sans confondre **évolution des métadonnées d’exécution** et **mutation du résultat final**. Les listes **`…/wallets/scans`** et **`…/tls/scans`** exposent chacune un **synopsis** de **sa** famille ; le **détail** est **`GET …/wallets/scans/{scan_id}`** ou **`GET …/tls/scans/{scan_id}`**.

---

## 4. Surfaces API (alignées sur la convention **§0**)

Les **URL complètes** sont définies en **§0.1** (Discovery) et **§0.2** / **§0.3** (CPM). Les tableaux ci‑dessous utilisent la notation **`…/ressource`** = suffixe après le préfixe **public** retenu.

**Contrainte explicite :** **pas de rétrocompatibilité** ; les **anciennes entrées sont supprimées**, pas conservées ni servies en parallèle.

- **Points d’entrée HTTP** remplacés par ce modèle : **suppression effective** du handler / de la route / de la configuration edge concernés dès livraison du remplaçant (pas de synonyme pérenne de l’ancien chemin pour les anciens clients).
- **Données persistées** que le nouveau modèle n’expose plus ou qui violent les invariants fixés (ex. enregistrements sans `scan_id` lorsque le produit l’exige pour la persistance « par scan ») : traitement par **suppression** (éventuellement après export interne hors contrat produit si la conformité l’exige — hors périmètre de conservation applicative).

La coordination release (frontend, scripts, intégrations) reste nécessaire, mais elle **organise la coupure**, elle ne **maintient pas** les anciennes surfaces.

### 4.1 Wallets CAFE — **`GET …/wallets`** + **`GET|DELETE …/wallets/{wallet_id}`**

| Élément | Décision de travail |
|--------|---------------------|
| Liste | **`GET …/wallets`** — équivalent **`GET /api/discovery/v1/wallets`** (**§0.1**) ; liste des **identifiants** (ou synopsis + **`id`**) des wallets **CAFE** du user. |
| Détail | **`GET …/wallets/{wallet_id}`** — **`{wallet_id}`** = identifiant interne wallet CAFE (**pas** l’adresse Ethereum sauf doc explicite — **discouraged**). |
| Effacement | **`DELETE …/wallets/{wallet_id}`** — **`204`** \| **`404`** \| **`409`** (conflit métier **`WALLET_REFERENCED_BY_POLICY`** — **§5.4.8**) ; **idempotence** **`404`** sur second **`DELETE`** (**§5.4.8**). |
| Auth | JWT ; **owner-scoped**. |
| Frontière scan-only | Une adresse **uniquement** observée via **scan wallet** mais **sans** enregistrement wallet CAFE **n’apparaît pas** sous **`GET …/wallets`** (liste CAFE) ; elle est consultable via **`wallets/scans`** (**`?address=`** [, **`chain_id`**] ou **`GET …/wallets/scans/{scan_id}`**). Les exécutions **TLS** : **`tls/scans/`** — **§2.2**. |

### 4.2 Scans Discovery — **`…/wallets/scans`** (wallet) et **`…/tls/scans`** (TLS)

#### 4.2.1 Wallet — **`GET …/wallets/scans`** + **`GET|DELETE …/wallets/scans/{scan_id}`**

| Élément | Décision de travail |
|--------|---------------------|
| Liste | **`GET …/wallets/scans`** — **`200`** + **`items`**, **`total`**, **`limit`**, **`offset`**. **`ScanListItem`** wallet : **§2.2** ; **`status`** = **lifecycle uniquement** (OpenAPI : *status is lifecycle metadata, not the crypto posture result*). **Tri par défaut** : **`created_at`** desc, puis **`scan_id`** desc (tie-breaker) — **§2.2**. |
| Filtre **`address`** (multi-chaînes) | **`GET …/wallets/scans?address=0x…`** — **même enveloppe** ; **toutes chaînes** ; synopsis **`chain_ids`** quand applicable (**§2.2**). |
| Filtre **`address`** + **`chain_id`** | **`GET …/wallets/scans?address=…&chain_id=…`** — **même enveloppe** ; raffinement mono-chaîne (**§2.2**). **`GET …/wallets/scans?chain_id=…`** **sans** **`address`** → **`400 Bad Request`** (**§2.2**). **Pas** de **`/wallets/scans/0x…`** (adresse **uniquement** en query). |
| **Dernier scan `completed`** | **`GET …/wallets/scans?address=0x…&latest=true`** — dernier scan **`completed`** seulement (**§2.2 W2**). **`total: 0`** s’il n’existe **aucun** **`completed`** pour l’adresse (même si des lignes **`failed`** ou en cours existent). |
| **Garde CPM lifecycle (W7)** | Explore/persist : refus si le plus récent n’est pas **`completed`** (**`LATEST_SCAN_NOT_COMPLETED`**) — **§2.2 W7**. **Ne s’applique pas** à **`POST …/scan`**. |
| **Garde rescan (W8)** | **`POST …/scan`** : refus si le plus récent est **`requested`** / **`started`** (**`SCAN_IN_PROGRESS`**). **Indépendant** de **W7** — **§2.2 W8**. |
| Détail | **`GET …/wallets/scans/{scan_id}`** — **`scan_id`**, **`status`** (lifecycle), **`result`** métier wallet — **exemple JSON** ci‑dessous ; **404** si absent / hors scope. |
| Lifecycle + immutabilité | **`scan_id`** alloué en **`requested`** (**§2.2**) ; **`result`** immutable après terminal. **Re-scan** = **nouvelle ligne** + **nouveau **`scan_id`**** **seulement** si **aucune** policy **ni** **draft** CPM pour la **`target_address`** (**§2.2 W1**). |
| **POST scan** (wallet) | **`POST …/scan`** avec **`address`** : **`409`** si **policy persistée** pour la cible (**W1**) — **pas** si **draft** seul (**IMM-W1-4**) ; sinon flux existant (**§8.3**). |
| **CBOM** | **`GET …/wallets/scans/{scan_id}/cbom`** — **404** sauf scan **`completed` success** (**G4**, **IMM-6b-7**). |
| Effacement | **`DELETE …/wallets/scans/{scan_id}`** — **`204`** \| **`404`** \| **`409`** ; **`409`** **`SCAN_REFERENCED_BY_POLICY`** si policy référence ce **`scan_id`**. **Parcours utilisateur** : **`GET …/policies?scan_id=`** → **`DELETE …/policies?id=`** → **`DELETE`** scan (**§2.2 W3**, rationale **§4.2**). **Idempotence** : second **`DELETE`** → **`404`** (**§5.4.6**). |
| Auth | JWT ; **owner-scoped** (+ authz **`scan`** / AUTH‑02 où défini). |
| Ancien `liste wallet historique (hors v1)` (**`id`** = adresse) | **Supprimé** ; remplacé par la liste **synopsis** + **`scan_id`** + filtres **§2.2**. |

**Exemple indicatif — `GET …/wallets/scans`** (même schéma avec **`?address`**, **`&chain_id`**) :

```json
{
  "items": [
    {
      "scan_id": "550e8400-e29b-41d4-a716-446655440000",
      "target_address": "0x…",
      "chain_ids": [1, 8453],
      "created_at": "2026-05-11T12:34:56Z",
      "status": "completed"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

#### 4.2.2 TLS — **`GET …/tls/scans`** + **`GET|DELETE …/tls/scans/{scan_id}`**

| Élément | Décision de travail |
|--------|---------------------|
| Liste | **`GET …/tls/scans`** — **même enveloppe** pagination ; **`TLSListItem`** — **§2.2** ; **`status`** = lifecycle seul. **Pas** de **`address`** / **`chain_id`**. **Même tri par défaut** que **`…/wallets/scans`** — **§2.2**. |
| Détail | **`GET …/tls/scans/{scan_id}`** — DTO avec **`result`** métier TLS (schéma **distinct** du wallet en OpenAPI). |
| Effacement | **`DELETE …/tls/scans/{scan_id}`** — **`204`** \| **`404`** \| **`409`** ; corps **`SCAN_REFERENCED_BY_POLICY`** identique **§4.2** (bloc JSON sous **`DELETE …/wallets/scans/{scan_id}`**). |
| Ancien `liste TLS historique (hors v1)` (**`id`** = URL) | **Supprimé** ; remplacé par synopsis + **`scan_id`** + **`GET|DELETE …/tls/scans/{scan_id}`**. |

**Exemple indicatif — `GET …/tls/scans`** :

```json
{
  "items": [
    {
      "scan_id": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
      "endpoint": "https://example.com",
      "created_at": "2026-05-11T11:00:00Z",
      "status": "requested"
    }
  ],
  "total": 7,
  "limit": 20,
  "offset": 0
}
```

**`GET …/wallets/scans/{scan_id}` — corps indicatif (DTO détail wallet)** — la **posture** et le **verdict métier** vivent sous **`result`** ; **`status`** reste le **lifecycle** uniquement :

```json
{
  "scan_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": {
    "current_pq_posture": "…",
    "wallet_type": "…",
    "observations": []
  }
}
```

**OpenAPI (description à porter sur les schémas)** : *status is lifecycle metadata, not the crypto posture result.*

**OpenAPI (listes `GET …/wallets/scans` et `GET …/tls/scans`)** : *Default sort order: `created_at` descending, then `scan_id` descending as deterministic tie-breaker.*

**`DELETE …/wallets/scans/{scan_id}`** ou **`DELETE …/tls/scans/{scan_id}` — `409 Conflict`** — si au moins une **instance persistée CPM** (**`DELETE …/policies?id=…`** / préfixe **§0.2** ou **§0.3**) référence ce **`scan_id`** :

```json
{
  "error": "SCAN_REFERENCED_BY_POLICY",
  "message": "This scan is referenced by one or more persisted crypto policies."
}
```

**Rationale :** une **instance persistée** est une **décision métier** ; la supprimer en **cascade** avec le scan **détruirait** la **traçabilité**. Le client appelle d’abord **`DELETE …/policies?id=…`** pour chaque instance, puis **`DELETE …/wallets/scans/{scan_id}`** ou **`DELETE …/tls/scans/{scan_id}`** selon la famille d’exécution.

### 4.3 Lecture catalogue — chemins **§0.2** ou **§0.3**

| Élément | Décision de travail |
|--------|---------------------|
| Rôle | **Référentiel** fichier (catalogue + **templates** + **instances de référence**). |
| Chemins cible | **`GET /api/cpm/v1/policies/catalog`**, **`templates`**, **`instances`** (**§0.2**). |
| Chemins rollout | **`GET /api/cpm/v1/policies/catalog`**, **`templates`**, **`instances`** (**§0.3**). |
| Exploration | **`POST …/policies/decisions/explore`** — **§0.2** ou **§0.3** selon déploiement. |
| Auth | **JWT** requis ; détail **AUTH** = matrices existantes. |

### 4.4 Instances persistées + brouillons — chemins **§0.2** ou **§0.3**

| Élément | Décision de travail |
|--------|---------------------|
| Rôle | **`POST` \| `GET` \| `DELETE …/policies`** (instances, query **`id`** pour **GET**/**DELETE**) ; **`POST` \| `GET` \| `DELETE …/drafts`** (brouillons, query **`id`** pour **GET**/**DELETE**). Préfixes **§0.2** (`/api/cpm/v1/`) ou **§0.3** (`alias CPM de transition `). |
| Corps **`POST`** | **`{ id, scan_id?, payload }`** — règles **`scan_id`** / **AUTH-02** : **§2.4**, OpenAPI. |
| **`DELETE …/policies?id=…`** | **`204`** \| **`404`** uniquement (**idempotence** **§5.4.9**) ; **pas** de **`409`**. Ne supprime **pas** le **`scan_result`** Discovery (**§2.4**). |
| **`DELETE …/drafts?id=…`** | **`204`** \| **`404`** uniquement (**idempotence** **§5.4.9**) ; ne supprime **pas** le scan Discovery. Débloque **W1** pour **`POST …/scan`** après suppression du brouillon plateforme (**§2.2**). |
| Relecture par **`scan_id`** | **`GET …/policies?id=…`** (une instance) ; **`GET …/policies?scan_id=…`** (liste) — **§5.2** ; combinaison **`id`** + **`scan_id`** → **`400`**. |
| Traçabilité CPM | **`scan_id`** (**UUID**) aligné Discovery / **Option A** ; **`DELETE`** **`…/wallets/scans/{scan_id}`** ou **`…/tls/scans/{scan_id}`** bloqué **`409`** tant que référence — **§4.2**. |
| Contrat brouillon plateforme (gel) | **[`CPM_DRAFT_1_PR.md`](./CPM_DRAFT_1_PR.md)** — **§4.4.1** ci-dessous ; PRs **CPM-DRAFT-1A/B/C** puis adoption frontend **CPM-DRAFT-2**. |

#### 4.4.1 Platform drafts — contrat gelé (CPM-DRAFT-1)

> **Diagnostic validé :** le frontend envoyait `{ draft: … }` alors que le backend attend `{ id, scan_id?, payload }` → **`400`**. Pas de patch frontend temporaire : stabiliser le contrat CPM d’abord.

| Décision | Choix |
|----------|--------|
| **`draft_id`** | **Option A** — `id` **fourni par le client** (obligatoire sur chaque `POST`). Premier save : générer un UUID côté client ; updates : même `id`. |
| **`draft_version`** | **Non exposé** tant qu’il n’existe pas de version sémantique en store. Réponse `POST` minimale : `draft_id`, `saved_at`, `status: "server_draft"`. |
| **`GET /drafts`** | **Un seul** brouillon par query **`id`** (obligatoire). **Pas** de liste owner-scoped sans `id`. |
| **`DELETE /drafts?id=…`** | **`204`** si supprimé ; **`404`** si inconnu / hors scope / **déjà supprimé**. Effet métier **idempotent** (état final = plus de brouillon) malgré le statut HTTP sur le 2ᵉ delete. |

**`POST /api/cpm/v1/drafts` — `DraftUpsertRequest`**

```json
{ "id": "client-uuid", "scan_id": "uuid-optionnel", "payload": { } }
```

| Champ | Règle |
|-------|--------|
| `id` | Obligatoire, non vide. |
| `payload` | Obligatoire, objet JSON (peut être `{}`). |
| `scan_id` | Si présent : UUID valide. |
| `binding` | **Interdit** sur brouillons CPM-DRAFT-1 → **`400`** `DRAFT_BINDING_FORBIDDEN`. |
| `owner_user_id`, `tenant_id` | **Interdits** → **`400`** `DRAFT_OWNER_FIELDS_FORBIDDEN`. |

**Réponse 200 — `DraftUpsertResponse`**

```json
{ "draft_id": "…", "saved_at": "RFC3339", "status": "server_draft" }
```

**`GET /api/cpm/v1/drafts?id=…` — `DraftRecord`**

```json
{ "id": "…", "scan_id": "…?", "payload": { }, "created_at": "…", "updated_at": "…" }
```

**Erreurs structurées (brouillons)** — corps `{ code, message, details, request_id }` :

| `code` | HTTP | Quand |
|--------|------|--------|
| `DRAFT_ID_REQUIRED` | 400 | `POST` sans `id` ; `GET`/`DELETE` sans query `id` |
| `DRAFT_PAYLOAD_REQUIRED` | 400 | `POST` sans `payload` ou pas un objet |
| `DRAFT_SCAN_ID_INVALID` | 400 | `scan_id` présent mais pas un UUID |
| `DRAFT_BINDING_FORBIDDEN` | 400 | `binding` présent dans le corps d’un upsert draft |
| `DRAFT_OWNER_FIELDS_FORBIDDEN` | 400 | `owner_user_id` ou `tenant_id` dans le corps |
| `DRAFT_NOT_FOUND` | 404 | `GET` inconnu ; `DELETE` inconnu / déjà supprimé (owner-safe) |

Exemples **curl** : [`CPM_DRAFT_1_PR.md`](./CPM_DRAFT_1_PR.md) et [`README.md`](../README.md) § Platform drafts.

---

## 5. Décisions finales avant acceptation

### 5.1 Catalogue vs exploration

**Décision :** le **triple GET** catalogue couvre le besoin de **lecture statique** du référentiel de crypto policies pour cette release (suffixes après préfixe **§0.2** ou **§0.3**) :

- **`GET …/policies/catalog`**
- **`GET …/policies/templates`**
- **`GET …/policies/instances`**

Ces endpoints servent **uniquement** la lecture du référentiel : graphe, templates, instances de référence ou exemples. Ils ne portent **pas** de décision contextualisée par un scan utilisateur.

**Décision :** **`POST …/policies/decisions/explore`** reste le **seul** endpoint d’**exploration décisionnelle** pour cette release.

Rôle de **`explore`** :

- prendre un contexte utilisateur / scan / posture comme entrée ;
- retourner les chemins ou politiques applicables ;
- préparer la sélection frontend ;
- **ne pas** persister une instance finale.

Aucun autre verbe d’exploration n’est ajouté dans cette remise à plat. Si un besoin futur apparaît, il devra être ajouté comme endpoint **explicitement nommé** sous la même famille **`policies/decisions/...`**, sans modifier la sémantique du catalogue.

### 5.2 Règles sur `scan_id` pour `POST|GET|DELETE …/policies`

**Décision :** une instance persistée de crypto policy reste représentée par **`PolicyRecord`** et exposée via (préfixe **§0.2** ou **§0.3**) :

- **`POST …/policies`**
- **`GET …/policies?id=...`**
- **`DELETE …/policies?id=...`**

**Décision :** **`scan_id`** est **obligatoire** pour toute instance persistée issue d’un flux **Discovery → CPM**.

Cela inclut notamment :

- une policy créée après exploration depuis un **scan wallet** ;
- *(hors flux produit actuel)* une policy liée à un scan **TLS** — **pas** de parcours assessment/remediation TLS CPM pour cette release (**§5.4.6**) ;
- une policy destinée à servir de base à une **remediation** liée à un scan.

**Décision :** **`scan_id`** peut rester **absent** uniquement pour des objets **non** liés à un scan utilisateur, par exemple :

- instance de **référence** du catalogue ;
- **brouillon** purement manuel non encore attaché à un scan ;
- cas de **test** ou **fixture** explicitement hors flux Discovery → CPM.

Ces cas doivent être distingués dans **OpenAPI** et dans les **tests**. Ils ne doivent **pas** être acceptés silencieusement dans les parcours produit « par scan ».

**Décision :** ajouter **`GET …/policies?scan_id=...`** dans le **contrat cible**.

**Rationale :** le client doit pouvoir résoudre proprement un **`409`** **`SCAN_REFERENCED_BY_POLICY`** lors d’un **`DELETE …/wallets/scans/{scan_id}`** ou **`DELETE …/tls/scans/{scan_id}`**. Sans relecture par **`scan_id`**, le frontend ne peut pas lister les instances à supprimer avant de retenter l’effacement du scan.

Comportement attendu :

- **`GET …/policies?id=...`** retourne **une** instance ;
- **`GET …/policies?scan_id=...`** retourne une **liste** (paginée ou non, à figer OpenAPI) d’instances **owner-scoped** ;
- **`id`** et **`scan_id`** ne doivent **pas** être combinés dans la même requête ;
- si les deux sont fournis : **`400 Bad Request`**.

**Décision :** **`DELETE …/policies?id=...`** supprime l’instance persistée de la **vue produit** utilisateur, mais **ne supprime jamais** le scan Discovery référencé.

Effets :

- le scan reste disponible ;
- le prochain **`DELETE …/wallets/scans/{scan_id}`** peut réussir si plus aucune policy ne référence ce **`scan_id`** ;
- **aucune** cascade automatique policy → scan.

**Décision :** les anciennes lignes de policies qui violent le nouvel invariant **`scan_id`** sont **supprimées** dans le cadre de cette bascule, sauf obligation légale ou audit interne contraire.

Comme le document fixe déjà un principe **sans rétrocompatibilité** produit, ces lignes ne sont **pas** exposées sous un ancien contrat. Si un besoin d’audit existe, il est traité **hors** contrat API public par export interne ou journalisation.

### 5.3 Suppression systématique

**Décision :** le principe de **suppression systématique** est **accepté**.

Toute route, réponse ou donnée persistée correspondant à **l’ancien modèle** est **retirée** à la bascule.

Cela inclut notamment :

- l’ancien **la liste wallet historique** où **`id`** représente une **adresse** ;
- l’ancien **la liste TLS historique** où **`id`** représente une **URL** ;
- toute route de détail **ambiguë** où une adresse ou une URL pourrait être interprétée comme identifiant de scan ;
- **la façade policy-context historique** si les nouvelles routes **`wallets/scans`**, **`wallets/scans/{scan_id}`** et CPM **`policies?scan_id=`** couvrent le besoin frontend.

**Décision :** après coupure, les **routes hors contrat §0** ne sont **plus** servies par l’**edge**.

Comportement recommandé :

- route **retirée** de nginx / edge ;
- route **retirée** de l’**OpenAPI** publié ;
- tests **QA** mis à jour pour vérifier que l’ancien contrat n’est plus utilisé ;
- **pas** de handler de compatibilité durable.

### 5.4 Wallets CAFE, adresses observées, scans wallet et scans TLS

#### 5.4.1 Enums de lifecycle

**Décision :** **`status`** est une **enum fermée** côté backend.

Valeurs autorisées :

```text
requested
started
completed
failed
```

Aucune valeur **`unknown`** n’est **produite** par le backend. Le frontend peut rester **défensif** face à une valeur inconnue, mais **OpenAPI** doit documenter **uniquement** ces **quatre** valeurs pour cette release.

#### 5.4.2 Champs de synopsis wallet

**Décision :** **`ScanListItem`** wallet contient :

```json
{
  "scan_id": "uuid",
  "target_address": "0x...",
  "chain_ids": [],
  "created_at": "ISO-8601",
  "status": "requested|started|completed|failed"
}
```

**Règles :**

- **`scan_id`** obligatoire ;
- **`created_at`** obligatoire ;
- **`status`** obligatoire ;
- **`target_address`** obligatoire pour les scans wallet déclenchés par adresse ;
- **`chain_ids`** obligatoire, mais peut être **`[]`** tant que les chaînes ne sont pas encore connues ;
- après **`completed`**, **`chain_ids`** doit refléter les chaînes effectivement observées si l’information est disponible ;
- **`status`** reste **lifecycle-only**, jamais posture crypto.

#### 5.4.3 Champs de synopsis TLS

**Décision :** **`TLSListItem`** contient :

```json
{
  "scan_id": "uuid",
  "endpoint": "https://example.com",
  "created_at": "ISO-8601",
  "status": "requested|started|completed|failed"
}
```

**Règles :**

- **`endpoint`** obligatoire ;
- **pas** de **`address`** ;
- **pas** de **`chain_id`** ;
- **même tri** que les scans wallet : **`created_at`** desc, puis **`scan_id`** desc (**§2.2**).

#### 5.4.4 DTO détail `result` wallet

**Décision :** le détail wallet expose le résultat métier sous **`result`**.

Structure minimale cible :

```json
{
  "scan_id": "uuid",
  "status": "completed",
  "result": {
    "target_address": "0x...",
    "chain_ids": [1, 8453],
    "wallet_type": "eoa|smart_account|contract|unknown",
    "current_pq_posture": "pq_ready|hybrid|not_pq_ready|unknown",
    "observations": []
  }
}
```

Les enums exactes de **`wallet_type`** et **`current_pq_posture`** doivent être alignées avec les contrats existants CPM / Discovery, mais le principe est **figé** : ces champs vivent dans **`result`**, pas dans le synopsis de liste.

#### 5.4.5 DTO détail `result` TLS

**Décision :** le détail TLS expose un schéma **`result`** **distinct** du wallet.

Structure minimale cible :

```json
{
  "scan_id": "uuid",
  "status": "completed",
  "result": {
    "endpoint": "https://example.com",
    "tls_version": "TLS1.3",
    "cipher_suite": "...",
    "key_exchange": "...",
    "certificate_summary": {},
    "current_pq_posture": "pq_ready|hybrid|not_pq_ready|unknown",
    "observations": []
  }
}
```

Ce schéma est volontairement distinct du wallet. Les deux familles partagent **`scan_id`**, **`status`**, lifecycle, immutabilité et pagination, mais **pas** le même **`result`**.

#### 5.4.6 Scans TLS — Discovery vs CPM (périmètre produit actuel)

**Décision :** les scans **TLS** restent une responsabilité **Discovery** pour **historique**, **CBOM** (optionnel), **observation** et **inventaire de risque**. Le flux produit **CPM assessment/remediation** actuel (**Option A**, **W1–W8**) est **wallet-only** (scans **wallet / EVM** uniquement).

**TLS scope (tranché) :**

- **oui** : **`scan_id`** stable, historique, résultat terminal immutable, list/detail/delete, CBOM optionnel ;
- **oui** : inventaire de risque / observation / historique Discovery ;
- **non** (flux CPM produit actuel) : cible d’**assessment** ou de **remediation** CPM ; pas de migration/remediation TLS dans CPM pour cette release ;
- **défensif** : si un **`scan_id`** TLS est référencé par une policy persistée (cas technique ou futur), **`DELETE …/tls/scans/{scan_id}`** retourne **`409`** **`SCAN_REFERENCED_BY_POLICY`** — même corps d’erreur que wallet.

**Conséquence :**

- **`DELETE …/wallets/scans/{scan_id}`** → **`409`** si une policy référence ce scan **wallet** ;
- **`DELETE …/tls/scans/{scan_id}`** → **`409`** si une policy référence ce scan **TLS** (garde défensive) ;
- **W1**, **W2**, **W7**, **W8** et lookup **IMM-9b** : **wallet / EVM** uniquement — **pas** d’équivalent TLS sur le parcours CPM courant.

#### 5.4.7 Croisement wallet CAFE et adresse observée

**Décision :** un **wallet CAFE** et une **adresse observée** par scan restent **deux notions distinctes**.

**Règles :**

- une adresse observée dans **`GET …/wallets/scans?address=...`** ne crée **pas** automatiquement un wallet CAFE ;
- un wallet CAFE peut référencer une ou plusieurs adresses, mais cette relation est **produit** et doit être **explicite** ;
- la **promotion** d’une adresse observée en wallet CAFE **n’est pas** couverte par cette remise à plat ;
- **aucune** fusion automatique entre **`wallets`** et **`wallets/scans`**.

Si un futur besoin apparaît, il devra être ajouté via une action explicite, par exemple **`POST …/wallets`** ou **`POST …/wallets/{wallet_id}/addresses`** — **hors périmètre** de ce document.

#### 5.4.8 `DELETE` wallet CAFE

**Décision :** **`DELETE …/wallets/{wallet_id}`** **ne supprime pas** les scans associés.

**Rationale :** un scan est une **observation historique**. Le supprimer automatiquement lors de la suppression d’un wallet CAFE détruirait la **traçabilité**.

**Comportement :**

- le wallet CAFE est supprimé de la vue produit utilisateur ;
- les scans restent consultables via **`…/wallets/scans`** et **`…/wallets/scans/{scan_id}`** s’ils appartiennent au **même owner** ;
- si un scan portait un **`wallet_id`**, la relation peut être supprimée ou rendue **nullable** côté persistance ;
- le **`scan_id`** de chaque ligne de résultat de scan **déjà persistée** reste inchangé (**identifiant de cette ligne** ; une **future** exécution crée une **autre** ligne avec un **autre** `scan_id`) ;
- les policies CPM référencent le **`scan_id`**, pas nécessairement le **`wallet_id`**.

**Décision :** **`DELETE …/wallets/{wallet_id}`** retourne **`409`** **uniquement** si une instance CPM référence **explicitement** ce **`wallet_id`**. Si les policies référencent seulement des **`scan_id`**, alors la suppression du wallet **ne doit pas** être bloquée par ces policies.

Erreur recommandée si conflit :

```json
{
  "error": "WALLET_REFERENCED_BY_POLICY",
  "message": "This wallet is referenced by one or more persisted crypto policies."
}
```

#### 5.4.9 `DELETE` policies et audit

**Décision :** **`DELETE …/policies?id=...`** est une suppression **produit** **owner-scoped**.

**Comportement API :**

- **`204`** si supprimée ;
- **`404`** si inconnue, hors scope ou déjà supprimée ;
- **pas** de **`409`** ;
- ne supprime **pas** le scan lié ;
- ne supprime **pas** le wallet lié.

**Décision :** l’**audit** interne, la **journalisation** et les exigences **RGPD** ne changent **pas** le contrat HTTP de cette release. Si nécessaire, le backend peut conserver un événement interne ou une **tombstone** technique, mais cette donnée n’est **pas** exposée comme instance persistée **active** dans l’API produit.

### 5.5 Edge / nginx

**Décision :** les routes edge à publier sont celles de **§0**.

**Discovery cible** (**§0.1**) :

```text
GET    /api/discovery/v1/wallets
GET    /api/discovery/v1/wallets/{wallet_id}
DELETE /api/discovery/v1/wallets/{wallet_id}

GET    /api/discovery/v1/wallets/scans
GET    /api/discovery/v1/wallets/scans/{scan_id}
DELETE /api/discovery/v1/wallets/scans/{scan_id}

GET    /api/discovery/v1/tls/scans
GET    /api/discovery/v1/tls/scans/{scan_id}
DELETE /api/discovery/v1/tls/scans/{scan_id}

POST   /api/discovery/v1/scan
```

**CPM cible** (**§0.2**) :

```text
GET    /api/cpm/v1/policies/catalog
GET    /api/cpm/v1/policies/templates
GET    /api/cpm/v1/policies/instances
POST   /api/cpm/v1/policies/decisions/explore

POST   /api/cpm/v1/policies
GET    /api/cpm/v1/policies?id=...
GET    /api/cpm/v1/policies?scan_id=...
DELETE /api/cpm/v1/policies?id=...

POST   /api/cpm/v1/drafts
GET    /api/cpm/v1/drafts?id=...
DELETE /api/cpm/v1/drafts?id=...
```

La variante **§0.3** reste **uniquement** un mécanisme de **transition ingress / déploiement**. Elle ne doit **pas** être documentée comme contrat produit **long terme**.

---

## 6. Liens avec l’existant (sans préjuger d’impl)

| Aujourd’hui (indicatif) | Lecture utile |
|-------------------------|---------------|
| la liste wallet historique (actuel) | **À supprimer** à la bascule. Remplacement : **`GET …/wallets/scans`** (+ synopsis **`scan_id`**) + **`?address=`** [, **`chain_id`**] + **`GET` / `DELETE …/wallets/scans/{scan_id}`**. **Pas** d’**adresse** en path sous **`wallets/scans`**. Liste CAFE : **`GET …/wallets`** — **§0.1**, **§2.1**. |
| la liste TLS historique (actuel) | **À supprimer** à la bascule. Remplacement : **`GET …/tls/scans`** + **`GET` / `DELETE …/tls/scans/{scan_id}`** (**§0.1**, **§4.2.2**). |
| la façade policy-context historique | Si **`wallets/scans`** + filtres **`address`** / **`chain_id`** + **`GET …/wallets/scans/{scan_id}`** rendent cette façade redondante, **supprimer** ; sinon trancher au sign-off. |
| CPM catalogue (**§0.3**) ; cible **§0.2** | **Rollout** : **`GET /api/cpm/v1/policies/*`**. **Cible** : **`GET /api/cpm/v1/policies/*`**. Évolution = contrat ou contenu fichier ; migration de préfixe = **§0**. |
| CPM instances (**§0.3**) ; cible **§0.2** | **`POST`**, **`GET`** existants sur **`…/cpm/policies`** ; **ajouter** **`DELETE`** **`?id=…`** (owner) si absent (**§2.4**, **§4.4**). **Cible** : **`/api/cpm/v1/policies`**. |

---

## 7. Non-objectifs de ce document

- Implémenter des handlers, migrations DB, ou PRs.
- Définir une **politique de dépréciation** ou **conserver les routes hors contrat §0** après bascule (hors périmètre : à la livraison, **suppression** des entrées évincées).
- Spécifier les schémas JSON finaux **champ par champ** (détail d’annexe après sign-off — l’**inventaire des types** OpenAPI est en **§8.3**).
- Remplacer `CPM_OPTION_A_PR_PLAN.md` : ce document est **complémentaire** (chemins publics **§0** : Discovery **§0.1** — **`…/wallets`**, **`…/wallets/scans`**, **`…/tls/scans`**, **`POST …/scan`** ; CPM **§0.2** ou **§0.3** selon rollout).

---

## 8. Critères d’acceptation du document (gate avant code)

Le document est **acceptable** lorsque les éléments ci-dessous sont **validés explicitement** par **produit**, **architecture** et **sécurité**. Le **chapitre 5** fait foi pour les **décisions fonctionnelles** ; le présent chapitre sert de **checklist de sign-off** avant **OpenAPI** et **implémentation** (il ne rouvre pas les arbitrages de **§5**).

### 8.1 Surface API canonique validée

- **Discovery** cible validée :
  - `GET /api/discovery/v1/wallets`
  - `GET /api/discovery/v1/wallets/{wallet_id}`
  - `DELETE /api/discovery/v1/wallets/{wallet_id}`
  - `GET /api/discovery/v1/wallets/scans`
  - `GET /api/discovery/v1/wallets/scans/{scan_id}`
  - `DELETE /api/discovery/v1/wallets/scans/{scan_id}`
  - `GET /api/discovery/v1/tls/scans`
  - `GET /api/discovery/v1/tls/scans/{scan_id}`
  - `DELETE /api/discovery/v1/tls/scans/{scan_id}`
  - `POST /api/discovery/v1/scan`

- **CPM** cible validée :
  - `GET /api/cpm/v1/policies/catalog`
  - `GET /api/cpm/v1/policies/templates`
  - `GET /api/cpm/v1/policies/instances`
  - `POST /api/cpm/v1/policies/decisions/explore`
  - `POST /api/cpm/v1/policies`
  - `GET /api/cpm/v1/policies?id=...`
  - `GET /api/cpm/v1/policies?scan_id=...`
  - `DELETE /api/cpm/v1/policies?id=...`
  - `POST /api/cpm/v1/drafts`
  - `GET /api/cpm/v1/drafts?id=...`
  - `DELETE /api/cpm/v1/drafts?id=...`

- La variante rollout **§0.3** est **confirmée** comme **transition ingress / déploiement** uniquement, **sans** statut de contrat produit pérenne.
- La **suppression** de **§0.3** après bascule vers **§0.2** est **acceptée** (chemins **non** servis, **non** documentés comme supportés — **§0**).

### 8.2 Ambiguïtés de routage éliminées

- Les routes **`wallets/scans`** et **`wallets/scans/{scan_id}`** sont enregistrées **avant** **`wallets/{wallet_id}`**.
- Aucune **adresse Ethereum** n’est acceptée en **path** sous **`wallets/scans`**.
- Aucune **URL TLS** n’est acceptée en **path** sous **`tls/scans`**.
- **`GET …/wallets/scans?chain_id=...`** **sans** **`address`** retourne **`400 Bad Request`**.
- **`id`** et **`scan_id`** ne peuvent **pas** être combinés sur **`GET …/policies`** ; si les deux sont présents, la réponse est **`400 Bad Request`**.

### 8.3 Schémas OpenAPI à produire

**OpenAPI** doit définir explicitement :

- `ScanListItem`
- `TLSListItem`
- `WalletScanDetail`
- `TLSScanDetail`
- `ScanLifecycleStatus`
- `PostScanRequest`
- `PostScanResponse`
- `PolicyRecord`
- `PolicyListByScanIdResponse`
- `ErrorResponse`
- `SCAN_REFERENCED_BY_POLICY`
- `WALLET_REFERENCED_BY_POLICY`

**`PostScanResponse`** doit indiquer au minimum (exemple **wallet**) :

```json
{
  "scan_id": "uuid",
  "scan_family": "wallet",
  "status": "requested",
  "location": "/api/discovery/v1/wallets/scans/{scan_id}"
}
```

Exemple **TLS** :

```json
{
  "scan_id": "uuid",
  "scan_family": "tls",
  "status": "requested",
  "location": "/api/discovery/v1/tls/scans/{scan_id}"
}
```

### 8.4 Invariants fonctionnels validés

Les invariants suivants sont **acceptés** comme comportement produit :

- **`scan_id`** est alloué dès **`requested`**, avant publication vers le pipeline asynchrone.
- Une **nouvelle** exécution de scan crée **toujours** un **nouveau résultat de scan persisté** ; ce résultat a **son propre **`scan_id`** (nouvelle ligne, nouvel identifiant).
- Le **lifecycle** autorisé est :
  - `requested` → `started` → `completed`
  - `requested` → `started` → `failed`
  - `requested` → `failed`
- **`status`** est une **enum fermée** : `requested`, `started`, `completed`, `failed`.
- **`status`** est une métadonnée de **lifecycle**, jamais une **posture crypto**.
- **`result`** est **absent** ou **partiel** avant état **terminal**.
- **`result`** est **immutable** après état **terminal**.
- Les **listes** sont triées par défaut par **`created_at`** desc, puis **`scan_id`** desc.

### 8.5 Comportements `DELETE` validés

- **`DELETE …/wallets/{wallet_id}`** :
  - **`204`** si supprimé ;
  - **`404`** si inconnu, hors scope ou déjà supprimé ;
  - **`409`** **`WALLET_REFERENCED_BY_POLICY`** **seulement** si une policy référence **explicitement** **`wallet_id`**.

- **`DELETE …/wallets/scans/{scan_id}`** :
  - **`204`** si supprimé ;
  - **`404`** si inconnu, hors scope ou déjà supprimé ;
  - **`409`** **`SCAN_REFERENCED_BY_POLICY`** si une policy référence ce **`scan_id`**.

- **`DELETE …/tls/scans/{scan_id}`** : **même sémantique** que scan wallet.

- **`DELETE …/policies?id=...`** :
  - **`204`** si supprimée ;
  - **`404`** si inconnue, hors scope ou déjà supprimée ;
  - **jamais** de cascade vers Discovery ;
  - **pas** de **`409`** dans cette release.

- **`DELETE …/drafts?id=...`** :
  - **`204`** si supprimé ;
  - **`404`** si inconnu, hors scope ou déjà supprimé ;
  - **ne supprime pas** le scan Discovery référencé ;
  - **pas** de **`409`** dans cette release ;
  - permet de satisfaire **W1** et débloquer **`POST …/scan`** (rescan) après suppression du brouillon plateforme (**§2.2**).

### 8.6 Contrat CPM validé

- Le **triple GET** catalogue reste **lecture statique**.
- **`POST …/policies/decisions/explore`** est le **seul** endpoint d’**exploration décisionnelle**.
- **`explore`** **ne persiste rien**.
- **`POST …/policies`** persiste une **instance finale**.
- **`scan_id`** est **obligatoire** pour toute instance issue d’un flux **Discovery → CPM**.
- Tant que la **newest row** n’est pas **`completed`**, **aucune** nouvelle policy / **explore** (**§2.2 W7**). **`POST …/scan`** : **W8** — interdit si scan en cours ; si newest **`failed`**, retry autorisé sous **W1** (**§2.2 W8**, **W1**).
- **`scan_id`** pour **explore** / **persist** = **`scan_id`** du **`GET …/wallets/scans?address=&latest=true`** (dernier **`completed`** — **§2.2 W2**) ; sinon **`400`**.
- **`DELETE …/drafts?id=...`** : **`204`** \| **`404`** ; supprime le brouillon plateforme pour débloquer **W1** / **`POST …/scan`** (**§2.4**).
- **`GET …/policies?scan_id=...`** est **requis** pour le parcours utilisateur avant **`DELETE`** scan (**§2.2 W3**, **`SCAN_REFERENCED_BY_POLICY`**).
- **`GET …/policies?id=...`** retourne une **instance unique**.
- **`GET …/policies?scan_id=...`** retourne une **liste** **owner-scoped**.

### 8.7 Inventaire de suppression validé

Les routes suivantes sont **retirées** à la bascule :

- ancien **la liste wallet historique** avec **`id` = adresse** ;
- ancien **la liste TLS historique** avec **`id` = URL** ;
- toute route de détail **ambiguë** par adresse ou URL ;
- **la façade policy-context historique** si **confirmé** redondant après **`wallets/scans`**, **`wallets/scans/{scan_id}`** et **`policies?scan_id=`** (sinon : décision explicite au sign-off — **§6**).

Les **routes hors contrat §0** doivent être retirées :

- du **code** ;
- de l’**edge** / nginx ;
- de l’**OpenAPI** ;
- des **tests** frontend / backend ;
- des **scripts** et **fixtures**.

### 8.8 Tests attendus avant merge

Les **PRs** d’implémentation doivent couvrir au minimum :

- pagination des listes **wallet** et **TLS** ;
- tri **`created_at`** desc, puis **`scan_id`** desc ;
- **`chain_id`** sans **`address`** → **`400`** ;
- **`id` + `scan_id`** sur **`GET …/policies`** → **`400`** ;
- **`DELETE`** idempotence → second **`DELETE`** → **`404`** ;
- **`DELETE`** scan avec policy référente → **`409`** ;
- **`GET …/policies?scan_id=...`** retourne les policies **owner-scoped** ;
- re-scan **même cible** **sans** policy **ni** draft CPM (**W1**) → **nouvelle** ligne → nouveau **`scan_id`** ;
- **`POST …/scan`** avec **policy persistée** sur la cible → **`409`** (**W1**) — **pas** si **draft** seul (**IMM-W1-4**) ;
- **`DELETE …/drafts?id=...`** → **`204`** \| **`404`** ; débloque **W1** pour rescan ;
- explore/persist avec **`scan_id`** non latest → **`400`** **`SCAN_ID_NOT_LATEST_FOR_TARGET`** ;
- **W7** CPM : newest row (pas **`latest=true`**) ; **W2** CPM : **`GET …/wallets/scans?address=&latest=true`** (pas **`limit=1`** seul) ;
- aucune réponse API n’expose **`RUNNING`** ou **`running`** (**lifecycle** : **`started`**) ;
- TLS : hors scope assessment/remediation CPM produit actuel ; garde **`409`** défensive sur **`DELETE`** scan TLS si policy référence le **`scan_id`** ;
- **`GET …/wallets/scans?address=&latest=true`** → dernier **`completed`** ; peut être **`total: 1`** avec un **`failed`** plus récent dans l’historique ;
- **`POST …/scan`** avec newest **`requested`** / **`started`** → **`409`** **`SCAN_IN_PROGRESS`** (**W8**) ; avec newest **`failed`** → **autorisé** (nouvelle ligne, **W8** + **W1**) ;
- explore/persist avec newest non **`completed`** (`failed` ou en cours) → **`400`** (**W7**) ; cas **`completed` A + `failed` B** (B plus récent) : CPM **400** (**W7**), **`POST …/scan`** **OK** (**W8** + **W1** OK) ;
- **`GET …/wallets/scans/{scan_id}/cbom`** pour scan terminal owner ;
- **`DELETE`** scan : **`409`** si policy référence ; **`204`** après **`DELETE`** policies ;
- **`status`** **lifecycle-only** ;
- séparation **`result`** wallet / **`result`** TLS ;
- routes hors contrat §0 **supprimées** ou **non routées**.

### 8.9 Propriétaires de surfaces

- **Discovery** possède : wallets ; wallet scans ; TLS scans ; **`POST /scan`** ; lifecycle scan ; ownership des **scan results**.
- **CPM** possède : catalogue ; **`decisions/explore`** ; policies persistées ; drafts ; référence à **`scan_id`** ; refus de suppression scan via **`SCAN_REFERENCED_BY_POLICY`**.
- **`cafe-deploy` / edge** possède : exposition des **préfixes publics** ; migration **§0.3** → **§0.2** ; suppression des **routes hors contrat §0**.

### 8.10 Sign-off

Le document peut passer en phase **OpenAPI / PRs** lorsque les **validations** suivantes sont **obtenues** (coche explicite recommandée) :

- **produit** : parcours utilisateur et suppressions validés ;
- **architecture** : ressources, invariants et responsabilités validés ;
- **sécurité** : **JWT** + **owner-scope**, non-divulgation **cross-user**, matrices **AUTH-02 / `scan_id`** où applicable, erreurs validées ;
- **frontend** : listes, détails, **`explore`**, **`policies?scan_id`** et suppressions validés ;
- **déploiement** : edge cible et retrait des routes hors contrat §0 validés.

---

## 9. Prochaines étapes (après sign-off **§8**)

Une fois la **checklist §8** validée (et **§5** inchangé comme référence décisionnelle), enchaîner :

1. Rédiger **OpenAPI** — alignée sur **§8.3** et les chemins **§0** — **`ScanListItem`**, **`TLSListItem`**, DTO détail **`WalletScanDetail`** / **`TLSScanDetail`**, **`POST …/scan`**, **`POST`/`GET`/`DELETE …/policies`** + drafts (**§0**, **§2.2**, **§2.4**, **§4.2**, **§4.4**).
2. Découper en **PRs** (Discovery **§0.1** : **wallets**, **`wallets/scans`**, **`tls/scans`**, **`POST …/scan`**) ; **CPM** : **`DELETE …/policies`** si absent + **`scan_id`**, AUTH, **`DELETE`** scans **`409`** ; aligner **edge** vers **§0.2** quand prêt (**§0**).
3. Mettre à jour **Option A** (CPM / **`scan_id`**) et les **scripts** vers les **URLs acceptées**, en bloc (pas de filet pour l’ancien contrat).

