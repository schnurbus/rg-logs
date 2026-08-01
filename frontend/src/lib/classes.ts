/** WoW player class slugs as returned by the API. */
export type PlayerClass =
  | 'deathknight'
  | 'druid'
  | 'hunter'
  | 'mage'
  | 'paladin'
  | 'priest'
  | 'rogue'
  | 'shaman'
  | 'warlock'
  | 'warrior'

const CLASS_SET = new Set<string>([
  'deathknight',
  'druid',
  'hunter',
  'mage',
  'paladin',
  'priest',
  'rogue',
  'shaman',
  'warlock',
  'warrior',
])

export function asPlayerClass(v: unknown): PlayerClass | undefined {
  if (typeof v !== 'string' || !CLASS_SET.has(v)) return undefined
  return v as PlayerClass
}

/** Official-ish WoW class colors (Retail/Classic shared palette). */
export const CLASS_COLORS: Record<PlayerClass, string> = {
  deathknight: 'var(--color-class-deathknight)',
  druid: 'var(--color-class-druid)',
  hunter: 'var(--color-class-hunter)',
  mage: 'var(--color-class-mage)',
  paladin: 'var(--color-class-paladin)',
  priest: 'var(--color-class-priest)',
  rogue: 'var(--color-class-rogue)',
  shaman: 'var(--color-class-shaman)',
  warlock: 'var(--color-class-warlock)',
  warrior: 'var(--color-class-warrior)',
}

export function classColor(cls?: PlayerClass | null): string {
  if (!cls) return 'var(--color-accent)'
  return CLASS_COLORS[cls]
}

/** Tailwind-safe inline style helpers for class-tinted text/bars. */
export function classTextStyle(cls?: PlayerClass | null): { color: string } {
  return { color: classColor(cls) }
}

export function classBarStyle(
  cls: PlayerClass | null | undefined,
  ratio: number,
): { width: string; backgroundColor: string } {
  const pct = Math.max(0, Math.min(1, ratio)) * 100
  return {
    width: `${pct}%`,
    backgroundColor: classColor(cls),
  }
}
