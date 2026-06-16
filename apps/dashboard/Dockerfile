FROM node:22-alpine AS builder
WORKDIR /app

# Install pnpm
RUN corepack enable && corepack prepare pnpm@9.15.0 --activate

# Copy workspace manifests so pnpm can resolve workspaces
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml* ./
COPY packages/types/package.json ./packages/types/package.json
COPY apps/dashboard/package.json ./apps/dashboard/package.json

RUN pnpm install --frozen-lockfile

# Copy source
COPY packages/types ./packages/types
COPY apps/dashboard ./apps/dashboard

# Build types then dashboard
RUN pnpm --filter @cascade/types build
RUN pnpm --filter @cascade/dashboard build

FROM nginx:1.27-alpine
COPY --from=builder /app/apps/dashboard/dist /usr/share/nginx/html
EXPOSE 3000
CMD ["nginx", "-g", "daemon off;"]
