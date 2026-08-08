# rg-logs Backend

Go/Fiber API für WotLK-Combat-Log-Analyse. Auth und Storage laufen über Supabase.
In Produktion liefert derselbe Prozess auch die Vite-SPA (eingebettet unter `web/`).

## Voraussetzungen

- Go 1.25+
- [Supabase CLI](https://supabase.com/docs/guides/cli) (`supabase start`) für lokales Dev
- Optional: Docker Compose für Mailpit (`docker compose up -d`)

## Start (lokal)

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

Ohne Frontend-Build liefert der Server keine SPA (`web/` enthält nur einen Platzhalter). Für die UI lokal weiter Vite nutzen (`cd frontend && npm run dev`, Proxy `/api` → `:3000`).

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

CORS Default: `http://localhost:5173` (Vite). Überschreiben via `CORS_ORIGINS` (CSV). Bearer-Token = Supabase Access Token.

## Docker / Cloud Supabase

Das Root-[`Dockerfile`](../Dockerfile) baut Frontend + Backend in ein Image. Die SPA wird vom Go-Prozess auf `:8080` ausgeliefert; Supabase bleibt ein Cloud-Service.

### Cloud-Setup (einmalig)

1. Supabase-Projekt anlegen; Connection String (Pooler + `sslmode=require`), URL, Anon- und Service-Role-Key notieren.
2. Storage-Bucket `combat-logs` anlegen (privat, max. ~100 MiB; MIME wie lokal: `text/plain`, `application/octet-stream`, `application/zip`).
3. Auth → Redirect URLs: `{APP_ORIGIN}/auth/callback` (plus Site URL = App-Origin).
4. OAuth-Provider (Google/Discord) im Cloud-Dashboard konfigurieren.

### Image bauen & starten

```bash
docker build \
  --build-arg VITE_SUPABASE_URL=https://<ref>.supabase.co \
  --build-arg VITE_SUPABASE_ANON_KEY=<anon> \
  -t rg-logs .

docker run --rm -p 8080:8080 \
  -e DATABASE_URL='postgresql://...' \
  -e SUPABASE_URL=https://<ref>.supabase.co \
  -e SUPABASE_ANON_KEY=<anon> \
  -e SUPABASE_SERVICE_ROLE_KEY=<service_role> \
  rg-logs
```

CI pusht nach `ghcr.io/<owner>/rg-logs` (siehe `.github/workflows/docker-publish.yml`). Dafür Repo-Variables/Secrets `VITE_SUPABASE_URL` und `VITE_SUPABASE_ANON_KEY` setzen.

## Tests

```bash
cd backend
go test ./internal/parser/ -count=1
```
