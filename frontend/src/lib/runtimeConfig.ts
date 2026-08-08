export type RuntimeConfig = {
  supabaseUrl: string
  supabaseAnonKey: string
}

function fromViteEnv(): RuntimeConfig | null {
  const supabaseUrl = import.meta.env.VITE_SUPABASE_URL as string | undefined
  const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY as
    | string
    | undefined
  if (!supabaseUrl || !supabaseAnonKey) return null
  return { supabaseUrl, supabaseAnonKey }
}

/**
 * Loads public config from the API (container runtime env) with optional
 * Vite env fallback for local frontend-only workflows.
 */
export async function loadRuntimeConfig(): Promise<RuntimeConfig> {
  try {
    const res = await fetch('/api/config')
    if (res.ok) {
      const data = (await res.json()) as {
        supabaseUrl?: string
        supabaseAnonKey?: string
      }
      if (data.supabaseUrl && data.supabaseAnonKey) {
        return {
          supabaseUrl: data.supabaseUrl,
          supabaseAnonKey: data.supabaseAnonKey,
        }
      }
    }
  } catch {
    // fall through to Vite env
  }

  const fromEnv = fromViteEnv()
  if (fromEnv) return fromEnv

  throw new Error(
    'Supabase config missing: set SUPABASE_URL/SUPABASE_ANON_KEY on the API or VITE_SUPABASE_* for Vite',
  )
}
