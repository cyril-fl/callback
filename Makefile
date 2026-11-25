.PHONY: openapi test

# After an API change (swag comments): docs/swagger.yaml, then the CLI's client. `bun run types` in web/ next.
openapi:
	go tool swag init --v3.1 --requiredByDefault -d core/api,core/domain,core/services -g router.go -o docs --outputTypes yaml
	@if [ -d cli/services/apiclient ]; then go tool oapi-codegen -generate types,client -package apiclient -o cli/services/apiclient/client.gen.go docs/swagger.yaml; fi

test:
	go vet ./... && go test ./...
