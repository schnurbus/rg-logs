import { Link, NavLink } from 'react-router-dom'
import { formatDuration } from '../lib/format'
import type { FightDetail } from '../types/api'

export type FightSide = 'players' | 'enemies'

type FightHeaderProps = {
  fight: FightDetail
  view: 'analyze' | 'events'
  side?: FightSide
  onSideChange?: (side: FightSide) => void
}

export function FightHeader({
  fight,
  view,
  side = 'players',
  onSideChange,
}: FightHeaderProps) {
  return (
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
      <div className="mt-2 flex flex-wrap items-end justify-between gap-3">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">
            {fight.title || 'Fight'}
          </h1>
          <p className="mt-1 text-sm text-text-muted">
            Dauer {formatDuration(fight.durationMs)}
            {fight.kill ? ' · Kill' : ''}
            {' · '}
            {fight.participantCount} Teilnehmer
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {view === 'analyze' && onSideChange ? (
            <div
              role="tablist"
              aria-label="Seite"
              className="flex gap-1 rounded border border-border p-0.5"
            >
              <button
                type="button"
                role="tab"
                aria-selected={side === 'players'}
                onClick={() => onSideChange('players')}
                className={[
                  'rounded px-3 py-1.5 text-sm transition-colors',
                  side === 'players'
                    ? 'bg-surface-overlay text-text'
                    : 'text-text-muted hover:text-text',
                ].join(' ')}
              >
                Spieler
              </button>
              <button
                type="button"
                role="tab"
                aria-selected={side === 'enemies'}
                onClick={() => onSideChange('enemies')}
                className={[
                  'rounded px-3 py-1.5 text-sm transition-colors',
                  side === 'enemies'
                    ? 'bg-surface-overlay text-text'
                    : 'text-text-muted hover:text-text',
                ].join(' ')}
              >
                Gegner
              </button>
            </div>
          ) : null}
          <div
            role="tablist"
            aria-label="Ansicht"
            className="flex gap-1 rounded border border-border p-0.5"
          >
            <NavLink
              to={`/fights/${fight.id}`}
              end
              role="tab"
              aria-selected={view === 'analyze'}
              className={[
                'rounded px-3 py-1.5 text-sm transition-colors',
                view === 'analyze'
                  ? 'bg-surface-overlay text-text'
                  : 'text-text-muted hover:text-text',
              ].join(' ')}
            >
              Auswertung
            </NavLink>
            <NavLink
              to={`/fights/${fight.id}/events`}
              role="tab"
              aria-selected={view === 'events'}
              className={[
                'rounded px-3 py-1.5 text-sm transition-colors',
                view === 'events'
                  ? 'bg-surface-overlay text-text'
                  : 'text-text-muted hover:text-text',
              ].join(' ')}
            >
              Events
            </NavLink>
          </div>
        </div>
      </div>
    </div>
  )
}
