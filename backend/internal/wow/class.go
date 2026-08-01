package wow

// Class is a WoW character class slug used in the API and UI.
type Class string

const (
	ClassDeathKnight Class = "deathknight"
	ClassDruid       Class = "druid"
	ClassHunter      Class = "hunter"
	ClassMage        Class = "mage"
	ClassPaladin     Class = "paladin"
	ClassPriest      Class = "priest"
	ClassRogue       Class = "rogue"
	ClassShaman      Class = "shaman"
	ClassWarlock     Class = "warlock"
	ClassWarrior     Class = "warrior"
)

// DetectClass scores signature spell IDs (weighted by total) and returns the best match.
// Returns empty string when no signature spells are found.
func DetectClass(spellTotals map[int]int64) Class {
	if len(spellTotals) == 0 {
		return ""
	}

	scores := make(map[Class]int64, 10)
	for spellID, total := range spellTotals {
		cls, ok := signatureSpells[spellID]
		if !ok || total <= 0 {
			continue
		}
		scores[cls] += total
	}

	var best Class
	var bestScore int64
	for cls, score := range scores {
		if score > bestScore {
			best = cls
			bestScore = score
		}
	}
	return best
}

// signatureSpells maps WotLK (and overlapping) spell IDs to classes.
// Prefer rank-specific IDs from combat logs; shared utility spells are omitted.
var signatureSpells = map[int]Class{
	// --- Death Knight ---
	49909: ClassDeathKnight, // Icy Touch
	49921: ClassDeathKnight, // Plague Strike
	49924: ClassDeathKnight, // Death Strike
	49930: ClassDeathKnight, // Blood Strike
	45470: ClassDeathKnight, // Death Strike (heal)
	47632: ClassDeathKnight, // Death Coil
	48982: ClassDeathKnight, // Rune Tap
	50401: ClassDeathKnight, // Razorice
	50463: ClassDeathKnight, // Blood-Caked Strike
	50475: ClassDeathKnight, // Blood Presence (proc)
	50526: ClassDeathKnight, // Wandering Plague
	50536: ClassDeathKnight, // Unholy Blight
	51411: ClassDeathKnight, // Howling Blast
	51425: ClassDeathKnight, // Obliterate
	51460: ClassDeathKnight, // Necrosis
	52212: ClassDeathKnight, // Death and Decay
	53365: ClassDeathKnight, // Unholy Strength
	55078: ClassDeathKnight, // Blood Plague
	55095: ClassDeathKnight, // Frost Fever
	55268: ClassDeathKnight, // Frost Strike
	56815: ClassDeathKnight, // Rune Strike
	59754: ClassDeathKnight, // Rune Tap heal
	66962: ClassDeathKnight, // Frost Strike
	66974: ClassDeathKnight, // Obliterate

	// --- Druid ---
	48461: ClassDruid, // Wrath
	48463: ClassDruid, // Moonfire
	48465: ClassDruid, // Starfire
	48468: ClassDruid, // Insect Swarm
	48505: ClassDruid, // Starfall
	53190: ClassDruid, // Starfall
	53195: ClassDruid, // Starfall
	53227: ClassDruid, // Typhoon
	61384: ClassDruid, // Typhoon
	71023: ClassDruid, // Languish
	48438: ClassDruid, // Wild Growth
	50464: ClassDruid, // Nourish
	48378: ClassDruid, // Healing Touch
	48443: ClassDruid, // Regrowth
	48441: ClassDruid, // Rejuvenation
	48574: ClassDruid, // Rake
	48572: ClassDruid, // Shred
	48577: ClassDruid, // Ferocious Bite
	49800: ClassDruid, // Rip
	48568: ClassDruid, // Lacerate
	48564: ClassDruid, // Mangle (Bear)
	48566: ClassDruid, // Mangle (Cat)
	48480: ClassDruid, // Maul
	48562: ClassDruid, // Swipe (Bear)

	// --- Hunter ---
	75:    ClassHunter, // Auto Shot
	49001: ClassHunter, // Serpent Sting
	49048: ClassHunter, // Multi-Shot
	49052: ClassHunter, // Steady Shot
	49065: ClassHunter, // Explosive Trap
	53352: ClassHunter, // Explosive Shot
	58433: ClassHunter, // Volley
	61006: ClassHunter, // Kill Shot
	63672: ClassHunter, // Black Arrow
	60053: ClassHunter, // Explosive Shot
	49045: ClassHunter, // Arcane Shot
	49050: ClassHunter, // Aimed Shot
	53209: ClassHunter, // Chimera Shot
	19503: ClassHunter, // Scatter Shot
	34490: ClassHunter, // Silencing Shot

	// --- Mage ---
	42833: ClassMage, // Fireball
	42873: ClassMage, // Fire Blast
	42891: ClassMage, // Pyroblast
	42897: ClassMage, // Arcane Blast
	42921: ClassMage, // Arcane Explosion
	42926: ClassMage, // Flamestrike
	42938: ClassMage, // Blizzard
	42845: ClassMage, // Arcane Missiles
	12654: ClassMage, // Ignite
	55360: ClassMage, // Living Bomb
	55362: ClassMage, // Living Bomb detonation
	43044: ClassMage, // Molten Armor
	42859: ClassMage, // Scorch
	42940: ClassMage, // Blizzard
	47610: ClassMage, // Frostfire Bolt
	42842: ClassMage, // Frostbolt
	42914: ClassMage, // Ice Lance
	44572: ClassMage, // Deep Freeze
	44425: ClassMage, // Arcane Barrage
	44457: ClassMage, // Living Bomb

	// --- Paladin ---
	10308: ClassPaladin, // Hammer of Justice
	20267: ClassPaladin, // Judgement of Light
	48782: ClassPaladin, // Holy Light
	48785: ClassPaladin, // Flash of Light
	48806: ClassPaladin, // Hammer of Wrath
	48817: ClassPaladin, // Holy Wrath
	48819: ClassPaladin, // Consecration
	48821: ClassPaladin, // Holy Shock
	48827: ClassPaladin, // Avenger's Shield
	48952: ClassPaladin, // Holy Shield
	53595: ClassPaladin, // Hammer of the Righteous
	53652: ClassPaladin, // Beacon of Light
	53654: ClassPaladin, // Beacon of Light
	53733: ClassPaladin, // Judgement of Corruption
	53739: ClassPaladin, // Seal of Corruption
	53742: ClassPaladin, // Blood Corruption
	54158: ClassPaladin, // Judgement
	54968: ClassPaladin, // Glyph of Holy Light
	61411: ClassPaladin, // Shield of Righteousness
	67485: ClassPaladin, // Hand of Reckoning
	31803: ClassPaladin, // Holy Vengeance
	35395: ClassPaladin, // Crusader Strike
	53385: ClassPaladin, // Divine Storm
	20473: ClassPaladin, // Holy Shock
	31842: ClassPaladin, // Divine Illumination

	// --- Priest ---
	48068: ClassPriest, // Renew
	48071: ClassPriest, // Flash Heal
	48072: ClassPriest, // Prayer of Healing
	48076: ClassPriest, // Holy Nova
	48078: ClassPriest, // Holy Nova
	48089: ClassPriest, // Circle of Healing
	48125: ClassPriest, // Shadow Word: Pain
	48127: ClassPriest, // Mind Blast
	48135: ClassPriest, // Holy Fire
	48156: ClassPriest, // Mind Flay
	48160: ClassPriest, // Vampiric Touch
	48063: ClassPriest, // Greater Heal
	48066: ClassPriest, // Power Word: Shield
	53023: ClassPriest, // Mind Sear
	56161: ClassPriest, // Glyph of Prayer of Healing
	63544: ClassPriest, // Empowered Renew
	34914: ClassPriest, // Vampiric Touch
	15407: ClassPriest, // Mind Flay
	32379: ClassPriest, // Shadow Word: Death
	48158: ClassPriest, // Shadow Word: Death

	// --- Rogue ---
	48664: ClassRogue, // Mutilate
	48665: ClassRogue, // Mutilate
	48676: ClassRogue, // Garrote
	57965: ClassRogue, // Instant Poison IX
	57970: ClassRogue, // Deadly Poison IX
	57993: ClassRogue, // Envenom
	48638: ClassRogue, // Sinister Strike
	48668: ClassRogue, // Eviscerate
	48691: ClassRogue, // Ambush
	48657: ClassRogue, // Backstab
	51723: ClassRogue, // Fan of Knives
	51690: ClassRogue, // Killing Spree
	14183: ClassRogue, // Premeditation
	13750: ClassRogue, // Adrenaline Rush

	// --- Shaman ---
	49233: ClassShaman, // Flame Shock
	49238: ClassShaman, // Lightning Bolt
	49240: ClassShaman, // Lightning Bolt overload
	49271: ClassShaman, // Chain Lightning
	49269: ClassShaman, // Chain Lightning
	60043: ClassShaman, // Lava Burst
	49236: ClassShaman, // Earth Shock
	49231: ClassShaman, // Earth Shock
	59159: ClassShaman, // Thunderstorm
	61654: ClassShaman, // Fire Nova
	49276: ClassShaman, // Lesser Healing Wave
	49273: ClassShaman, // Healing Wave
	61301: ClassShaman, // Riptide
	49279: ClassShaman, // Lightning Shield
	51505: ClassShaman, // Lava Burst
	17364: ClassShaman, // Stormstrike
	60103: ClassShaman, // Lava Lash
	49281: ClassShaman, // Lightning Shield

	// --- Warlock ---
	47809: ClassWarlock, // Shadow Bolt
	47811: ClassWarlock, // Immolate
	47813: ClassWarlock, // Corruption
	47825: ClassWarlock, // Soul Fire
	47838: ClassWarlock, // Incinerate
	47855: ClassWarlock, // Drain Soul
	47893: ClassWarlock, // Fel Armor
	54181: ClassWarlock, // Fel Synergy
	25228: ClassWarlock, // Soul Link
	47843: ClassWarlock, // Unstable Affliction
	59172: ClassWarlock, // Haunt
	47864: ClassWarlock, // Curse of Agony
	47867: ClassWarlock, // Curse of Doom
	61290: ClassWarlock, // Shadowflame
	17962: ClassWarlock, // Conflagrate
	50796: ClassWarlock, // Chaos Bolt

	// --- Warrior ---
	1680:  ClassWarrior, // Whirlwind
	12721: ClassWarrior, // Deep Wounds
	12809: ClassWarrior, // Concussion Blow
	20253: ClassWarrior, // Intercept
	20647: ClassWarrior, // Execute
	23880: ClassWarrior, // Bloodthirst
	23881: ClassWarrior, // Bloodthirst
	44949: ClassWarrior, // Whirlwind off-hand
	47450: ClassWarrior, // Heroic Strike
	47465: ClassWarrior, // Rend
	47488: ClassWarrior, // Shield Slam
	47498: ClassWarrior, // Devastate
	47502: ClassWarrior, // Thunder Clap
	47520: ClassWarrior, // Cleave
	50783: ClassWarrior, // Slam
	57755: ClassWarrior, // Heroic Throw
	57823: ClassWarrior, // Revenge
	59653: ClassWarrior, // Damage Shield
	47486: ClassWarrior, // Mortal Strike
	47436: ClassWarrior, // Battle Shout
	46924: ClassWarrior, // Bladestorm
	64382: ClassWarrior, // Shattering Throw
	12294: ClassWarrior, // Mortal Strike
	23922: ClassWarrior, // Shield Slam
}
