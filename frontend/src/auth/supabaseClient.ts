import { createClient, type SupabaseClient } from '@supabase/supabase-js'
import { loadRuntimeConfig } from '../lib/runtimeConfig'

let client: SupabaseClient | null = null
let initPromise: Promise<SupabaseClient> | null = null

export function getSupabase(): Promise<SupabaseClient> {
  if (client) return Promise.resolve(client)
  if (!initPromise) {
    initPromise = loadRuntimeConfig().then((cfg) => {
      client = createClient(cfg.supabaseUrl, cfg.supabaseAnonKey)
      return client
    })
  }
  return initPromise
}
