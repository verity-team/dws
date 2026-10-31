FROM node:20-alpine AS base

# INSTALL DEPENDENCIES
FROM base AS deps
RUN apk add --no-cache libc6-compat python3 make eudev-dev
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

# Build-time configuration (.env files are kept out of the image). ARGs are
# only visible to the build; an unset ARG stays unset, so code defaults apply.
# NEXT_PUBLIC_* values are inlined into the client bundle; the private ones
# are used when pages are prerendered and must also be set at runtime.
ARG NEXT_PUBLIC_DONATE_PUBKEY
ARG NEXT_PUBLIC_MIN_ETH
ARG NEXT_PUBLIC_MIN_USDT
ARG NEXT_PUBLIC_API_TIMEOUT
ARG NEXT_PUBLIC_LAUNCH_TIME
ARG NEXT_PUBLIC_UDATA_REFRESH_TIME_DEFAULT
ARG NEXT_PUBLIC_UDATA_REFRESH_TIME_UNCONFIRMED
ARG NEXT_PUBLIC_TARGET_NETWORK_ID
ARG NEXT_PUBLIC_TARGET_NETWORK_ALIAS
ARG NEXT_PUBLIC_TARGET_NETWORK_RPC
ARG NEXT_PUBLIC_WC_PROJECT_ID
ARG NEXT_PUBLIC_EMAIL_API_URL
ARG NEXT_PUBLIC_GALACTICA_API_URL
ARG NEXT_PUBLIC_HOST_URL
ARG NEXT_PUBLIC_FILE_MAX
ARG NEXT_PUBLIC_TARGET_SALE
ARG DWS_API_URL
ARG HOST_URL
ARG API_TIMEOUT

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