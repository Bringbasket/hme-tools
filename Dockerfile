FROM node:22-alpine AS web

ARG GOKEEP_VERSION=0.1.0
ARG GOKEEP_REVISION=dev
ENV VITE_APP_VERSION=${GOKEEP_VERSION} \
    VITE_APP_REVISION=${GOKEEP_REVISION}

WORKDIR /src
RUN corepack enable
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
COPY apps/web/package.json ./apps/web/package.json
RUN pnpm install --frozen-lockfile
COPY apps/web ./apps/web
RUN pnpm --filter @gokeep/web build

FROM golang:1.25-alpine AS backend

ARG GOKEEP_VERSION=0.1.0
ARG GOKEEP_REVISION=dev

WORKDIR /src/server
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server ./
COPY --from=web /src/server/webui/dist ./webui/dist
RUN CGO_ENABLED=0 go build -tags embed -trimpath \
    -ldflags="-s -w -X main.buildVersion=${GOKEEP_VERSION} -X main.buildRevision=${GOKEEP_REVISION}" \
    -o /out/gokeep-server ./cmd/server

FROM alpine:3.22

ARG GOKEEP_VERSION=0.1.0
ARG GOKEEP_REVISION=dev

LABEL org.opencontainers.image.source="https://github.com/Bringbasket/hme-tools" \
      org.opencontainers.image.version="${GOKEEP_VERSION}" \
      org.opencontainers.image.revision="${GOKEEP_REVISION}"

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 gokeep \
    && adduser -S -D -H -u 10001 -G gokeep gokeep \
    && mkdir -p /data/mail /data/system \
    && chown -R gokeep:gokeep /data

COPY --from=backend /out/gokeep-server /usr/local/bin/gokeep-server

ENV APP_ENV=production \
    SERVER_ADDR=:8080 \
    DATA_DIR=/data \
    MAIL_DATA_DIR=/data/mail \
    GOKEEP_VERSION=${GOKEEP_VERSION} \
    GOKEEP_REVISION=${GOKEEP_REVISION}

USER gokeep:gokeep
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=4s --start-period=10s --retries=4 CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1
ENTRYPOINT ["/usr/local/bin/gokeep-server"]
