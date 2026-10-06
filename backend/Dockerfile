# syntax=docker/dockerfile:1
# Builds one image that contains the API plus both embedded SPAs (front + ops).
# Build context is the repository root so the web workspace is available.

ARG GO_VERSION=1.26
ARG NODE_VERSION=24

# --- frontend (front + ops) -----------------------------------------------
FROM node:${NODE_VERSION}-alpine AS web
WORKDIR /src
RUN corepack enable
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY lib/package.json lib/
COPY front/package.json front/
COPY ops/package.json ops/
RUN pnpm install --frozen-lockfile
COPY lib ./lib
COPY front ./front
COPY ops ./ops
RUN pnpm --filter @openshop/front build && pnpm --filter @openshop/ops build

# --- backend ---------------------------------------------------------------
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend ./
# Drop in the freshly built SPAs so //go:embed picks them up.
COPY --from=web /src/front/dist ./internal/http/web/front/dist
COPY --from=web /src/ops/dist ./internal/http/web/ops/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/seed /app/seed
COPY migrations /app/migrations

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
