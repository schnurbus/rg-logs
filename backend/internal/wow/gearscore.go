package wow

import "math"

// EquippedItem is one worn item for GearScoreLite scoring.
type EquippedItem struct {
	ItemLevel int
	Quality   int // 0=poor .. 7=heirloom (Blizzard rarity)
	SlotBak   int // INVTYPE slotbak (1=head, 17=2H, …)
}

// inventory slot IDs used by the GearScoreLite GetScore loop
const (
	invSlotShirt   = 4
	invSlotMainHand = 16
	invSlotOffHand  = 17
	invSlotRanged   = 18
)

// slotMOD mirrors GS_ItemTypes[].SlotMOD from GearScoreLite / TacoTip.
var slotMOD = map[int]float64{
	1:  1.0,    // INVTYPE_HEAD
	2:  0.5625, // INVTYPE_NECK
	3:  0.75,   // INVTYPE_SHOULDER
	4:  0,      // INVTYPE_BODY
	5:  1.0,    // INVTYPE_CHEST
	6:  0.75,   // INVTYPE_WAIST
	7:  1.0,    // INVTYPE_LEGS
	8:  0.75,   // INVTYPE_FEET
	9:  0.5625, // INVTYPE_WRIST
	10: 0.75,   // INVTYPE_HAND
	11: 0.5625, // INVTYPE_FINGER
	12: 0.5625, // INVTYPE_TRINKET
	13: 1.0,    // INVTYPE_WEAPON
	14: 1.0,    // INVTYPE_SHIELD
	15: 0.3164, // INVTYPE_RANGED
	16: 0.5625, // INVTYPE_CLOAK
	17: 2.0,    // INVTYPE_2HWEAPON
	20: 1.0,    // INVTYPE_ROBE
	21: 1.0,    // INVTYPE_WEAPONMAINHAND
	22: 1.0,    // INVTYPE_WEAPONOFFHAND
	23: 1.0,    // INVTYPE_HOLDABLE
	25: 0.3164, // INVTYPE_THROWN
	26: 0.3164, // INVTYPE_RANGEDRIGHT
	28: 0.3164, // INVTYPE_RELIC
}

type gsFormula struct{ A, B float64 }

var (
	gsFormulaA = map[int]gsFormula{
		4: {91.45, 0.65},
		3: {81.375, 0.8125},
		2: {73.0, 1.0},
	}
	gsFormulaB = map[int]gsFormula{
		4: {26.0, 1.2},
		3: {0.75, 1.8},
		2: {8.0, 2.0},
		1: {0.0, 2.25},
	}
	gsFormulaC = map[int]gsFormula{
		4: {0.25, 1.6275},
	}
)

const gsScale = 1.8618

// ItemScore returns the GearScoreLite score for a single item (no hunter/TG adjust).
func ItemScore(item EquippedItem) int {
	ilvl := float64(item.ItemLevel)
	rarity := item.Quality
	qualityScale := 1.0

	switch rarity {
	case 5: // legendary
		qualityScale = 1.3
		rarity = 4
	case 1: // common
		qualityScale = 0.005
		rarity = 2
	case 0: // poor
		qualityScale = 0.005
		rarity = 2
	case 7: // heirloom
		rarity = 3
		ilvl = 187.05
	}

	mod, ok := slotMOD[item.SlotBak]
	if !ok || mod == 0 {
		return 0
	}

	var table map[int]gsFormula
	switch {
	case ilvl < 100 && rarity == 4:
		table = gsFormulaC
	case ilvl < 168 && rarity == 4:
		table = gsFormulaB
	case ilvl < 148 && rarity == 3:
		table = gsFormulaB
	case ilvl < 138 && rarity == 2:
		table = gsFormulaB
	case ilvl <= 120:
		table = gsFormulaB
	default:
		table = gsFormulaA
	}

	f, ok := table[rarity]
	if !ok || rarity < 2 || rarity > 4 {
		return 0
	}

	score := math.Floor(((ilvl - f.A) / f.B) * mod * gsScale * qualityScale)
	if score < 0 {
		return 0
	}
	return int(score)
}

// isTwoHand reports INVTYPE_2HWEAPON.
func isTwoHand(slotBak int) bool { return slotBak == 17 }

// isWeaponLike reports main/off-hand weapon INVTYPEs used for hunter MH penalty.
func isWeaponLike(slotBak int) bool {
	switch slotBak {
	case 13, 17, 21, 22, 23: // weapon, 2H, MH, OH, holdable
		return true
	default:
		return false
	}
}

func isRanged(slotBak int) bool {
	switch slotBak {
	case 15, 25, 26: // ranged, thrown, rangedright
		return true
	default:
		return false
	}
}

// GearScore computes total GearScoreLite for equipped items.
// byInvSlot maps inventory slot (1–18) → item. Shirt (4) is ignored.
// class is our Class slug (hunter gets weapon/ranged multipliers).
func GearScore(byInvSlot map[int]EquippedItem, class Class) int {
	if len(byInvSlot) == 0 {
		return 0
	}

	titanGrip := 1.0
	mh, hasMH := byInvSlot[invSlotMainHand]
	oh, hasOH := byInvSlot[invSlotOffHand]
	if hasMH && hasOH && (isTwoHand(mh.SlotBak) || isTwoHand(oh.SlotBak)) {
		titanGrip = 0.5
	}

	total := 0.0
	hunter := class == ClassHunter

	if hasOH {
		sc := float64(ItemScore(oh))
		if hunter {
			sc *= 0.3164
		}
		total += sc * titanGrip
	}

	for slot := 1; slot <= 18; slot++ {
		if slot == invSlotShirt || slot == invSlotOffHand {
			continue
		}
		item, ok := byInvSlot[slot]
		if !ok {
			continue
		}
		sc := float64(ItemScore(item))
		if hunter {
			switch {
			case slot == invSlotMainHand || (slot != invSlotRanged && isWeaponLike(item.SlotBak)):
				sc *= 0.3164
			case slot == invSlotRanged || isRanged(item.SlotBak):
				sc *= 5.3224
			}
		}
		if slot == invSlotMainHand {
			sc *= titanGrip
		}
		total += sc
	}

	return int(math.Floor(total))
}

// ClassFromWoWClassID maps Blizzard class IDs used by AoWow profiles.
func ClassFromWoWClassID(id int) Class {
	switch id {
	case 1:
		return ClassWarrior
	case 2:
		return ClassPaladin
	case 3:
		return ClassHunter
	case 4:
		return ClassRogue
	case 5:
		return ClassPriest
	case 6:
		return ClassDeathKnight
	case 7:
		return ClassShaman
	case 8:
		return ClassMage
	case 9:
		return ClassWarlock
	case 11:
		return ClassDruid
	default:
		return ""
	}
}
