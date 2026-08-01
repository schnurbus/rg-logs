import { createClient, type Session, type User } from '@supabase/supabase-js'
import {
  createContext,
  useContext,
  useEffect,
  useEffectEvent,
  useState,
  type ReactNode,
} from 'react'
import { setApiAccessToken } from '../api/token'

const supabaseUrl = import.meta.env.VITE_SUPABASE_URL as string | undefined
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY as
  | string
  | undefined

if (!supabaseUrl || !supabaseAnonKey) {
  console.warn(
    'VITE_SUPABASE_URL / VITE_SUPABASE_ANON_KEY missing — auth will not work',
  )
}

export const supabase = createClient(
  supabaseUrl || 'http://127.0.0.1:54321',
  supabaseAnonKey || 'public-anon-key',
)

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

  const onAuth = useEffectEvent((next: Session | null) => {
    setSession(next)
    setApiAccessToken(next?.access_token ?? null)
    setLoading(false)
  })

  useEffect(() => {
    let active = true
    void supabase.auth.getSession().then(({ data }) => {
      if (active) onAuth(data.session)
    })
    const { data: sub } = supabase.auth.onAuthStateChange((_event, next) => {
      onAuth(next)
    })
    return () => {
      active = false
      sub.subscription.unsubscribe()
    }
  }, [])

  const value: AuthState = {
    session,
    user: session?.user ?? null,
    loading,
    accessToken: session?.access_token ?? null,
    async signInWithMagicLink(email: string) {
      const { error } = await supabase.auth.signInWithOtp({
        email,
        options: {
          emailRedirectTo: `${window.location.origin}/auth/callback`,
        },
      })
      if (error) throw error
    },
    async signInWithOAuth(provider) {
      const { error } = await supabase.auth.signInWithOAuth({
        provider,
        options: {
          redirectTo: `${window.location.origin}/auth/callback`,
        },
      })
      if (error) throw error
    },
    async signOut() {
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
