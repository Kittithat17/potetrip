# Production image: the Go API also serves the built web app, so the app and
# /api share one origin (the session cookie needs that).

# ---- web app ----
FROM node:22-alpine AS web
WORKDIR /src/frontend
RUN corepack enable && corepack prepare pnpm@10.17.1 --activate
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY frontend/ ./
RUN pnpm build

# ---- API ----
FROM golang:1.26-alpine AS api
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- runtime: no shell, no package manager, non-root ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=api /out/api /app/api
COPY --from=web /src/frontend/dist /app/web
ENV STATIC_DIR=/app/web \
    PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
