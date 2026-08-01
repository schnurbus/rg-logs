import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ApiError, getFight, getFightSpells } from '../api/client'
import { ErrorMessage, Loading } from '../components/ErrorMessage'
import { MeterBar } from '../components/MeterBar'
import { SpellStatTooltip } from '../components/SpellStatTooltip'
import {
  asPlayerClass,
  classTextStyle,
  type PlayerClass,
} from '../lib/classes'
import {
  formatDuration,
  formatNumber,
  formatPercent,
  formatRate,
} from '../lib/format'
import type { FightDetail, Participant, SpellStat } from '../types/api'

type MetricTab = 'damage' | 'healing' | 'taken'

const TABS: { id: MetricTab; label: string }[] = [
  { id: 'damage', label: 'Damage Done' },
  { id: 'healing', label: 'Healing Done' },
  { id: 'taken', label: 'Damage Taken' },
]

type MeterRow = {
  player: Participant
  /** Player stats with pet damage/healing/taken rolled into totals */
  totals: Participant
}

function amountFor(p: Participant, tab: MetricTab): number {
  switch (tab) {
    case 'damage':
      return p.damageDone
    case 'healing':
      return p.healingDone
    case 'taken':
      return p.damageTaken
  }
}

function rateFor(
  p: Participant,
  tab: MetricTab,
  durationMs: number,
): number | undefined {
  const seconds = durationMs > 0 ? durationMs / 1000 : 0
  if (seconds <= 0) return undefined
  if (tab === 'damage') return p.damageDone / seconds
  if (tab === 'healing') {
    const effective = Math.max(0, p.healingDone - p.overheal)
    return effective / seconds
  }
  return p.damageTaken / seconds
}

function rateLabel(tab: MetricTab): string {
  if (tab === 'healing') return 'HPS'
  if (tab === 'taken') return 'DTPS'
  return 'DPS'
}

function playerClassOf(p: Participant): PlayerClass | undefined {
  return asPlayerClass(p.class)
}

/** Roll pet damage into player totals. Pets are not listed separately. */
function buildMeterRows(
  participants: Participant[],
  durationMs: number,
): MeterRow[] {
  const players = participants.filter((p) => p.isPlayer)
  const byGuid = new Map(players.map((p) => [p.guid, p]))

  const petsByOwner = new Map<string, Participant[]>()
  for (const p of participants) {
    if (p.isPlayer || !p.ownerGuid) continue
    if (!byGuid.has(p.ownerGuid)) continue
    const list = petsByOwner.get(p.ownerGuid) ?? []
    list.push(p)
    petsByOwner.set(p.ownerGuid, list)
  }

  const secs = durationMs > 0 ? durationMs / 1000 : 0

  return players.map((player) => {
    const pets = petsByOwner.get(player.guid) ?? []
    const damageDone =
      player.damageDone + pets.reduce((s, pet) => s + pet.damageDone, 0)
    const healingDone =
      player.healingDone + pets.reduce((s, pet) => s + pet.healingDone, 0)
    const overheal =
      player.overheal + pets.reduce((s, pet) => s + pet.overheal, 0)
    const damageTaken =
      player.damageTaken + pets.reduce((s, pet) => s + pet.damageTaken, 0)

    const totals: Participant = {
      ...player,
      damageDone,
      healingDone,
      overheal,
      damageTaken,
      dps: secs > 0 ? damageDone / secs : undefined,
      hps: secs > 0 ? Math.max(0, healingDone - overheal) / secs : undefined,
    }

    return { player, totals }
  })
}

export function FightDetailPage() {
  const { fightId } = useParams<{ fightId: string }>()
  const [fight, setFight] = useState<FightDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [tab, setTab] = useState<MetricTab>('damage')
  const [selected, setSelected] = useState<Participant | null>(null)
  const [spells, setSpells] = useState<SpellStat[] | null>(null)
  const [spellsError, setSpellsError] = useState<string | null>(null)
  const [spellsLoading, setSpellsLoading] = useState(false)

  const load = useCallback(async () => {
    if (!fightId) return
    try {
      const data = await getFight(fightId)
      setFight(data)
      setError(null)
    } catch (err) {
      setError(
        err instanceof ApiError
          ? `Fight nicht geladen (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Unbekannter Fehler',
      )
    }
  }, [fightId])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    setSelected(null)
    setSpells(null)
    setSpellsError(null)
  }, [tab])

  const rows = useMemo(() => {
    if (!fight) return []
    return buildMeterRows(fight.participants, fight.durationMs)
      .filter((row) => amountFor(row.totals, tab) > 0)
      .sort((a, b) => amountFor(b.totals, tab) - amountFor(a.totals, tab))
  }, [fight, tab])

  const total = useMemo(
    () => rows.reduce((sum, row) => sum + amountFor(row.totals, tab), 0),
    [rows, tab],
  )

  /** Top player amount — bar widths are relative to this (WCL-style). */
  const maxAmount = useMemo(
    () =>
      rows.reduce((max, row) => Math.max(max, amountFor(row.totals, tab)), 0),
    [rows, tab],
  )

  const openSpells = async (p: Participant) => {
    if (!fightId) return
    setSelected(p)
    setSpells(null)
    setSpellsError(null)
    setSpellsLoading(true)
    try {
      const data = await getFightSpells(fightId, p.actorId)
      // Prefer spells matching current metric; fall back to all
      const metricFilter =
        tab === 'damage'
          ? 'damage'
          : tab === 'healing'
            ? 'healing'
            : 'damage_taken'
      const filtered = data.filter((s) => s.metric === metricFilter)
      const list = (filtered.length > 0 ? filtered : data).sort(
        (a, b) => b.total - a.total,
      )
      setSpells(list)
    } catch (err) {
      setSpellsError(
        err instanceof ApiError
          ? `Spell-Breakdown fehlgeschlagen (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Unbekannter Fehler',
      )
    } finally {
      setSpellsLoading(false)
    }
  }

  if (error && !fight) {
    return <ErrorMessage message={error} onRetry={load} />
  }

  if (!fight) {
    return <Loading label="Fight laden…" />
  }

  const selectedClass = selected ? playerClassOf(selected) : undefined

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs text-text-muted">
          {fight.uploadId ? (
            <>
              <Link to={`/uploads/${fight.uploadId}`}>Upload</Link>
              <span className="mx-1.5">/</span>
            </>
          ) : (
            <>
              <Link to="/uploads">Uploads</Link>
              <span className="mx-1.5">/</span>
            </>
          )}
          <span className="text-text">{fight.title}</span>
        </p>
        <h1 className="mt-2 text-xl font-semibold tracking-tight">
          {fight.title || 'Fight'}
        </h1>
        <p className="mt-1 text-sm text-text-muted">
          Dauer {formatDuration(fight.durationMs)}
          {fight.kill ? ' · Kill' : ''}
          {' · '}
          {fight.participantCount} Teilnehmer
        </p>
      </div>

      <div
        role="tablist"
        className="flex flex-wrap gap-1 border-b border-border pb-px"
      >
        {TABS.map((t) => (
          <button
            key={t.id}
            type="button"
            role="tab"
            aria-selected={tab === t.id}
            onClick={() => setTab(t.id)}
            className={[
              'rounded-t px-3 py-2 text-sm transition-colors',
              tab === t.id
                ? 'bg-surface-overlay text-text ring-1 ring-border'
                : 'text-text-muted hover:text-text',
            ].join(' ')}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="overflow-x-auto rounded border border-border">
        <table className="w-full min-w-[640px] border-collapse text-left text-sm">
          <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
            <tr>
              <th className="px-3 py-2 font-medium w-8">#</th>
              <th className="px-3 py-2 font-medium min-w-[12rem]">Name</th>
              <th className="px-3 py-2 font-medium text-right">Amount</th>
              <th className="px-3 py-2 font-medium text-right">
                {rateLabel(tab)}
              </th>
              <th className="px-3 py-2 font-medium text-right">%</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => {
              const amount = amountFor(row.totals, tab)
              const rate = rateFor(row.totals, tab, fight.durationMs)
              const pct = total > 0 ? amount / total : 0
              const barRatio = maxAmount > 0 ? amount / maxAmount : 0
              const cls = playerClassOf(row.player)
              const active = selected?.actorId === row.player.actorId

              return (
                <tr
                  key={row.player.actorId || `${row.player.name}-${i}`}
                  onClick={() => void openSpells(row.player)}
                  className={[
                    'border-t border-border-subtle cursor-pointer',
                    active
                      ? 'bg-surface-overlay'
                      : 'hover:bg-surface-overlay/50',
                  ].join(' ')}
                >
                  <td className="px-3 py-2 text-text-muted">{i + 1}</td>
                  <td className="px-1 py-1.5">
                    <MeterBar ratio={barRatio} playerClass={cls}>
                      <span
                        className="font-medium"
                        style={classTextStyle(cls)}
                      >
                        {row.player.name}
                      </span>
                    </MeterBar>
                  </td>
                  <td className="px-3 py-2 text-right font-mono">
                    {formatNumber(amount)}
                  </td>
                  <td className="px-3 py-2 text-right font-mono text-text-muted">
                    {formatRate(rate)}
                  </td>
                  <td className="px-3 py-2 text-right font-mono text-text-muted">
                    {formatPercent(pct)}
                  </td>
                </tr>
              )
            })}
            {rows.length === 0 ? (
              <tr>
                <td
                  colSpan={5}
                  className="px-3 py-6 text-center text-text-muted"
                >
                  Keine Teilnehmer.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      {selected ? (
        <section className="rounded border border-border bg-surface-raised p-4">
          <div className="mb-3 flex items-center justify-between gap-3">
            <h2 className="text-sm font-semibold">
              Spell-Breakdown:{' '}
              <span style={classTextStyle(selectedClass)}>{selected.name}</span>
            </h2>
            <button
              type="button"
              className="text-xs text-text-muted hover:text-text"
              onClick={() => {
                setSelected(null)
                setSpells(null)
                setSpellsError(null)
              }}
            >
              Schließen
            </button>
          </div>

          {spellsLoading ? <Loading label="Spells laden…" /> : null}
          {spellsError ? <ErrorMessage message={spellsError} /> : null}

          {spells && spells.length === 0 ? (
            <p className="text-sm text-text-muted">Keine Spell-Stats.</p>
          ) : null}

          {spells && spells.length > 0 ? (
            <div>
              <table className="w-full min-w-[480px] border-collapse text-left text-sm">
                <thead className="text-xs uppercase tracking-wide text-text-muted">
                  <tr>
                    <th className="px-2 py-1.5 font-medium min-w-[10rem]">
                      Spell
                    </th>
                    <th className="px-2 py-1.5 font-medium text-right">
                      Total
                    </th>
                    <th className="px-2 py-1.5 font-medium text-right">Hits</th>
                    <th className="px-2 py-1.5 font-medium text-right">
                      Crits
                    </th>
                    <th className="px-2 py-1.5 font-medium text-right">%</th>
                  </tr>
                </thead>
                <tbody>
                  {(() => {
                    const spellTotal = spells.reduce((s, x) => s + x.total, 0)
                    const spellMax = spells.reduce(
                      (m, x) => Math.max(m, x.total),
                      0,
                    )
                    return spells.map((s) => (
                      <tr
                        key={`${s.spellId}-${s.metric}-${s.spellName}`}
                        className="border-t border-border-subtle"
                      >
                        <td className="px-0.5 py-1">
                          <MeterBar
                            ratio={spellMax > 0 ? s.total / spellMax : 0}
                            playerClass={selectedClass}
                            tooltip={<SpellStatTooltip spell={s} />}
                          >
                            <span className="text-text">{s.spellName}</span>
                            {s.pet ? (
                              <span className="ml-2 text-xs text-text-muted">
                                Pet
                              </span>
                            ) : (
                              <span className="ml-2 font-mono text-xs text-text-muted">
                                {s.spellId}
                              </span>
                            )}
                          </MeterBar>
                        </td>
                        <td className="px-2 py-1.5 text-right font-mono">
                          {formatNumber(s.total)}
                        </td>
                        <td className="px-2 py-1.5 text-right font-mono text-text-muted">
                          {formatNumber(s.hits)}
                        </td>
                        <td className="px-2 py-1.5 text-right font-mono text-text-muted">
                          {formatNumber(s.crits)}
                        </td>
                        <td className="px-2 py-1.5 text-right font-mono text-text-muted">
                          {formatPercent(
                            spellTotal > 0 ? s.total / spellTotal : 0,
                          )}
                        </td>
                      </tr>
                    ))
                  })()}
                </tbody>
              </table>
            </div>
          ) : null}
        </section>
      ) : (
        <p className="text-xs text-text-muted">
          Zeile anklicken für Spell-Breakdown.
        </p>
      )}
    </div>
  )
}
