# rg-logs Backend

Go/Fiber API für WotLK-Combat-Log-Analyse. Auth und Storage laufen über Supabase.

## Voraussetzungen

- Go 1.25+
- [Supabase CLI](https://supabase.com/docs/guides/cli) (`supabase start`)
- Optional: Docker Compose für Mailpit (`docker compose up -d`)

## Start

```bash
# 1) Supabase lokal (DB, Auth, Storage, Mailpit :54324)
supabase start

# 2) Env einmalig anlegen (Repo-Root)
cp .env.example .env
# ANON_KEY / SERVICE_ROLE_KEY / JWT_SECRET aus `supabase status` eintragen

# Optional: zusätzliches Mailpit
docker compose up -d

# 3) Backend (lädt automatisch ../.env)
cd backend
go run ./cmd/server
# oder: air
```

Migrationen laufen automatisch beim Start (`internal/db/migrations`).

## API

| Methode | Pfad | Auth | Zweck |
|---------|------|------|--------|
| `POST` | `/api/uploads` | ja | Multipart `file` + optional `is_private` |
| `GET` | `/api/uploads` | optional | Öffentliche + eigene (`?mine=1`) |
| `GET` | `/api/uploads/:id` | optional | Detail + Fights (private nur Owner) |
| `DELETE` | `/api/uploads/:id` | Owner | Löscht DB + Storage-Objekt |
| `GET` | `/api/fights/:id` | optional | Meta + Participants |
| `GET` | `/api/fights/:id/spells?actorId=` | optional | Spell-Breakdown |
| `GET` | `/api/health` | nein | Healthcheck |

CORS erlaubt `http://localhost:5173` (Vite). Bearer-Token = Supabase Access Token.

## Tests

```bash
cd backend
go test ./internal/parser/ -count=1
```
