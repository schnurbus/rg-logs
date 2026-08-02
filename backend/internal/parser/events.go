package parser

import "time"

// Combat event_type values persisted in combat_events.event_type.
const (
	EventDamage      int16 = 1
	EventHeal        int16 = 2
	EventMiss        int16 = 3
	EventDeath       int16 = 4
	EventSummon      int16 = 5
	EventAuraApplied int16 = 6
	EventAuraRemoved int16 = 7
	EventAuraRefresh int16 = 8
	EventInterrupt   int16 = 9
	EventDispel      int16 = 10
)

// Event flag bits persisted in combat_events.flags.
const (
	EventFlagCrit       = 1 << 0
	EventFlagGlancing   = 1 << 1
	EventFlagCrushing   = 1 << 2
	EventFlagPeriodic   = 1 << 3
	EventFlagAuraBuff   = 1 << 4
	EventFlagAuraDebuff = 1 << 5
)

// Miss type values persisted in combat_events.miss_type.
const (
	MissTypeMiss    int16 = 1
	MissTypeDodge   int16 = 2
	MissTypeParry   int16 = 3
	MissTypeBlock   int16 = 4
	MissTypeEvade   int16 = 5
	MissTypeImmune  int16 = 6
	MissTypeDeflect int16 = 7
	MissTypeAbsorb  int16 = 8
	MissTypeReflect int16 = 9
	MissTypeResist  int16 = 10
)

// AbilityInfo is a spell dictionary entry for an upload.
type AbilityInfo struct {
	SpellID int
	Name    string
	School  int
}

// CombatEvent is one normalized combat-log row (no repeated name strings).
type CombatEvent struct {
	Ts         time.Time
	OffsetMs   int
	EventType  int16
	SourceGUID string
	TargetGUID string
	SpellID    int
	Amount     int
	Overkill   int
	Overheal   int
	Absorbed   int
	Resisted   int
	Blocked    int
	Flags      int
	MissType   *int16
	Extra      int
}
