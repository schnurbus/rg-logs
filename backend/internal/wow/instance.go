package wow

import "strings"

// DetectInstance guesses a WotLK raid/instance name from fight titles (boss names).
// Returns "" when no known bosses are found.
func DetectInstance(fightTitles []string) string {
	scores := make(map[string]int)
	for _, title := range fightTitles {
		key := normalizeBoss(title)
		if key == "" || key == "trash" {
			continue
		}
		if inst, ok := bossInstances[key]; ok {
			scores[inst]++
			continue
		}
		if id := EncounterID(title); id != "" {
			if inst := encounterInstance[id]; inst != "" {
				scores[inst]++
			}
		}
	}
	var best string
	var bestScore int
	for inst, score := range scores {
		if score > bestScore {
			best = inst
			bestScore = score
		}
	}
	return best
}

// IsKnownBoss reports whether title matches a known WotLK boss (EN/DE) in bossInstances
// or a recognized multi-NPC encounter unit (e.g. gunship adds).
func IsKnownBoss(title string) bool {
	key := normalizeBoss(title)
	if key == "" || key == "trash" {
		return false
	}
	if _, ok := bossInstances[key]; ok {
		return true
	}
	return EncounterID(title) != ""
}

// FightTitle returns a canonical fight label for multi-NPC encounters, or the
// original NPC name for single-boss fights.
func FightTitle(npcName string) string {
	key := normalizeBoss(npcName)
	if t, ok := bossFightTitles[key]; ok {
		return t
	}
	if id := EncounterID(npcName); id != "" {
		if t, ok := encounterTitles[id]; ok {
			return t
		}
	}
	return npcName
}

// EncounterID groups NPCs that belong to the same multi-boss encounter.
// Empty means a standalone boss (or unknown NPC).
func EncounterID(name string) string {
	key := normalizeBoss(name)
	if id, ok := bossEncounterIDs[key]; ok {
		return id
	}
	if isGunshipUnit(key) {
		return encICCGunship
	}
	return ""
}

func normalizeBoss(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "`", "'")
	return s
}

func isGunshipUnit(key string) bool {
	switch {
	case strings.Contains(key, "der himmelsbrecher"),
		strings.Contains(key, "der kor'kron"),
		strings.HasSuffix(key, "skybreaker"),
		strings.Contains(key, "skybreaker "):
		return true
	}
	switch key {
	case "muradin bronzebart", "muradin bronzebeard",
		"hochfürst saurfang", "high overlord saurfang",
		"orgrims hammer", "orgrim's hammer",
		"die himmelsbrecher", "the skybreaker",
		"luftschiffrumpf", "gunship hull",
		"kanone des hordenkanonenboots", "kanone des allianzkanonenboots",
		"horde gunship cannon", "alliance gunship cannon":
		return true
	}
	return false
}

const (
	encICCGunship      = "icc-gunship"
	encICCBloodCouncil = "icc-blood-council"
	encICCValithria    = "icc-valithria"
)

var encounterInstance = map[string]string{
	encICCGunship:      "Eiskronenzitadelle",
	encICCBloodCouncil: "Eiskronenzitadelle",
	encICCValithria:    "Eiskronenzitadelle",
}

var encounterTitles = map[string]string{
	encICCGunship:      "Kanonenschiffsschlacht von der Eiskrone",
	encICCBloodCouncil: "Rat der Blutprinzen",
	encICCValithria:    "Valithria Traumwandler",
}

// bossFightTitles maps normalized NPC names to preferred fight titles.
var bossFightTitles = map[string]string{
	"prinz valanar":             "Rat der Blutprinzen",
	"prince valanar":            "Rat der Blutprinzen",
	"prinz keleseth":            "Rat der Blutprinzen",
	"prince keleseth":           "Rat der Blutprinzen",
	"prinz taldaram":            "Rat der Blutprinzen",
	"prince taldaram":           "Rat der Blutprinzen",
	"steuerung der blutkugel":   "Rat der Blutprinzen",
	"blood orb controller":      "Rat der Blutprinzen",
	"rat der blutprinzen":       "Rat der Blutprinzen",
	"blood prince council":      "Rat der Blutprinzen",
	"blutrat der blutfürsten":   "Rat der Blutprinzen",
	"valithria traumwandler":    "Valithria Traumwandler",
	"valithria dreamwalker":     "Valithria Traumwandler",
}

// bossEncounterIDs maps bosses that share a multi-NPC encounter.
var bossEncounterIDs = map[string]string{
	"prinz valanar":           encICCBloodCouncil,
	"prince valanar":          encICCBloodCouncil,
	"prinz keleseth":          encICCBloodCouncil,
	"prince keleseth":         encICCBloodCouncil,
	"prinz taldaram":          encICCBloodCouncil,
	"prince taldaram":         encICCBloodCouncil,
	"steuerung der blutkugel": encICCBloodCouncil,
	"blood orb controller":    encICCBloodCouncil,
	"rat der blutprinzen":     encICCBloodCouncil,
	"blood prince council":    encICCBloodCouncil,
	"blutrat der blutfürsten": encICCBloodCouncil,
	"valithria traumwandler":  encICCValithria,
	"valithria dreamwalker":   encICCValithria,
}

// IsGunshipEncounter reports whether the NPC belongs to the ICC gunship battle.
func IsGunshipEncounter(name string) bool {
	return EncounterID(name) == encICCGunship
}

// IsHealEncounterBoss reports whether the NPC is the objective of a heal fight
// (currently Valithria Dreamwalker).
func IsHealEncounterBoss(name string) bool {
	return EncounterID(name) == encICCValithria
}

// DeathEndsEncounter reports whether UNIT_DIED of this NPC should close the fight.
// Multi-NPC encounters keep going through add/prince deaths until a combat gap,
// a success spell, or a different encounter engages.
func DeathEndsEncounter(name string) bool {
	if !IsKnownBoss(name) {
		return false
	}
	return EncounterID(name) == ""
}

// UnitDeathCountsAsKill reports whether death of this NPC alone may mark the fight
// as a kill. Gunship crew deaths are wave progress, not an encounter kill.
func UnitDeathCountsAsKill(name string) bool {
	if EncounterID(name) == encICCGunship {
		return false
	}
	return IsKnownBoss(name)
}

// ValithriaSuccessSpellID is Dreamwalker's Rage — cast when the heal fight succeeds.
const ValithriaSuccessSpellID = 71189

// GunshipSuccessSpellID is Teleport to Deathbringer's Rise — applied when the
// gunship battle is won (APPLIED/REMOVED both count; APPLIED is often missing).
const GunshipSuccessSpellID = 70858

// bossInstances maps normalized EN/DE boss names to display instance names (German).
var bossInstances = map[string]string{
	// Naxxramas
	"anub'rekhan":            "Naxxramas",
	"großwitwe faerlina":     "Naxxramas",
	"grand widow faerlina":   "Naxxramas",
	"maexxna":                "Naxxramas",
	"noth der seuchenfürst":  "Naxxramas",
	"noth the plaguebringer": "Naxxramas",
	"heigan der unreine":     "Naxxramas",
	"heigan the unclean":     "Naxxramas",
	"loatheb":                "Naxxramas",
	"instrukteur razuvious":  "Naxxramas",
	"instructor razuvious":   "Naxxramas",
	"gothik der ernter":      "Naxxramas",
	"gothik the harvester":   "Naxxramas",
	"die vier reiter":        "Naxxramas",
	"the four horsemen":      "Naxxramas",
	"thane korth'azz":        "Naxxramas",
	"sir zeliek":             "Naxxramas",
	"lady blaumeux":          "Naxxramas",
	"baron rivendare":        "Naxxramas",
	"patchwerk":              "Naxxramas",
	"grobbulus":              "Naxxramas",
	"gluth":                  "Naxxramas",
	"thaddius":               "Naxxramas",
	"sapphiron":              "Naxxramas",
	"kel'thuzad":             "Naxxramas",

	// Obsidiansanktum
	"sartharion": "Obsidiansanktum",

	// Auge der Ewigkeit
	"malygos": "Auge der Ewigkeit",

	// Archavons Kammer
	"archavon der steinwächter":  "Archavons Kammer",
	"archavon the stone watcher": "Archavons Kammer",
	"emalon der sturmwächter":    "Archavons Kammer",
	"emalon the storm watcher":   "Archavons Kammer",
	"koralon der flammenwächter": "Archavons Kammer",
	"koralon the flame watcher":  "Archavons Kammer",
	"toravon der eiswächter":     "Archavons Kammer",
	"toravon the ice watcher":    "Archavons Kammer",

	// Ulduar
	"flammenleviathan":           "Ulduar",
	"flame leviathan":            "Ulduar",
	"ignis der ofenlord":         "Ulduar",
	"ignis the furnace master":   "Ulduar",
	"klingenschuppe":             "Ulduar",
	"razorscale":                 "Ulduar",
	"xt-002 dekonstruktor":       "Ulduar",
	"xt-002 deconstructor":       "Ulduar",
	"die versammlung des eisens": "Ulduar",
	"the assembly of iron":       "Ulduar",
	"stahlbrecher":               "Ulduar",
	"steelbreaker":               "Ulduar",
	"runenmeister molgeim":       "Ulduar",
	"runemaster molgeim":         "Ulduar",
	"sturmrufer brundir":         "Ulduar",
	"stormcaller brundir":        "Ulduar",
	"kologarn":                   "Ulduar",
	"auriaya":                    "Ulduar",
	"hodir":                      "Ulduar",
	"thorim":                     "Ulduar",
	"freya":                      "Ulduar",
	"mimiron":                    "Ulduar",
	"general vezax":              "Ulduar",
	"yogg-saron":                 "Ulduar",
	"algalon der beobachter":     "Ulduar",
	"algalon the observer":       "Ulduar",

	// Prüfung des Kreuzfahrers
	"die bestien von nordend": "Prüfung des Kreuzfahrers",
	"northrend beasts":        "Prüfung des Kreuzfahrers",
	"gormok der pfähler":      "Prüfung des Kreuzfahrers",
	"gormok the impaler":      "Prüfung des Kreuzfahrers",
	"säureschlund":            "Prüfung des Kreuzfahrers",
	"acidmaw":                 "Prüfung des Kreuzfahrers",
	"schreckensschuppe":       "Prüfung des Kreuzfahrers",
	"dreadscale":              "Prüfung des Kreuzfahrers",
	"eisheuler":               "Prüfung des Kreuzfahrers",
	"icehowl":                 "Prüfung des Kreuzfahrers",
	"lord jaraxxus":           "Prüfung des Kreuzfahrers",
	"fraktionschampions":      "Prüfung des Kreuzfahrers",
	"faction champions":       "Prüfung des Kreuzfahrers",
	"zwillingsval'kyr":        "Prüfung des Kreuzfahrers",
	"twin val'kyr":            "Prüfung des Kreuzfahrers",
	"fjola lightbane":         "Prüfung des Kreuzfahrers",
	"fjola lichtbann":         "Prüfung des Kreuzfahrers",
	"eydis darkbane":          "Prüfung des Kreuzfahrers",
	"eydis nachtbann":         "Prüfung des Kreuzfahrers",
	"anub'arak":               "Prüfung des Kreuzfahrers",

	// Onyxias Hort
	"onyxia": "Onyxias Hort",

	// Eiskronenzitadelle
	"lord mark'gar":                           "Eiskronenzitadelle",
	"lord marrowgar":                          "Eiskronenzitadelle",
	"lady todeswisper":                        "Eiskronenzitadelle",
	"lady deathwhisper":                       "Eiskronenzitadelle",
	"kanonenschiffsschlacht von der eiskrone": "Eiskronenzitadelle",
	"icecrown gunship battle":                 "Eiskronenzitadelle",
	"todesbringer saurfang":                   "Eiskronenzitadelle",
	"todbringer saurfang":                     "Eiskronenzitadelle", // legacy typo alias
	"deathbringer saurfang":                   "Eiskronenzitadelle",
	"fauldarm":                                "Eiskronenzitadelle",
	"festergut":                               "Eiskronenzitadelle",
	"modermiene":                              "Eiskronenzitadelle",
	"seuchenbeutel":                           "Eiskronenzitadelle", // legacy wrong DE alias
	"rotface":                                 "Eiskronenzitadelle",
	"professor seuchenmord":                   "Eiskronenzitadelle",
	"professor putricide":                     "Eiskronenzitadelle",
	"blutrat der blutfürsten":                 "Eiskronenzitadelle",
	"rat der blutprinzen":                     "Eiskronenzitadelle",
	"blood prince council":                    "Eiskronenzitadelle",
	"prinz valanar":                           "Eiskronenzitadelle",
	"prince valanar":                          "Eiskronenzitadelle",
	"prinz keleseth":                          "Eiskronenzitadelle",
	"prince keleseth":                         "Eiskronenzitadelle",
	"prinz taldaram":                          "Eiskronenzitadelle",
	"prince taldaram":                         "Eiskronenzitadelle",
	"steuerung der blutkugel":                 "Eiskronenzitadelle",
	"blood orb controller":                    "Eiskronenzitadelle",
	"blutkönigin lana'thel":                   "Eiskronenzitadelle",
	"blutprinzessin lana'thel":                "Eiskronenzitadelle", // legacy wrong DE alias
	"blood-queen lana'thel":                   "Eiskronenzitadelle",
	"valithria traumwandler":                  "Eiskronenzitadelle",
	"valithria dreamwalker":                   "Eiskronenzitadelle",
	"sindragosa":                              "Eiskronenzitadelle",
	"der lichkönig":                           "Eiskronenzitadelle",
	"the lich king":                           "Eiskronenzitadelle",

	// Rubinsanktum
	"halion": "Rubinsanktum",
}
