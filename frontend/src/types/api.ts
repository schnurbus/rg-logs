/**
 * Assumed API contract for rg-logs backend.
 *
 * Preferred JSON: camelCase (as below). The client also accepts Go-style
 * snake_case via normalize helpers in `api/client.ts`.
 *
 * Endpoints:
 *   POST /api/uploads              → Upload (multipart field: "file")
 *   GET  /api/uploads              → Upload[]
 *   GET  /api/uploads/:id          → Upload (with fights)
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
  filename: string
  sizeBytes: number
  status: UploadStatus
  error?: string | null
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
}
