FROM node:20-alpine AS base

# INSTALL DEPENDENCIES
FROM base AS deps
RUN apk add --no-cache libc6-compat
WORKDIR /app

# Install dependencies based on the preferred package manager
COPY package.json package-lock.json ./
RUN npm ci

# BUILD NEXT APP
FROM base AS builder
WORKDIR /app

COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NODE_ENV production
ENV NEXT_TELEMETRY_DISABLED 1
ENV NEXT_SHARP_PATH /app/node_modules/sharp

RUN npm run build

# RUN NEXT APP
FROM base as runner
WORKDIR /app

ENV NEXT_SHARP_PATH /app/node_modules/sharp
ENV NODE_ENV production

COPY --from=builder /app/public ./public

RUN addgroup --system --gid 1001 nodejs
RUN adduser --system --uid 1001 nextjs

# Set the correct permission for prerender cache
RUN mkdir .next
RUN chown nextjs:nodejs .next

# Automatically leverage output traces to reduce image size
# https://nextjs.org/docs/advanced-features/output-file-tracing
COPY --from=builder --chown=nextjs:nodejs /app/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT 3000
ENV HOSTNAME "0.0.0.0"

CMD ["node", "server.js"]