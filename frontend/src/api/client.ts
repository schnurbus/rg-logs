import type {
  FightDetail,
  FightSummary,
  Participant,
  SpellStat,
  Upload,
  UploadStatus,
} from '../types/api'
import { asPlayerClass } from '../lib/classes'

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

  return {
    actorId: asString(pick(o, 'actorId', 'actor_id')),
    guid: asString(pick(o, 'guid', 'guid')),
    name: asString(pick(o, 'name', 'name'), 'Unknown'),
    isPlayer: asBool(pick(o, 'isPlayer', 'is_player')),
    ownerGuid,
    class: asPlayerClass(pick(o, 'class', 'class')),
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
    filename: asString(pick(o, 'filename', 'filename')),
    sizeBytes: asNumber(pick(o, 'sizeBytes', 'size_bytes')),
    status: asStatus(pick(o, 'status', 'status')),
    error: (() => {
      const e = pick<unknown>(o, 'error', 'error')
      if (e == null || e === '') return null
      return asString(e)
    })(),
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
  const res = await fetch(path, init)
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

export async function listUploads(): Promise<Upload[]> {
  const data = await request('/api/uploads')
  if (!Array.isArray(data)) return []
  return data.map(normalizeUpload)
}

export async function getUpload(id: string): Promise<Upload> {
  const data = await request(`/api/uploads/${encodeURIComponent(id)}`)
  return normalizeUpload(data)
}

export async function uploadFile(file: File): Promise<Upload> {
  const form = new FormData()
  form.append('file', file)
  const data = await request('/api/uploads', {
    method: 'POST',
    body: form,
  })
  return normalizeUpload(data)
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
