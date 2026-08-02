import { useCallback, useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ApiError, getFight, getFightEvents } from '../api/client'
import { ErrorMessage, Loading } from '../components/ErrorMessage'
import { FightHeader } from '../components/FightHeader'
import { SpecIcon } from '../components/SpecIcon'
import { asPlayerClass, classTextStyle } from '../lib/classes'
import { formatDuration, formatNumber } from '../lib/format'
import type {
  CombatEventList,
  CombatEventRow,
  CombatEventTypeFilter,
  FightDetail,
} from '../types/api'

const PAGE_SIZE = 100

const TYPE_FILTERS: { id: CombatEventTypeFilter; label: string }[] = [
  { id: '', label: 'Alle' },
  { id: 'damage', label: 'Schaden' },
  { id: 'heal', label: 'Heilung' },
  { id: 'miss', label: 'Miss' },
  { id: 'death', label: 'Tod' },
  { id: 'aura', label: 'Aura' },
  { id: 'interrupt', label: 'Interrupt' },
  { id: 'dispel', label: 'Dispel' },
  { id: 'summon', label: 'Summon' },
]

function eventTypeLabel(t: number): string {
  switch (t) {
    case 1:
      return 'Schaden'
    case 2:
      return 'Heilung'
    case 3:
      return 'Miss'
    case 4:
      return 'Tod'
    case 5:
      return 'Summon'
    case 6:
      return 'Aura+'
    case 7:
      return 'Aura−'
    case 8:
      return 'Aura↻'
    case 9:
      return 'Interrupt'
    case 10:
      return 'Dispel'
    default:
      return String(t)
  }
}

function ActorCell({
  name,
  playerClass,
  spec,
}: {
  name?: string
  playerClass?: string
  spec?: string
}) {
  if (!name) return <span className="text-text-muted">—</span>
  const cls = asPlayerClass(playerClass)
  return (
    <span
      className="inline-flex items-center gap-1.5"
      style={classTextStyle(cls)}
    >
      {(spec || playerClass) && (
        <SpecIcon spec={spec} playerClass={playerClass} name={name} />
      )}
      {name}
    </span>
  )
}

function amountLabel(ev: CombatEventRow): string {
  if (ev.eventType === 1 || ev.eventType === 2) {
    return formatNumber(ev.amount)
  }
  if (ev.eventType === 3 && ev.missType != null) {
    return `miss:${ev.missType}`
  }
  if (ev.extra > 0 && (ev.eventType === 9 || ev.eventType === 10)) {
    return ev.extraSpellName || String(ev.extra)
  }
  if (ev.extra > 0 && (ev.eventType === 6 || ev.eventType === 7 || ev.eventType === 8)) {
    return `×${ev.extra}`
  }
  return ev.amount > 0 ? formatNumber(ev.amount) : '—'
}

export function FightEventsPage() {
  const { fightId } = useParams<{ fightId: string }>()
  const [fight, setFight] = useState<FightDetail | null>(null)
  const [fightError, setFightError] = useState<string | null>(null)
  const [typeFilter, setTypeFilter] = useState<CombatEventTypeFilter>('')
  const [page, setPage] = useState(0)
  const [list, setList] = useState<CombatEventList | null>(null)
  const [listError, setListError] = useState<string | null>(null)
  const [listLoading, setListLoading] = useState(false)

  const loadFight = useCallback(async () => {
    if (!fightId) return
    try {
      const data = await getFight(fightId)
      setFight(data)
      setFightError(null)
    } catch (err) {
      setFightError(
        err instanceof ApiError
          ? `Fight nicht geladen (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Unbekannter Fehler',
      )
    }
  }, [fightId])

  useEffect(() => {
    void loadFight()
  }, [loadFight])

  useEffect(() => {
    setPage(0)
  }, [typeFilter])

  useEffect(() => {
    if (!fightId) return
    let cancelled = false
    setListLoading(true)
    setListError(null)
    void (async () => {
      try {
        const data = await getFightEvents(fightId, {
          limit: PAGE_SIZE,
          offset: page * PAGE_SIZE,
          type: typeFilter || undefined,
        })
        if (!cancelled) setList(data)
      } catch (err) {
        if (!cancelled) {
          setListError(
            err instanceof ApiError
              ? `Events nicht geladen (${err.status})`
              : err instanceof Error
                ? err.message
                : 'Unbekannter Fehler',
          )
        }
      } finally {
        if (!cancelled) setListLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [fightId, page, typeFilter])

  if (fightError && !fight) {
    return <ErrorMessage message={fightError} onRetry={loadFight} />
  }
  if (!fight) {
    return <Loading label="Fight laden…" />
  }

  const total = list?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <div className="space-y-6">
      <FightHeader fight={fight} view="events" />

      <div className="flex flex-wrap items-center gap-2">
        <label className="text-xs text-text-muted" htmlFor="event-type-filter">
          Typ
        </label>
        <select
          id="event-type-filter"
          value={typeFilter}
          onChange={(e) =>
            setTypeFilter(e.target.value as CombatEventTypeFilter)
          }
          className="rounded border border-border bg-surface-raised px-2 py-1.5 text-sm"
        >
          {TYPE_FILTERS.map((f) => (
            <option key={f.id || 'all'} value={f.id}>
              {f.label}
            </option>
          ))}
        </select>
        <span className="text-xs text-text-muted">
          {formatNumber(total)} Events
        </span>
      </div>

      {listLoading ? <Loading label="Events laden…" /> : null}
      {listError ? <ErrorMessage message={listError} /> : null}

      {!listLoading && !listError && list ? (
        <>
          <div className="overflow-x-auto rounded border border-border">
            <table className="w-full min-w-[800px] border-collapse text-left text-sm">
              <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
                <tr>
                  <th className="px-3 py-2 font-medium">Zeit</th>
                  <th className="px-3 py-2 font-medium">Typ</th>
                  <th className="px-3 py-2 font-medium">Quelle</th>
                  <th className="px-3 py-2 font-medium">Ziel</th>
                  <th className="px-3 py-2 font-medium">Spell</th>
                  <th className="px-3 py-2 font-medium text-right">Amount</th>
                </tr>
              </thead>
              <tbody>
                {list.events.map((ev) => (
                  <tr
                    key={ev.id}
                    className="border-t border-border-subtle"
                  >
                    <td className="px-3 py-1.5 font-mono text-text-muted">
                      {formatDuration(ev.offsetMs)}
                    </td>
                    <td className="px-3 py-1.5">{eventTypeLabel(ev.eventType)}</td>
                    <td className="px-3 py-1.5">
                      <ActorCell
                        name={ev.sourceName}
                        playerClass={ev.sourceClass}
                        spec={ev.sourceSpec}
                      />
                    </td>
                    <td className="px-3 py-1.5">
                      <ActorCell
                        name={ev.targetName}
                        playerClass={ev.targetClass}
                        spec={ev.targetSpec}
                      />
                    </td>
                    <td className="px-3 py-1.5">
                      {ev.spellName || (ev.spellId > 0 ? String(ev.spellId) : '—')}
                      {ev.spellId > 0 ? (
                        <span className="ml-2 font-mono text-xs text-text-muted">
                          {ev.spellId}
                        </span>
                      ) : null}
                    </td>
                    <td className="px-3 py-1.5 text-right font-mono">
                      {amountLabel(ev)}
                    </td>
                  </tr>
                ))}
                {list.events.length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="px-3 py-8 text-center text-text-muted"
                    >
                      Keine Event-Daten für diesen Fight. Log ggf. erneut
                      hochladen.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>

          <div className="flex flex-wrap items-center justify-between gap-3 text-sm">
            <button
              type="button"
              disabled={page <= 0}
              onClick={() => setPage((p) => Math.max(0, p - 1))}
              className="rounded border border-border px-3 py-1.5 disabled:opacity-40"
            >
              Zurück
            </button>
            <span className="text-text-muted">
              Seite {page + 1} / {totalPages}
            </span>
            <button
              type="button"
              disabled={page + 1 >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="rounded border border-border px-3 py-1.5 disabled:opacity-40"
            >
              Weiter
            </button>
          </div>
        </>
      ) : null}
    </div>
  )
}
