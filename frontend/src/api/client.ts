import type {
  AuraKind,
  AuraStat,
  CastCountStat,
  CombatEventList,
  CombatEventRow,
  CombatEventTypeFilter,
  FightDetail,
  FightSummary,
  Participant,
  SpellStat,
  TimelineMode,
  TimelinePlayers,
  TimelinePlayerSeries,
  TimelineSeriesPoint,
  TimelineSummary,
  TimelineSummaryPoint,
  Upload,
  UploadStatus,
} from '../types/api'
import { asPlayerClass } from '../lib/classes'
import { asPlayerSpec } from '../lib/specs'
import { getApiAccessToken } from './token'

type Raw = Record<string, unknown>

function isObject(v: unknown): v is Raw {
  return typeof v === 'object' && v !== null && !Array.isArray(v)
}

/** Pick camelCase or snake_case field from a raw JSON object. */
function pick<T>(obj: Raw, camel: string, snake: string): T | undefined {
  if (camel in obj) return obj[camel] as T
  if (snake in obj) return obj[snake] as T
  return undefined
}

function asString(v: unknown, fallback = ''): string {
  if (v == null) return fallback
  return String(v)
}

function asNumber(v: unknown, fallback = 0): number {
  if (typeof v === 'number' && Number.isFinite(v)) return v
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v)
    if (Number.isFinite(n)) return n
  }
  return fallback
}

function asBool(v: unknown, fallback = false): boolean {
  if (typeof v === 'boolean') return v
  return fallback
}

function asStatus(v: unknown): UploadStatus {
  const s = asString(v, 'pending')
  if (s === 'pending' || s === 'processing' || s === 'ready' || s === 'failed') {
    return s
  }
  return 'pending'
}

export function normalizeFightSummary(raw: unknown): FightSummary {
  const o = isObject(raw) ? raw : {}
  return {
    id: asString(pick(o, 'id', 'id')),
    startTs: asString(pick(o, 'startTs', 'start_ts')),
    endTs: asString(pick(o, 'endTs', 'end_ts')),
    durationMs: asNumber(pick(o, 'durationMs', 'duration_ms')),
    title: asString(pick(o, 'title', 'title'), 'Trash'),
    kill: asBool(pick(o, 'kill', 'kill')),
    participantCount: asNumber(
      pick(o, 'participantCount', 'participant_count'),
    ),
  }
}

export function normalizeParticipant(raw: unknown): Participant {
  const o = isObject(raw) ? raw : {}
  const damageDone = asNumber(pick(o, 'damageDone', 'damage_done'))
  const healingDone = asNumber(pick(o, 'healingDone', 'healing_done'))
  const activeTimeMs = asNumber(pick(o, 'activeTimeMs', 'active_time_ms'))
  const seconds = activeTimeMs > 0 ? activeTimeMs / 1000 : 0

  const dpsRaw = pick<unknown>(o, 'dps', 'dps')
  const hpsRaw = pick<unknown>(o, 'hps', 'hps')
  const ownerRaw = pick<unknown>(o, 'ownerGuid', 'owner_guid')
  const ownerGuid =
    ownerRaw != null && asString(ownerRaw) !== ''
      ? asString(ownerRaw)
      : undefined

  const gearRaw = pick<unknown>(o, 'gearScore', 'gear_score')
  const gearScore =
    gearRaw != null && asNumber(gearRaw) > 0 ? asNumber(gearRaw) : undefined

  return {
    actorId: asString(pick(o, 'actorId', 'actor_id')),
    guid: asString(pick(o, 'guid', 'guid')),
    name: asString(pick(o, 'name', 'name'), 'Unknown'),
    isPlayer: asBool(pick(o, 'isPlayer', 'is_player')),
    flags: asNumber(pick(o, 'flags', 'flags')),
    ownerGuid,
    class: asPlayerClass(pick(o, 'class', 'class')),
    spec: asPlayerSpec(pick(o, 'spec', 'spec')),
    gearScore,
    damageDone,
    healingDone,
    overheal: asNumber(pick(o, 'overheal', 'overheal')),
    damageTaken: asNumber(pick(o, 'damageTaken', 'damage_taken')),
    activeTimeMs,
    dps:
      dpsRaw !== undefined
        ? asNumber(dpsRaw)
        : seconds > 0
          ? damageDone / seconds
          : undefined,
    hps:
      hpsRaw !== undefined
        ? asNumber(hpsRaw)
        : seconds > 0
          ? healingDone / seconds
          : undefined,
  }
}

export function normalizeUpload(raw: unknown): Upload {
  const o = isObject(raw) ? raw : {}
  const fightsRaw = pick<unknown>(o, 'fights', 'fights')
  return {
    id: asString(pick(o, 'id', 'id')),
    userId: asString(pick(o, 'userId', 'user_id')) || undefined,
    name: asString(pick(o, 'name', 'name')) || undefined,
    filename: asString(pick(o, 'filename', 'filename')),
    sizeBytes: asNumber(pick(o, 'sizeBytes', 'size_bytes')),
    status: asStatus(pick(o, 'status', 'status')),
    error: (() => {
      const e = pick<unknown>(o, 'error', 'error')
      if (e == null || e === '') return null
      return asString(e)
    })(),
    contentHash: asString(pick(o, 'contentHash', 'content_hash')) || undefined,
    isPrivate: asBool(pick(o, 'isPrivate', 'is_private')),
    includeTrash: asBool(pick(o, 'includeTrash', 'include_trash')),
    createdAt: asString(pick(o, 'createdAt', 'created_at')),
    processedAt: (() => {
      const p = pick<unknown>(o, 'processedAt', 'processed_at')
      if (p == null || p === '') return null
      return asString(p)
    })(),
    fights: Array.isArray(fightsRaw)
      ? fightsRaw.map(normalizeFightSummary)
      : undefined,
  }
}

export function normalizeFightDetail(raw: unknown): FightDetail {
  const summary = normalizeFightSummary(raw)
  const o = isObject(raw) ? raw : {}
  const participantsRaw = pick<unknown>(o, 'participants', 'participants')
  const uploadId = pick<unknown>(o, 'uploadId', 'upload_id')
  return {
    ...summary,
    uploadId: uploadId != null ? asString(uploadId) : undefined,
    participants: Array.isArray(participantsRaw)
      ? participantsRaw.map(normalizeParticipant)
      : [],
  }
}

export function normalizeSpellStat(raw: unknown): SpellStat {
  const o = isObject(raw) ? raw : {}
  return {
    spellId: asNumber(pick(o, 'spellId', 'spell_id')),
    spellName: asString(pick(o, 'spellName', 'spell_name'), 'Unknown'),
    school: (pick(o, 'school', 'school') as number | string) ?? 0,
    metric: asString(pick(o, 'metric', 'metric'), 'damage'),
    total: asNumber(pick(o, 'total', 'total')),
    hits: asNumber(pick(o, 'hits', 'hits')),
    crits: asNumber(pick(o, 'crits', 'crits')),
    ticks: asNumber(pick(o, 'ticks', 'ticks')),
    misses: asNumber(pick(o, 'misses', 'misses')),
    glancing: asNumber(pick(o, 'glancing', 'glancing')),
    normalHits: asNumber(pick(o, 'normalHits', 'normal_hits')),
    normalTotal: asNumber(pick(o, 'normalTotal', 'normal_total')),
    normalMin: asNumber(pick(o, 'normalMin', 'normal_min')),
    normalMax: asNumber(pick(o, 'normalMax', 'normal_max')),
    critTotal: asNumber(pick(o, 'critTotal', 'crit_total')),
    critMin: asNumber(pick(o, 'critMin', 'crit_min')),
    critMax: asNumber(pick(o, 'critMax', 'crit_max')),
    glancingTotal: asNumber(pick(o, 'glancingTotal', 'glancing_total')),
    glancingMin: asNumber(pick(o, 'glancingMin', 'glancing_min')),
    glancingMax: asNumber(pick(o, 'glancingMax', 'glancing_max')),
    pet: Boolean(pick(o, 'pet', 'pet')),
  }
}

export class ApiError extends Error {
  status: number
  body: string

  constructor(status: number, body: string) {
    super(`API ${status}: ${body || 'request failed'}`)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

async function request(path: string, init?: RequestInit): Promise<unknown> {
  const headers = new Headers(init?.headers)
  const token = getApiAccessToken()
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const res = await fetch(path, { ...init, headers })
  if (!res.ok) {
    const text = await res.text().catch(() => '')
    throw new ApiError(res.status, text)
  }
  if (res.status === 204) return null
  const contentType = res.headers.get('content-type') ?? ''
  if (contentType.includes('application/json')) {
    return res.json()
  }
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return text
  }
}

export async function listUploads(opts?: { mine?: boolean }): Promise<Upload[]> {
  const qs = opts?.mine ? '?mine=1' : ''
  const data = await request(`/api/uploads${qs}`)
  if (!Array.isArray(data)) return []
  return data.map(normalizeUpload)
}

export async function getUpload(id: string): Promise<Upload> {
  const data = await request(`/api/uploads/${encodeURIComponent(id)}`)
  return normalizeUpload(data)
}

export async function uploadFile(
  file: File,
  opts?: { isPrivate?: boolean; includeTrash?: boolean; name?: string },
): Promise<Upload> {
  const form = new FormData()
  form.append('file', file)
  if (opts?.isPrivate) {
    form.append('is_private', 'true')
  }
  if (opts?.includeTrash) {
    form.append('include_trash', 'true')
  }
  if (opts?.name?.trim()) {
    form.append('name', opts.name.trim())
  }
  try {
    const data = await request('/api/uploads', {
      method: 'POST',
      body: form,
    })
    return normalizeUpload(data)
  } catch (err) {
    if (err instanceof ApiError && err.status === 409) {
      try {
        const parsed = JSON.parse(err.body) as { upload?: unknown }
        if (parsed.upload) return normalizeUpload(parsed.upload)
      } catch {
        /* fall through */
      }
    }
    throw err
  }
}

export async function renameUpload(id: string, name: string): Promise<Upload> {
  const data = await request(`/api/uploads/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name }),
  })
  return normalizeUpload(data)
}

export async function deleteUpload(id: string): Promise<void> {
  await request(`/api/uploads/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export async function getFight(id: string): Promise<FightDetail> {
  const data = await request(`/api/fights/${encodeURIComponent(id)}`)
  return normalizeFightDetail(data)
}

export async function getFightSpells(
  fightId: string,
  actorId: string,
): Promise<SpellStat[]> {
  const qs = new URLSearchParams({ actorId })
  const data = await request(
    `/api/fights/${encodeURIComponent(fightId)}/spells?${qs}`,
  )
  if (!Array.isArray(data)) return []
  return data.map(normalizeSpellStat)
}

export function normalizeAuraStat(raw: unknown): AuraStat {
  const o = isObject(raw) ? raw : {}
  return {
    spellId: asNumber(pick(o, 'spellId', 'spell_id')),
    spellName: asString(pick(o, 'spellName', 'spell_name'), 'Unknown'),
    school: asNumber(pick(o, 'school', 'school')),
    applications: asNumber(pick(o, 'applications', 'applications')),
    refreshes: asNumber(pick(o, 'refreshes', 'refreshes')),
    targets: asNumber(pick(o, 'targets', 'targets')),
    uptimeMs: asNumber(pick(o, 'uptimeMs', 'uptime_ms')),
    uptimePct: asNumber(pick(o, 'uptimePct', 'uptime_pct')),
  }
}

export function normalizeCastCountStat(raw: unknown): CastCountStat {
  const o = isObject(raw) ? raw : {}
  return {
    actorId: asString(pick(o, 'actorId', 'actor_id')),
    actorName: asString(pick(o, 'actorName', 'actor_name'), 'Unknown'),
    class: asPlayerClass(pick(o, 'class', 'class')),
    spec: asPlayerSpec(pick(o, 'spec', 'spec')),
    spellId: asNumber(pick(o, 'spellId', 'spell_id')),
    spellName: asString(pick(o, 'spellName', 'spell_name'), 'Unknown'),
    extraSpellId: asNumber(pick(o, 'extraSpellId', 'extra_spell_id')),
    extraSpellName: asString(
      pick(o, 'extraSpellName', 'extra_spell_name'),
      '',
    ),
    count: asNumber(pick(o, 'count', 'count')),
  }
}

export async function getFightAuras(
  fightId: string,
  kind: AuraKind,
): Promise<AuraStat[]> {
  const qs = new URLSearchParams({ kind })
  const data = await request(
    `/api/fights/${encodeURIComponent(fightId)}/auras?${qs}`,
  )
  if (!Array.isArray(data)) return []
  return data.map(normalizeAuraStat)
}

export async function getFightInterrupts(
  fightId: string,
): Promise<CastCountStat[]> {
  const data = await request(
    `/api/fights/${encodeURIComponent(fightId)}/interrupts`,
  )
  if (!Array.isArray(data)) return []
  return data.map(normalizeCastCountStat)
}

export async function getFightDispels(
  fightId: string,
): Promise<CastCountStat[]> {
  const data = await request(
    `/api/fights/${encodeURIComponent(fightId)}/dispels`,
  )
  if (!Array.isArray(data)) return []
  return data.map(normalizeCastCountStat)
}

export function normalizeTimelineSummary(raw: unknown): TimelineSummary {
  const o = isObject(raw) ? raw : {}
  const pointsRaw = pick<unknown>(o, 'points', 'points')
  const points: TimelineSummaryPoint[] = Array.isArray(pointsRaw)
    ? pointsRaw.map((p) => {
        const po = isObject(p) ? p : {}
        return {
          t: asNumber(pick(po, 't', 't')),
          damage: asNumber(pick(po, 'damage', 'damage')),
          healing: asNumber(pick(po, 'healing', 'healing')),
          taken: asNumber(pick(po, 'taken', 'taken')),
        }
      })
    : []
  return {
    bucketMs: asNumber(pick(o, 'bucketMs', 'bucket_ms'), 1000),
    points,
  }
}

export function normalizeTimelinePlayers(raw: unknown): TimelinePlayers {
  const o = isObject(raw) ? raw : {}
  const seriesRaw = pick<unknown>(o, 'series', 'series')
  const series: TimelinePlayerSeries[] = Array.isArray(seriesRaw)
    ? seriesRaw.map((s) => {
        const so = isObject(s) ? s : {}
        const ptsRaw = pick<unknown>(so, 'points', 'points')
        const points: TimelineSeriesPoint[] = Array.isArray(ptsRaw)
          ? ptsRaw.map((p) => {
              const po = isObject(p) ? p : {}
              return {
                t: asNumber(pick(po, 't', 't')),
                amount: asNumber(pick(po, 'amount', 'amount')),
              }
            })
          : []
        return {
          actorId: asString(pick(so, 'actorId', 'actor_id')),
          name: asString(pick(so, 'name', 'name'), 'Unknown'),
          class: asPlayerClass(pick(so, 'class', 'class')),
          points,
          total: asNumber(pick(so, 'total', 'total')),
        }
      })
    : []
  return {
    bucketMs: asNumber(pick(o, 'bucketMs', 'bucket_ms'), 1000),
    series,
  }
}

export type TimelineSide = 'players' | 'enemies'

export async function getFightTimelineSummary(
  fightId: string,
  side: TimelineSide = 'players',
): Promise<TimelineSummary> {
  const qs = new URLSearchParams({ mode: 'summary', side })
  const data = await request(
    `/api/fights/${encodeURIComponent(fightId)}/timeline?${qs}`,
  )
  return normalizeTimelineSummary(data)
}

export async function getFightTimelinePlayers(
  fightId: string,
  mode: Exclude<TimelineMode, 'summary'>,
  side: TimelineSide = 'players',
): Promise<TimelinePlayers> {
  const qs = new URLSearchParams({ mode, side })
  const data = await request(
    `/api/fights/${encodeURIComponent(fightId)}/timeline?${qs}`,
  )
  return normalizeTimelinePlayers(data)
}

export function normalizeCombatEventRow(raw: unknown): CombatEventRow {
  const o = isObject(raw) ? raw : {}
  const sourceId = pick<unknown>(o, 'sourceId', 'source_id')
  const targetId = pick<unknown>(o, 'targetId', 'target_id')
  const missType = pick<unknown>(o, 'missType', 'miss_type')
  return {
    id: asNumber(pick(o, 'id', 'id')),
    offsetMs: asNumber(pick(o, 'offsetMs', 'offset_ms')),
    eventType: asNumber(pick(o, 'eventType', 'event_type')),
    sourceId:
      sourceId != null && asString(sourceId) !== ''
        ? asString(sourceId)
        : undefined,
    sourceName: asString(pick(o, 'sourceName', 'source_name')) || undefined,
    sourceClass: asPlayerClass(pick(o, 'sourceClass', 'source_class')),
    sourceSpec: asPlayerSpec(pick(o, 'sourceSpec', 'source_spec')),
    targetId:
      targetId != null && asString(targetId) !== ''
        ? asString(targetId)
        : undefined,
    targetName: asString(pick(o, 'targetName', 'target_name')) || undefined,
    targetClass: asPlayerClass(pick(o, 'targetClass', 'target_class')),
    targetSpec: asPlayerSpec(pick(o, 'targetSpec', 'target_spec')),
    spellId: asNumber(pick(o, 'spellId', 'spell_id')),
    spellName: asString(pick(o, 'spellName', 'spell_name')) || undefined,
    amount: asNumber(pick(o, 'amount', 'amount')),
    overkill: asNumber(pick(o, 'overkill', 'overkill')),
    overheal: asNumber(pick(o, 'overheal', 'overheal')),
    absorbed: asNumber(pick(o, 'absorbed', 'absorbed')),
    flags: asNumber(pick(o, 'flags', 'flags')),
    missType:
      missType != null && asString(missType) !== ''
        ? asNumber(missType)
        : undefined,
    extra: asNumber(pick(o, 'extra', 'extra')),
    extraSpellName:
      asString(pick(o, 'extraSpellName', 'extra_spell_name')) || undefined,
  }
}

export async function getFightEvents(
  fightId: string,
  opts: {
    limit?: number
    offset?: number
    type?: CombatEventTypeFilter
    actorId?: string
  } = {},
): Promise<CombatEventList> {
  const qs = new URLSearchParams()
  if (opts.limit != null) qs.set('limit', String(opts.limit))
  if (opts.offset != null) qs.set('offset', String(opts.offset))
  if (opts.type) qs.set('type', opts.type)
  if (opts.actorId) qs.set('actorId', opts.actorId)
  const q = qs.toString()
  const data = await request(
    `/api/fights/${encodeURIComponent(fightId)}/events${q ? `?${q}` : ''}`,
  )
  const o = isObject(data) ? data : {}
  const eventsRaw = pick<unknown>(o, 'events', 'events')
  return {
    total: asNumber(pick(o, 'total', 'total')),
    limit: asNumber(pick(o, 'limit', 'limit'), opts.limit ?? 100),
    offset: asNumber(pick(o, 'offset', 'offset'), opts.offset ?? 0),
    events: Array.isArray(eventsRaw)
      ? eventsRaw.map(normalizeCombatEventRow)
      : [],
  }
}
