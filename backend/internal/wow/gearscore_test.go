package wow

import "testing"

func TestItemScoreEpicHead232(t *testing.T) {
	// Hellscream's Helmet of Conquest: ilvl 232, epic, head
	got := ItemScore(EquippedItem{ItemLevel: 232, Quality: 4, SlotBak: 1})
	// floor(((232-91.45)/0.65)*1.0*1.8618) = floor(216.15*1.8618/… wait:
	// ((232 - 91.45) / 0.65) * 1 * 1.8618 = 216.230769… * 1.8618 ≈ 402.57 → 402
	if got != 402 {
		t.Fatalf("got %d want 402", got)
	}
}

func TestItemScoreNeckLowerMod(t *testing.T) {
	got := ItemScore(EquippedItem{ItemLevel: 232, Quality: 4, SlotBak: 2})
	// same base * 0.5625 → floor(402.57 * 0.5625) via formula:
	// ((232-91.45)/0.65)*0.5625*1.8618 = floor(226.4…) → 226
	if got != 226 {
		t.Fatalf("got %d want 226", got)
	}
}

func TestItemScoreTwoHand(t *testing.T) {
	got := ItemScore(EquippedItem{ItemLevel: 232, Quality: 4, SlotBak: 17})
	// * 2.0 → ~805
	if got != 805 {
		t.Fatalf("got %d want 805", got)
	}
}

func TestGearScoreBasic(t *testing.T) {
	items := map[int]EquippedItem{
		1:  {ItemLevel: 232, Quality: 4, SlotBak: 1},
		2:  {ItemLevel: 232, Quality: 4, SlotBak: 2},
		16: {ItemLevel: 232, Quality: 4, SlotBak: 13}, // 1H
		17: {ItemLevel: 232, Quality: 4, SlotBak: 14}, // shield
	}
	got := GearScore(items, ClassWarrior)
	want := ItemScore(items[1]) + ItemScore(items[2]) + ItemScore(items[16]) + ItemScore(items[17])
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestGearScoreTitanGrip(t *testing.T) {
	items := map[int]EquippedItem{
		16: {ItemLevel: 264, Quality: 4, SlotBak: 17}, // 2H MH
		17: {ItemLevel: 264, Quality: 4, SlotBak: 17}, // 2H OH (TG)
	}
	got := GearScore(items, ClassWarrior)
	mh := float64(ItemScore(items[16])) * 0.5
	oh := float64(ItemScore(items[17])) * 0.5
	want := int(mh + oh)
	if got != want {
		t.Fatalf("got %d want %d", got, want)
	}
}

func TestGearScoreHunterRangedBoost(t *testing.T) {
	items := map[int]EquippedItem{
		16: {ItemLevel: 232, Quality: 4, SlotBak: 13},
		18: {ItemLevel: 232, Quality: 4, SlotBak: 15},
	}
	warrior := GearScore(items, ClassWarrior)
	hunter := GearScore(items, ClassHunter)
	if hunter <= warrior {
		t.Fatalf("hunter GS %d should exceed warrior %d due to ranged weight", hunter, warrior)
	}
}

func TestClassFromWoWClassID(t *testing.T) {
	if ClassFromWoWClassID(3) != ClassHunter {
		t.Fatal("expected hunter")
	}
	if ClassFromWoWClassID(1) != ClassWarrior {
		t.Fatal("expected warrior")
	}
}
