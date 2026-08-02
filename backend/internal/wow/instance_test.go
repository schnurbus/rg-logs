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
}

func TestIsKnownBoss(t *testing.T) {
	if !IsKnownBoss("Sindragosa") {
		t.Fatal("expected Sindragosa known")
	}
	if !IsKnownBoss("Flame Leviathan") {
		t.Fatal("expected Flame Leviathan known")
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
