import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { getSupabase } from '../auth/supabaseClient'

export function AuthCallbackPage() {
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    void (async () => {
      try {
        const supabase = await getSupabase()
        const { error: exchangeError } =
          await supabase.auth.exchangeCodeForSession(window.location.href)
        if (!active) return
        if (exchangeError) {
          // Hash-based magic links may already be handled by detectSessionInUrl.
          const { data } = await supabase.auth.getSession()
          if (data.session) {
            navigate('/', { replace: true })
            return
          }
          setError(exchangeError.message)
          return
        }
        navigate('/', { replace: true })
      } catch (err) {
        if (!active) return
        setError(err instanceof Error ? err.message : String(err))
      }
    })()
    return () => {
      active = false
    }
  }, [navigate])

  if (error) {
    return (
      <div className="space-y-3">
        <h1 className="text-xl font-semibold">Login fehlgeschlagen</h1>
        <p className="text-sm text-danger">{error}</p>
      </div>
    )
  }

  return <p className="text-sm text-text-muted">Anmeldung wird abgeschlossen…</p>
}
