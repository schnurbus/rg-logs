import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ApiError, deleteUpload, listUploads } from '../api/client'
import { useAuth } from '../auth/AuthProvider'
import { ErrorMessage, Loading } from '../components/ErrorMessage'
import { StatusBadge } from '../components/StatusBadge'
import { formatBytes, formatDateTime } from '../lib/format'
import { uploadDisplayName } from '../lib/upload'
import type { Upload } from '../types/api'

export function UploadsListPage() {
  const { user } = useAuth()
  const [uploads, setUploads] = useState<Upload[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [mineOnly, setMineOnly] = useState(false)
  const [deletingId, setDeletingId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setError(null)
    try {
      const data = await listUploads({ mine: mineOnly })
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
  }, [mineOnly])

  useEffect(() => {
    void load()
  }, [load])

  const onDelete = async (id: string) => {
    if (!window.confirm('Upload wirklich löschen?')) return
    setDeletingId(id)
    try {
      await deleteUpload(id)
      setUploads((prev) => prev?.filter((u) => u.id !== id) ?? null)
    } catch (err) {
      setError(
        err instanceof ApiError
          ? `Löschen fehlgeschlagen (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Löschen fehlgeschlagen',
      )
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Uploads</h1>
          <p className="mt-1 text-sm text-text-muted">
            Status und Fights der hochgeladenen Logs.
          </p>
        </div>
        <div className="flex items-center gap-3">
          {user ? (
            <label className="flex items-center gap-2 text-sm text-text-muted">
              <input
                type="checkbox"
                checked={mineOnly}
                onChange={(e) => setMineOnly(e.target.checked)}
              />
              Nur meine
            </label>
          ) : null}
          <Link
            to="/"
            className="rounded bg-surface-overlay px-3 py-1.5 text-sm text-text no-underline ring-1 ring-border hover:bg-surface-raised"
          >
            Neuer Upload
          </Link>
        </div>
      </div>

      {error ? <ErrorMessage message={error} onRetry={load} /> : null}
      {!error && uploads === null ? <Loading /> : null}

      {uploads && uploads.length === 0 ? (
        <p className="text-sm text-text-muted">Noch keine Uploads.</p>
      ) : null}

      {uploads && uploads.length > 0 ? (
        <div className="overflow-x-auto rounded border border-border">
          <table className="w-full min-w-[720px] border-collapse text-left text-sm">
            <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
              <tr>
                <th className="px-3 py-2 font-medium">Name</th>
                <th className="px-3 py-2 font-medium">Status</th>
                <th className="px-3 py-2 font-medium">Sichtbarkeit</th>
                <th className="px-3 py-2 font-medium">Größe</th>
                <th className="px-3 py-2 font-medium">Erstellt</th>
                <th className="px-3 py-2 font-medium">Fights</th>
                <th className="px-3 py-2 font-medium" />
              </tr>
            </thead>
            <tbody>
              {uploads.map((u) => {
                const owned = Boolean(user && u.userId === user.id)
                return (
                  <tr
                    key={u.id}
                    className="border-t border-border-subtle hover:bg-surface-overlay/50"
                  >
                    <td className="px-3 py-2">
                      <Link
                        to={`/uploads/${u.id}`}
                        className="font-medium text-accent"
                      >
                        {uploadDisplayName(u)}
                      </Link>
                      {u.name?.trim() && u.filename ? (
                        <p className="mt-0.5 font-mono text-xs text-text-muted">
                          {u.filename}
                        </p>
                      ) : null}
                      {u.error ? (
                        <p className="mt-0.5 text-xs text-danger">{u.error}</p>
                      ) : null}
                    </td>
                    <td className="px-3 py-2">
                      <StatusBadge status={u.status} />
                    </td>
                    <td className="px-3 py-2 text-text-muted">
                      {u.isPrivate ? 'Privat' : 'Öffentlich'}
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
                    <td className="px-3 py-2 text-right">
                      {owned ? (
                        <button
                          type="button"
                          disabled={deletingId === u.id}
                          onClick={() => void onDelete(u.id)}
                          className="text-xs text-danger hover:underline disabled:opacity-40"
                        >
                          {deletingId === u.id ? '…' : 'Löschen'}
                        </button>
                      ) : null}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  )
}
