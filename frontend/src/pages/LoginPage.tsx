import { useState, type FormEvent } from 'react'
import { Link } from 'react-router-dom'
import { useAuth } from '../auth/AuthProvider'

export function LoginPage() {
  const { signInWithMagicLink, signInWithOAuth, user, loading } = useAuth()
  const [email, setEmail] = useState('')
  const [sent, setSent] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!loading && user) {
    return (
      <div className="space-y-4">
        <h1 className="text-xl font-semibold tracking-tight">Angemeldet</h1>
        <p className="text-sm text-text-muted">
          Du bist als {user.email ?? user.id} angemeldet.
        </p>
        <Link to="/" className="text-sm text-accent">
          Zum Upload
        </Link>
      </div>
    )
  }

  const onMagic = async (e: FormEvent) => {
    e.preventDefault()
    if (!email.trim() || busy) return
    setBusy(true)
    setError(null)
    try {
      await signInWithMagicLink(email.trim())
      setSent(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login fehlgeschlagen')
    } finally {
      setBusy(false)
    }
  }

  const onOAuth = async (provider: 'google' | 'discord') => {
    setBusy(true)
    setError(null)
    try {
      await signInWithOAuth(provider)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'OAuth fehlgeschlagen')
      setBusy(false)
    }
  }

  return (
    <div className="mx-auto max-w-md space-y-6">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Anmelden</h1>
        <p className="mt-1 text-sm text-text-muted">
          Magic Link oder Social Login über Supabase.
        </p>
      </div>

      {sent ? (
        <p className="rounded border border-success/40 bg-emerald-950/20 px-4 py-3 text-sm text-success">
          Magic Link gesendet. Prüfe Mailpit (
          <a href="http://127.0.0.1:54324" target="_blank" rel="noreferrer">
            Supabase
          </a>{' '}
          oder{' '}
          <a href="http://localhost:8025" target="_blank" rel="noreferrer">
            Compose
          </a>
          ).
        </p>
      ) : (
        <form onSubmit={onMagic} className="space-y-3">
          <label className="block text-sm">
            <span className="text-text-muted">E-Mail</span>
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className="mt-1 w-full rounded border border-border bg-surface-raised px-3 py-2 text-text outline-none focus:border-accent"
              placeholder="du@example.com"
            />
          </label>
          <button
            type="submit"
            disabled={busy}
            className="w-full rounded bg-accent px-4 py-2 text-sm font-medium text-surface disabled:opacity-40 hover:enabled:bg-accent-hover"
          >
            {busy ? 'Senden…' : 'Magic Link senden'}
          </button>
        </form>
      )}

      <div className="relative text-center text-xs text-text-muted">
        <span className="bg-surface px-2">oder</span>
        <div className="absolute inset-x-0 top-1/2 -z-10 border-t border-border" />
      </div>

      <div className="grid gap-2 sm:grid-cols-2">
        <button
          type="button"
          disabled={busy}
          onClick={() => void onOAuth('google')}
          className="rounded border border-border bg-surface-raised px-4 py-2 text-sm hover:bg-surface-overlay disabled:opacity-40"
        >
          Google
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => void onOAuth('discord')}
          className="rounded border border-border bg-surface-raised px-4 py-2 text-sm hover:bg-surface-overlay disabled:opacity-40"
        >
          Discord
        </button>
      </div>

      {error ? (
        <p className="rounded border border-danger/40 bg-red-950/30 px-4 py-3 text-sm text-danger">
          {error}
        </p>
      ) : null}
    </div>
  )
}
