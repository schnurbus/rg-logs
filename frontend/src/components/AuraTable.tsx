import { formatDuration, formatNumber, formatPercent } from '../lib/format'
import type { AuraStat } from '../types/api'

type AuraTableProps = {
  rows: AuraStat[]
  emptyLabel?: string
}

export function AuraTable({
  rows,
  emptyLabel = 'Keine Event-Daten für diesen Fight. Log ggf. erneut hochladen.',
}: AuraTableProps) {
  return (
    <div className="overflow-x-auto rounded border border-border">
      <table className="w-full min-w-[560px] border-collapse text-left text-sm">
        <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
          <tr>
            <th className="px-3 py-2 font-medium min-w-[12rem]">Fähigkeit</th>
            <th className="px-3 py-2 font-medium text-right">Anwendungen</th>
            <th className="px-3 py-2 font-medium text-right">Ziele</th>
            <th className="px-3 py-2 font-medium text-right">Uptime %</th>
            <th className="px-3 py-2 font-medium text-right">Uptime</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.spellId}
              className="border-t border-border-subtle"
            >
              <td className="px-3 py-2">
                <span className="font-medium">{row.spellName}</span>
                <span className="ml-2 font-mono text-xs text-text-muted">
                  {row.spellId}
                </span>
              </td>
              <td className="px-3 py-2 text-right font-mono">
                {formatNumber(row.applications)}
                {row.refreshes > 0 ? (
                  <span className="ml-1 text-xs text-text-muted">
                    (+{formatNumber(row.refreshes)} Ref)
                  </span>
                ) : null}
              </td>
              <td className="px-3 py-2 text-right font-mono text-text-muted">
                {formatNumber(row.targets)}
              </td>
              <td className="px-3 py-2 text-right font-mono">
                {formatPercent(row.uptimePct)}
              </td>
              <td className="px-3 py-2 text-right font-mono text-text-muted">
                {formatDuration(row.uptimeMs)}
              </td>
            </tr>
          ))}
          {rows.length === 0 ? (
            <tr>
              <td
                colSpan={5}
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
