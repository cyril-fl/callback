# MCP : donner les candidatures à un agent

callback sert les candidatures à un agent (Claude Code, Claude Desktop, l'app Claude, tout client MCP) par six outils.
Deux transports, les mêmes outils :

- **stdio** : `callback mcp`, lancé par le client sur la machine. C'est un client de l'API comme le reste de la CLI :
  chaque outil devient une requête au serveur, avec le jeton de la session.
- **HTTP** : `callback-server serve --mcp-addr <adresse>` sert `/mcp` (« streamable HTTP ») et sa connexion OAuth, pour
  un agent qui ne peut pas lancer de commande (l'app Claude appelle depuis le cloud d'Anthropic). Les outils y
  appellent l'API dans le même processus, sans réseau.

## 1. En stdio : `callback mcp`

Le client démarre `callback mcp` comme sous-processus et échange du JSON-RPC sur son entrée et sa sortie standard :
on ne le lance pas à la main. Il suffit d'une session, une fois, sous l'utilisateur Unix qui lancera l'agent :

```sh
go install github.com/cyril-fl/callback/cmd/callback@latest
callback login https://callback.example.com     # le mot de passe, demandé ou lu dans CALLBACK_PASSWORD
callback whoami                            # si ça répond, le MCP marchera
claude mcp add callback -- "$(go env GOPATH)/bin/callback" mcp
```

`.mcp.json` ou Claude Desktop (le chemin en absolu : un client lancé depuis le Finder n'a pas le `PATH` du shell) :

```json
{ "mcpServers": { "callback": { "command": "/Users/<moi>/go/bin/callback", "args": ["mcp"] } } }
```

La session est dans `~/.config/callback/session.json` (`0600`). **`CALLBACK_URL` et `CALLBACK_SESSION` passent avant le fichier**,
chacune à part : un agent dans un conteneur, sans shell pour `callback login`, prend le serveur et un jeton par son
environnement. Le piège inverse : un bloc `env` recopié chez le client envoie le MCP vers un autre serveur.

Le mieux pour un agent est un **jeton d'API** plutôt que la session : il se révoque seul, et peut être en lecture
seule.

```sh
CALLBACK_SESSION=$(callback token add claude --read-only --expires 30d)
```

Ce jeton se traite comme un mot de passe : jamais dans un fichier versionné. Avec un jeton en lecture seule,
`callback mcp` n'annonce que les outils de lecture.

## 2. En HTTP : `/mcp`

`callback-server serve --mcp-addr :8081` sert, sur une adresse à part, `/mcp` et les routes OAuth (`/.well-known/…`,
`/register`, `/authorize`, `/token`). C'est cette adresse seule qu'on ouvre à Internet (Tailscale Funnel, Cloudflare
Tunnel, un proxy HTTPS qui garde le `Host`) ; l'API et l'app restent derrière.

Chaque requête porte `Authorization: Bearer <jeton>`, de l'une de deux sources :

- **Un jeton d'API** (`callback token add "Claude téléphone" --expires 30d`, montré une fois), pour un client qui
  accepte un en-tête fixe :

  ```sh
  claude mcp add --transport http callback https://callback.example.com/mcp --header "Authorization: Bearer callback_…"
  ```

- **OAuth** : le client ne reçoit que l'URL `https://callback.example.com/mcp`, ouvre la page de connexion de callback, et
  reçoit ses jetons seul. Pour l'app Claude (Réglages > Connecteurs > « Add custom connector ») et ChatGPT.

Sans jeton, avec un jeton inconnu, expiré, révoqué, ou un jeton de **session**, la réponse est `401` avant tout
outil, avec `WWW-Authenticate: Bearer resource_metadata="…/.well-known/oauth-protected-resource"` qui dit au client
où se connecter. Un jeton en lecture seule (`--read-only`, ou la case de la page OAuth) ne reçoit que les outils de
lecture, et l'API lui refuse toute écriture (403). Aucune session MCP n'est gardée entre deux requêtes.

Pour vérifier sans client :

```sh
curl -s https://callback.example.com/mcp -H 'Authorization: Bearer callback_…' \
  -H 'Content-Type: application/json' -H 'Accept: application/json, text/event-stream' \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

## 3. Les outils

Tous en lot : **un appel d'outil = une requête à l'API = une transaction**, jamais un appel par candidature.

| Outil | Ce qu'il fait | Annotations |
|---|---|---|
| `list_applications` | Les candidatures des filtres (`status` liste, `source`, `q`, `applied_before`, `applied_after`, `untouched_since`), **compactes** : `id`, `company`, `status`, `applied_at`, et `total`. `fields` ajoute des colonnes, `count_only` ne rend que le total, `sort`, `limit` (50 par défaut, 0 pour toutes), `offset` | lecture |
| `read_applications` | Les candidatures entières de `ids` (50 au plus), dans l'ordre demandé ; `max_chars` coupe les commentaires (`truncated`). Les champs identiques sur toutes sont dits une fois dans `common`, les ids introuvables dans `missing` | lecture |
| `application_stats` | Le total, le nombre par statut et par source (`by_source`, pour réutiliser les noms existants) et le taux de réponse | lecture |
| `status_history` | Les statuts atteints et quand, du plus ancien au plus récent (`changes`), des candidatures de `ids` et/ou depuis `since` (ce jour inclus) : délais de réponse, ce qui a bougé récemment | lecture |
| `add_applications` | Crée `items` (500 au plus) ; rend seulement les `ids` créés et les éléments en échec (`failed`, avec leur `index`) : les valides sont écrits quand même | écriture |
| `update_applications` | `ops`, chacun choisit des candidatures par `ids` ou par filtres et donne `set` (`status`, `rating`, `comment`…) ; `dry_run` n'écrit rien et rend ce qui changerait, `summary` retire les ids | écriture, idempotent |
| `delete_applications` | Supprime **définitivement** les candidatures de `ids` ou des filtres ; `dry_run`, `summary`. Un filtre vide est refusé | destructif, idempotent |

Les valeurs : `status` parmi `applied`, `responded`, `interview`, `offer`, `rejected` ; `source` : `spontaneous`
(par défaut), `job_post` (avec l'URL de l'annonce en `source_detail`), `recruiter` (avec son e-mail) ou un nom libre
(60 caractères au plus, `source_detail` libre) ; l'adresse : `location` est la ville, `street`, `postal_code` et
`country` la complètent (aussi dans `fields` et `q`) ; `interview_at`, la date et l'heure locales de l'entretien
(`2026-01-31T14:30`) ; `contact_name` et `contact_email` (une adresse valide), cherchés aussi par `q` ; `rating` de
0 à 5 ; les jours en `2026-01-31`, `applied_before` et `applied_after` stricts. `""` efface `interview_at` et le
contact.

Les réponses sont du JSON compact (sans indentation, champs vides omis). `readOnlyHint` sur les lectures : le client
les lance sans demander ; `destructiveHint` sur `delete_applications` : le client demande avant.

**Peu d'outils, des descriptions d'une phrase.** Chaque outil annoncé coûte du contexte à chaque requête.
`TestToolListSize` (`cli/cmd-mcp/cmd_test.go`) mesure le nom, la description et le schéma d'entrée de chaque outil, et
échoue au-delà de 8 outils ou 6 000 octets (6 outils et 4,1 Ko au départ, 4,8 Ko avec l'entretien et le contact, 7 outils et 5,2 Ko avec l'historique) :
des outils complets qui évitent des allers-retours valent quelques octets, mais optimiser l'existant avant d'ajouter.
`go test -v -run Size ./cli/cmd-mcp/` affiche les chiffres.

## 4. Les consignes

Le serveur donne aussi des consignes, que le client met dans son prompt système (`cli/cmd-mcp/instructions.go`) : les
recettes tiennent là, une fois, plutôt que dans les descriptions.

- **Compter avant de lire** : `application_stats`, ou `list_applications` avec `count_only`.
- **Filtrer sur le serveur** plutôt que tout lire, et ne demander en `fields` que les colonnes utiles ; paginer avec
  `limit` et `offset`.
- **Lire par lots** : `read_applications` prend 50 ids, avec `max_chars` pour les longs commentaires.
- **L'historique filtré** : `status_history` avec `ids` ou `since`, pas tout l'historique.
- **Réutiliser les sources** de `by_source` (`application_stats`) plutôt que créer un quasi-doublon.
- **Écrire par lots** : un `add_applications` pour toutes les nouvelles, un `update_applications` par lot, un groupe
  par élément de `ops`.
- **`dry_run` d'abord** sur des filtres, puis le vrai appel ; `summary` quand les compteurs suffisent. Les écritures
  rendent les ids et les échecs : inutile de relire.
- Les sous-agents ne reçoivent pas les consignes : l'agent principal leur passe ce qu'il leur faut.

Avec un jeton en lecture seule, les consignes d'écriture sont retirées.

## 5. La CLI, pour un agent qui a un shell

Un agent avec un shell peut aussi scripter `callback` : chaque commande prend `--json`, et `ls`, `edit`, `rm` prennent
les mêmes filtres que les outils (`edit` les nomme `--where-status` et `--where-source`, `--status` et `--source` y
étant les champs à changer).

```sh
callback ls --status applied --untouched-since 2026-01-01 --count
callback edit --where-status applied --untouched-since 2026-01-01 --status rejected --dry-run
callback add --file apps.json        # un tableau JSON, un seul appel
```
