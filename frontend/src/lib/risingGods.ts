/** Rising Gods AoWow character profiler URL. */
export function risingGodsProfileURL(characterName: string): string {
  const base = characterName.split('-')[0]?.trim() || characterName
  return `https://db.rising-gods.de/?profile=eu.rising-gods.${base.toLowerCase()}`
}
