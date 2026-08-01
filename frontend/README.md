# Frontend (rg-logs)

Vite + React 19 + TypeScript + Tailwind v4 UI for the WoW Combat Log Analyzer.

## Run

```bash
cd frontend
npm install
npm run dev
```

Dev server proxies `/api` → `http://localhost:3000` (Fiber backend).

## Routes

| Path | Page |
|------|------|
| `/` | Upload (drag-and-drop) |
| `/uploads` | Uploads list with status badges |
| `/uploads/:uploadId` | Fight list for one upload (polls while pending/processing) |
| `/fights/:fightId` | Fight detail — Damage / Healing / Taken + spell breakdown |

## Assumed API contract

Preferred JSON **camelCase**. Client also accepts **snake_case** (Go `json` tags) via normalizers in `src/api/client.ts`. Canonical TypeScript shapes live in `src/types/api.ts`.

| Method | Path | Body / query | Response |
|--------|------|--------------|----------|
| `POST` | `/api/uploads` | multipart field `file` | `Upload` |
| `GET` | `/api/uploads` | — | `Upload[]` |
| `GET` | `/api/uploads/:id` | — | `Upload` (+ `fights`) |
| `GET` | `/api/fights/:id` | — | `FightDetail` |
| `GET` | `/api/fights/:id/spells` | `actorId` | `SpellStat[]` |
| `GET` | `/api/health` | — | health |

### Types (camelCase)

```ts
Upload: {
  id, filename, sizeBytes, status, // pending|processing|ready|failed
  error?, createdAt, processedAt?, fights?: FightSummary[]
}

FightSummary: {
  id, startTs, endTs, durationMs, title, kill, participantCount
}

FightDetail: FightSummary & {
  uploadId?, participants: Participant[]
}

Participant: {
  actorId, name, isPlayer,
  damageDone, healingDone, overheal, damageTaken, activeTimeMs,
  dps?, hps?  // computed client-side if missing
}

SpellStat: {
  spellId, spellName, school, metric, // damage|healing|damage_taken
  total, hits, crits
}
```

If the backend uses different field names, update the normalizers in `src/api/client.ts` — UI code consumes only the normalized camelCase types.
