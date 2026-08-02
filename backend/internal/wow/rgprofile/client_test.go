package rgprofile

import (
	"testing"

	"rg-logs/internal/wow"
)

func TestNormalizeName(t *testing.T) {
	if got := NormalizeName("Muhximus"); got != "muhximus" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeName("Muhximus-RisingGods"); got != "muhximus" {
		t.Fatalf("got %q", got)
	}
}

func TestParseLoadPayload(t *testing.T) {
	body := []byte(`
g_items.add(48388, {"name_enus":"Helm","quality":4,"classs":4,"jsonequip":{"level":232,"slotbak":1,"slot":1,"gearscore":161.3}});
g_items.add(50228, {"name_enus":"Neck","quality":4,"classs":4,"jsonequip":{"level":232,"slotbak":2,"slot":2,"gearscore":108.8}});
g_items.add(42334, {"name_enus":"Sword","quality":4,"classs":2,"jsonequip":{"level":245,"slotbak":13,"slot":13,"gearscore":200}});
WowheadProfiler.registerProfile({"id":1,"classs":1,"name":"Muhximus","inventory":{"1":[48388,0,0,0,0,0,0,0,0],"2":[50228,0,0,0,0,0,0,0,0],"16":[42334,0,0,0,0,0,0,0,0]}});
`)
	ch, err := parseLoadPayload(body)
	if err != nil {
		t.Fatal(err)
	}
	if ch == nil {
		t.Fatal("expected character")
	}
	if ch.Class != wow.ClassWarrior {
		t.Fatalf("class %q (must use profile classs, not item classs)", ch.Class)
	}
	if len(ch.Inventory) != 3 {
		t.Fatalf("inventory size %d", len(ch.Inventory))
	}
	if ch.Inventory[1].ItemLevel != 232 || ch.Inventory[1].SlotBak != 1 {
		t.Fatalf("head %+v", ch.Inventory[1])
	}
	gs := wow.GearScore(ch.Inventory, ch.Class)
	if gs <= 0 {
		t.Fatalf("expected positive gearscore, got %d", gs)
	}
}

func TestParseLoadPayloadMissing(t *testing.T) {
	ch, err := parseLoadPayload([]byte("nothing here"))
	if err != nil {
		t.Fatal(err)
	}
	if ch != nil {
		t.Fatal("expected nil")
	}
}
