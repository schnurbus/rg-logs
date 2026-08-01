package wow

import "testing"

func TestDetectClass(t *testing.T) {
	tests := []struct {
		name   string
		spells map[int]int64
		want   Class
	}{
		{
			name:   "empty",
			spells: nil,
			want:   "",
		},
		{
			name:   "mage fire",
			spells: map[int]int64{42833: 100, 42891: 80, 55360: 50},
			want:   ClassMage,
		},
		{
			name:   "warrior bloodthirst",
			spells: map[int]int64{23880: 200, 47450: 90, 12721: 150},
			want:   ClassWarrior,
		},
		{
			name:   "paladin consecration wins over shared noise",
			spells: map[int]int64{48819: 500, 53739: 200, 1: 999},
			want:   ClassPaladin,
		},
		{
			name:   "unknown only",
			spells: map[int]int64{1: 1000, 999999: 50},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectClass(tt.spells)
			if got != tt.want {
				t.Fatalf("DetectClass() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDetectClassSampleRaid(t *testing.T) {
	cases := []struct {
		name   string
		want   Class
		spells map[int]int64
	}{
		{"Azulogie", ClassMage, map[int]int64{42833: 60, 42891: 56, 55360: 54, 12654: 49}},
		{"Deaklot", ClassPaladin, map[int]int64{48819: 199, 53739: 109, 53742: 64, 53595: 30}},
		{"Diedot", ClassWarlock, map[int]int64{54181: 180, 47893: 77, 47811: 67, 47813: 52, 47809: 40}},
		{"Floriniâ", ClassMage, map[int]int64{42938: 106, 42926: 26, 42897: 25, 42845: 24}},
		{"Fofinho", ClassShaman, map[int]int64{49233: 43, 49238: 36, 49240: 22, 49271: 11, 60043: 8}},
		{"Jartok", ClassHunter, map[int]int64{58433: 45, 75: 39, 53352: 26, 49001: 18, 63672: 16}},
		{"Lashback", ClassPriest, map[int]int64{48076: 181, 48078: 137, 48068: 40, 56161: 18}},
		{"Muhximus", ClassWarrior, map[int]int64{12721: 135, 23880: 111, 47450: 44, 23881: 35}},
		{"Nhgaming", ClassRogue, map[int]int64{57965: 193, 57970: 49, 48676: 35, 48665: 27}},
		{"Osirris", ClassDeathKnight, map[int]int64{50475: 82, 50536: 76, 51460: 54, 55095: 33}},
		{"Ruskodk", ClassDeathKnight, map[int]int64{55095: 25, 49909: 20, 55078: 19, 49921: 12}},
		{"Sensenkarl", ClassDeathKnight, map[int]int64{50475: 325, 51460: 95, 50401: 83, 52212: 81}},
		{"Skriptxjk", ClassDruid, map[int]int64{48461: 42, 48463: 32, 48465: 29, 71023: 27}},
		{"Stateless", ClassWarrior, map[int]int64{12721: 140, 47498: 54, 47450: 41, 47488: 25}},
		{"Zycaria", ClassPaladin, map[int]int64{20267: 837, 54968: 143, 48782: 48, 53652: 44}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectClass(tt.spells)
			if got != tt.want {
				t.Fatalf("DetectClass() = %q, want %q", got, tt.want)
			}
		})
	}
}
