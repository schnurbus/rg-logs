import { Link, NavLink } from 'react-router-dom'
import { formatDuration } from '../lib/format'
import type { FightDetail } from '../types/api'

type FightHeaderProps = {
  fight: FightDetail
  view: 'analyze' | 'events'
}

export function FightHeader({ fight, view }: FightHeaderProps) {
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
  )
}
