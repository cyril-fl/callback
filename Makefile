.PHONY: setup up rebuild down logs compose web server dev backup openapi test

# CALLBACK_PASSWORD (and others) from .env, if there is one.
-include .env
CALLBACK_PASSWORD ?= change-me
export

# Dev ports apart from tide's API, which holds 127.0.0.1:8080-8081 in Docker.
API_PORT ?= 8082
MCP_PORT ?= 8083

# Git hooks (lefthook is a go tool of go.mod).
setup:
	go tool lefthook install

# Docker: the server image built locally, API and web on :8080, MCP on :8081. CALLBACK_PASSWORD from .env, else change-me.
# `up` reuses the image (compose builds it only when missing): `rebuild` after a code change.
up:
	docker compose up -d

rebuild:
	VERSION=$$(git describe --tags --always --dirty) docker compose up -d --build

down:
	docker compose down

logs:
	docker compose logs -f

# Any other compose command: make compose c="ps".
compose:
	docker compose $(c)

# The Nuxt dev server, on the API of `make server`.
web:
	cd web && NUXT_PUBLIC_API_BASE=http://localhost:$(API_PORT) bun run dev

# The Go server on API_PORT and MCP_PORT, without docker. The password defaults to change-me.
server:
	go run ./cmd/callback-server serve --db ./data/callback.db --origin http://localhost:3000 --addr :$(API_PORT) --mcp-addr :$(MCP_PORT)

# The whole dev stack without docker: the server and the Nuxt dev server (Ctrl-C stops both).
dev:
	@trap 'kill 0' INT TERM EXIT; $(MAKE) server & $(MAKE) web & wait

# A consistent copy of the base (VACUUM INTO, safe while serving) in the volume.
backup:
	docker compose exec callback /callback-server --db /data/callback.db backup /data/backup-$$(date +%Y%m%d-%H%M%S).db

# After an API change (swag comments): docs/swagger.yaml, then the CLI's client. `bun run types` in web/ next.
openapi:
	go tool swag init --v3.1 --requiredByDefault -d core/api,core/domain,core/services -g router.go -o docs --outputTypes yaml
	@if [ -d cli/services/apiclient ]; then go tool oapi-codegen -generate types,client -package apiclient -o cli/services/apiclient/client.gen.go docs/swagger.yaml; fi

test:
	go vet ./... && go test ./...
