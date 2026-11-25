# The server image: the API and the web app on one origin. The build stages run on the build machine and
# cross-compile, so an arm64 image costs no emulation. `--target server` stops before the web app: the API alone.
FROM --platform=$BUILDPLATFORM oven/bun:1.4.4 AS web
WORKDIR /src
COPY web/package.json web/bun.lock* ./
RUN bun install --frozen-lockfile
COPY web .
# Empty API base: the app calls /api on its own origin.
RUN NUXT_PUBLIC_API_BASE= bun --bun run generate

FROM --platform=$BUILDPLATFORM golang:1.27 AS api
ARG TARGETOS TARGETARCH
WORKDIR /src
# Modules first: a code change does not download them again.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The context has no .git for go build to read the version from.
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath \
    -ldflags "-s -w -X github.com/cyril-fl/callback/config.version=$VERSION" -o /callback-server ./cmd/callback-server
# scratch has no mkdir: /data holds the base, /tmp is for SQLite temp files.
RUN install -d -m 755 /image-data /image-tmp

FROM scratch AS server
COPY --from=api /callback-server /callback-server
COPY --from=api --chown=65534:65534 /image-data /data
COPY --from=api --chown=65534:65534 /image-tmp /tmp
USER 65534
EXPOSE 8080
ENTRYPOINT ["/callback-server"]
CMD ["serve", "--db", "/data/callback.db", "--addr", ":8080"]

FROM server
COPY --from=web /src/.output/public /web
CMD ["serve", "--db", "/data/callback.db", "--web", "/web", "--addr", ":8080"]
