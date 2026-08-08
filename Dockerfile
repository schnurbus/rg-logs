# syntax=docker/dockerfile:1

# --- Frontend (Vite) ---------------------------------------------------------
FROM node:22-alpine AS frontend

WORKDIR /app

ARG VITE_SUPABASE_URL
ARG VITE_SUPABASE_ANON_KEY

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ ./
RUN VITE_SUPABASE_URL="$VITE_SUPABASE_URL" \
    VITE_SUPABASE_ANON_KEY="$VITE_SUPABASE_ANON_KEY" \
    npm run build

# --- Backend (Go) ------------------------------------------------------------
FROM golang:1.25-alpine AS backend

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Overlay Vite production assets into web/ (kept embed.go from the COPY above).
COPY --from=frontend /app/dist/ ./web/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

# --- Runtime -----------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=backend /out/server /server

ENV HTTP_ADDR=:8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/server"]
