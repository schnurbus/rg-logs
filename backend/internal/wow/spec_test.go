package wow

import "testing"

func TestDetectSpec(t *testing.T) {
	tests := []struct {
		name   string
		spells map[int]int64
		want   Spec
	}{
		{name: "empty", spells: nil, want: ""},
		{
			name:   "mage fire",
			spells: map[int]int64{42833: 100, 42891: 80, 55360: 50},
			want:   SpecMageFire,
		},
		{
			name:   "warrior fury",
			spells: map[int]int64{23880: 200, 1680: 90, 44949: 50},
			want:   SpecWarriorFury,
		},
		{
			name:   "warrior protection",
			spells: map[int]int64{47488: 100, 47498: 80, 57823: 40},
			want:   SpecWarriorProtection,
		},
		{
			name:   "druid guardian vs feral",
			spells: map[int]int64{48480: 200, 48568: 150, 48562: 80},
			want:   SpecDruidGuardian,
		},
		{
			name:   "druid feral cat",
			spells: map[int]int64{48572: 200, 49800: 150, 48574: 80},
			want:   SpecDruidFeral,
		},
		{
			name:   "unknown only",
			spells: map[int]int64{1: 1000, 999999: 50},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSpec(tt.spells)
			if got != tt.want {
				t.Fatalf("DetectSpec() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRoleFromSpec(t *testing.T) {
	if RoleFromSpec(SpecWarriorProtection) != RoleTank {
		t.Fatal("prot warrior should be tank")
	}
	if RoleFromSpec(SpecPriestHoly) != RoleHealer {
		t.Fatal("holy priest should be healer")
	}
	if RoleFromSpec(SpecMageFire) != RoleDPS {
		t.Fatal("fire mage should be dps")
	}
	if RoleFromSpec("") != RoleDPS {
		t.Fatal("empty spec defaults to dps")
	}
}

func TestDetectSpecSampleRaid(t *testing.T) {
	cases := []struct {
		name   string
		want   Spec
		spells map[int]int64
	}{
		{"Azulogie", SpecMageFire, map[int]int64{42833: 60, 42891: 56, 55360: 54, 12654: 49}},
		{"Deaklot", SpecPaladinProtection, map[int]int64{48819: 199, 53739: 109, 53742: 64, 53595: 30}},
		{"Diedot", SpecWarlockDemonology, map[int]int64{54181: 180, 47893: 77, 47811: 67, 47813: 52, 47809: 40}},
		{"Floriniâ", SpecMageFrost, map[int]int64{42938: 106, 42926: 26, 42897: 25, 42845: 24}},
		{"Fofinho", SpecShamanElemental, map[int]int64{49233: 43, 49238: 36, 49240: 22, 49271: 11, 60043: 8}},
		{"Jartok", SpecHunterSurvival, map[int]int64{58433: 45, 75: 39, 53352: 26, 49001: 18, 63672: 16}},
		{"Lashback", SpecPriestHoly, map[int]int64{48076: 181, 48078: 137, 48068: 40, 56161: 18}},
		{"Muhximus", SpecWarriorFury, map[int]int64{12721: 135, 23880: 111, 47450: 44, 23881: 35}},
		{"Nhgaming", SpecRogueAssassination, map[int]int64{57965: 193, 57970: 49, 48676: 35, 48665: 27}},
		{"Osirris", SpecDeathKnightUnholy, map[int]int64{50475: 82, 50536: 76, 51460: 54, 55095: 33}},
		{"Sensenkarl", SpecDeathKnightUnholy, map[int]int64{50475: 325, 51460: 95, 50401: 83, 52212: 81}},
		{"Skriptxjk", SpecDruidBalance, map[int]int64{48461: 42, 48463: 32, 48465: 29, 71023: 27}},
		{"Stateless", SpecWarriorProtection, map[int]int64{12721: 140, 47498: 54, 47450: 41, 47488: 25}},
		{"Zycaria", SpecPaladinHoly, map[int]int64{20267: 837, 54968: 143, 48782: 48, 53652: 44}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSpec(tt.spells)
			if got != tt.want {
				t.Fatalf("DetectSpec() = %q, want %q", got, tt.want)
			}
		})
	}
}
