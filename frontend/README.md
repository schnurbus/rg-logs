# Frontend (rg-logs)

Vite + React 19 + TypeScript + Tailwind v4 UI for the WoW Combat Log Analyzer.

## Run

Voraussetzungen: lokales Supabase (`supabase start`) und Backend auf `:3000`.

```bash
cd frontend
cp ../.env.example .env.local   # optional: VITE_SUPABASE_* als Fallback

npm install
npm run dev
```

Dev server proxies `/api` → `http://localhost:3000` (Fiber backend). Auth-Config kommt bevorzugt von `GET /api/config` (Backend-Env); `VITE_SUPABASE_*` nur als Fallback.

In Produktion wird dieses Frontend vom Go-Backend (Docker-Image) ausgeliefert — kein separater Vite-Server.

## Auth

- Magic Link / Google / Discord über Supabase
- Magic-Link-Mails lokal in Mailpit (`http://127.0.0.1:54324` oder Compose `:8025`)
- Callback: `/auth/callback`

## Routes

| Path | Page |
|------|------|
| `/` | Upload (Auth erforderlich, optional privat) |
| `/login` | Magic Link + Social Login |
| `/auth/callback` | OAuth / Magic-Link Callback |
| `/uploads` | Uploads (öffentlich + eigene; Löschen für Owner) |
| `/uploads/:uploadId` | Fight list (pollt während Processing) |
| `/fights/:fightId` | Fight detail — Damage / Healing / Taken + Spells |
