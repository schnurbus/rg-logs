import { useCallback, useRef, useState, type DragEvent } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { ApiError, uploadFile } from '../api/client'
import { useAuth } from '../auth/AuthProvider'
import { formatBytes } from '../lib/format'

export function UploadPage() {
  const navigate = useNavigate()
  const { user, loading } = useAuth()
  const inputRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [name, setName] = useState('')
  const [isPrivate, setIsPrivate] = useState(false)
  const [includeTrash, setIncludeTrash] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const onFile = useCallback((f: File | null) => {
    setError(null)
    setFile(f)
  }, [])

  const handleDrop = useCallback(
    (e: DragEvent) => {
      e.preventDefault()
      setDragging(false)
      const f = e.dataTransfer.files?.[0]
      if (f) onFile(f)
    },
    [onFile],
  )

  const handleUpload = async () => {
    if (!file || uploading || !user) return
    setUploading(true)
    setError(null)
    setProgress('Hochladen…')
    try {
      const result = await uploadFile(file, {
        isPrivate,
        includeTrash,
        name: name.trim() || undefined,
      })
      setProgress('Fertig — Weiterleitung…')
      navigate(`/uploads/${result.id}`)
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? `Upload fehlgeschlagen (${err.status}): ${err.body || err.message}`
          : err instanceof Error
            ? err.message
            : 'Upload fehlgeschlagen'
      setError(msg)
      setProgress(null)
    } finally {
      setUploading(false)
    }
  }

  if (!loading && !user) {
    return (
      <div className="space-y-4">
        <h1 className="text-xl font-semibold tracking-tight">
          Combat Log hochladen
        </h1>
        <p className="text-sm text-text-muted">
          Zum Hochladen bitte{' '}
          <Link to="/login" className="text-accent">
            anmelden
          </Link>
          .
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">
          Combat Log hochladen
        </h1>
        <p className="mt-1 text-sm text-text-muted">
          WotLK Classic Combat Log (.txt oder .zip) per Drag-and-Drop oder
          Dateiauswahl. ZIP-Archive werden beim Analysieren entpackt.
        </p>
      </div>

      <div
        onDragOver={(e) => {
          e.preventDefault()
          setDragging(true)
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={handleDrop}
        className={[
          'rounded border-2 border-dashed px-6 py-14 text-center transition-colors',
          dragging
            ? 'border-accent bg-surface-overlay'
            : 'border-border bg-surface-raised hover:border-border/80',
        ].join(' ')}
      >
        <p className="text-sm text-text-muted">Datei hierher ziehen oder</p>
        <button
          type="button"
          className="mt-3 rounded bg-accent px-4 py-2 text-sm font-medium text-surface hover:bg-accent-hover"
          onClick={() => inputRef.current?.click()}
        >
          Datei wählen
        </button>
        <input
          ref={inputRef}
          type="file"
          accept=".txt,.zip,text/plain,application/zip,application/x-zip-compressed"
          className="hidden"
          onChange={(e) => onFile(e.target.files?.[0] ?? null)}
        />
        {file ? (
          <p className="mt-4 font-mono text-sm text-text">
            {file.name}{' '}
            <span className="text-text-muted">({formatBytes(file.size)})</span>
          </p>
        ) : (
          <p className="mt-4 text-xs text-text-muted">
            Erwartet: WoWCombatLog.txt oder .zip
          </p>
        )}
      </div>

      <label className="block text-sm">
        <span className="text-text-muted">Name (optional)</span>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          maxLength={200}
          placeholder="Leer = Instanz aus Bossen erkennen"
          className="mt-1 w-full max-w-md rounded border border-border bg-surface-raised px-3 py-2 text-text outline-none focus:border-accent"
        />
      </label>

      <label className="flex items-center gap-2 text-sm text-text-muted">
        <input
          type="checkbox"
          checked={isPrivate}
          onChange={(e) => setIsPrivate(e.target.checked)}
          className="rounded border-border"
        />
        Privat (nur für dich sichtbar)
      </label>

      <label className="flex items-center gap-2 text-sm text-text-muted">
        <input
          type="checkbox"
          checked={includeTrash}
          onChange={(e) => setIncludeTrash(e.target.checked)}
          className="rounded border-border"
        />
        Trash-Mobs mitauswerten
      </label>

      {error ? (
        <p className="rounded border border-danger/40 bg-red-950/30 px-4 py-3 text-sm text-danger">
          {error}
        </p>
      ) : null}

      {progress ? (
        <p className="text-sm text-text-muted" role="status">
          {progress}
        </p>
      ) : null}

      <button
        type="button"
        disabled={!file || uploading || !user}
        onClick={handleUpload}
        className="rounded bg-accent px-5 py-2 text-sm font-medium text-surface disabled:cursor-not-allowed disabled:opacity-40 hover:enabled:bg-accent-hover"
      >
        {uploading ? 'Wird hochgeladen…' : 'Hochladen'}
      </button>
    </div>
  )
}
