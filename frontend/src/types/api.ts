/**
 * Assumed API contract for rg-logs backend.
 *
 * Preferred JSON: camelCase (as below). The client also accepts Go-style
 * snake_case via normalize helpers in `api/client.ts`.
 *
 * Endpoints:
 *   POST /api/uploads              → Upload (multipart: file, is_private, name) [auth]
 *   GET  /api/uploads              → Upload[] (public + own; ?mine=1)
 *   GET  /api/uploads/:id          → Upload (with fights)
 *   PATCH /api/uploads/:id         → Upload { name } [owner]
 *   DELETE /api/uploads/:id        → 204 [owner]
 *   GET  /api/fights/:id           → FightDetail
 *   GET  /api/fights/:id/spells?actorId= → SpellStat[]
 *   GET  /api/health
 */

export type UploadStatus = 'pending' | 'processing' | 'ready' | 'failed'

export interface FightSummary {
  id: string
  startTs: string
  endTs: string
  durationMs: number
  title: string
  kill: boolean
  participantCount: number
}

export interface Upload {
  id: string
  userId?: string
  name?: string
  filename: string
  sizeBytes: number
  status: UploadStatus
  error?: string | null
  contentHash?: string
  isPrivate?: boolean
  createdAt: string
  processedAt?: string | null
  fights?: FightSummary[]
}

export interface Participant {
  actorId: string
  guid: string
  name: string
  isPlayer: boolean
  /** Present for pets/guardians/summons owned by a player */
  ownerGuid?: string
  /** Inferred from signature spell IDs (WotLK). */
  class?: string
  damageDone: number
  healingDone: number
  overheal: number
  damageTaken: number
  activeTimeMs: number
  /** Optional; computed client-side from damageDone / (activeTimeMs/1000) if missing */
  dps?: number
  /** Optional; computed client-side from healingDone / (activeTimeMs/1000) if missing */
  hps?: number
}

export interface FightDetail extends FightSummary {
  uploadId?: string
  participants: Participant[]
}

export type SpellMetric = 'damage' | 'healing' | 'damage_taken'

export interface SpellStat {
  spellId: number
  spellName: string
  school: number | string
  metric: SpellMetric | string
  total: number
  hits: number
  crits: number
  ticks?: number
  /** Count of MISS outcomes (not dodge/parry/etc.). */
  misses: number
  /** Count of glancing hits. */
  glancing: number
  normalHits: number
  normalTotal: number
  normalMin: number
  normalMax: number
  critTotal: number
  critMin: number
  critMax: number
  glancingTotal: number
  glancingMin: number
  glancingMax: number
  /** Aggregated pet/summon contribution under the owner. */
  pet?: boolean
}
