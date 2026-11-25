.PHONY: setup up down web backup openapi test

# Git hooks (lefthook is a go tool of go.mod).
setup:
	go tool lefthook install

# The dev stack: the server image built locally, API and web on :8080, MCP on :8081. Needs JAT_PASSWORD in .env.
up:
	VERSION=$$(git describe --tags --always --dirty) docker compose up -d --build

down:
	docker compose down

# The Nuxt dev server.
web:
	cd web && bun run dev

# A consistent copy of the base (VACUUM INTO, safe while serving) in the volume.
backup:
	docker compose exec jat /jat-server --db /data/jat.db backup /data/backup-$$(date +%Y%m%d-%H%M%S).db

# After an API change (swag comments): docs/swagger.yaml, then the CLI's client. `bun run types` in web/ next.
openapi:
	go tool swag init --v3.1 --requiredByDefault -d core/api,core/domain,core/services -g router.go -o docs --outputTypes yaml
	@if [ -d cli/services/apiclient ]; then go tool oapi-codegen -generate types,client -package apiclient -o cli/services/apiclient/client.gen.go docs/swagger.yaml; fi

test:
	go vet ./... && go test ./...
