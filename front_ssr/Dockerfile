# The server-rendered storefront is a Node application, so unlike the SPA it
# cannot be embedded in the Go binary: it is built and shipped on its own.
# Build from the repository root:
#   docker build -f front_ssr/Dockerfile .

FROM node:24-alpine AS deps
WORKDIR /app
RUN corepack enable
# Only the workspace manifests are needed to resolve the dependency graph.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY front_ssr/package.json ./front_ssr/package.json
RUN pnpm install --frozen-lockfile --filter @openshop/front-ssr...

FROM node:24-alpine AS build
WORKDIR /app
RUN corepack enable
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/front_ssr/node_modules ./front_ssr/node_modules
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY front_ssr ./front_ssr
# Every page is rendered on demand, so the build needs no API access.
RUN pnpm --filter @openshop/front-ssr build

FROM node:24-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
# Next's standalone output is a self-contained server plus its traced deps.
COPY --from=build /app/front_ssr/.next/standalone ./
COPY --from=build /app/front_ssr/.next/static ./front_ssr/.next/static
COPY --from=build /app/front_ssr/public ./front_ssr/public
USER node
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s \
  CMD wget -qO- http://127.0.0.1:3000/ >/dev/null || exit 1
CMD ["node", "front_ssr/server.js"]
