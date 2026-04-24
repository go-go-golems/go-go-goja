# syntax=docker/dockerfile:1

# Stage 1: Build frontend
FROM node:22-slim AS web-builder
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN corepack enable && corepack prepare pnpm@10.15.0 --activate
RUN pnpm install --frozen-lockfile
COPY web/ .
RUN pnpm build

# Stage 2: Build Go binary
FROM golang:1.26-bookworm AS go-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist/public ./web/dist/public
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/goja-repl ./cmd/goja-repl

# Stage 3: Runtime
FROM debian:12-slim
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd --gid 65532 goja \
    && useradd --uid 65532 --gid goja --shell /usr/sbin/nologin --create-home goja

WORKDIR /app
COPY --from=go-builder /app/bin/goja-repl /app/goja-repl
COPY --from=web-builder /app/web/dist/public /app/web/dist/public

RUN mkdir -p /data && chown goja:goja /data

ENV GOJA_REPL_ESSAY_WEB_DIST=/app/web/dist/public
EXPOSE 8080
USER goja:goja
ENTRYPOINT ["/app/goja-repl"]
CMD ["essay", "--addr", ":8080", "--db-path", "/data/goja-repl.sqlite"]
