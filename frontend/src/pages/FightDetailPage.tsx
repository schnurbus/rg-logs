import { useCallback, useEffect, useMemo, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  ApiError,
  getFight,
  getFightAuras,
  getFightDispels,
  getFightInterrupts,
  getFightSpells,
  getFightTimelinePlayers,
  getFightTimelineSummary,
} from '../api/client'
import { AuraTable } from '../components/AuraTable'
import { CastCountTable } from '../components/CastCountTable'
import { ErrorMessage, Loading } from '../components/ErrorMessage'
import { FightHeader, type FightSide } from '../components/FightHeader'
import { FightPlayerTimelineChart } from '../components/FightPlayerTimelineChart'
import { FightTimelineChart } from '../components/FightTimelineChart'
import { MeterBar } from '../components/MeterBar'
import { RaidComposition } from '../components/RaidComposition'
import { SpecIcon } from '../components/SpecIcon'
import { SpellStatTooltip } from '../components/SpellStatTooltip'
import {
  asPlayerClass,
  classTextStyle,
  type PlayerClass,
} from '../lib/classes'
import {
  formatNumber,
  formatPercent,
  formatRate,
} from '../lib/format'
import { risingGodsProfileURL } from '../lib/risingGods'
import type {
  AuraStat,
  CastCountStat,
  FightDetail,
  Participant,
  SpellStat,
  TimelinePlayers,
  TimelineSummary,
} from '../types/api'

type FightTab =
  | 'summary'
  | 'damage'
  | 'taken'
  | 'healing'
  | 'threat'
  | 'buffs'
  | 'debuffs'
  | 'deaths'
  | 'interrupts'
  | 'dispels'

type MeterTab = 'damage' | 'healing' | 'taken'

const TABS: { id: FightTab; label: string }[] = [
  { id: 'summary', label: 'Zusammenfassung' },
  { id: 'damage', label: 'Verursachter Schaden' },
  { id: 'taken', label: 'Erlittener Schaden' },
  { id: 'healing', label: 'Heilung' },
  { id: 'threat', label: 'Bedrohung' },
  { id: 'buffs', label: 'Buffs' },
  { id: 'debuffs', label: 'Debuffs' },
  { id: 'deaths', label: 'Tode' },
  { id: 'interrupts', label: 'Unterbrechungen' },
  { id: 'dispels', label: 'Bannungen' },
]

const PLACEHOLDER_TABS = new Set<FightTab>(['threat', 'deaths'])

type MeterRow = {
  player: Participant
  /** Player stats with pet damage/healing/taken rolled into totals */
  totals: Participant
}

function isMeterTab(tab: FightTab): tab is MeterTab {
  return tab === 'damage' || tab === 'healing' || tab === 'taken'
}

function amountFor(p: Participant, tab: MeterTab): number {
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
  tab: MeterTab,
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

function rateLabel(tab: MeterTab): string {
  if (tab === 'healing') return 'HPS'
  if (tab === 'taken') return 'DTPS'
  return 'DPS'
}

function playerClassOf(p: Participant): PlayerClass | undefined {
  return asPlayerClass(p.class)
}

function ownedByPlayer(
  p: Participant,
  participants: Participant[],
): boolean {
  if (!p.ownerGuid) return false
  return participants.some(
    (o) => o.isPlayer && o.guid === p.ownerGuid,
  )
}

/** Enemy = non-player without a player owner (raid pets stay with players). */
function isEnemy(p: Participant, participants: Participant[]): boolean {
  return !p.isPlayer && !ownedByPlayer(p, participants)
}

/** Roll pet damage into player totals. Pets are not listed separately. */
function buildPlayerMeterRows(
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

/** Enemy rows without pet rollup; each NPC GUID is its own row. */
function buildEnemyMeterRows(
  participants: Participant[],
  durationMs: number,
): MeterRow[] {
  const secs = durationMs > 0 ? durationMs / 1000 : 0
  return participants
    .filter((p) => isEnemy(p, participants))
    .map((enemy) => {
      const totals: Participant = {
        ...enemy,
        dps: secs > 0 ? enemy.damageDone / secs : undefined,
        hps:
          secs > 0
            ? Math.max(0, enemy.healingDone - enemy.overheal) / secs
            : undefined,
      }
      return { player: enemy, totals }
    })
}

function buildMeterRows(
  participants: Participant[],
  durationMs: number,
  side: FightSide,
): MeterRow[] {
  return side === 'enemies'
    ? buildEnemyMeterRows(participants, durationMs)
    : buildPlayerMeterRows(participants, durationMs)
}

function sortedMeterRows(
  allRows: MeterRow[],
  metric: MeterTab,
): MeterRow[] {
  return allRows
    .filter((row) => amountFor(row.totals, metric) > 0)
    .sort(
      (a, b) => amountFor(b.totals, metric) - amountFor(a.totals, metric),
    )
}

type CompactMeterProps = {
  title: string
  metric: MeterTab
  rows: MeterRow[]
  durationMs: number
  side: FightSide
}

function CompactMeterTable({
  title,
  metric,
  rows,
  durationMs,
  side,
}: CompactMeterProps) {
  const total = rows.reduce((sum, row) => sum + amountFor(row.totals, metric), 0)
  const maxAmount = rows.reduce(
    (max, row) => Math.max(max, amountFor(row.totals, metric)),
    0,
  )
  const showPlayerChrome = side === 'players'

  return (
    <div className="overflow-hidden rounded border border-border">
      <div className="border-b border-border bg-surface-overlay px-3 py-2">
        <h3 className="text-sm font-semibold">{title}</h3>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[320px] border-collapse text-left text-sm">
          <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
            <tr>
              <th className="px-3 py-2 font-medium min-w-[10rem]">Name</th>
              <th className="px-3 py-2 font-medium text-right">%</th>
              <th className="px-3 py-2 font-medium text-right">Amount</th>
              <th className="px-3 py-2 font-medium text-right">
                {rateLabel(metric)}
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row, i) => {
              const amount = amountFor(row.totals, metric)
              const rate = rateFor(row.totals, metric, durationMs)
              const pct = total > 0 ? amount / total : 0
              const barRatio = maxAmount > 0 ? amount / maxAmount : 0
              const cls = showPlayerChrome
                ? playerClassOf(row.player)
                : undefined
              const profileURL = showPlayerChrome
                ? risingGodsProfileURL(row.player.name)
                : undefined

              return (
                <tr
                  key={row.player.actorId || `${row.player.name}-${i}`}
                  className="border-t border-border-subtle"
                >
                  <td className="px-1 py-1.5">
                    <MeterBar ratio={barRatio} playerClass={cls}>
                      <span className="inline-flex items-center gap-1.5">
                        {showPlayerChrome ? (
                          <>
                            <SpecIcon
                              spec={row.player.spec}
                              playerClass={row.player.class}
                              name={row.player.name}
                            />
                            <a
                              href={profileURL}
                              target="_blank"
                              rel="noopener noreferrer"
                              title="Rising Gods Profil"
                              className="font-medium hover:underline"
                              style={classTextStyle(cls)}
                            >
                              {row.player.name}
                            </a>
                          </>
                        ) : (
                          <span className="font-medium">{row.player.name}</span>
                        )}
                      </span>
                    </MeterBar>
                  </td>
                  <td className="px-3 py-2 text-right font-mono text-text-muted">
                    {formatPercent(pct)}
                  </td>
                  <td className="px-3 py-2 text-right font-mono">
                    {formatNumber(amount)}
                  </td>
                  <td className="px-3 py-2 text-right font-mono text-text-muted">
                    {formatRate(rate)}
                  </td>
                </tr>
              )
            })}
            {rows.length === 0 ? (
              <tr>
                <td
                  colSpan={4}
                  className="px-3 py-6 text-center text-text-muted"
                >
                  Keine Daten.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>
    </div>
  )
}

type FullMeterProps = {
  metric: MeterTab
  rows: MeterRow[]
  durationMs: number
  selected: Participant | null
  onSelect: (p: Participant) => void
  side: FightSide
}

function FullMeterTable({
  metric,
  rows,
  durationMs,
  selected,
  onSelect,
  side,
}: FullMeterProps) {
  const total = rows.reduce((sum, row) => sum + amountFor(row.totals, metric), 0)
  const maxAmount = rows.reduce(
    (max, row) => Math.max(max, amountFor(row.totals, metric)),
    0,
  )
  const showPlayerChrome = side === 'players'

  return (
    <div className="overflow-x-auto rounded border border-border">
      <table className="w-full min-w-[640px] border-collapse text-left text-sm">
        <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
          <tr>
            <th className="px-3 py-2 font-medium w-8">#</th>
            <th className="px-3 py-2 font-medium min-w-[12rem]">Name</th>
            <th className="px-3 py-2 font-medium text-right">GS</th>
            <th className="px-3 py-2 font-medium text-right">Amount</th>
            <th className="px-3 py-2 font-medium text-right">
              {rateLabel(metric)}
            </th>
            <th className="px-3 py-2 font-medium text-right">%</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => {
            const amount = amountFor(row.totals, metric)
            const rate = rateFor(row.totals, metric, durationMs)
            const pct = total > 0 ? amount / total : 0
            const barRatio = maxAmount > 0 ? amount / maxAmount : 0
            const cls = showPlayerChrome
              ? playerClassOf(row.player)
              : undefined
            const active = selected?.actorId === row.player.actorId
            const profileURL = showPlayerChrome
              ? risingGodsProfileURL(row.player.name)
              : undefined

            return (
              <tr
                key={row.player.actorId || `${row.player.name}-${i}`}
                onClick={() => onSelect(row.player)}
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
                    <span className="inline-flex items-center gap-1.5">
                      {showPlayerChrome ? (
                        <>
                          <SpecIcon
                            spec={row.player.spec}
                            playerClass={row.player.class}
                            name={row.player.name}
                          />
                          <a
                            href={profileURL}
                            target="_blank"
                            rel="noopener noreferrer"
                            title="Rising Gods Profil"
                            onClick={(e) => e.stopPropagation()}
                            className="font-medium hover:underline"
                            style={classTextStyle(cls)}
                          >
                            {row.player.name}
                          </a>
                        </>
                      ) : (
                        <span className="font-medium">{row.player.name}</span>
                      )}
                    </span>
                  </MeterBar>
                </td>
                <td className="px-3 py-2 text-right font-mono text-text-muted">
                  {showPlayerChrome && row.player.gearScore != null
                    ? formatNumber(row.player.gearScore)
                    : '—'}
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
                colSpan={6}
                className="px-3 py-6 text-center text-text-muted"
              >
                Keine Teilnehmer.
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
  )
}

export function FightDetailPage() {
  const { fightId } = useParams<{ fightId: string }>()
  const [fight, setFight] = useState<FightDetail | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [tab, setTab] = useState<FightTab>('summary')
  const [side, setSide] = useState<FightSide>('players')
  const [selected, setSelected] = useState<Participant | null>(null)
  const [spells, setSpells] = useState<SpellStat[] | null>(null)
  const [spellsError, setSpellsError] = useState<string | null>(null)
  const [spellsLoading, setSpellsLoading] = useState(false)

  const [auras, setAuras] = useState<AuraStat[] | null>(null)
  const [aurasError, setAurasError] = useState<string | null>(null)
  const [aurasLoading, setAurasLoading] = useState(false)

  const [castRows, setCastRows] = useState<CastCountStat[] | null>(null)
  const [castError, setCastError] = useState<string | null>(null)
  const [castLoading, setCastLoading] = useState(false)

  const [summaryTimeline, setSummaryTimeline] =
    useState<TimelineSummary | null>(null)
  const [summaryTimelineError, setSummaryTimelineError] = useState<
    string | null
  >(null)
  const [summaryTimelineLoading, setSummaryTimelineLoading] = useState(false)

  const [playerTimeline, setPlayerTimeline] = useState<TimelinePlayers | null>(
    null,
  )
  const [playerTimelineError, setPlayerTimelineError] = useState<string | null>(
    null,
  )
  const [playerTimelineLoading, setPlayerTimelineLoading] = useState(false)

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

  useEffect(() => {
    if (!fightId) return
    if (tab !== 'buffs' && tab !== 'debuffs') return

    let cancelled = false
    setAuras(null)
    setAurasError(null)
    setAurasLoading(true)
    void (async () => {
      try {
        const kind = tab === 'buffs' ? 'buff' : 'debuff'
        const data = await getFightAuras(fightId, kind)
        if (!cancelled) setAuras(data)
      } catch (err) {
        if (!cancelled) {
          setAurasError(
            err instanceof ApiError
              ? `Auras nicht geladen (${err.status})`
              : err instanceof Error
                ? err.message
                : 'Unbekannter Fehler',
          )
        }
      } finally {
        if (!cancelled) setAurasLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [fightId, tab])

  useEffect(() => {
    if (!fightId) return
    if (tab !== 'interrupts' && tab !== 'dispels') return

    let cancelled = false
    setCastRows(null)
    setCastError(null)
    setCastLoading(true)
    void (async () => {
      try {
        const data =
          tab === 'interrupts'
            ? await getFightInterrupts(fightId)
            : await getFightDispels(fightId)
        if (!cancelled) setCastRows(data)
      } catch (err) {
        if (!cancelled) {
          setCastError(
            err instanceof ApiError
              ? `Daten nicht geladen (${err.status})`
              : err instanceof Error
                ? err.message
                : 'Unbekannter Fehler',
          )
        }
      } finally {
        if (!cancelled) setCastLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [fightId, tab])

  useEffect(() => {
    if (!fightId || tab !== 'summary') return
    let cancelled = false
    setSummaryTimeline(null)
    setSummaryTimelineError(null)
    setSummaryTimelineLoading(true)
    void (async () => {
      try {
        const data = await getFightTimelineSummary(fightId, side)
        if (!cancelled) setSummaryTimeline(data)
      } catch (err) {
        if (!cancelled) {
          setSummaryTimelineError(
            err instanceof ApiError
              ? `Timeline nicht geladen (${err.status})`
              : err instanceof Error
                ? err.message
                : 'Unbekannter Fehler',
          )
        }
      } finally {
        if (!cancelled) setSummaryTimelineLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [fightId, tab, side])

  useEffect(() => {
    if (!fightId || !isMeterTab(tab)) return
    let cancelled = false
    setPlayerTimeline(null)
    setPlayerTimelineError(null)
    setPlayerTimelineLoading(true)
    void (async () => {
      try {
        const data = await getFightTimelinePlayers(fightId, tab, side)
        if (!cancelled) setPlayerTimeline(data)
      } catch (err) {
        if (!cancelled) {
          setPlayerTimelineError(
            err instanceof ApiError
              ? `Timeline nicht geladen (${err.status})`
              : err instanceof Error
                ? err.message
                : 'Unbekannter Fehler',
          )
        }
      } finally {
        if (!cancelled) setPlayerTimelineLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [fightId, tab, side])

  const allRows = useMemo(() => {
    if (!fight) return []
    return buildMeterRows(fight.participants, fight.durationMs, side)
  }, [fight, side])

  const meterRows = useMemo(() => {
    if (!isMeterTab(tab)) return []
    return sortedMeterRows(allRows, tab)
  }, [allRows, tab])

  const damageSummaryRows = useMemo(
    () => sortedMeterRows(allRows, 'damage'),
    [allRows],
  )
  const healingSummaryRows = useMemo(
    () => sortedMeterRows(allRows, 'healing'),
    [allRows],
  )

  const openSpells = async (p: Participant) => {
    if (!fightId || !isMeterTab(tab)) return
    setSelected(p)
    setSpells(null)
    setSpellsError(null)
    setSpellsLoading(true)
    try {
      const data = await getFightSpells(fightId, p.actorId)
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
      <FightHeader
        fight={fight}
        view="analyze"
        side={side}
        onSideChange={(next) => {
          setSide(next)
          setSelected(null)
          setSpells(null)
        }}
      />

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

      {tab === 'summary' ? (
        <div className="space-y-4">
          {summaryTimelineLoading ? <Loading label="Timeline laden…" /> : null}
          {summaryTimelineError ? (
            <ErrorMessage message={summaryTimelineError} />
          ) : null}
          {!summaryTimelineLoading &&
          !summaryTimelineError &&
          summaryTimeline ? (
            <FightTimelineChart data={summaryTimeline} />
          ) : null}
          <RaidComposition players={fight.participants} />
          <div className="grid gap-4 lg:grid-cols-2">
            <CompactMeterTable
              title="Verursachter Schaden"
              metric="damage"
              rows={damageSummaryRows}
              durationMs={fight.durationMs}
              side={side}
            />
            <CompactMeterTable
              title="Heilung"
              metric="healing"
              rows={healingSummaryRows}
              durationMs={fight.durationMs}
              side={side}
            />
          </div>
        </div>
      ) : null}

      {isMeterTab(tab) ? (
        <>
          {playerTimelineLoading ? <Loading label="Timeline laden…" /> : null}
          {playerTimelineError ? (
            <ErrorMessage message={playerTimelineError} />
          ) : null}
          {!playerTimelineLoading &&
          !playerTimelineError &&
          playerTimeline ? (
            <FightPlayerTimelineChart
              data={playerTimeline}
              title={
                tab === 'damage'
                  ? 'Schaden über Zeit'
                  : tab === 'healing'
                    ? 'Heilung über Zeit'
                    : 'Erlittener Schaden über Zeit'
              }
            />
          ) : null}
          <FullMeterTable
            metric={tab}
            rows={meterRows}
            durationMs={fight.durationMs}
            selected={selected}
            onSelect={(p) => void openSpells(p)}
            side={side}
          />

          {selected ? (
            <section className="rounded border border-border bg-surface-raised p-4">
              <div className="mb-3 flex items-center justify-between gap-3">
                <h2 className="text-sm font-semibold">
                  Spell-Breakdown:{' '}
                  <span style={classTextStyle(selectedClass)}>
                    {selected.name}
                  </span>
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
                        <th className="px-2 py-1.5 font-medium text-right">
                          Hits
                        </th>
                        <th className="px-2 py-1.5 font-medium text-right">
                          Crits
                        </th>
                        <th className="px-2 py-1.5 font-medium text-right">
                          %
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {(() => {
                        const spellTotal = spells.reduce(
                          (s, x) => s + x.total,
                          0,
                        )
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
                                ratio={
                                  spellMax > 0 ? s.total / spellMax : 0
                                }
                                playerClass={selectedClass}
                                tooltip={<SpellStatTooltip spell={s} />}
                              >
                                <span className="text-text">
                                  {s.spellName}
                                </span>
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
        </>
      ) : null}

      {tab === 'buffs' || tab === 'debuffs' ? (
        <div className="space-y-3">
          {aurasLoading ? <Loading label="Auras laden…" /> : null}
          {aurasError ? <ErrorMessage message={aurasError} /> : null}
          {!aurasLoading && !aurasError && auras ? (
            <AuraTable rows={auras} />
          ) : null}
        </div>
      ) : null}

      {tab === 'interrupts' || tab === 'dispels' ? (
        <div className="space-y-3">
          {castLoading ? <Loading label="Daten laden…" /> : null}
          {castError ? <ErrorMessage message={castError} /> : null}
          {!castLoading && !castError && castRows ? (
            <CastCountTable
              rows={castRows}
              extraColumnLabel={
                tab === 'interrupts' ? 'Unterbrochener Spell' : 'Gebannter Spell'
              }
            />
          ) : null}
        </div>
      ) : null}

      {PLACEHOLDER_TABS.has(tab) ? (
        <div className="rounded border border-border bg-surface-raised px-4 py-10 text-center text-sm text-text-muted">
          Noch nicht verfügbar.
        </div>
      ) : null}
    </div>
  )
}
