import type { Session, User } from '@supabase/supabase-js'
import {
  createContext,
  useContext,
  useEffect,
  useEffectEvent,
  useState,
  type ReactNode,
} from 'react'
import { setApiAccessToken } from '../api/token'
import { getSupabase } from './supabaseClient'

type AuthState = {
  session: Session | null
  user: User | null
  loading: boolean
  accessToken: string | null
  signInWithMagicLink: (email: string) => Promise<void>
  signInWithOAuth: (provider: 'google' | 'discord') => Promise<void>
  signOut: () => Promise<void>
}

const AuthContext = createContext<AuthState | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null)
  const [loading, setLoading] = useState(true)
  const [ready, setReady] = useState(false)
  const [initError, setInitError] = useState<string | null>(null)

  const onAuth = useEffectEvent((next: Session | null) => {
    setSession(next)
    setApiAccessToken(next?.access_token ?? null)
    setLoading(false)
  })

  useEffect(() => {
    let active = true
    let unsubscribe: (() => void) | undefined

    void (async () => {
      try {
        const supabase = await getSupabase()
        if (!active) return
        setReady(true)
        const { data } = await supabase.auth.getSession()
        if (active) onAuth(data.session)
        const { data: sub } = supabase.auth.onAuthStateChange((_event, next) => {
          onAuth(next)
        })
        unsubscribe = () => sub.subscription.unsubscribe()
      } catch (err) {
        if (!active) return
        setInitError(err instanceof Error ? err.message : String(err))
        setLoading(false)
      }
    })()

    return () => {
      active = false
      unsubscribe?.()
    }
  }, [])

  if (initError) {
    return (
      <div className="mx-auto max-w-lg p-6 text-sm text-danger">
        Auth-Konfiguration fehlgeschlagen: {initError}
      </div>
    )
  }

  if (!ready) {
    return <p className="p-6 text-sm text-text-muted">Lade Konfiguration…</p>
  }

  const value: AuthState = {
    session,
    user: session?.user ?? null,
    loading,
    accessToken: session?.access_token ?? null,
    async signInWithMagicLink(email: string) {
      const supabase = await getSupabase()
      const { error } = await supabase.auth.signInWithOtp({
        email,
        options: {
          emailRedirectTo: `${window.location.origin}/auth/callback`,
        },
      })
      if (error) throw error
    },
    async signInWithOAuth(provider) {
      const supabase = await getSupabase()
      const { error } = await supabase.auth.signInWithOAuth({
        provider,
        options: {
          redirectTo: `${window.location.origin}/auth/callback`,
        },
      })
      if (error) throw error
    },
    async signOut() {
      const supabase = await getSupabase()
      const { error } = await supabase.auth.signOut()
      if (error) throw error
    },
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
