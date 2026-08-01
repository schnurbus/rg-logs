# rg-logs Backend

Go/Fiber API für WotLK-Combat-Log-Analyse.

## Voraussetzungen

- Go 1.22+
- Docker (PostgreSQL)

## Start

```bash
# Postgres
docker compose up -d

# Backend (aus Repo-Root oder backend/)
cd backend
export DATABASE_URL='postgres://rglogs:rglogs@localhost:5432/rglogs?sslmode=disable'
export UPLOAD_DIR=../uploads
export HTTP_ADDR=:3000
go run ./cmd/server
```

Migrationen laufen automatisch beim Start (`internal/db/migrations`).

## API

| Methode | Pfad | Zweck |
|---------|------|--------|
| `POST` | `/api/uploads` | Multipart-Feld `file` |
| `GET` | `/api/uploads` | Liste |
| `GET` | `/api/uploads/:id` | Detail + Fights |
| `GET` | `/api/fights/:id` | Meta + Participants (`?sort=damage\|healing\|taken`) |
| `GET` | `/api/fights/:id/spells?actorId=` | Spell-Breakdown |
| `GET` | `/api/health` | Healthcheck |

CORS erlaubt `http://localhost:5173` (Vite).

## Tests

```bash
cd backend
go test ./internal/parser/ -count=1
```
