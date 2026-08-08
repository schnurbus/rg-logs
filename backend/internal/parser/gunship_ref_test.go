package parser_test

import (
	"os"
	"strings"
	"testing"

	"rg-logs/internal/parser"
)

func TestICCReferenceGunshipSingleSegment(t *testing.T) {
	path := "../../../references/combatlogs/Hogareth_Icc25_260802_WoWCombatLog.txt"
	f, err := os.Open(path)
	if err != nil {
		t.Skip(err)
	}
	defer f.Close()
	res, err := parser.Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	var gunship []*parser.FightResult
	for _, fight := range res.Fights {
		if strings.Contains(fight.Title, "Kanonenschiff") {
			gunship = append(gunship, fight)
			t.Logf("%s %s->%s dur=%ds kill=%v events=%d",
				fight.Title,
				fight.StartTs.Format("15:04:05"),
				fight.EndTs.Format("15:04:05"),
				fight.DurationMs/1000,
				fight.Kill,
				fight.EventCount,
			)
		}
	}
	if len(gunship) != 1 {
		t.Fatalf("expected 1 gunship segment, got %d", len(gunship))
	}
	if !gunship[0].Kill {
		t.Fatal("expected successful gunship kill")
	}
}
