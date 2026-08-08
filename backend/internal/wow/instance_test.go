package wow

import "testing"

func TestDetectInstance(t *testing.T) {
	got := DetectInstance([]string{"Trash", "Lord Mark'gar", "Lady Todeswisper", "Sindragosa"})
	if got != "Eiskronenzitadelle" {
		t.Fatalf("got %q", got)
	}
	got = DetectInstance([]string{"Flame Leviathan", "Hodir", "Yogg-Saron"})
	if got != "Ulduar" {
		t.Fatalf("got %q", got)
	}
	got = DetectInstance([]string{"Zombie", "Ghoul"})
	if got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
	got = DetectInstance([]string{"Kanonenschiffsschlacht von der Eiskrone", "Rat der Blutprinzen"})
	if got != "Eiskronenzitadelle" {
		t.Fatalf("got %q", got)
	}
}

func TestIsKnownBoss(t *testing.T) {
	known := []string{
		"Sindragosa",
		"Flame Leviathan",
		"Todesbringer Saurfang",
		"Modermiene",
		"Blutkönigin Lana'thel",
		"Valithria Traumwandler",
		"Prinz Valanar",
		"Steuerung der Blutkugel",
		"Rat der Blutprinzen",
		"Marinesoldat der Himmelsbrecher",
		"Scharfschütze der Himmelsbrecher",
		"Muradin Bronzebart",
		"Kanonenschiffsschlacht von der Eiskrone",
	}
	for _, name := range known {
		if !IsKnownBoss(name) {
			t.Fatalf("expected %q known", name)
		}
	}
	if IsKnownBoss("Trash") {
		t.Fatal("Trash must not be a boss")
	}
	if IsKnownBoss("Zombie") {
		t.Fatal("Zombie must not be a boss")
	}
	if IsKnownBoss("") {
		t.Fatal("empty title must not be a boss")
	}
}

func TestFightTitle(t *testing.T) {
	if got := FightTitle("Prinz Valanar"); got != "Rat der Blutprinzen" {
		t.Fatalf("got %q", got)
	}
	if got := FightTitle("Marinesoldat der Himmelsbrecher"); got != "Kanonenschiffsschlacht von der Eiskrone" {
		t.Fatalf("got %q", got)
	}
	if got := FightTitle("Sindragosa"); got != "Sindragosa" {
		t.Fatalf("got %q", got)
	}
}

func TestEncounterID(t *testing.T) {
	if EncounterID("Prinz Valanar") != EncounterID("Prinz Taldaram") {
		t.Fatal("blood princes should share encounter")
	}
	if EncounterID("Todesbringer Saurfang") != "" {
		t.Fatal("Saurfang is standalone")
	}
	if EncounterID("Muradin Bronzebart") != encICCGunship {
		t.Fatal("Muradin is gunship")
	}
}

func TestGunshipDeathAndKillRules(t *testing.T) {
	if DeathEndsEncounter("Marinesoldat der Himmelsbrecher") {
		t.Fatal("gunship add death must not end encounter")
	}
	if DeathEndsEncounter("Prinz Valanar") {
		t.Fatal("blood council death must not end encounter")
	}
	if !DeathEndsEncounter("Sindragosa") {
		t.Fatal("single boss death should end encounter")
	}
	if UnitDeathCountsAsKill("Marinesoldat der Himmelsbrecher") {
		t.Fatal("gunship add death must not count as kill")
	}
	if !UnitDeathCountsAsKill("Prinz Valanar") {
		t.Fatal("blood prince death may count as kill")
	}
	if !IsGunshipEncounter("Muradin Bronzebart") {
		t.Fatal("Muradin should be gunship")
	}
}
