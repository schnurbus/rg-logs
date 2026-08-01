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

func normalizeBoss(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, "’", "'")
	s = strings.ReplaceAll(s, "`", "'")
	return s
}

// bossInstances maps normalized EN/DE boss names to display instance names (German).
var bossInstances = map[string]string{
	// Naxxramas
	"anub'rekhan":                 "Naxxramas",
	"großwitwe faerlina":          "Naxxramas",
	"grand widow faerlina":        "Naxxramas",
	"maexxna":                     "Naxxramas",
	"noth der seuchenfürst":       "Naxxramas",
	"noth the plaguebringer":      "Naxxramas",
	"heigan der unreine":          "Naxxramas",
	"heigan the unclean":          "Naxxramas",
	"loatheb":                     "Naxxramas",
	"instrukteur razuvious":       "Naxxramas",
	"instructor razuvious":        "Naxxramas",
	"gothik der ernter":           "Naxxramas",
	"gothik the harvester":        "Naxxramas",
	"die vier reiter":             "Naxxramas",
	"the four horsemen":           "Naxxramas",
	"thane korth'azz":             "Naxxramas",
	"sir zeliek":                  "Naxxramas",
	"lady blaumeux":               "Naxxramas",
	"baron rivendare":             "Naxxramas",
	"patchwerk":                   "Naxxramas",
	"grobbulus":                   "Naxxramas",
	"gluth":                       "Naxxramas",
	"thaddius":                    "Naxxramas",
	"sapphiron":                   "Naxxramas",
	"kel'thuzad":                  "Naxxramas",

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
	"lord mark'gar":                 "Eiskronenzitadelle",
	"lord marrowgar":                "Eiskronenzitadelle",
	"lady todeswisper":              "Eiskronenzitadelle",
	"lady deathwhisper":             "Eiskronenzitadelle",
	"kanonenschiffsschlacht von der eiskrone": "Eiskronenzitadelle",
	"icecrown gunship battle":       "Eiskronenzitadelle",
	"todbringer saurfang":           "Eiskronenzitadelle",
	"deathbringer saurfang":         "Eiskronenzitadelle",
	"fauldarm":                      "Eiskronenzitadelle",
	"festergut":                     "Eiskronenzitadelle",
	"seuchenbeutel":                 "Eiskronenzitadelle",
	"rotface":                       "Eiskronenzitadelle",
	"professor seuchenmord":         "Eiskronenzitadelle",
	"professor putricide":           "Eiskronenzitadelle",
	"blutrat der blutfürsten":       "Eiskronenzitadelle",
	"blood prince council":          "Eiskronenzitadelle",
	"prinz valanar":                 "Eiskronenzitadelle",
	"prince valanar":                "Eiskronenzitadelle",
	"prinz keleseth":                "Eiskronenzitadelle",
	"prince keleseth":               "Eiskronenzitadelle",
	"prinz taldaram":                "Eiskronenzitadelle",
	"prince taldaram":               "Eiskronenzitadelle",
	"blutprinzessin lana'thel":      "Eiskronenzitadelle",
	"blood-queen lana'thel":         "Eiskronenzitadelle",
	"valithria traumwandler":        "Eiskronenzitadelle",
	"valithria dreamwalker":         "Eiskronenzitadelle",
	"sindragosa":                    "Eiskronenzitadelle",
	"der lichkönig":                 "Eiskronenzitadelle",
	"the lich king":                 "Eiskronenzitadelle",

	// Rubinsanktum
	"halion": "Rubinsanktum",
}
