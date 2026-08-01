import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, listUploads } from '../api/client'
import { ErrorMessage, Loading } from '../components/ErrorMessage'
import { StatusBadge } from '../components/StatusBadge'
import { formatBytes, formatDateTime } from '../lib/format'
import type { Upload } from '../types/api'

export function UploadsListPage() {
  const [uploads, setUploads] = useState<Upload[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setError(null)
    try {
      const data = await listUploads()
      setUploads(data)
    } catch (err) {
      setUploads(null)
      setError(
        err instanceof ApiError
          ? `Konnte Uploads nicht laden (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Unbekannter Fehler',
      )
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <div className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Uploads</h1>
          <p className="mt-1 text-sm text-text-muted">
            Status und Fights der hochgeladenen Logs.
          </p>
        </div>
        <Link
          to="/"
          className="rounded bg-surface-overlay px-3 py-1.5 text-sm text-text no-underline ring-1 ring-border hover:bg-surface-raised"
        >
          Neuer Upload
        </Link>
      </div>

      {error ? <ErrorMessage message={error} onRetry={load} /> : null}
      {!error && uploads === null ? <Loading /> : null}

      {uploads && uploads.length === 0 ? (
        <p className="text-sm text-text-muted">Noch keine Uploads.</p>
      ) : null}

      {uploads && uploads.length > 0 ? (
        <div className="overflow-x-auto rounded border border-border">
          <table className="w-full min-w-[640px] border-collapse text-left text-sm">
            <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
              <tr>
                <th className="px-3 py-2 font-medium">Datei</th>
                <th className="px-3 py-2 font-medium">Status</th>
                <th className="px-3 py-2 font-medium">Größe</th>
                <th className="px-3 py-2 font-medium">Erstellt</th>
                <th className="px-3 py-2 font-medium">Fights</th>
              </tr>
            </thead>
            <tbody>
              {uploads.map((u) => (
                <tr
                  key={u.id}
                  className="border-t border-border-subtle hover:bg-surface-overlay/50"
                >
                  <td className="px-3 py-2">
                    <Link
                      to={`/uploads/${u.id}`}
                      className="font-medium text-accent"
                    >
                      {u.filename || u.id}
                    </Link>
                    {u.error ? (
                      <p className="mt-0.5 text-xs text-danger">{u.error}</p>
                    ) : null}
                  </td>
                  <td className="px-3 py-2">
                    <StatusBadge status={u.status} />
                  </td>
                  <td className="px-3 py-2 font-mono text-text-muted">
                    {formatBytes(u.sizeBytes)}
                  </td>
                  <td className="px-3 py-2 text-text-muted">
                    {formatDateTime(u.createdAt)}
                  </td>
                  <td className="px-3 py-2 text-text-muted">
                    {u.fights?.length ?? (u.status === 'ready' ? '→' : '—')}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  )
}
