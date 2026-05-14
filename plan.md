# Plan de migration fctl v3 vers fctl v4

Statut: brouillon de travail pour la migration complete.

Source d'inventaire:

- commandes v3 sous `cmd/`;
- architecture v4 sous `docs/rfcs/0001-fctl-v4-architecture.md`;
- ADR v4 sous `docs/adr/`;
- design des commandes sous `docs/cli-v4/command-design.md`;
- manifeste de compatibilite sous `docs/cli-v4/compatibility-manifest.md`;
- migration de configuration sous `docs/cli-v4/migration-from-v3.md`.

Objectif du document:

- donner une vision exhaustive des commandes v3 a migrer;
- definir le nom canonique v4, les alias de compatibilite et les changements d'arguments;
- servir de base a la documentation utilisateur "v3 vers v4";
- servir de checklist de review pendant l'implementation;
- cadrer les tests unitaires, integration et end-to-end necessaires.

## Principes v4

### Modele utilisateur

La v4 ne doit plus supposer que l'utilisateur dispose d'un compte Formance Cloud ou d'un membership.

Les commandes parlent a un contexte:

- `stack`: stack locale ou self-hosted;
- `cloud`: controle plane Formance Cloud;
- `cloud-stack`: stack Formance Cloud selectionnee par organisation et stack.

L'authentification est une propriete du contexte, pas une hypothese globale du CLI.

### Resolution des versions API

Les commandes exposent une intention produit stable:

```bash
fctl ledger transactions list
```

Elles ne doivent pas exposer l'espace de noms SDK comme UX primaire:

```bash
# Non canonique
fctl ledger v2 transactions list
fctl ledger transactions list-v2
```

Le runtime v4 choisit l'API comme suit:

1. appeler `/versions`;
2. lire la version du composant, par exemple `ledger=2.3.4`;
3. convertir la version composant en namespaces API supportes via le manifeste de compatibilite;
4. intersecter avec les handlers disponibles dans le CLI;
5. choisir par defaut le namespace le plus recent compatible;
6. autoriser un override explicite avec `--api-version ledger=v1` ou `--api-version v1` dans une commande produit.

### Regles de naming

| Cas v3 | Regle v4 | Exemple |
| --- | --- | --- |
| Commandes avec `_` | kebab-case canonique, alias underscore deprecie | `transfer_initiation` -> `transfer-initiation` |
| `get` et `describe` melanges | `show` canonique, aliases `get`/`describe` si existants | `payments payments get` -> `payments payments show` |
| Abreviations peu claires | flag explicite canonique, alias court deprecie | `--ik` -> `--idempotency-key` |
| Commandes produit historiques avec `_` | grouper sous le service parent quand cela clarifie l'UX | `cloud_stacks` devient `cloud stacks`, avec alias deprecie |
| Flags API-specifiques | flag produit stable | `--account` reste `--account` meme si l'API v2 attend `address` |
| Flags v3 tres utilises | conserver comme alias cache ou deprecie | `--src` -> `--source`, `--dst` -> `--destination` |
| Saisie fichier | `--file <path>|-` quand l'objet principal n'est pas naturellement positionnel | `create <file>|-` peut rester alias |
| Pagination | `--cursor`, `--page-size` | `ledger transactions list --page-size 20` |
| Metadata | `--metadata key=value` repetable, `--metadata-file <path>|-` si utile | toutes familles |
| Confirmation | `--confirm` pour scripts, prompt interactif seulement si TTY | commandes destructives ou mutantes |

### Politique d'alias

La v4 etant une version majeure, elle peut casser des commandes, mais les chemins v3 les plus courants doivent rester disponibles comme aliases de migration quand cela ne complique pas l'implementation.

Regles:

- alias visibles pour les raccourcis utiles (`list`, `ls`);
- alias deprecie pour les anciens noms (`get`, `transfer_initiation`, `bank_accounts`, `update_status`);
- alias cache seulement pour les formes tres anciennes ou incoherentes;
- chaque alias deprecie conserve doit afficher un warning sur stderr qui indique la commande canonique a utiliser;
- erreur claire si un ancien flag ne peut pas etre mappe sans ambiguite;
- documentation generee a partir de cette table pour chaque suppression volontaire.

### Format global des flags

| v3 | v4 canonique | Compatibilite | Notes |
| --- | --- | --- | --- |
| `--profile`, `-p` | `--context`, `-c` si non conflictuel | `--profile` alias deprecie | La v3 utilise `-c` pour config dir, donc la v4 doit eviter une collision si `-c` reste config. |
| `--config-dir`, `-c` | `--config-dir` | garder `-c` seulement si `--context` n'a pas `-c` | Preferer `--context` sans short pour eviter ambiguite. |
| `--debug`, `-d` | `--debug` | garder `-d` | Active logs techniques sur stderr. |
| `--output`, `-o plain,json` | `--output`, `-o plain,json,yaml` | extension compatible | Les sorties structurees doivent etre stables. |
| `--insecure-tls` | `--insecure-tls` dans le contexte ou flag override | garder flag | Ne pas persister implicitement sans action explicite. |
| `--telemetry` | `--telemetry` / config | a confirmer | La v4 doit documenter opt-in/opt-out. |
| absent | `--non-interactive` | nouveau | Aucune question, erreurs propres. |
| absent | `--api-version` | nouveau | Pin produit ou commande, ex: `ledger=v2`. |
| absent | `--no-color` | nouveau | Necessaire pour CI et golden tests. |
| absent | `--quiet` | nouveau | Ne sortir que la donnee principale ou l'identifiant cree. |

Implementation v4:

- `--profile <name>` est conserve comme alias deprecie de `--context <name>`; fournir les deux produit une erreur explicite.
- `--config-dir <dir>` conserve le short `-c`, car `--context` n'a volontairement pas de short pour eviter l'ambiguite avec la v3;
- `--insecure-tls` est conserve comme override runtime explicite et non persistant; il configure le client HTTP partage par `/versions`, les SDK stack, les SDK Cloud et les flux `client_credentials`.

## Migration configuration et session

| v3 | v4 canonique | Changements | Tests critiques |
| --- | --- | --- | --- |
| `fctl login --membership-uri <url>` | `fctl auth login cloud --cloud-url <url>` | Cloud devient un provider d'auth, pas une condition pour la stack. | migration de token, erreurs sans browser, non-interactif. |
| aucun equivalent local propre | `fctl auth login token --token <token>` | Pour self-hosted et CI. | secret jamais ecrit en clair dans config si keyring disponible. |
| aucun equivalent local propre | `fctl auth login client-credentials --issuer-url --client-id --client-secret` | Auth machine-to-machine. | renouvellement, expiration, erreurs OAuth. |
| aucun equivalent local propre | `fctl auth login oidc --issuer-url --client-id` | OIDC generique. | scopes, device flow si supporte. |
| aucun equivalent local propre | `fctl auth login none` | Local/dev sans auth. | refuse sur contexte non local sauf confirmation explicite. |
| `fctl profiles list` | `fctl context list` | `profiles` alias deprecie. | format table/json/yaml stable. |
| `fctl profiles show <name>` | `fctl context show <name>` | meme argument. | masque les secrets. |
| `fctl profiles use <name>` | `fctl context use <name>` | current context. | ecriture atomique. |
| `fctl profiles delete <name>` | `fctl context delete <name>` | refuse si current sans `--force`. | deletion config + references credentials. |
| `fctl profiles rename <old> <new>` | `fctl context rename <old> <new>` | meme forme. | conserve current si necessaire. |
| `fctl profiles reset <name>` | `fctl context unset-defaults <name> --confirm` | Clarifie le reset comme suppression des defaults de contexte uniquement; ne supprime jamais les credentials. Alias deprecie `profiles reset`. | confirmation et non-interactif. |
| `fctl profiles set-default-organization <org>` | `fctl context set <name> --organization <org>` | Peut prendre current context par defaut. | validation kind cloud/cloud-stack. |
| `fctl profiles set-default-stack <stack>` | `fctl context set <name> --stack <stack>` | Peut prendre current context par defaut. | validation kind cloud-stack. |
| aucun equivalent v3 | `fctl context create stack <name> --stack-url <url> [--auth ...]` | Base self-hosted/local. | config schema, auth method. |
| aucun equivalent v3 | `fctl context create cloud <name> --cloud-url <url>` | Controle plane Cloud. | aucun stack requis. |
| aucun equivalent v3 | `fctl context create cloud-stack <name> --cloud-url --organization --stack` | Stack Cloud data-plane. | resolution d'URL stack. |
| aucun equivalent v3 | `fctl target inspect` | Montre target, auth, `/versions`, API choisies. | mock `/versions`. |
| aucun equivalent v3 | `fctl config migrate-v3` | Importe les profiles v3 sans les modifier. | fixtures v3, keyring fake, idempotence. |
| `fctl prompt` | `fctl setup` ou `fctl context wizard` | Garder `prompt` alias cache/deprecie. | ne jamais bloquer en `--non-interactive`. |
| `fctl version` | `fctl version` | Ajouter build metadata v4. | stdout stable. |
| `fctl ui` | `fctl ui [--print]` | Garder pour les contextes Cloud uniquement; `--print` donne une sortie scriptable sans navigateur. | detection browser/TTY. |

Implementation v4:

- `auth login token` met a jour le contexte selectionne et peut stocker le token dans un `--credential-dir` explicite via `--token` ou `--token-stdin`;
- `auth login client-credentials` met a jour le contexte selectionne et peut stocker le secret client dans un `--credential-dir` explicite via `--client-secret` ou `--client-secret-stdin`;
- `auth login none` desactive l'auth sur un contexte `stack`; sur `cloud`/`cloud-stack`, `--confirm` est requis pour eviter une desactivation accidentelle;
- `auth status` affiche la methode d'auth du contexte courant sans exposer les secrets;
- `auth token` imprime le token d'acces resolu pour les contextes authentifies, afin de faciliter CI et debug;
- `auth logout --confirm` supprime les credentials stockes localement quand ils utilisent un ref gere par le CLI, puis repasse le contexte en `none`.
- `context unset-defaults [name] --confirm` supprime les defaults du contexte sans toucher a l'auth ni aux credentials; les aliases deprecies `profiles reset`, `profiles set-default-organization` et `profiles set-default-stack` restent disponibles pour les migrations peu couteuses.
- `ui [--print]` reste disponible sur les contextes `cloud`/`cloud-stack`; il lit `/_info.consoleURL`, ouvre le navigateur seulement en mode interactif, et refuse les contextes `stack`.

## Mapping commandes Cloud

Les commandes Cloud restent sous `cloud`, mais elles doivent utiliser un contexte `cloud` ou `cloud-stack`. Elles ne doivent pas etre requises pour utiliser les produits stack (`ledger`, `payments`, etc.) contre une stack locale.

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `cloud generate-personal-token` | `cloud personal-tokens create` | Sortir le token en `--output json` sans decoration. | Garder ancien nom alias si simple. |
| `cloud me info` | `cloud me show` | `info` alias. | |
| `cloud me invitations list` | identique | pagination normalisee si disponible. | |
| `cloud me invitations accept <id>` | identique | `--confirm` si action irreversible. | |
| `cloud me invitations decline <id>` | identique | `--confirm` si action irreversible. | |
| `cloud organizations create <name> --default-stack-role --default-organization-role` | identique | flags kebab-case deja OK; valider enums. | |
| `cloud organizations list` | identique | `--cursor/--page-size` si API le supporte, sinon adapter page interne. | |
| `cloud organizations describe <organizationId>` | `cloud organizations show <organization-id>` | alias `describe`; argument kebab dans docs. | |
| `cloud organizations update <organizationId> --name --default-policy-id` | `cloud organizations update <organization-id>` | flags repetables documentes. | |
| `cloud organizations delete <organization-id>` | identique | `--confirm` obligatoire en non-interactif. | |
| `cloud organizations history` | identique | Sortie evenement structuree. | |
| `cloud organizations applications list` | identique | | |
| `cloud organizations applications show <application-id>` | identique | | |
| `cloud organizations authentication-provider show` | `cloud organizations authentication-provider show` | | |
| `cloud organizations authentication-provider configure <type> <name> <client-id> <client-secret>` | `cloud organizations authentication-provider configure --type --name --client-id --client-secret` | Eviter secret positionnel dans shell history; accepter ancien format alias. | Secret via prompt ou `--client-secret-stdin`. |
| `cloud organizations authentication-provider delete` | identique | `--confirm`. | |
| `cloud organizations invitations list` | identique | | |
| `cloud organizations invitations send <email>` | identique | flags role/policy si presents dans API. | |
| `cloud organizations invitations delete <id>` | identique | `--confirm`. | |
| `cloud organizations oauth-clients create` | identique | preferer flags explicites ou `--file`. | |
| `cloud organizations oauth-clients list` | identique | | |
| `cloud organizations oauth-clients show <client_id>` | `cloud organizations oauth-clients show <client-id>` | alias underscore. | |
| `cloud organizations oauth-clients update <clientId>` | `cloud organizations oauth-clients update <client-id>` | normaliser argument. | |
| `cloud organizations oauth-clients delete <client_id>` | `cloud organizations oauth-clients delete <client-id>` | `--confirm`. | |
| `cloud organizations policies create <name>` | identique | scopes via `--scope` repetable. | |
| `cloud organizations policies list` | identique | | |
| `cloud organizations policies show <policy-id>` | identique | | |
| `cloud organizations policies update <policy-id>` | identique | | |
| `cloud organizations policies delete <policy-id>` | identique | `--confirm`. | |
| `cloud organizations policies add-scope <policy-id> <scope-id>` | identique | | |
| `cloud organizations policies remove-scope <policy-id> <scope-id>` | identique | `--confirm` si necessaire. | |
| `cloud organizations users list` | identique | | |
| `cloud organizations users show <user-id>` | identique | | |
| `cloud organizations users link <user-id>` | identique | | |
| `cloud organizations users unlink <user-id>` | identique | `--confirm`. | |
| `cloud regions create` | identique | payload flags ou `--file`. | |
| `cloud regions list` | identique | | |
| `cloud regions show` | identique | | |
| `cloud regions delete` | identique | `--confirm`. | |
| `cloud apps create` | identique | Clarifier si app appartient org/stack/contexte. | |
| `cloud apps list` | identique | | |
| `cloud apps show` | identique | | |
| `cloud apps delete` | identique | `--confirm`. | |
| `cloud apps deploy` | identique | Documenter source: cwd, image, manifest. | |
| `cloud apps runs list` | identique | | |
| `cloud apps runs show` | identique | | |
| `cloud apps runs logs` | identique | `--follow`, `--since`, `--tail` si API. | |
| `cloud apps versions list` | identique | | |
| `cloud apps versions show` | identique | | |
| `cloud apps versions show-manifest` | `cloud apps versions manifest` | alias `show-manifest`. | |
| `cloud apps versions show-archive` | `cloud apps versions archive show` ou garder `show-archive` | Choisir selon API; documenter. | |
| `cloud apps versions archive` | identique si existant | `--confirm`. | |
| `cloud apps variables list` | identique | | |
| `cloud apps variables create` | identique | Secret via stdin/prompt si sensible. | |
| `cloud apps variables delete` | identique | `--confirm`. | |

Implementation v4 initiale:

- `cloud apps` est implemente sous le contexte `cloud`/`cloud-stack` et utilise un client deployserver separe du client membership;
- `--organization` reste optionnel avec un contexte `cloud-stack` et obligatoire avec un contexte `cloud`;
- `--deploy-url` permet de cibler un deployserver non-production sans changer l'URL membership du contexte Cloud;
- les commandes destructives ajoutees dans cette tranche exigent `--confirm` quand le plan le demande;
- `cloud apps variables create` accepte `--value-stdin` pour eviter d'exposer une variable sensible dans l'historique shell;
- `cloud apps versions manifest` est le nom canonique, avec `show-manifest` alias deprecie cache.
- `cloud apps versions archive show` est le nom canonique pour lire l'archive, avec `show-archive` alias deprecie cache.

Points bloques / non applicables avec le SDK actuel:

- `cloud personal-tokens create` n'est pas implemente pour l'instant: la v3 depend de claims Cloud, de `EnsureStackAccess`, puis d'un token exchange contre l'Auth de la stack. Le runtime v4 ne porte pas encore le modele d'acces Cloud stack/cache de token necessaire, et il ne faut pas recreer une dependance Cloud pour les commandes stack locales.
- `cloud apps versions archive` comme action mutante n'est pas expose par le deployserver SDK actuel; seule la lecture de l'archive est disponible et exposee via `cloud apps versions archive show`.

## Mapping Cloud stacks lifecycle

Les commandes `stack` v3 sont Cloud-control-plane. En v4, elles doivent etre clairement distinguees des commandes qui parlent a une stack data-plane.
Le chemin canonique v4 est `cloud stacks ...`, car ces operations appartiennent naturellement au controle plane Cloud.
L'ancien chemin v4 intermediaire `cloud_stacks ...` et les anciens chemins v3 `stacks ...` et `stack ...` restent des aliases deprecies pendant la v4 quand ils sont peu couteux a maintenir, avec warning indiquant la commande `cloud stacks ...`.
Ces aliases pourront etre supprimes en v5 ou dans une version mineure ulterieure si on decide de durcir la migration.

Implementation v4:

- `cloud stacks history <stack-id>` est implemente avec le meme service d'audit que `cloud organizations history`, en imposant le filtre `stackId` au niveau membership.
- `target proxy --port <port>` est implemente pour les contextes `stack` directs; le proxy Cloud stack reste separe tant que la resolution d'URI data-plane Cloud n'est pas dans le runtime v4.

| v3 | v4 canonique | Changements | Notes |
| --- | --- | --- | --- |
| `stack create` | `cloud stacks create` | `cloud_stacks create`, `stack create` et `stacks create` aliases deprecies avec warning. | Ne doit pas exister pour contexte `stack` local. |
| `stack list` | `cloud stacks list` | aliases deprecies avec warning. | |
| `stack show` | `cloud stacks show <stack-id>` | aliases deprecies avec warning. | |
| `stack update` | `cloud stacks update <stack-id>` | aliases deprecies avec warning. | |
| `stack delete` | `cloud stacks delete <stack-id>` | `--confirm`; aliases deprecies avec warning. | |
| `stack enable` | `cloud stacks enable <stack-id>` | aliases deprecies avec warning. | |
| `stack disable` | `cloud stacks disable <stack-id>` | `--confirm`; aliases deprecies avec warning. | |
| `stack restore` | `cloud stacks restore <stack-id>` | `--confirm`; aliases deprecies avec warning. | |
| `stack upgrade` | `cloud stacks upgrade <stack-id>` | `--confirm`, afficher target version; aliases deprecies avec warning. | |
| `stack history` | `cloud stacks history <stack-id>` | aliases deprecies avec warning. | |
| `stack proxy` | `target proxy` ou `cloud stacks proxy <stack-id>` | aliases deprecies avec warning. | Clarifier usage: proxy data-plane vs Cloud. |
| `stack users list` | `cloud stacks users list <stack-id>` | aliases deprecies avec warning. | |
| `stack users link <user-id>` | `cloud stacks users link <stack-id> <user-id>` | stack explicite ou contexte courant; aliases deprecies avec warning. | |
| `stack users unlink <user-id>` | `cloud stacks users unlink <stack-id> <user-id>` | `--confirm`; aliases deprecies avec warning. | |
| `stack modules list` | `cloud stacks modules list <stack-id>` | aliases deprecies avec warning. | |
| `stack modules enable` | `cloud stacks modules enable <stack-id> <module>` | aliases deprecies avec warning. | |
| `stack modules disable` | `cloud stacks modules disable <stack-id> <module>` | `--confirm`; aliases deprecies avec warning. | |

## Mapping Ledger

Le ledger est la premiere famille a beneficier de la resolution API automatique. Les commandes v4 doivent construire des inputs canoniques, puis laisser `internal/runtime` choisir `ledger.v1`, `ledger.v2`, `ledger.v3`, etc.

Regles de flags ledger:

- `--ledger` reste le selecteur produit, avec defaut depuis le contexte;
- `--account` reste le terme CLI canonique pour une adresse de compte;
- si une API appelle le champ `address`, l'adapter v4 fait la traduction;
- `--src` et `--dst` deviennent aliases deprecies de `--source` et `--destination`;
- les timestamps acceptent RFC3339 et doivent produire des erreurs explicites;
- les commandes qui n'existent qu'en v3+ doivent etre visibles avec une note `requires ledger API v3+`.

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `ledger --ledger <name>` | `ledger --ledger <name>` | Defaut depuis `context.defaults.ledger`. | Pas de Cloud obligatoire. |
| `ledger list` | identique | | Liste des ledgers si API exposee. |
| `ledger create <name> --bucket --features --metadata` | identique | `--metadata` repetable, `--feature` repetable; `--confirm` si destructif non applicable. | Adapter selon API. |
| `ledger import <ledger name> <file path> --input --resume` | `ledger import <ledger> --file <path>|-` | ancien positionnel garde; clarifier `--input` vs `--file`. | Tester reprise/resume. |
| `ledger export --output <file>` | `ledger export --file <path>|-` | `--output` global ne doit pas entrer en conflit; utiliser `--file` pour fichier. | `--output json` reste rendu CLI. |
| `ledger server-infos` | `ledger info` | alias `server-infos`. | |
| `ledger stats` | identique | | |
| `ledger send [source] <destination> <amount> <asset> --metadata --reference` | `ledger transactions send --source --destination --amount --asset` | garder `ledger send` alias; source positionnelle depreciee. | Evite ambiguite des positionnels. |
| `ledger set-metadata <ledger-name> key=value...` | identique | ajouter `--metadata-file`. | |
| `ledger delete-metadata <ledger-name> <key>` | identique | `--confirm` non necessaire si API idempotente, a verifier. | |
| `ledger accounts list --address --metadata --page-size` | `ledger accounts list --account --metadata --page-size` | `--address` alias deprecie de `--account`; l'adapter traduit vers `address` si l'API l'attend. | `--account` reste le vocabulaire CLI canonique. |
| `ledger accounts show <address>` | identique | | |
| `ledger accounts set-metadata <address> key=value...` | identique | | |
| `ledger accounts delete-metadata <address> <key>` | identique | | |
| `ledger transactions list --account --dst --src --reference --metadata --page-size --start --end` | `ledger transactions list --account --destination --source --reference --metadata --page-size --start --end` | aliases `--dst`, `--src`; adapter `account/address` selon API. | Deja commence en v4, etendre avec tous filtres. |
| aucun equivalent v3 direct | `ledger transactions count --account --destination --source --reference` | retourne le header API `Count`; aliases `--dst`, `--src`. | Commande read-only exposee par les API Ledger v1/v2. |
| `ledger transactions show <transaction-id>` | identique | ID type string cote CLI, adapter int/uuid selon API. | |
| `ledger transactions num -|<filename>` | `ledger transactions run-script --file <path>|-` ou `ledger transactions create --script-file <path>|-` | ne pas mapper vers `count`; garder `num` alias deprecie uniquement pour l'execution Numscript. | Nom canonique a choisir avant implementation. |
| `ledger transactions revert <transaction-id> --at-effective-date --force` | identique | `--force` devient alias ou complement de `--confirm`; date RFC3339. | |
| `ledger transactions set-metadata <transaction-id> key=value...` | identique | | |
| `ledger transactions delete-metadata <transaction-id> <key>` | identique | | |
| `ledger volumes list --pit --oot --use-insertion-date --group-by --address --metadata --cursor --page-size` | `ledger volumes list --start-time --end-time --use-insertion-date --group-by --account --metadata --cursor --page-size` | `--address` alias deprecie de `--account`, `--oot` alias de `--start-time`, `--pit` alias de `--end-time`; `--group-by` enum validee. | |

Implementation v4:

- le nom canonique retenu est `ledger transactions run-script --file <path>|-`;
- `ledger transactions num <file>` est conserve comme alias deprecie avec warning;
- la commande reutilise la resolution API Ledger et mappe vers `createTransaction` v1/v2 avec payload Numscript.
- `ledger accounts query <query-id> --schema-version <version>` est implemente via `v2RunQuery`;
- la commande reste product-oriented: elle force le resource template `accounts`, accepte `--var key=value`, et garde `--api-version` comme override technique uniquement.
- `ledger set-metadata <ledger> [key=value]... --metadata-file <path>|-` accepte un objet JSON de metadata et laisse les `key=value` explicites surcharger le fichier.

Commandes nouvelles possibles si l'API Ledger v3 les expose:

| Nouvelle commande | Condition | Comportement si cible trop ancienne |
| --- | --- | --- |
| `ledger transactions explain <id>` | `ledger API v3+` | erreur `requires ledger API v3+`, avec version cible courante. |
| `ledger schemas list/show/insert` | selon manifeste | commande visible, validation runtime. |
| `ledger accounts query` | selon manifeste | proposer `--api-version` seulement comme override technique. |

## Mapping Payments

Regles:

- utiliser kebab-case pour les resources composees;
- conserver les anciens chemins avec underscores comme aliases deprecies;
- les payloads JSON passent par `--file <path>|-` tout en acceptant l'ancien positionnel;
- `get` devient `show`;
- les connecteurs gardent leurs noms metier exacts.

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `payments versions` | `payments versions` ou `target inspect --product payments` | Garder pour compat si utile. | Le runtime centralise deja `/versions`. |
| `payments accounts create <file>|-` | `payments accounts create --file <path>|-` | ancien positionnel alias. | Valider JSON par schema si possible. |
| `payments accounts list` | identique | pagination et filtres normalises. | |
| `payments accounts get <accountID>` | `payments accounts show <account-id>` | alias `get`; argument kebab dans docs. | |
| `payments accounts balances <accountID>` | `payments accounts balances <account-id>` | | |
| `payments bank_accounts create <file>|-` | `payments bank-accounts create --file <path>|-` | alias `bank_accounts`. | |
| `payments bank_accounts list` | `payments bank-accounts list` | | |
| `payments bank_accounts get <bankAccountID>` | `payments bank-accounts show <bank-account-id>` | alias `get`. | |
| `payments bank_accounts forward <bankAccountID> <connectorID>` | `payments bank-accounts forward <bank-account-id> <connector-id>` | | |
| `payments bank_accounts update-metadata <bankAccountID> key=value...` | `payments bank-accounts set-metadata <bank-account-id> key=value...` | `update-metadata` alias si API reste ainsi. | Harmoniser avec autres produits. |
| `payments payments create <file>|-` | `payments payments create --file <path>|-` | ancien positionnel alias. | Nom double conserve mais documenter. |
| `payments payments list` | identique | | |
| `payments payments get <paymentID>` | `payments payments show <payment-id>` | alias `get`. | |
| `payments payments set-metadata <paymentID> key=value...` | `payments payments set-metadata <payment-id> key=value...` | | |
| `payments pools create <file>|-` | `payments pools create --file <path>|-` | | |
| `payments pools list` | identique | | |
| `payments pools get <poolID>` | `payments pools show <pool-id>` | alias `get`. | |
| `payments pools delete <poolID>` | `payments pools delete <pool-id>` | `--confirm`. | |
| `payments pools add-account <poolID> <accountID>` | `payments pools add-account <pool-id> <account-id>` | | v3 file name is `add_accounts.go` but command is singular. |
| `payments pools remove-account <poolID> <accountID>` | `payments pools remove-account <pool-id> <account-id>` | `--confirm` if destructive. | |
| `payments pools update-query <poolID> <file>|-` | `payments pools update-query <pool-id> --file <path>|-` | | |
| `payments pools balances <poolID> <at>` | `payments pools balances <pool-id> --at <time>` | old positional `<at>` alias. | |
| `payments pools latest-balances <poolID>` | identique | | |
| `payments tasks show` | `payments tasks show <task-id>` | verifier v3 args exacts. | |
| `payments transfer_initiation create <file>|-` | `payments transfer-initiation create --file <path>|-` | alias underscore. | |
| `payments transfer_initiation list` | `payments transfer-initiation list` | | |
| `payments transfer_initiation get <transferID>` | `payments transfer-initiation show <transfer-id>` | alias `get`. | |
| `payments transfer_initiation approve <transferInitiationID>` | `payments transfer-initiation approve <transfer-initiation-id>` | | |
| `payments transfer_initiation reject <transferInitiationID>` | `payments transfer-initiation reject <transfer-initiation-id>` | | |
| `payments transfer_initiation retry <transferID>` | `payments transfer-initiation retry <transfer-id>` | | |
| `payments transfer_initiation reverse <transferID> <file>|-` | `payments transfer-initiation reverse <transfer-id> --file <path>|-` | | |
| `payments transfer_initiation delete <transferID>` | `payments transfer-initiation delete <transfer-id>` | `--confirm`. | |
| `payments transfer_initiation update_status <transferID> <status>` | `payments transfer-initiation update-status <transfer-id> <status>` | alias underscore. | validate status enum. |

### Connecteurs Payments

Connecteurs v3 inventories:

- `adyen`
- `atlar`
- `bankingcircle`
- `coinbaseprime`
- `column`
- `currencycloud`
- `fireblocks`
- `generic`
- `increase`
- `mangopay`
- `modulr`
- `moneycorp`
- `plaid`
- `powens`
- `qonto`
- `stripe`
- `tink`
- `wise`

| v3 | v4 canonique | Changements |
| --- | --- | --- |
| `payments connectors list` | identique | sortie stable, inclure type, id, status. |
| `payments connectors uninstall` | `payments connectors uninstall <connector-id>` | `--confirm`. |
| `payments connectors install <connector> <file>|-` | `payments connectors install <connector> --file <path>|-` | garder ancien positionnel. |
| `payments connectors update-config <connector> <file>|- --connector-id <id>` | `payments connectors config update <connector-id> --file <path>|-` | l'identifiant cible est toujours le connector ID; le type connector v3 devient un detail d'adapter ou une option de compatibilite. |
| `payments connectors update-config get-config --connector-id <id>` | `payments connectors config show <connector-id>` | la lecture de config cible un connector ID en v4. |

Decisions d'implementation:

- `install` prend un type de connecteur (`stripe`, `qonto`, etc.) et retourne un connector ID;
- `config update` prend toujours un connector ID;
- les anciens sous-chemins par type, comme `update-config stripe`, peuvent rester aliases deprecies quand ils aident l'autocompletion, mais ils doivent exiger `--connector-id` et afficher un warning;
- ne pas ajouter d'alias court qui masque le connector ID cible;
- ne pas accepter de forme qui laisse croire que le type connecteur est l'identifiant cible; l'objet modifie est toujours le connector ID;
- eviter de multiplier les sous-commandes generees si une commande generique avec schema OpenAPI suffit;
- garder les commandes par connecteur comme aliases pour l'autocompletion et la documentation.

## Mapping Wallets

Regles:

- conserver `wallets`;
- `--ik` devient `--idempotency-key`;
- `credit` et `debit` prennent toujours un wallet cible explicite en v4;
- commandes mutantes gardent `--confirm` si elles ont deja une confirmation v3;
- identifiants documentes en kebab-case dans l'aide, sans changer la valeur attendue.

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `wallets create <name> --metadata --ik` | `wallets create <name> --metadata --idempotency-key` | alias `--ik`. | |
| `wallets list` | identique | pagination/filtres si disponibles. | |
| `wallets show` | `wallets show <wallet-id>` | verifier si v3 lit un flag implicite; rendre explicite. | |
| `wallets update <wallet-id>` | identique | metadata/name via flags ou `--file`. | |
| `wallets credit <amount> <asset>` | `wallets credit <wallet-id> --amount <amount> --asset <asset>` | wallet cible explicite obligatoire; ancienne forme seulement alias deprecie si elle peut etre resolue sans ambiguite. | |
| `wallets debit <amount> <asset>` | `wallets debit <wallet-id> --amount <amount> --asset <asset>` | wallet cible explicite obligatoire; ancienne forme seulement alias deprecie si elle peut etre resolue sans ambiguite. | |
| `wallets balances create <balance-name>` | identique | | |
| `wallets balances list` | identique | | |
| `wallets balances show <balance-name>` | identique | | |
| `wallets holds list` | identique | | |
| `wallets holds show <hold-id>` | identique | | |
| `wallets holds void <hold-id>` | identique | `--confirm`. | |
| `wallets holds confirm <hold-id>` | identique | `--confirm` si irreversible. | |
| `wallets transactions list` | identique | pagination/filtres. | |

## Mapping Flows

L'ancien produit `orchestration` doit etre expose sous le nom canonique `flows` en v4.
Le chemin `orchestration ...` peut rester comme alias deprecie pendant la phase de migration si cela ne complique pas le routeur Cobra.

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `orchestration workflows create <file>|-` | `flows workflows create --file <path>|-` | ancien positionnel alias; ancien prefixe `orchestration` alias deprecie. | |
| `orchestration workflows list` | `flows workflows list` | prefixe v3 alias deprecie. | |
| `orchestration workflows show <id>` | `flows workflows show <id>` | prefixe v3 alias deprecie. | |
| `orchestration workflows run <id>` | `flows workflows run <id>` | payload vars via `--input`/`--file` si API; prefixe v3 alias deprecie. | |
| `orchestration workflows delete <workflow-id>` | `flows workflows delete <workflow-id>` | `--confirm`; prefixe v3 alias deprecie. | |
| `orchestration instances list` | `flows instances list` | prefixe v3 alias deprecie. | |
| `orchestration instances show <instance-id>` | `flows instances show <instance-id>` | prefixe v3 alias deprecie. | |
| `orchestration instances describe <instance-id>` | `flows instances inspect <instance-id>` | alias `describe`; prefixe v3 alias deprecie. | Choisir selon distinction show/inspect. |
| `orchestration instances send-event <instance-id> <event>` | `flows instances send-event <instance-id> <event>` | event JSON via `--event` ou `--file` si necessaire; prefixe v3 alias deprecie. | |
| `orchestration instances stop <instance-id>` | `flows instances stop <instance-id>` | `--confirm`; prefixe v3 alias deprecie. | |
| `orchestration triggers create <event> <workflow-id>` | `flows triggers create <event> <workflow-id>` | flags pour conditions si API; prefixe v3 alias deprecie. | |
| `orchestration triggers list` | `flows triggers list` | prefixe v3 alias deprecie. | |
| `orchestration triggers show <trigger-id>` | `flows triggers show <trigger-id>` | prefixe v3 alias deprecie. | |
| `orchestration triggers delete <trigger-id>` | `flows triggers delete <trigger-id>` | `--confirm`; prefixe v3 alias deprecie. | |
| `orchestration triggers test <trigger-id> <event>` | `flows triggers test <trigger-id> <event>` | event JSON/file si possible; prefixe v3 alias deprecie. | |
| `orchestration triggers occurrences list` | `flows triggers occurrences list` | prefixe v3 alias deprecie. | |

## Mapping Reconciliation

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `reconciliation list` | identique | pagination/filtres normalises. | |
| `reconciliation get <reconciliationID>` | `reconciliation show <reconciliation-id>` | alias `get`. | |
| `reconciliation policies create <file>|-` | `reconciliation policies create --file <path>|-` | ancien positionnel alias. | |
| `reconciliation policies list` | identique | | |
| `reconciliation policies get <policyID>` | `reconciliation policies show <policy-id>` | alias `get`. | |
| `reconciliation policies delete <policyID>` | `reconciliation policies delete <policy-id>` | `--confirm`. | |
| `reconciliation policies reconcile <policyID> <atLedger> <atPayments>` | `reconciliation policies reconcile <policy-id> --ledger-at <time> --payments-at <time>` | anciens positionnels alias. | |

## Mapping Auth service

La v3 utilise `auth` pour le service Auth de la stack. La v4 garde `auth` comme nom canonique du service, car `identity` pourra devenir un produit Formance distinct plus tard.
Les commandes de session CLI peuvent vivre sous `auth login/status/logout/token`, mais les commandes du service restent `auth clients ...` et `auth users ...`.

Decision:

- `auth login/status/logout/token` = session CLI;
- `auth clients ...` et `auth users ...` = service Auth de la stack;
- ne pas introduire `identity` comme alias ou nom canonique dans cette migration.

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `auth clients create <name>` | identique | flags scopes/redirects explicites. | |
| `auth clients list` | identique | | |
| `auth clients show <client-id>` | identique | | |
| `auth clients update <client-id>` | identique | | |
| `auth clients delete <client-id>` | identique | `--confirm`. | |
| `auth clients secrets create <client-id> <secret-name>` | identique | afficher secret seulement en sortie structuree controlee. | |
| `auth clients secrets delete <client-id> <secret-id>` | identique | `--confirm`. | |
| `auth users list` | identique | | |
| `auth users show <user-id>` | identique | | |

## Mapping Webhooks

| v3 | v4 canonique | Changements d'arguments | Notes |
| --- | --- | --- | --- |
| `webhooks create <endpoint> [event-type...] --secret` | identique | permettre `--secret-stdin`; secret masque. | |
| `webhooks list` | identique | | |
| `webhooks activate <config-id>` | identique | | |
| `webhooks deactivate <config-id>` | identique | `--confirm` si necessaire. | |
| `webhooks delete <config-id>` | identique | `--confirm`. | |
| `webhooks change-secret <config-id> <secret>` | `webhooks secret rotate <config-id>` ou `webhooks change-secret` | Preferer `--secret-stdin`; garder ancien nom. | |

## Commandes a retirer ou a requalifier

| v3 | Decision proposee | Raison |
| --- | --- | --- |
| `prompt` | remplacer par `setup`/`context wizard`, garder alias cache | Le nom ne decrit pas l'intention utilisateur. |
| `cloud_stacks ...`, `stack ...` et `stacks ...` a la racine | deprecie vers `cloud stacks ...` avec warning | `stack` doit pouvoir signifier target data-plane; les operations lifecycle sont Cloud. |
| `search ...` et alias `se` | supprimer | Le produit n'existe plus en v4. |
| `payments ... get` | deprecie vers `show` | Coherence globale. |
| commandes avec underscores | aliases deprecies avec warning | Coherence shell. |

Les aliases conserves sont une aide de migration v4, pas une promesse de compatibilite long terme. Leur suppression pourra etre planifiee dans une version ulterieure (`v4.1`, `v4.2` ou `v5`) selon le cout de maintenance et l'usage observe.

## Documentation a produire depuis ce plan

1. `docs/cli-v4/migration-v3-v4.md`
   - guide utilisateur;
   - sections "ce qui change", "commandes renommees", "flags renommes";
   - exemples avant/apres.

2. `docs/cli-v4/command-reference.md`
   - reference v4 canonique;
   - generee en partie depuis Cobra pour eviter la derive.

3. `docs/cli-v4/compatibility-aliases.md`
   - liste des aliases v3;
   - statut: visible, deprecie, cache, supprime;
   - date/version cible de suppression si applicable.

4. `docs/cli-v4/testing-strategy.md`
   - strategie de tests decrite ci-dessous;
   - comment lancer les mocks localement;
   - comment ajouter une commande avec fixtures.

## Strategie de tests

### Tests unitaires critiques

| Zone | Ce qu'il faut tester |
| --- | --- |
| `internal/config` | parsing config, defaults, validation kind/auth, ecriture atomique, migration v3 idempotente. |
| `internal/credentials` | keyring fake, fallback insecure explicite, aucun secret en clair par defaut. |
| `internal/auth` | token statique, client credentials, OIDC/cloud device flow mocke, expiration/refresh. |
| `internal/capabilities` | parsing `/versions`, ranges semver, choix du namespace le plus recent, override `--api-version`, erreurs si aucune version compatible. |
| `internal/runtime` | resolution context -> target -> auth -> versions -> SDK client; erreurs sans contexte; local no-auth. |
| `internal/render` | plain/json/yaml, stderr vs stdout, no-color, quiet, erreurs structurees. |
| `internal/prompt` | jamais appele en `--non-interactive`, fallback propre si pas de TTY. |
| `internal/commands/ledger` | inputs canoniques, adapters v1/v2/v3, aliases `--src/--dst`, validation de dates. |
| `internal/commands/payments` | payload `--file|-`, normalisation kebab/underscore, connecteurs. |
| `internal/commands/wallets` | `--idempotency-key`/`--ik`, wallet subject resolution. |
| `internal/commands/cloud` | contexte requis `cloud`/`cloud-stack`, erreurs propres sur contexte `stack`. |

### Tests d'integration CLI

Approche:

- construire le binaire v4 une fois par package de tests;
- executer de vraies commandes avec config temporaire;
- utiliser des serveurs HTTP `httptest` pour stack, membership et auth;
- capturer stdout, stderr, exit code;
- comparer les sorties structurees avec golden files;
- interdire les prompts avec `--non-interactive`;
- verifier que les erreurs sont scriptables.

Scenarios minimaux:

| Scenario | Commandes |
| --- | --- |
| contexte local no-auth | `context create stack`, `context use`, `target inspect`, `ledger transactions list`. |
| contexte token self-hosted | `auth login token`, `target inspect`, commande produit. |
| contexte client credentials | `auth login client-credentials`, refresh token, commande produit. |
| contexte Cloud stack | `auth login cloud`, `cloud organizations list`, `ledger transactions list`. |
| migration v3 | `config migrate-v3`, puis `context list/show/use`. |
| aliases v3 | `profiles list`, `payments transfer_initiation list`, `payments bank_accounts list`, `ledger transactions list --src --dst`. |
| non-interactif | commandes mutantes sans `--confirm` doivent echouer proprement. |
| sortie stable | chaque famille avec `--output json` et `--output yaml`. |

### Mock OpenAPI stack et membership

Specs disponibles:

- stack: `https://github.com/formancehq/stack/releases/download/v3.2.4/generate.json`;
- SDK public: `https://github.com/formancehq/formance-sdk-go`;
- membership/deployserver: verifier les specs locales sous `openapi/` si presentes dans le repo.

Plan de mock:

1. Ajouter `v4/internal/testserver/openapi`.
2. Charger les specs OpenAPI via `github.com/getkin/kin-openapi/openapi3`.
3. Router les requetes avec `routers/gorillamux` ou equivalent.
4. Valider method/path/query/body quand le schema est disponible.
5. Retourner des fixtures JSON stockees sous `v4/testdata/responses/<product>/<operation>.json`.
6. Exposer `/versions` avec une fixture parametrable par test.
7. Fournir un mock membership minimal pour:
   - device flow/cloud login;
   - organisations;
   - stacks;
   - tokens personnels;
   - erreurs 401/403/404.
8. Fournir un mock auth/OIDC minimal pour:
   - discovery `.well-known/openid-configuration`;
   - token endpoint;
   - refresh;
   - JWKS si necessaire.

Le mock ne doit pas devenir une implementation complete de la stack. Il doit valider que le CLI:

- appelle le bon endpoint pour la version API resolue;
- envoie les bons parametres apres adaptation v3/v4;
- gere correctement les erreurs API;
- produit une sortie stable.

### Tests contractuels OpenAPI

Pour chaque commande v4 implementee:

- declarer le produit, la feature et les operation IDs supportes;
- verifier que les operation IDs existent dans le manifeste genere;
- verifier que chaque handler versionne a une operation correspondante;
- verifier que les commandes marquees `requires API vN+` echouent proprement sur des fixtures `/versions` plus anciennes.

Exemples:

| Commande | Contrat |
| --- | --- |
| `ledger transactions list` | operation list transactions existe pour chaque namespace handler. |
| `payments transfer-initiation create` | payload fixture valide contre le schema. |
| `cloud organizations list` | commande refuse contexte non Cloud avant tout appel reseau. |
| `auth clients create` | operation auth service presente dans stack spec. |

### E2E de bout en bout

Les E2E doivent rester peu nombreux mais couvrir les chemins de valeur:

1. Installer/initialiser un contexte local.
2. Inspecter la cible.
3. Creer/importer un ledger ou fixture equivalente.
4. Lister les transactions avec API latest compatible.
5. Forcer une API plus ancienne avec `--api-version`.
6. Migrer une config v3 vers v4.
7. Executer une commande Cloud contre membership mock.
8. Verifier que les anciens aliases produisent soit la meme sortie, soit un warning de deprecation sur stderr.

Les E2E contre une vraie stack peuvent etre ajoutes separement derriere un tag Go:

```bash
go test ./... -tags=e2e_real_stack
```

Ils ne doivent pas etre requis pour les PR ordinaires.

## Strategie d'implementation

### Phase 0: inventaire automatique

- Ajouter un outil interne qui parcourt l'arbre Cobra v3 et ecrit `v4/testdata/v3-command-inventory.json`.
- Capturer pour chaque commande:
  - chemin complet;
  - aliases;
  - flags;
  - args;
  - hidden/deprecated;
  - short/long help.
- Utiliser ce fichier pour verifier que ce plan ne perd aucune commande.

### Phase 1: fondations transverses

- Finaliser contextes/auth/credentials/runtime/render.
- Stabiliser `--context`, `--output`, `--non-interactive`, `--api-version`.
- Ajouter le mock OpenAPI minimal et le harness CLI.
- Ajouter les tests de migration config.

### Phase 2: Ledger complet

- Migrer toutes les commandes ledger.
- Ajouter adapters v1/v2/v3.
- Tester les changements de parametres `account/address`, `src/source`, `dst/destination`.
- Ajouter les commandes v3+ uniquement si elles existent dans le manifeste.

### Phase 3: produits stack restants

Ordre recommande:

1. `payments`, car beaucoup de payloads et connecteurs;
2. `wallets`, car sujet/wallet implicite a clarifier;
3. `flows` pour l'ancien `orchestration`;
4. `reconciliation`;
5. `webhooks`;
6. `auth` service pour `auth clients/users`.

Chaque famille doit etre livree avec:

- commandes canoniques;
- aliases v3 documentes;
- tests adapters;
- tests CLI avec mock;
- golden outputs.

### Phase 4: Cloud control plane

- Migrer `cloud ...`;
- migrer `cloud_stacks ...`, `stack ...` et `stacks ...` vers `cloud stacks ...` avec warnings de deprecation;
- garder aliases v3 quand non ambigus;
- couvrir membership mock.

### Phase 5: documentation et compatibilite

- Generer la reference v4;
- rediger le guide v3 -> v4;
- ajouter warnings de deprecation;
- verifier tous les aliases de ce plan;
- produire une matrice "commande v3 / commande v4 / statut".

### Phase 6: cutover

- Quand v4 est fonctionnellement complet:
  - executer les tests v4;
  - executer les tests v3 encore presents pour verifier absence de regression avant suppression;
  - deplacer v4 a la racine selon l'ADR de cutover;
  - supprimer l'implementation v3 dans un commit dedie;
  - mettre a jour packaging, CI, goreleaser, completions.

## Checklist de review par commande

Pour chaque commande migree:

- [ ] le chemin v4 canonique suit ce plan;
- [ ] les aliases v3 utiles existent ou la suppression est documentee;
- [ ] les flags v3 renommes ont un alias deprecie si possible;
- [ ] l'input Cobra est converti en modele canonique;
- [ ] les adapters API versionnes sont testes;
- [ ] la selection d'API vient du runtime, pas de la commande;
- [ ] le contexte requis est valide avant appel reseau;
- [ ] les prompts sont desactives en `--non-interactive`;
- [ ] stdout contient la donnee, stderr contient logs/warnings;
- [ ] JSON/YAML sont stables;
- [ ] une erreur API est rendue proprement;
- [ ] les fixtures mock couvrent au moins succes + une erreur;
- [ ] la documentation avant/apres est mise a jour.

## Decisions actees

1. Le service Auth garde le nom canonique `auth`; ne pas utiliser `identity` dans cette migration.
2. Les operations Cloud de lifecycle de stack utilisent `cloud stacks ...`; l'ancien chemin `cloud_stacks ...` et les anciens chemins `stacks ...` et `stack ...` sont des aliases deprecies avec warning.
3. `wallets credit` et `wallets debit` prennent un wallet cible explicite en v4.
4. `payments connectors config update` cible toujours un connector ID.
5. Ne pas ajouter d'alias court comme `fctl transaction list`; toujours garder le nom du service, par exemple `fctl ledger transactions list`.
6. Les aliases v3 peu couteux peuvent rester pendant la v4, mais chaque utilisation doit afficher un warning avec la commande canonique. Leur suppression pourra arriver dans une version ulterieure, potentiellement `v4.1`, `v4.2` ou `v5` selon le cout de maintenance et l'usage observe.
7. Les noms internes, variables d'environnement et identifiants generes doivent toujours inclure le service: `FCTL_LedgerTransactionList`, jamais `FCTL_TransactionList`.
