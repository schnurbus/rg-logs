package parser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rg-logs/internal/parser"
)

func TestSplitAndAggregateSampleSnippet(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Auferstandener Zombie",0xa48,1035,405,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:17.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Auferstandener Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:18.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Auferstandener Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:19.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Auferstandener Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Auferstandener Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:21.240  SPELL_DAMAGE,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000B9,"Auferstandener Zombie",0xa48,42921,"Arkane Explosion",0x40,1320,690,64,0,0,0,nil,nil,nil`,
		`8/1 08:45:21.240  SPELL_DAMAGE,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BC,"Auferstandener Zombie",0xa48,42921,"Arkane Explosion",0x40,2450,1820,64,0,0,0,1,nil,nil`,
		`8/1 08:45:22.000  SPELL_DAMAGE,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BC,"Auferstandener Zombie",0xa48,42921,"Arkane Explosion",0x40,1,0,64,0,0,0,nil,nil,nil`,
		`8/1 08:45:23.000  SPELL_DAMAGE,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BC,"Auferstandener Zombie",0xa48,42921,"Arkane Explosion",0x40,1,0,64,0,0,0,nil,nil,nil`,
		`8/1 08:45:38.597  SPELL_HEAL,0x0000000000811FFF,"Lashback",0x512,0x00000000002A0928,"Deaklot",0x512,48076,"Heilige Nova",0x2,2418,0,0,nil`,
		`8/1 08:45:38.597  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130006C590000B9,"Auferstandener Zombie",0xa48`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 1 {
		t.Fatalf("expected 1 fight, got %d", len(res.Fights))
	}
	f := res.Fights[0]
	if f.Title != "Auferstandener Zombie" {
		t.Fatalf("title=%q", f.Title)
	}
	if !f.Kill {
		t.Fatal("expected kill=true")
	}

	var floriniDmg, deaklotDmg, lashbackHeal int64
	nameByGUID := map[string]string{}
	for _, a := range res.Actors {
		nameByGUID[a.GUID] = a.Name
	}
	for guid, agg := range f.Actors {
		switch nameByGUID[guid] {
		case "Floriniâ":
			floriniDmg = agg.DamageDone
		case "Deaklot":
			deaklotDmg = agg.DamageDone
		case "Lashback":
			lashbackHeal = agg.HealingDone
		}
	}
	if floriniDmg != 1320+2450+1+1 {
		t.Fatalf("florini damage=%d", floriniDmg)
	}
	if deaklotDmg != 1035+1+1+1+1 {
		t.Fatalf("deaklot damage=%d", deaklotDmg)
	}
	if lashbackHeal != 2418 {
		t.Fatalf("lashback heal=%d", lashbackHeal)
	}
}

func TestPetDamageStaysOnPetWithPlayerOwner(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SPELL_SUMMON,0x0000000000811FFF,"Lashback",0x512,0xF130004CD400035B,"Schattengeist",0x1112,34433,"Schattengeist",0x20`,
		`8/1 08:45:17.000  SPELL_DAMAGE,0xF130004CD400035B,"Schattengeist",0x1112,0xF130006C590000BA,"Zombie",0xa48,1,"Nahkampf",0x1,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:18.000  SPELL_DAMAGE,0xF130004CD400035B,"Schattengeist",0x1112,0xF130006C590000BA,"Zombie",0xa48,1,"Nahkampf",0x1,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:19.000  SPELL_DAMAGE,0xF130004CD400035B,"Schattengeist",0x1112,0xF130006C590000BA,"Zombie",0xa48,1,"Nahkampf",0x1,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:20.000  SPELL_DAMAGE,0xF130004CD400035B,"Schattengeist",0x1112,0xF130006C590000BA,"Zombie",0xa48,1,"Nahkampf",0x1,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:21.000  SPELL_DAMAGE,0xF130004CD400035B,"Schattengeist",0x1112,0xF130006C590000BA,"Zombie",0xa48,1,"Nahkampf",0x1,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:22.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,100,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:23.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,100,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:24.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,100,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:25.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,100,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:26.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,100,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:40.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130006C590000BA,"Zombie",0xa48`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 1 {
		t.Fatalf("expected 1 fight, got %d", len(res.Fights))
	}
	f := res.Fights[0]

	const (
		playerGUID = "0x0000000000811FFF"
		petGUID    = "0xF130004CD400035B"
	)

	var petOwner string
	for _, a := range res.Actors {
		if a.GUID == petGUID {
			petOwner = a.OwnerGUID
		}
	}
	if petOwner != playerGUID {
		t.Fatalf("pet owner=%q, want %q", petOwner, playerGUID)
	}

	petAgg := f.Actors[petGUID]
	if petAgg == nil {
		t.Fatal("expected pet actor agg")
	}
	if petAgg.DamageDone != 2500 {
		t.Fatalf("pet damage=%d, want 2500 (not remapped to owner)", petAgg.DamageDone)
	}
	playerAgg := f.Actors[playerGUID]
	if playerAgg == nil {
		t.Fatal("expected player actor agg")
	}
	if playerAgg.DamageDone != 500 {
		t.Fatalf("player own damage=%d, want 500", playerAgg.DamageDone)
	}
	if f.ParticipantCount != 1 {
		t.Fatalf("participantCount=%d, want 1 (player only, not pet)", f.ParticipantCount)
	}
}

func TestFightGapSegmentation(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:25.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:30.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:35.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:40.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:45.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:50.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:55.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:00.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:05.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		// 50s gap
		`8/1 08:46:55.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:00.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:05.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:10.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:15.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:25.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:30.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:35.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:40.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:45.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BB,"Ghoul",0xa48,200,0,1,0,0,0,nil,nil,nil`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 2 {
		t.Fatalf("expected 2 fights, got %d", len(res.Fights))
	}
	if res.Fights[0].Title != "Zombie" {
		t.Fatalf("fight0 title=%q", res.Fights[0].Title)
	}
	if res.Fights[1].Title != "Ghoul" {
		t.Fatalf("fight1 title=%q", res.Fights[1].Title)
	}
}

func TestOOCHealsDoNotBridgeFightGap(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:25.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:30.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:35.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:40.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:45.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:50.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:55.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:00.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:05.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130006C590000BA,"Zombie",0xa48`,
		// OOC HoTs that previously bridged the gap
		`8/1 08:46:20.000  SPELL_PERIODIC_HEAL,0x0000000000817847,"Diedot",0x514,0x0000000000817847,"Diedot",0x514,47893,"Teufelsrüstung",0x20,500,500,0,nil`,
		`8/1 08:46:40.000  SPELL_PERIODIC_HEAL,0x0000000000817847,"Diedot",0x514,0x0000000000817847,"Diedot",0x514,47893,"Teufelsrüstung",0x20,500,500,0,nil`,
		`8/1 08:47:00.000  SPELL_PERIODIC_HEAL,0x0000000000817847,"Diedot",0x514,0x0000000000817847,"Diedot",0x514,47893,"Teufelsrüstung",0x20,500,500,0,nil`,
		// Boss pull >45s after last hostile event
		`8/1 08:47:10.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:15.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:25.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:30.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:35.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:40.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:45.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:50.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:47:55.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:48:00.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130009621000002,"Toravon der Eiswächter",0xa48`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 2 {
		t.Fatalf("expected 2 fights, got %d", len(res.Fights))
	}
	if res.Fights[0].Title != "Zombie" {
		t.Fatalf("fight0 title=%q", res.Fights[0].Title)
	}
	if res.Fights[1].Title != "Toravon der Eiswächter" {
		t.Fatalf("fight1 title=%q", res.Fights[1].Title)
	}
}

func TestBossEngageSplitsTrashSegment(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:25.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:30.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:35.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:40.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:45.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:50.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:55.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:00.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Wärter des Archavon",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		// Immediate boss pull (no 45s gap)
		`8/1 08:46:05.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:10.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:15.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:25.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:30.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:35.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:40.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:45.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:50.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130009621000002,"Toravon der Eiswächter",0xa48,500,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:46:55.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130009621000002,"Toravon der Eiswächter",0xa48`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 2 {
		t.Fatalf("expected 2 fights, got %d", len(res.Fights))
	}
	if res.Fights[0].Title != "Wärter des Archavon" {
		t.Fatalf("fight0 title=%q", res.Fights[0].Title)
	}
	if res.Fights[1].Title != "Toravon der Eiswächter" {
		t.Fatalf("fight1 title=%q", res.Fights[1].Title)
	}
}

func TestParseReferenceLogSmoke(t *testing.T) {
	path := filepath.Join("..", "..", "..", "references", "combatlogs", "WoWCombatLog.txt")
	f, err := os.Open(path)
	if err != nil {
		t.Skip("reference log not available:", err)
	}
	defer f.Close()

	res, err := parser.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) < 2 {
		t.Fatalf("expected >=2 fights (trash + boss), got %d", len(res.Fights))
	}
	var toravon *parser.FightResult
	for _, fight := range res.Fights {
		if fight.Title == "Toravon der Eiswächter" {
			toravon = fight
		}
	}
	if toravon == nil {
		t.Fatal("expected Toravon fight")
	}
	// Real pull is ~1.5min; previously OOC heals merged trash into a ~6:46 fight.
	if toravon.DurationMs < 60_000 || toravon.DurationMs > 150_000 {
		t.Fatalf("toravon duration=%dms, want roughly 60-150s", toravon.DurationMs)
	}
	t.Logf("fights=%d actors=%d", len(res.Fights), len(res.Actors))
	for i, f := range res.Fights {
		t.Logf("fight[%d] title=%s dur=%dms events=%d participants=%d kill=%v",
			i, f.Title, f.DurationMs, f.EventCount, f.ParticipantCount, f.Kill)
	}
}

func TestSpellHitDetails(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,1000,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:17.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,800,0,1,0,0,0,nil,1,nil`,
		`8/1 08:45:18.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,2000,0,1,0,0,0,1,nil,nil`,
		`8/1 08:45:19.000  SWING_MISSED,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,MISS`,
		`8/1 08:45:20.000  SWING_MISSED,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,DODGE`,
		`8/1 08:45:21.000  SPELL_DAMAGE,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BA,"Zombie",0xa48,42921,"Arkane Explosion",0x40,1320,0,64,0,0,0,nil,nil,nil`,
		`8/1 08:45:22.000  SPELL_DAMAGE,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BA,"Zombie",0xa48,42921,"Arkane Explosion",0x40,2450,0,64,0,0,0,1,nil,nil`,
		`8/1 08:45:23.000  SPELL_MISSED,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BA,"Zombie",0xa48,42921,"Arkane Explosion",0x40,MISS`,
		`8/1 08:45:24.000  SPELL_MISSED,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BA,"Zombie",0xa48,42921,"Arkane Explosion",0x40,RESIST`,
		`8/1 08:45:25.000  SPELL_DAMAGE,0x000000000070F5A4,"Floriniâ",0x512,0xF130006C590000BA,"Zombie",0xa48,42921,"Arkane Explosion",0x40,1500,0,64,0,0,0,nil,nil,nil`,
		`8/1 08:45:40.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130006C590000BA,"Zombie",0xa48`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 1 {
		t.Fatalf("expected 1 fight, got %d", len(res.Fights))
	}
	f := res.Fights[0]

	const (
		deaklot  = "0x00000000002A0928"
		florinia = "0x000000000070F5A4"
	)

	var melee, ae *parser.SpellAgg
	for _, sp := range f.Spells {
		if sp.ActorGUID == deaklot && sp.SpellID == parser.SpellMeleeID && sp.Metric == parser.MetricDamage {
			melee = sp
		}
		if sp.ActorGUID == florinia && sp.SpellID == 42921 && sp.Metric == parser.MetricDamage {
			ae = sp
		}
	}

	if melee == nil {
		t.Fatal("expected melee spell agg")
	}
	if melee.Hits != 3 || melee.Crits != 1 || melee.Glancing != 1 || melee.Misses != 1 {
		t.Fatalf("melee hits=%d crits=%d glancing=%d misses=%d", melee.Hits, melee.Crits, melee.Glancing, melee.Misses)
	}
	if melee.Total != 3800 {
		t.Fatalf("melee total=%d", melee.Total)
	}
	if melee.Normal.Hits != 1 || melee.Normal.Min != 1000 || melee.Normal.Max != 1000 {
		t.Fatalf("melee normal=%+v", melee.Normal)
	}
	if melee.Glance.Hits != 1 || melee.Glance.Min != 800 || melee.Glance.Max != 800 {
		t.Fatalf("melee glance=%+v", melee.Glance)
	}
	if melee.Crit.Hits != 1 || melee.Crit.Min != 2000 || melee.Crit.Max != 2000 {
		t.Fatalf("melee crit=%+v", melee.Crit)
	}

	if ae == nil {
		t.Fatal("expected AE spell agg")
	}
	if ae.Hits != 3 || ae.Crits != 1 || ae.Misses != 1 || ae.Glancing != 0 {
		t.Fatalf("ae hits=%d crits=%d misses=%d glancing=%d", ae.Hits, ae.Crits, ae.Misses, ae.Glancing)
	}
	if ae.Total != 1320+2450+1500 {
		t.Fatalf("ae total=%d", ae.Total)
	}
	if ae.Normal.Hits != 2 || ae.Normal.Min != 1320 || ae.Normal.Max != 1500 {
		t.Fatalf("ae normal=%+v", ae.Normal)
	}
	if ae.Crit.Hits != 1 || ae.Crit.Min != 2450 || ae.Crit.Max != 2450 {
		t.Fatalf("ae crit=%+v", ae.Crit)
	}
}
