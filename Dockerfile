FROM golang:1.26.4 AS ci
WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN go test ./internal/app -run TestNewPolicyStoreRejectsMemoryInProductionBuild && go test -tags dev ./...

FROM golang:1.26.4 AS build
WORKDIR /app

ARG APP_VERSION
ARG TARGETARCH
COPY go.mod ./
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
COPY --from=build /app/internal/domain/policy/testdata /app/policy
COPY --from=build /app/internal/domain/provider/testdata/provider_manifest_nicetry_v0_1.json /app/policy/provider_manifest_nicetry_v0_1.json
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/cafe-cpm"]
