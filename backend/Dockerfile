# Builds one image that contains the API plus both embedded SPAs (front + ops).
# Build context is the repository root so the web workspace is available.

ARG GO_VERSION=1.26
ARG NODE_VERSION=24

# --- frontend (front + ops) -----------------------------------------------
FROM node:${NODE_VERSION}-alpine AS web
WORKDIR /src
RUN npm install -g pnpm@11.25.0
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY lib/package.json lib/
COPY front/package.json front/
COPY ops/package.json ops/
COPY e2e/package.json e2e/
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
COPY --from=web /src/front/dist ./internal/http/front/assets/dist
COPY --from=web /src/ops/dist ./internal/http/ops/assets/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ops ./cmd/ops && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# --- runtime ---------------------------------------------------------------
# Alpine rather than distroless: it comes from the same registry as every other
# image CI pulls, so the build is reproducible. CA certificates are required for
# outbound TLS (payment providers, SMTP) and tzdata for local-time rendering.
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -u 65532 -h /app nonroot
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/ops /app/ops
COPY --from=build /out/migrate /app/migrate
COPY --from=build /out/seed /app/seed
# The context root is the repository root, so migrations live under backend/.
COPY backend/migrations /app/migrations

EXPOSE 8080
USER nonroot
ENTRYPOINT ["/app/server"]
