FROM node:22-alpine AS builder
WORKDIR /app

RUN corepack enable && corepack prepare pnpm@9.15.0 --activate

COPY package.json pnpm-workspace.yaml pnpm-lock.yaml* ./
COPY packages/types/package.json ./packages/types/package.json
COPY services/auth/package.json ./services/auth/package.json

RUN pnpm install --frozen-lockfile

COPY packages/types ./packages/types
COPY services/auth ./services/auth

RUN pnpm --filter @cascade/types build
RUN pnpm --filter @cascade/auth build

# pnpm deploy produces a self-contained dir with real (non-symlinked) node_modules
RUN pnpm deploy --filter @cascade/auth --prod /deploy

FROM node:22-alpine
RUN apk add --no-cache curl
WORKDIR /app

COPY --from=builder /deploy ./

EXPOSE 8083
CMD ["node", "dist/index.js"]
