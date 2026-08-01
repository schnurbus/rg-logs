import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { ApiError, getUpload } from '../api/client'
import { ErrorMessage, Loading } from '../components/ErrorMessage'
import { StatusBadge } from '../components/StatusBadge'
import {
  formatBytes,
  formatDateTime,
  formatDuration,
} from '../lib/format'
import type { Upload } from '../types/api'

const POLL_MS = 2000

export function UploadDetailPage() {
  const { uploadId } = useParams<{ uploadId: string }>()
  const [upload, setUpload] = useState<Upload | null>(null)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!uploadId) return
    try {
      const data = await getUpload(uploadId)
      setUpload(data)
      setError(null)
    } catch (err) {
      setError(
        err instanceof ApiError
          ? `Upload nicht gefunden oder API-Fehler (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Unbekannter Fehler',
      )
    }
  }, [uploadId])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    if (!upload) return
    if (upload.status !== 'pending' && upload.status !== 'processing') return
    const id = window.setInterval(() => {
      void load()
    }, POLL_MS)
    return () => window.clearInterval(id)
  }, [upload, load])

  if (error && !upload) {
    return <ErrorMessage message={error} onRetry={load} />
  }

  if (!upload) {
    return <Loading label="Upload laden…" />
  }

  const fights = upload.fights ?? []

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs text-text-muted">
          <Link to="/uploads">Uploads</Link>
          <span className="mx-1.5">/</span>
          <span className="text-text">{upload.filename}</span>
        </p>
        <div className="mt-2 flex flex-wrap items-center gap-3">
          <h1 className="text-xl font-semibold tracking-tight">
            {upload.filename}
          </h1>
          <StatusBadge status={upload.status} />
        </div>
        <dl className="mt-3 grid gap-2 text-sm text-text-muted sm:grid-cols-3">
          <div>
            <dt className="text-xs uppercase tracking-wide">Größe</dt>
            <dd className="font-mono text-text">{formatBytes(upload.sizeBytes)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-wide">Erstellt</dt>
            <dd>{formatDateTime(upload.createdAt)}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-wide">Verarbeitet</dt>
            <dd>{formatDateTime(upload.processedAt)}</dd>
          </div>
        </dl>
        {upload.error ? (
          <p className="mt-3 rounded border border-danger/40 bg-red-950/30 px-3 py-2 text-sm text-danger">
            {upload.error}
          </p>
        ) : null}
        {(upload.status === 'pending' || upload.status === 'processing') && (
          <p className="mt-3 text-sm text-warning">
            Parsing läuft — Status wird automatisch aktualisiert…
          </p>
        )}
      </div>

      <div>
        <h2 className="mb-3 text-sm font-semibold uppercase tracking-wide text-text-muted">
          Fights ({fights.length})
        </h2>

        {fights.length === 0 ? (
          <p className="text-sm text-text-muted">
            {upload.status === 'ready'
              ? 'Keine Fights erkannt.'
              : 'Noch keine Fights verfügbar.'}
          </p>
        ) : (
          <div className="overflow-x-auto rounded border border-border">
            <table className="w-full min-w-[560px] border-collapse text-left text-sm">
              <thead className="bg-surface-raised text-xs uppercase tracking-wide text-text-muted">
                <tr>
                  <th className="px-3 py-2 font-medium">Titel</th>
                  <th className="px-3 py-2 font-medium">Dauer</th>
                  <th className="px-3 py-2 font-medium">Teilnehmer</th>
                  <th className="px-3 py-2 font-medium">Kill</th>
                </tr>
              </thead>
              <tbody>
                {fights.map((f) => (
                  <tr
                    key={f.id}
                    className="border-t border-border-subtle hover:bg-surface-overlay/50"
                  >
                    <td className="px-3 py-2">
                      <Link
                        to={`/fights/${f.id}`}
                        className="font-medium text-accent"
                      >
                        {f.title || 'Trash'}
                      </Link>
                    </td>
                    <td className="px-3 py-2 font-mono text-text-muted">
                      {formatDuration(f.durationMs)}
                    </td>
                    <td className="px-3 py-2 text-text-muted">
                      {f.participantCount}
                    </td>
                    <td className="px-3 py-2 text-text-muted">
                      {f.kill ? 'Ja' : 'Nein'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}
