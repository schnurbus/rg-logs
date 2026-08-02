import { asPlayerClass, classTextStyle } from '../lib/classes'
import { formatNumber } from '../lib/format'
import { risingGodsProfileURL } from '../lib/risingGods'
import type { CastCountStat } from '../types/api'
import { SpecIcon } from './SpecIcon'

type CastCountTableProps = {
  rows: CastCountStat[]
  extraColumnLabel: string
  emptyLabel?: string
}

export function CastCountTable({
  rows,
  extraColumnLabel,
  emptyLabel = 'Keine Event-Daten für diesen Fight. Log ggf. erneut hochladen.',
}: CastCountTableProps) {
  return (
    <div className="overflow-x-auto rounded border border-border">
      <table className="w-full min-w-[640px] border-collapse text-left text-sm">
        <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
          <tr>
            <th className="px-3 py-2 font-medium min-w-[10rem]">Spieler</th>
            <th className="px-3 py-2 font-medium min-w-[10rem]">Fähigkeit</th>
            <th className="px-3 py-2 font-medium min-w-[10rem]">
              {extraColumnLabel}
            </th>
            <th className="px-3 py-2 font-medium text-right">Anzahl</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => {
            const cls = asPlayerClass(row.class)
            const profileURL = risingGodsProfileURL(row.actorName)
            return (
              <tr
                key={`${row.actorId}-${row.spellId}-${row.extraSpellId}-${i}`}
                className="border-t border-border-subtle"
              >
                <td className="px-3 py-2">
                  <a
                    href={profileURL}
                    target="_blank"
                    rel="noopener noreferrer"
                    title="Rising Gods Profil"
                    className="inline-flex items-center gap-1.5 font-medium hover:underline"
                    style={classTextStyle(cls)}
                  >
                    <SpecIcon
                      spec={row.spec}
                      playerClass={row.class}
                      name={row.actorName}
                    />
                    {row.actorName}
                  </a>
                </td>
                <td className="px-3 py-2">
                  <span>{row.spellName}</span>
                  <span className="ml-2 font-mono text-xs text-text-muted">
                    {row.spellId}
                  </span>
                </td>
                <td className="px-3 py-2">
                  {row.extraSpellName ? (
                    <>
                      <span>{row.extraSpellName}</span>
                      <span className="ml-2 font-mono text-xs text-text-muted">
                        {row.extraSpellId}
                      </span>
                    </>
                  ) : (
                    <span className="text-text-muted">—</span>
                  )}
                </td>
                <td className="px-3 py-2 text-right font-mono">
                  {formatNumber(row.count)}
                </td>
              </tr>
            )
          })}
          {rows.length === 0 ? (
            <tr>
              <td
                colSpan={4}
                className="px-3 py-8 text-center text-text-muted"
              >
                {emptyLabel}
              </td>
            </tr>
          ) : null}
        </tbody>
      </table>
    </div>
  )
}
