# syntax=docker/dockerfile:1.7

# --- Build Go binary ---
FROM golang:1.23 AS builder

WORKDIR /workspace

# Cache deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source
COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/hanko-web ./cmd/web

# --- Build CSS with Tailwind standalone ---
FROM debian:bookworm-slim AS assets

ARG TARGETARCH=amd64
ENV TAILWIND_VERSION=v3.4.14
ENV HTMX_VERSION=2.0.3

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Only what Tailwind needs
COPY tailwind.config.js ./tailwind.config.js
COPY assets/css ./assets/css
COPY templates ./templates

RUN set -eux; \
    case "${TARGETARCH}" in \
      amd64) TW_URL="https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-x64" ;; \
      arm64) TW_URL="https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-arm64" ;; \
      *) echo "Unsupported TARGETARCH=${TARGETARCH}"; exit 1 ;; \
    esac; \
    curl -fsSL "$TW_URL" -o /usr/local/bin/tailwindcss; \
    chmod +x /usr/local/bin/tailwindcss; \
    mkdir -p /out/public/assets/js; \
    tailwindcss -c tailwind.config.js -i assets/css/input.css -o /out/public/assets/app.css --minify; \
    curl -fsSL "https://cdn.jsdelivr.net/npm/htmx.org@${HTMX_VERSION}/dist/htmx.min.js" -o /out/public/assets/js/htmx.min.js

# --- Final image ---
FROM gcr.io/distroless/static-debian12:nonroot AS runner

ENV PORT=8080

WORKDIR /app

COPY --from=builder /out/hanko-web /usr/local/bin/hanko-web
COPY --from=assets /out/public /app/public
COPY templates /app/templates

EXPOSE 8080
USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/hanko-web","-addr",":8080","-public","/app/public","-templates","/app/templates"]
