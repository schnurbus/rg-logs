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
 *   GET  /api/fights/:id/auras?kind=buff|debuff → AuraStat[]
 *   GET  /api/fights/:id/interrupts → CastCountStat[]
 *   GET  /api/fights/:id/dispels → CastCountStat[]
 *   GET  /api/fights/:id/timeline?mode=summary|damage|healing|taken → Timeline
 *   GET  /api/fights/:id/events → CombatEventList
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
  includeTrash?: boolean
  createdAt: string
  processedAt?: string | null
  fights?: FightSummary[]
}

export interface Participant {
  actorId: string
  guid: string
  name: string
  isPlayer: boolean
  /** CLEU unit flags (pet=0x1000, guardian=0x2000). */
  flags?: number
  /** Present for pets/guardians/summons owned by a player */
  ownerGuid?: string
  /** Inferred from signature spell IDs (WotLK). */
  class?: string
  /** Talent-tree specialization inferred from signature spell IDs (WotLK). */
  spec?: string
  /** GearScoreLite from Rising Gods profile at ingest time. */
  gearScore?: number
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

export type AuraKind = 'buff' | 'debuff'

export interface AuraStat {
  spellId: number
  spellName: string
  school: number
  applications: number
  refreshes: number
  targets: number
  uptimeMs: number
  /** Average uptime ratio across targets (0–1). */
  uptimePct: number
}

export interface CastCountStat {
  actorId: string
  actorName: string
  class?: string
  spec?: string
  spellId: number
  spellName: string
  extraSpellId: number
  extraSpellName: string
  count: number
}

export type TimelineMode = 'summary' | 'damage' | 'healing' | 'taken'

export interface TimelineSummaryPoint {
  t: number
  damage: number
  healing: number
  taken: number
}

export interface TimelineSummary {
  bucketMs: number
  points: TimelineSummaryPoint[]
}

export interface TimelineSeriesPoint {
  t: number
  amount: number
}

export interface TimelinePlayerSeries {
  actorId: string
  name: string
  class?: string
  points: TimelineSeriesPoint[]
  total: number
}

export interface TimelinePlayers {
  bucketMs: number
  series: TimelinePlayerSeries[]
}

export type CombatEventTypeFilter =
  | ''
  | 'damage'
  | 'heal'
  | 'miss'
  | 'death'
  | 'summon'
  | 'aura'
  | 'interrupt'
  | 'dispel'

export interface CombatEventRow {
  id: number
  offsetMs: number
  eventType: number
  sourceId?: string
  sourceName?: string
  sourceClass?: string
  sourceSpec?: string
  targetId?: string
  targetName?: string
  targetClass?: string
  targetSpec?: string
  spellId: number
  spellName?: string
  amount: number
  overkill: number
  overheal: number
  absorbed: number
  flags: number
  missType?: number
  extra: number
  extraSpellName?: string
}

export interface CombatEventList {
  total: number
  limit: number
  offset: number
  events: CombatEventRow[]
}
