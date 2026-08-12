# syntax=docker/dockerfile:1.7

FROM node:24.17.0-alpine3.24 AS web-build
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.25.12-alpine3.24 AS api-build
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY backend/ ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/nythara-api ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -buildvcs=false -ldflags="-s -w" -o /out/nythara-migrate ./cmd/migrate

FROM alpine:3.24 AS api
RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S -g 10001 nythara && adduser -S -D -H -u 10001 -G nythara nythara
WORKDIR /app
COPY --from=api-build /out/nythara-api /out/nythara-migrate ./
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/app/nythara-api"]

FROM caddy:2.11.4-alpine AS web
COPY ops/Caddyfile /etc/caddy/Caddyfile
COPY --from=web-build /src/web/dist /srv
EXPOSE 80 443 443/udp
