package parser_test

import (
	"strings"
	"testing"

	"rg-logs/internal/parser"
)

func TestCombatEventsDamageHealMissDeath(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,1035,405,1,10,20,30,nil,nil,nil`,
		`8/1 08:45:17.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:18.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:19.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,1,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:21.000  SPELL_MISSED,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,47486,"Mortal Strike",0x1,DODGE`,
		`8/1 08:45:22.000  SPELL_HEAL,0x0000000000811FFF,"Lashback",0x512,0x00000000002A0928,"Deaklot",0x512,48076,"Heilige Nova",0x2,2418,100,5,nil`,
		`8/1 08:45:23.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,50,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:24.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,50,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:25.000  SPELL_DAMAGE,0x0000000000811FFF,"Lashback",0x512,0xF130006C590000BA,"Zombie",0xa48,585,"Smite",0x2,50,0,2,0,0,0,nil,nil,nil`,
		`8/1 08:45:26.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0x00000000002A0928,"Deaklot",0x512`,
		`8/1 08:45:27.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130006C590000BA,"Zombie",0xa48`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 1 {
		t.Fatalf("fights=%d", len(res.Fights))
	}
	f := res.Fights[0]

	var damage, heal, miss, death int
	var firstDamage *parser.CombatEvent
	for i := range f.Events {
		ev := &f.Events[i]
		switch ev.EventType {
		case parser.EventDamage:
			damage++
			if firstDamage == nil {
				firstDamage = ev
			}
		case parser.EventHeal:
			heal++
			if ev.Overheal != 100 || ev.Absorbed != 5 || ev.Amount != 2418 {
				t.Fatalf("heal event=%+v", ev)
			}
		case parser.EventMiss:
			miss++
			if ev.MissType == nil || *ev.MissType != parser.MissTypeDodge {
				t.Fatalf("miss type=%v", ev.MissType)
			}
			if ev.SpellID != 47486 {
				t.Fatalf("miss spell=%d", ev.SpellID)
			}
		case parser.EventDeath:
			death++
		}
	}
	if damage < 5 {
		t.Fatalf("damage events=%d", damage)
	}
	if heal != 1 || miss != 1 || death != 2 {
		t.Fatalf("heal=%d miss=%d death=%d", heal, miss, death)
	}
	if firstDamage == nil || firstDamage.Overkill != 405 || firstDamage.Resisted != 10 ||
		firstDamage.Blocked != 20 || firstDamage.Absorbed != 30 {
		t.Fatalf("first damage=%+v", firstDamage)
	}

	foundMelee := false
	foundMS := false
	for _, ab := range res.Abilities {
		if ab.SpellID == parser.SpellMeleeID && ab.Name == parser.SpellMeleeName {
			foundMelee = true
		}
		if ab.SpellID == 47486 && ab.Name == "Mortal Strike" {
			foundMS = true
		}
	}
	if !foundMelee || !foundMS {
		t.Fatalf("abilities missing melee=%v mortal=%v (n=%d)", foundMelee, foundMS, len(res.Abilities))
	}
}

func TestCombatEventsAuraInterruptDispel(t *testing.T) {
	snippet := strings.Join([]string{
		`8/1 08:45:16.653  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:17.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:18.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:19.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:20.000  SWING_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,100,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:21.000  SPELL_AURA_APPLIED,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,47465,"Rend",0x1,DEBUFF`,
		`8/1 08:45:21.500  SPELL_AURA_APPLIED_DOSE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,47465,"Rend",0x1,DEBUFF,3`,
		`8/1 08:45:22.000  SPELL_INTERRUPT,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,72,"Shield Bash",0x1,34063,"Lich Slap",0x20`,
		`8/1 08:45:23.000  SPELL_DISPEL,0x0000000000811FFF,"Lashback",0x512,0x00000000002A0928,"Deaklot",0x512,988,"Dispel Magic",0x2,47865,"Curse of Agony",0x20,DEBUFF`,
		`8/1 08:45:24.000  SPELL_AURA_REMOVED,0xF130006C590000BA,"Zombie",0xa48,0x00000000002A0928,"Deaklot",0x512,47465,"Rend",0x1,DEBUFF`,
		`8/1 08:45:25.000  SPELL_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,47450,"Heroic Strike",0x1,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:26.000  SPELL_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,47450,"Heroic Strike",0x1,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:27.000  SPELL_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,47450,"Heroic Strike",0x1,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:27.500  SPELL_DAMAGE,0x00000000002A0928,"Deaklot",0x512,0xF130006C590000BA,"Zombie",0xa48,47450,"Heroic Strike",0x1,200,0,1,0,0,0,nil,nil,nil`,
		`8/1 08:45:28.000  UNIT_DIED,0x0000000000000000,nil,0x80000000,0xF130006C590000BA,"Zombie",0xa48`,
	}, "\n")

	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 1 {
		t.Fatalf("fights=%d", len(res.Fights))
	}
	f := res.Fights[0]

	var auraApplied, auraRemoved, interrupt, dispel int
	for _, ev := range f.Events {
		switch ev.EventType {
		case parser.EventAuraApplied:
			auraApplied++
			if ev.Flags&parser.EventFlagAuraDebuff == 0 {
				t.Fatalf("aura applied missing debuff flag: %+v", ev)
			}
			if ev.SpellID == 47465 && ev.Extra == 3 && auraApplied == 2 {
				// dose event
			}
		case parser.EventAuraRemoved:
			auraRemoved++
		case parser.EventInterrupt:
			interrupt++
			if ev.SpellID != 72 || ev.Extra != 34063 {
				t.Fatalf("interrupt=%+v", ev)
			}
		case parser.EventDispel:
			dispel++
			if ev.SpellID != 988 || ev.Extra != 47865 {
				t.Fatalf("dispel=%+v", ev)
			}
			if ev.Flags&parser.EventFlagAuraDebuff == 0 {
				t.Fatalf("dispel missing debuff flag")
			}
		}
	}
	if auraApplied != 2 || auraRemoved != 1 || interrupt != 1 || dispel != 1 {
		t.Fatalf("auraApp=%d auraRem=%d interrupt=%d dispel=%d", auraApplied, auraRemoved, interrupt, dispel)
	}
}

func TestAuraOutsideFightIgnored(t *testing.T) {
	snippet := `8/1 08:45:16.653  SPELL_AURA_APPLIED,0x00000000002A0928,"Deaklot",0x512,0x00000000002A0928,"Deaklot",0x512,47440,"Commanding Shout",0x1,BUFF`
	res, err := parser.Parse(strings.NewReader(snippet))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Fights) != 0 {
		t.Fatalf("expected no fights, got %d", len(res.Fights))
	}
}
