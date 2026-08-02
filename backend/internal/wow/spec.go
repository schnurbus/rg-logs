package wow

// Spec is a WotLK talent-tree slug used in the API and UI.
type Spec string

const (
	SpecDeathKnightBlood  Spec = "deathknight-blood"
	SpecDeathKnightFrost  Spec = "deathknight-frost"
	SpecDeathKnightUnholy Spec = "deathknight-unholy"

	SpecDruidBalance     Spec = "druid-balance"
	SpecDruidFeral       Spec = "druid-feral"
	SpecDruidGuardian    Spec = "druid-guardian"
	SpecDruidRestoration Spec = "druid-restoration"

	SpecHunterBeastMastery Spec = "hunter-beastmastery"
	SpecHunterMarksmanship Spec = "hunter-marksmanship"
	SpecHunterSurvival     Spec = "hunter-survival"

	SpecMageArcane Spec = "mage-arcane"
	SpecMageFire   Spec = "mage-fire"
	SpecMageFrost  Spec = "mage-frost"

	SpecPaladinHoly        Spec = "paladin-holy"
	SpecPaladinProtection  Spec = "paladin-protection"
	SpecPaladinRetribution Spec = "paladin-retribution"

	SpecPriestDiscipline Spec = "priest-discipline"
	SpecPriestHoly       Spec = "priest-holy"
	SpecPriestShadow     Spec = "priest-shadow"

	SpecRogueAssassination Spec = "rogue-assassination"
	SpecRogueCombat        Spec = "rogue-combat"
	SpecRogueSubtlety      Spec = "rogue-subtlety"

	SpecShamanElemental   Spec = "shaman-elemental"
	SpecShamanEnhancement Spec = "shaman-enhancement"
	SpecShamanRestoration Spec = "shaman-restoration"

	SpecWarlockAffliction  Spec = "warlock-affliction"
	SpecWarlockDemonology  Spec = "warlock-demonology"
	SpecWarlockDestruction Spec = "warlock-destruction"

	SpecWarriorArms       Spec = "warrior-arms"
	SpecWarriorFury       Spec = "warrior-fury"
	SpecWarriorProtection Spec = "warrior-protection"
)

// Role is the raid role derived from a specialization.
type Role string

const (
	RoleTank   Role = "tank"
	RoleHealer Role = "healer"
	RoleDPS    Role = "dps"
)

// DetectSpec scores signature spell IDs (weighted by total) and returns the best match.
// Returns empty string when no signature spells are found.
func DetectSpec(spellTotals map[int]int64) Spec {
	if len(spellTotals) == 0 {
		return ""
	}

	scores := make(map[Spec]int64, 32)
	for spellID, total := range spellTotals {
		spec, ok := signatureSpecs[spellID]
		if !ok || total <= 0 {
			continue
		}
		scores[spec] += total
	}

	var best Spec
	var bestScore int64
	for spec, score := range scores {
		if score > bestScore {
			best = spec
			bestScore = score
		}
	}
	return best
}

// RoleFromSpec maps a specialization to tank / healer / dps.
func RoleFromSpec(spec Spec) Role {
	switch spec {
	case SpecDeathKnightBlood, SpecDruidGuardian, SpecPaladinProtection, SpecWarriorProtection:
		return RoleTank
	case SpecDruidRestoration, SpecPaladinHoly, SpecPriestDiscipline, SpecPriestHoly, SpecShamanRestoration:
		return RoleHealer
	default:
		if spec == "" {
			return RoleDPS
		}
		return RoleDPS
	}
}

// ClassFromSpec returns the class slug for a known specialization.
func ClassFromSpec(spec Spec) Class {
	switch spec {
	case SpecDeathKnightBlood, SpecDeathKnightFrost, SpecDeathKnightUnholy:
		return ClassDeathKnight
	case SpecDruidBalance, SpecDruidFeral, SpecDruidGuardian, SpecDruidRestoration:
		return ClassDruid
	case SpecHunterBeastMastery, SpecHunterMarksmanship, SpecHunterSurvival:
		return ClassHunter
	case SpecMageArcane, SpecMageFire, SpecMageFrost:
		return ClassMage
	case SpecPaladinHoly, SpecPaladinProtection, SpecPaladinRetribution:
		return ClassPaladin
	case SpecPriestDiscipline, SpecPriestHoly, SpecPriestShadow:
		return ClassPriest
	case SpecRogueAssassination, SpecRogueCombat, SpecRogueSubtlety:
		return ClassRogue
	case SpecShamanElemental, SpecShamanEnhancement, SpecShamanRestoration:
		return ClassShaman
	case SpecWarlockAffliction, SpecWarlockDemonology, SpecWarlockDestruction:
		return ClassWarlock
	case SpecWarriorArms, SpecWarriorFury, SpecWarriorProtection:
		return ClassWarrior
	default:
		return ""
	}
}

// IconFromSpec returns the Wowhead medium icon basename for a specialization.
func IconFromSpec(spec Spec) string {
	switch spec {
	case SpecDeathKnightBlood:
		return "spell_deathknight_bloodpresence"
	case SpecDeathKnightFrost:
		return "spell_deathknight_frostpresence"
	case SpecDeathKnightUnholy:
		return "spell_deathknight_unholypresence"
	case SpecDruidBalance:
		return "spell_nature_starfall"
	case SpecDruidFeral:
		return "ability_druid_catform"
	case SpecDruidGuardian:
		return "ability_racial_bearform"
	case SpecDruidRestoration:
		return "spell_nature_healingtouch"
	case SpecHunterBeastMastery:
		return "ability_hunter_bestialdiscipline"
	case SpecHunterMarksmanship:
		return "ability_hunter_focusedaim"
	case SpecHunterSurvival:
		return "ability_hunter_camouflage"
	case SpecMageArcane:
		return "spell_holy_magicalsentry"
	case SpecMageFire:
		return "spell_fire_firebolt02"
	case SpecMageFrost:
		return "spell_frost_frostbolt02"
	case SpecPaladinHoly:
		return "spell_holy_holybolt"
	case SpecPaladinProtection:
		return "spell_holy_devotionaura"
	case SpecPaladinRetribution:
		return "spell_holy_auraoflight"
	case SpecPriestDiscipline:
		return "spell_holy_powerwordshield"
	case SpecPriestHoly:
		return "spell_holy_guardianspirit"
	case SpecPriestShadow:
		return "spell_shadow_shadowwordpain"
	case SpecRogueAssassination:
		return "ability_rogue_eviscerate"
	case SpecRogueCombat:
		return "ability_backstab"
	case SpecRogueSubtlety:
		return "ability_stealth"
	case SpecShamanElemental:
		return "spell_nature_lightning"
	case SpecShamanEnhancement:
		return "spell_nature_lightningshield"
	case SpecShamanRestoration:
		return "spell_nature_magicimmunity"
	case SpecWarlockAffliction:
		return "spell_shadow_deathcoil"
	case SpecWarlockDemonology:
		return "spell_shadow_metamorphosis"
	case SpecWarlockDestruction:
		return "spell_shadow_rainoffire"
	case SpecWarriorArms:
		return "ability_warrior_savageblow"
	case SpecWarriorFury:
		return "ability_warrior_innerrage"
	case SpecWarriorProtection:
		return "ability_warrior_defensivestance"
	default:
		return "inv_misc_questionmark"
	}
}

// signatureSpecs maps WotLK spell IDs to talent-tree specializations.
// Prefer tree-defining abilities; shared class utilities are omitted or weakly weighted via lower totals.
var signatureSpecs = map[int]Spec{
	// --- Death Knight: Blood ---
	49924: SpecDeathKnightBlood, // Death Strike
	45470: SpecDeathKnightBlood, // Death Strike (heal)
	49930: SpecDeathKnightBlood, // Blood Strike
	48982: SpecDeathKnightBlood, // Rune Tap
	59754: SpecDeathKnightBlood, // Rune Tap heal
	50463: SpecDeathKnightBlood, // Blood-Caked Strike
	56815: SpecDeathKnightBlood, // Rune Strike
	55262: SpecDeathKnightBlood, // Heart Strike
	55233: SpecDeathKnightBlood, // Vampiric Blood

	// --- Death Knight: Frost ---
	51411: SpecDeathKnightFrost, // Howling Blast
	51425: SpecDeathKnightFrost, // Obliterate
	66974: SpecDeathKnightFrost, // Obliterate
	55268: SpecDeathKnightFrost, // Frost Strike
	66962: SpecDeathKnightFrost, // Frost Strike
	50401: SpecDeathKnightFrost, // Razorice
	49143: SpecDeathKnightFrost, // Frost Strike (base)
	49184: SpecDeathKnightFrost, // Howling Blast (base)

	// --- Death Knight: Unholy ---
	50536: SpecDeathKnightUnholy, // Unholy Blight
	51460: SpecDeathKnightUnholy, // Necrosis
	50526: SpecDeathKnightUnholy, // Wandering Plague
	52212: SpecDeathKnightUnholy, // Death and Decay
	53365: SpecDeathKnightUnholy, // Unholy Strength
	49206: SpecDeathKnightUnholy, // Summon Gargoyle
	55090: SpecDeathKnightUnholy, // Scourge Strike
	55265: SpecDeathKnightUnholy, // Scourge Strike
	55270: SpecDeathKnightUnholy, // Scourge Strike
	55271: SpecDeathKnightUnholy, // Scourge Strike

	// --- Druid: Balance ---
	48461: SpecDruidBalance, // Wrath
	48463: SpecDruidBalance, // Moonfire
	48465: SpecDruidBalance, // Starfire
	48468: SpecDruidBalance, // Insect Swarm
	48505: SpecDruidBalance, // Starfall
	53190: SpecDruidBalance, // Starfall
	53195: SpecDruidBalance, // Starfall
	53227: SpecDruidBalance, // Typhoon
	61384: SpecDruidBalance, // Typhoon
	71023: SpecDruidBalance, // Languish
	24858: SpecDruidBalance, // Moonkin Form

	// --- Druid: Feral (Cat) ---
	48574: SpecDruidFeral, // Rake
	48572: SpecDruidFeral, // Shred
	48577: SpecDruidFeral, // Ferocious Bite
	49800: SpecDruidFeral, // Rip
	48566: SpecDruidFeral, // Mangle (Cat)
	50213: SpecDruidFeral, // Tiger's Fury
	62078: SpecDruidFeral, // Swipe (Cat)

	// --- Druid: Guardian (Bear) ---
	48568: SpecDruidGuardian, // Lacerate
	48564: SpecDruidGuardian, // Mangle (Bear)
	48480: SpecDruidGuardian, // Maul
	48562: SpecDruidGuardian, // Swipe (Bear)
	62606: SpecDruidGuardian, // Savage Defense
	61336: SpecDruidGuardian, // Survival Instincts

	// --- Druid: Restoration ---
	48438: SpecDruidRestoration, // Wild Growth
	50464: SpecDruidRestoration, // Nourish
	48378: SpecDruidRestoration, // Healing Touch
	48443: SpecDruidRestoration, // Regrowth
	48441: SpecDruidRestoration, // Rejuvenation
	18562: SpecDruidRestoration, // Swiftmend
	33763: SpecDruidRestoration, // Lifebloom

	// --- Hunter: Beast Mastery ---
	34026: SpecHunterBeastMastery, // Kill Command
	19574: SpecHunterBeastMastery, // Bestial Wrath
	34471: SpecHunterBeastMastery, // The Beast Within

	// --- Hunter: Marksmanship ---
	53209: SpecHunterMarksmanship, // Chimera Shot
	49050: SpecHunterMarksmanship, // Aimed Shot
	34490: SpecHunterMarksmanship, // Silencing Shot
	19506: SpecHunterMarksmanship, // Trueshot Aura

	// --- Hunter: Survival ---
	53352: SpecHunterSurvival, // Explosive Shot
	60053: SpecHunterSurvival, // Explosive Shot
	63672: SpecHunterSurvival, // Black Arrow
	49065: SpecHunterSurvival, // Explosive Trap
	3674:  SpecHunterSurvival, // Black Arrow (base)

	// --- Mage: Arcane ---
	42897: SpecMageArcane, // Arcane Blast
	42845: SpecMageArcane, // Arcane Missiles
	44425: SpecMageArcane, // Arcane Barrage
	42921: SpecMageArcane, // Arcane Explosion
	12042: SpecMageArcane, // Arcane Power
	31589: SpecMageArcane, // Slow

	// --- Mage: Fire ---
	42833: SpecMageFire, // Fireball
	42891: SpecMageFire, // Pyroblast
	42873: SpecMageFire, // Fire Blast
	12654: SpecMageFire, // Ignite
	55360: SpecMageFire, // Living Bomb
	55362: SpecMageFire, // Living Bomb detonation
	44457: SpecMageFire, // Living Bomb
	42859: SpecMageFire, // Scorch
	42926: SpecMageFire, // Flamestrike
	47610: SpecMageFire, // Frostfire Bolt
	11129: SpecMageFire, // Combustion

	// --- Mage: Frost ---
	42842: SpecMageFrost, // Frostbolt
	42914: SpecMageFrost, // Ice Lance
	42938: SpecMageFrost, // Blizzard
	42940: SpecMageFrost, // Blizzard
	44572: SpecMageFrost, // Deep Freeze
	12472: SpecMageFrost, // Icy Veins
	11958: SpecMageFrost, // Cold Snap

	// --- Paladin: Holy ---
	48782: SpecPaladinHoly, // Holy Light
	48785: SpecPaladinHoly, // Flash of Light
	48821: SpecPaladinHoly, // Holy Shock
	20473: SpecPaladinHoly, // Holy Shock
	53652: SpecPaladinHoly, // Beacon of Light
	53654: SpecPaladinHoly, // Beacon of Light
	54968: SpecPaladinHoly, // Glyph of Holy Light
	31842: SpecPaladinHoly, // Divine Illumination
	20267: SpecPaladinHoly, // Judgement of Light (strong holy signal when healing)

	// --- Paladin: Protection ---
	48827: SpecPaladinProtection, // Avenger's Shield
	48952: SpecPaladinProtection, // Holy Shield
	53595: SpecPaladinProtection, // Hammer of the Righteous
	61411: SpecPaladinProtection, // Shield of Righteousness
	20925: SpecPaladinProtection, // Holy Shield (base)
	31935: SpecPaladinProtection, // Avenger's Shield (base)
	53600: SpecPaladinProtection, // Shield of Righteousness (base)

	// --- Paladin: Retribution ---
	35395: SpecPaladinRetribution, // Crusader Strike
	53385: SpecPaladinRetribution, // Divine Storm
	31884: SpecPaladinRetribution, // Avenging Wrath
	20066: SpecPaladinRetribution, // Repentance
	59578: SpecPaladinRetribution, // The Art of War
	53408: SpecPaladinRetribution, // Judgement of Command

	// --- Priest: Discipline ---
	48066: SpecPriestDiscipline, // Power Word: Shield
	47540: SpecPriestDiscipline, // Penance
	53007: SpecPriestDiscipline, // Penance
	47666: SpecPriestDiscipline, // Penance
	47750: SpecPriestDiscipline, // Penance
	33206: SpecPriestDiscipline, // Pain Suppression
	10060: SpecPriestDiscipline, // Power Infusion

	// --- Priest: Holy ---
	48089: SpecPriestHoly, // Circle of Healing
	48068: SpecPriestHoly, // Renew
	48072: SpecPriestHoly, // Prayer of Healing
	48071: SpecPriestHoly, // Flash Heal
	48063: SpecPriestHoly, // Greater Heal
	48076: SpecPriestHoly, // Holy Nova
	48078: SpecPriestHoly, // Holy Nova
	48135: SpecPriestHoly, // Holy Fire
	56161: SpecPriestHoly, // Glyph of Prayer of Healing
	63544: SpecPriestHoly, // Empowered Renew
	34861: SpecPriestHoly, // Circle of Healing (base)
	47788: SpecPriestHoly, // Guardian Spirit

	// --- Priest: Shadow ---
	48125: SpecPriestShadow, // Shadow Word: Pain
	48127: SpecPriestShadow, // Mind Blast
	48156: SpecPriestShadow, // Mind Flay
	48160: SpecPriestShadow, // Vampiric Touch
	34914: SpecPriestShadow, // Vampiric Touch
	15407: SpecPriestShadow, // Mind Flay
	32379: SpecPriestShadow, // Shadow Word: Death
	48158: SpecPriestShadow, // Shadow Word: Death
	53023: SpecPriestShadow, // Mind Sear
	15487: SpecPriestShadow, // Silence
	47585: SpecPriestShadow, // Dispersion

	// --- Rogue: Assassination ---
	48664: SpecRogueAssassination, // Mutilate
	48665: SpecRogueAssassination, // Mutilate
	57993: SpecRogueAssassination, // Envenom
	48676: SpecRogueAssassination, // Garrote
	57965: SpecRogueAssassination, // Instant Poison IX
	57970: SpecRogueAssassination, // Deadly Poison IX
	1329:  SpecRogueAssassination, // Mutilate (base)
	32645: SpecRogueAssassination, // Envenom (base)

	// --- Rogue: Combat ---
	48638: SpecRogueCombat, // Sinister Strike
	51690: SpecRogueCombat, // Killing Spree
	13750: SpecRogueCombat, // Adrenaline Rush
	13877: SpecRogueCombat, // Blade Flurry
	51723: SpecRogueCombat, // Fan of Knives
	48668: SpecRogueCombat, // Eviscerate

	// --- Rogue: Subtlety ---
	48657: SpecRogueSubtlety, // Backstab
	48691: SpecRogueSubtlety, // Ambush
	14183: SpecRogueSubtlety, // Premeditation
	36554: SpecRogueSubtlety, // Shadowstep
	51713: SpecRogueSubtlety, // Shadow Dance
	16511: SpecRogueSubtlety, // Hemorrhage

	// --- Shaman: Elemental ---
	49238: SpecShamanElemental, // Lightning Bolt
	49240: SpecShamanElemental, // Lightning Bolt overload
	49271: SpecShamanElemental, // Chain Lightning
	49269: SpecShamanElemental, // Chain Lightning
	60043: SpecShamanElemental, // Lava Burst
	51505: SpecShamanElemental, // Lava Burst
	49233: SpecShamanElemental, // Flame Shock
	59159: SpecShamanElemental, // Thunderstorm
	16166: SpecShamanElemental, // Elemental Mastery

	// --- Shaman: Enhancement ---
	17364: SpecShamanEnhancement, // Stormstrike
	60103: SpecShamanEnhancement, // Lava Lash
	51533: SpecShamanEnhancement, // Feral Spirit
	30823: SpecShamanEnhancement, // Shamanistic Rage
	49281: SpecShamanEnhancement, // Lightning Shield
	49279: SpecShamanEnhancement, // Lightning Shield
	61654: SpecShamanEnhancement, // Fire Nova

	// --- Shaman: Restoration ---
	49276: SpecShamanRestoration, // Lesser Healing Wave
	49273: SpecShamanRestoration, // Healing Wave
	61301: SpecShamanRestoration, // Riptide
	61295: SpecShamanRestoration, // Riptide
	974:   SpecShamanRestoration, // Earth Shield
	49284: SpecShamanRestoration, // Earth Shield
	16190: SpecShamanRestoration, // Mana Tide Totem
	51886: SpecShamanRestoration, // Cleanse Spirit

	// --- Warlock: Affliction ---
	47813: SpecWarlockAffliction, // Corruption
	47843: SpecWarlockAffliction, // Unstable Affliction
	59172: SpecWarlockAffliction, // Haunt
	47864: SpecWarlockAffliction, // Curse of Agony
	47855: SpecWarlockAffliction, // Drain Soul
	48181: SpecWarlockAffliction, // Haunt (base)
	30108: SpecWarlockAffliction, // Unstable Affliction (base)

	// --- Warlock: Demonology ---
	25228: SpecWarlockDemonology, // Soul Link
	54181: SpecWarlockDemonology, // Fel Synergy
	47893: SpecWarlockDemonology, // Fel Armor
	47193: SpecWarlockDemonology, // Demonic Empowerment
	59672: SpecWarlockDemonology, // Metamorphosis
	50589: SpecWarlockDemonology, // Immolation Aura
	47241: SpecWarlockDemonology, // Metamorphosis

	// --- Warlock: Destruction ---
	47809: SpecWarlockDestruction, // Shadow Bolt
	47811: SpecWarlockDestruction, // Immolate
	47838: SpecWarlockDestruction, // Incinerate
	47825: SpecWarlockDestruction, // Soul Fire
	17962: SpecWarlockDestruction, // Conflagrate
	50796: SpecWarlockDestruction, // Chaos Bolt
	61290: SpecWarlockDestruction, // Shadowflame
	47867: SpecWarlockDestruction, // Curse of Doom

	// --- Warrior: Arms ---
	47486: SpecWarriorArms, // Mortal Strike
	12294: SpecWarriorArms, // Mortal Strike
	46924: SpecWarriorArms, // Bladestorm
	64382: SpecWarriorArms, // Shattering Throw
	12328: SpecWarriorArms, // Sweeping Strikes
	50783: SpecWarriorArms, // Slam

	// --- Warrior: Fury ---
	23880: SpecWarriorFury, // Bloodthirst
	23881: SpecWarriorFury, // Bloodthirst
	1680:  SpecWarriorFury, // Whirlwind
	44949: SpecWarriorFury, // Whirlwind off-hand
	12292: SpecWarriorFury, // Death Wish
	60970: SpecWarriorFury, // Heroic Fury

	// --- Warrior: Protection ---
	47488: SpecWarriorProtection, // Shield Slam
	23922: SpecWarriorProtection, // Shield Slam
	47498: SpecWarriorProtection, // Devastate
	57823: SpecWarriorProtection, // Revenge
	12809: SpecWarriorProtection, // Concussion Blow
	59653: SpecWarriorProtection, // Damage Shield
	46968: SpecWarriorProtection, // Shockwave
	2565:  SpecWarriorProtection, // Shield Block
}
