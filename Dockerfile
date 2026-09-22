##############################
# Stage CI — Lint / Test / Vuln (use: docker build --target ci)
##############################
FROM golang:1.26.6 AS ci

WORKDIR /app

# Cache-bust: force reinstall of tools when Go version changes (must match image tag)
ENV GO_TOOLING_VERSION=1.26.6
ENV PATH=/go/bin:/usr/local/go/bin:/usr/local/bin:$PATH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go install golang.org/x/vuln/cmd/govulncheck@latest \
    && go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0

# Preserve CPM-specific test guards: prod memory rejection (!dev) + full suite with -tags dev
CMD ["sh", "-c", "go mod download && golangci-lint run ./... && go test ./internal/app -run TestNewPolicyStoreRejectsMemoryInProductionBuild && go test -tags dev ./... && govulncheck ./..."]


##############################
# Stage Build
##############################
FROM golang:1.26.6 AS build
WORKDIR /app

ARG APP_VERSION
ARG TARGETARCH
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN RESOLVED_VERSION="${APP_VERSION}" && \
    if [ -z "$RESOLVED_VERSION" ]; then \
      if [ -d .git ]; then \
        BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null | sed 's/[^a-zA-Z0-9]/-/g' || echo "dev") && \
        SHORT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "") && \
        RESOLVED_VERSION="${BRANCH}-${SHORT_SHA:-unknown}"; \
      else \
        RESOLVED_VERSION="dev"; \
      fi; \
    fi && \
    CGO_ENABLED=0 GOOS=linux GOARCH="${TARGETARCH}" go build \
      -ldflags "-X github.com/create2-labs/cafe-crypto-policy-mgt/internal/version.version=${RESOLVED_VERSION}" \
      -o /out/cafe-cpm ./cmd/cafe-cpm

FROM gcr.io/distroless/base-debian12:nonroot
ARG APP_VERSION
LABEL org.opencontainers.image.version="${APP_VERSION}"
COPY --from=build /out/cafe-cpm /usr/local/bin/cafe-cpm
# Flat catalogue dir: Crypto Policies + ProviderManifests (scanned at boot).
COPY --from=build /app/internal/domain/policy/testdata/ /app/policy/
COPY --from=build /app/internal/domain/provider/testdata/ /app/policy/
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/cafe-cpm"]
