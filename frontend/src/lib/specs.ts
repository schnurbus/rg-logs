import type { PlayerClass } from './classes'

/** WotLK talent-tree slugs as returned by the API. */
export type PlayerSpec =
  | 'deathknight-blood'
  | 'deathknight-frost'
  | 'deathknight-unholy'
  | 'druid-balance'
  | 'druid-feral'
  | 'druid-guardian'
  | 'druid-restoration'
  | 'hunter-beastmastery'
  | 'hunter-marksmanship'
  | 'hunter-survival'
  | 'mage-arcane'
  | 'mage-fire'
  | 'mage-frost'
  | 'paladin-holy'
  | 'paladin-protection'
  | 'paladin-retribution'
  | 'priest-discipline'
  | 'priest-holy'
  | 'priest-shadow'
  | 'rogue-assassination'
  | 'rogue-combat'
  | 'rogue-subtlety'
  | 'shaman-elemental'
  | 'shaman-enhancement'
  | 'shaman-restoration'
  | 'warlock-affliction'
  | 'warlock-demonology'
  | 'warlock-destruction'
  | 'warrior-arms'
  | 'warrior-fury'
  | 'warrior-protection'

export type RaidRole = 'tank' | 'healer' | 'dps'

const SPEC_SET = new Set<string>([
  'deathknight-blood',
  'deathknight-frost',
  'deathknight-unholy',
  'druid-balance',
  'druid-feral',
  'druid-guardian',
  'druid-restoration',
  'hunter-beastmastery',
  'hunter-marksmanship',
  'hunter-survival',
  'mage-arcane',
  'mage-fire',
  'mage-frost',
  'paladin-holy',
  'paladin-protection',
  'paladin-retribution',
  'priest-discipline',
  'priest-holy',
  'priest-shadow',
  'rogue-assassination',
  'rogue-combat',
  'rogue-subtlety',
  'shaman-elemental',
  'shaman-enhancement',
  'shaman-restoration',
  'warlock-affliction',
  'warlock-demonology',
  'warlock-destruction',
  'warrior-arms',
  'warrior-fury',
  'warrior-protection',
])

const TANK_SPECS = new Set<PlayerSpec>([
  'deathknight-blood',
  'druid-guardian',
  'paladin-protection',
  'warrior-protection',
])

const HEALER_SPECS = new Set<PlayerSpec>([
  'druid-restoration',
  'paladin-holy',
  'priest-discipline',
  'priest-holy',
  'shaman-restoration',
])

/** Wowhead medium icon basenames per specialization. */
const SPEC_ICONS: Record<PlayerSpec, string> = {
  'deathknight-blood': 'spell_deathknight_bloodpresence',
  'deathknight-frost': 'spell_deathknight_frostpresence',
  'deathknight-unholy': 'spell_deathknight_unholypresence',
  'druid-balance': 'spell_nature_starfall',
  'druid-feral': 'ability_druid_catform',
  'druid-guardian': 'ability_racial_bearform',
  'druid-restoration': 'spell_nature_healingtouch',
  'hunter-beastmastery': 'ability_hunter_bestialdiscipline',
  'hunter-marksmanship': 'ability_hunter_focusedaim',
  'hunter-survival': 'ability_hunter_camouflage',
  'mage-arcane': 'spell_holy_magicalsentry',
  'mage-fire': 'spell_fire_firebolt02',
  'mage-frost': 'spell_frost_frostbolt02',
  'paladin-holy': 'spell_holy_holybolt',
  'paladin-protection': 'spell_holy_devotionaura',
  'paladin-retribution': 'spell_holy_auraoflight',
  'priest-discipline': 'spell_holy_powerwordshield',
  'priest-holy': 'spell_holy_guardianspirit',
  'priest-shadow': 'spell_shadow_shadowwordpain',
  'rogue-assassination': 'ability_rogue_eviscerate',
  'rogue-combat': 'ability_backstab',
  'rogue-subtlety': 'ability_stealth',
  'shaman-elemental': 'spell_nature_lightning',
  'shaman-enhancement': 'spell_nature_lightningshield',
  'shaman-restoration': 'spell_nature_magicimmunity',
  'warlock-affliction': 'spell_shadow_deathcoil',
  'warlock-demonology': 'spell_shadow_metamorphosis',
  'warlock-destruction': 'spell_shadow_rainoffire',
  'warrior-arms': 'ability_warrior_savageblow',
  'warrior-fury': 'ability_warrior_innerrage',
  'warrior-protection': 'ability_warrior_defensivestance',
}

/** Fallback class icons when no spec is known. */
const CLASS_ICONS: Record<PlayerClass, string> = {
  deathknight: 'class_deathknight',
  druid: 'class_druid',
  hunter: 'class_hunter',
  mage: 'class_mage',
  paladin: 'class_paladin',
  priest: 'class_priest',
  rogue: 'class_rogue',
  shaman: 'class_shaman',
  warlock: 'class_warlock',
  warrior: 'class_warrior',
}

export function asPlayerSpec(v: unknown): PlayerSpec | undefined {
  if (typeof v !== 'string' || !SPEC_SET.has(v)) return undefined
  return v as PlayerSpec
}

export function roleFromSpec(spec?: PlayerSpec | null): RaidRole {
  if (!spec) return 'dps'
  if (TANK_SPECS.has(spec)) return 'tank'
  if (HEALER_SPECS.has(spec)) return 'healer'
  return 'dps'
}

export function specIconName(
  spec?: PlayerSpec | null,
  playerClass?: PlayerClass | null,
): string {
  if (spec) return SPEC_ICONS[spec]
  if (playerClass) return CLASS_ICONS[playerClass]
  return 'inv_misc_questionmark'
}

export function wowheadIconURL(iconName: string): string {
  return `https://wow.zamimg.com/images/wow/icons/medium/${iconName}.jpg`
}
