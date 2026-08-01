import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import {
  ApiError,
  deleteUpload,
  getUpload,
  renameUpload,
} from '../api/client'
import { useAuth } from '../auth/AuthProvider'
import { ErrorMessage, Loading } from '../components/ErrorMessage'
import { StatusBadge } from '../components/StatusBadge'
import {
  formatBytes,
  formatDateTime,
  formatDuration,
} from '../lib/format'
import { uploadDisplayName } from '../lib/upload'
import type { Upload } from '../types/api'

const POLL_MS = 2000

export function UploadDetailPage() {
  const { uploadId } = useParams<{ uploadId: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [upload, setUpload] = useState<Upload | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [editing, setEditing] = useState(false)
  const [draftName, setDraftName] = useState('')
  const [savingName, setSavingName] = useState(false)

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

  const onDelete = async () => {
    if (!upload || !window.confirm('Upload wirklich löschen?')) return
    setDeleting(true)
    try {
      await deleteUpload(upload.id)
      navigate('/uploads', { replace: true })
    } catch (err) {
      setError(
        err instanceof ApiError
          ? `Löschen fehlgeschlagen (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Löschen fehlgeschlagen',
      )
      setDeleting(false)
    }
  }

  const startEdit = () => {
    if (!upload) return
    setDraftName(uploadDisplayName(upload))
    setEditing(true)
  }

  const saveName = async () => {
    if (!upload) return
    const next = draftName.trim()
    if (!next) {
      setError('Name darf nicht leer sein')
      return
    }
    setSavingName(true)
    try {
      const updated = await renameUpload(upload.id, next)
      setUpload(updated)
      setEditing(false)
      setError(null)
    } catch (err) {
      setError(
        err instanceof ApiError
          ? `Umbenennen fehlgeschlagen (${err.status})`
          : err instanceof Error
            ? err.message
            : 'Umbenennen fehlgeschlagen',
      )
    } finally {
      setSavingName(false)
    }
  }

  if (error && !upload) {
    return <ErrorMessage message={error} onRetry={load} />
  }

  if (!upload) {
    return <Loading label="Upload laden…" />
  }

  const fights = upload.fights ?? []
  const owned = Boolean(user && upload.userId === user.id)
  const display = uploadDisplayName(upload)

  return (
    <div className="space-y-6">
      <div>
        <p className="text-xs text-text-muted">
          <Link to="/uploads">Uploads</Link>
          <span className="mx-1.5">/</span>
          <span className="text-text">{display}</span>
        </p>
        <div className="mt-2 flex flex-wrap items-center gap-3">
          {editing ? (
            <div className="flex flex-wrap items-center gap-2">
              <input
                type="text"
                value={draftName}
                onChange={(e) => setDraftName(e.target.value)}
                maxLength={200}
                className="rounded border border-border bg-surface-raised px-3 py-1.5 text-lg font-semibold text-text outline-none focus:border-accent"
                autoFocus
              />
              <button
                type="button"
                disabled={savingName}
                onClick={() => void saveName()}
                className="rounded bg-accent px-3 py-1.5 text-sm text-surface disabled:opacity-40"
              >
                {savingName ? '…' : 'Speichern'}
              </button>
              <button
                type="button"
                disabled={savingName}
                onClick={() => setEditing(false)}
                className="text-sm text-text-muted hover:text-text"
              >
                Abbrechen
              </button>
            </div>
          ) : (
            <h1 className="text-xl font-semibold tracking-tight">{display}</h1>
          )}
          <StatusBadge status={upload.status} />
          <span className="rounded bg-surface-overlay px-2 py-0.5 text-xs text-text-muted">
            {upload.isPrivate ? 'Privat' : 'Öffentlich'}
          </span>
          {owned && !editing ? (
            <button
              type="button"
              onClick={startEdit}
              className="text-sm text-text-muted hover:text-text hover:underline"
            >
              Umbenennen
            </button>
          ) : null}
          {owned ? (
            <button
              type="button"
              disabled={deleting}
              onClick={() => void onDelete()}
              className="ml-auto text-sm text-danger hover:underline disabled:opacity-40"
            >
              {deleting ? 'Löschen…' : 'Löschen'}
            </button>
          ) : null}
        </div>
        {upload.filename && upload.filename !== display ? (
          <p className="mt-1 font-mono text-xs text-text-muted">{upload.filename}</p>
        ) : null}
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
        {error ? (
          <p className="mt-3 rounded border border-danger/40 bg-red-950/30 px-3 py-2 text-sm text-danger">
            {error}
          </p>
        ) : null}
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
