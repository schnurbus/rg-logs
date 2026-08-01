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
