# Déployer callback et y connecter Claude

Du serveur nu à un Claude qui lit et modifie les candidatures par le MCP. La sécurité du MCP exposé : [MCP.md](../MCP.md#2-en-http--mcp).

## Une fois, publier l'image (depuis le poste de dev)

Un tag `v*` poussé fait construire et publier
`ghcr.io/cyril-fl/callback` par la CI (`git tag v0.3.0 && git push origin v0.3.0`).

## Une fois, sur le serveur

1. Dans un dossier, copier `compose.prod.yaml`, `Makefile.server` sous le nom `Makefile`, et `.env.example` sous le
   nom `.env`. Remplir `.env` :
   - `CALLBACK_PASSWORD` : long (c'est la seule barrière de la page OAuth ouverte à Internet), lu au premier démarrage
     seulement.
   - `CALLBACK_ORIGIN` : `https://<nom de la machine>.<tailnet>.ts.net`.
2. Si le paquet GitHub est privé : `docker login ghcr.io` avec un jeton qui lit les paquets.
3. Tailscale : `sudo tailscale set --operator=$USER` (pour lancer `tailscale` sans sudo), et Funnel autorisé pour la
   machine dans la console d'admin (Access controls, attribut `funnel`).
4. `make up`, puis `make serve` (l'app sur le tailnet, à `CALLBACK_ORIGIN`) et `make mcp` (le MCP sur Internet, à
   `CALLBACK_ORIGIN:8443/mcp`, l'adresse qu'il affiche).
5. `make check` depuis n'importe où : `401` sur `/mcp`, et l'`issuer` doit être `https://…ts.net:8443`.

## Connecter Claude

- **App Claude** (web, bureau, mobile) : Réglages > Connecteurs > « Add custom connector », nom `callback`, URL
  `https://<machine>.<tailnet>.ts.net:8443/mcp`. « Connect » ouvre la page de callback : vérifier qu'elle renvoie
  vers `claude.ai`, taper le mot de passe, cocher « Lecture seule » si l'agent ne doit rien changer.
- **Claude Code** : `claude mcp add --transport http callback https://<machine>.<tailnet>.ts.net:8443/mcp`, puis
  `/mcp` dans Claude Code > callback > Authenticate.

N'autoriser une connexion que si on l'a lancée soi-même : l'inscription des clients est ouverte, n'importe qui peut
fabriquer un lien vers cette page.

## Ensuite

- Une connexion se révoque seule dans les réglages de l'app (les jetons) ; `make mcp-off` ferme le MCP à Internet.
- Mise à jour : `make update` (sauvegarde dans `backups/`, puis la dernière image). `make backup` à la demande.
- `make logs`, `make status`, `make reset-password`.
