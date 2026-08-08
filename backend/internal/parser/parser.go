package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"rg-logs/internal/wow"
)

const (
	FightGap          = 45 * time.Second
	GunshipFightGap   = 3 * time.Minute // cannon/portal lulls between boarding waves
	MinFightDur       = 5 * time.Second
	MinFightEvents    = 10
	FlagPlayer        = 0x400
	FlagNPC           = 0x800
	FlagPet           = 0x1000
	FlagGuardian      = 0x2000
	SpellMeleeID      = 1
	SpellMeleeName    = "Nahkampf"
)

// Spells that prove a player↔pet ownership link when SUMMON is missing
// (pet already active when combat logging started).
var petOwnerLinkSpells = map[int]struct{}{
	25228: {}, // Soul Link
	35696: {}, // Demonic Knowledge
	35706: {}, // Master Demonologist
	54181: {}, // Fel Synergy
}

type Metric string

const (
	MetricDamage      Metric = "damage"
	MetricHealing     Metric = "healing"
	MetricDamageTaken Metric = "damage_taken"
)

type ActorInfo struct {
	GUID      string
	Name      string
	Flags     int64
	IsPlayer  bool
	OwnerGUID string
}

type HitSpectrum struct {
	Hits  int
	Total int64
	Min   int64
	Max   int64
}

func (h *HitSpectrum) add(amount int64) {
	if h.Hits == 0 {
		h.Min = amount
		h.Max = amount
	} else {
		if amount < h.Min {
			h.Min = amount
		}
		if amount > h.Max {
			h.Max = amount
		}
	}
	h.Total += amount
	h.Hits++
}

type SpellAgg struct {
	ActorGUID string
	SpellID   int
	SpellName string
	School    int
	Metric    Metric
	Total     int64
	Hits      int
	Crits     int
	Ticks     int
	Misses    int
	Glancing  int
	Normal    HitSpectrum
	Crit      HitSpectrum
	Glance    HitSpectrum
}

type ActorAgg struct {
	GUID        string
	DamageDone  int64
	HealingDone int64
	Overheal    int64
	DamageTaken int64
	FirstEvent  time.Time
	LastEvent   time.Time
	Seen        bool
}

type FightResult struct {
	ID               uuid.UUID
	StartTs          time.Time
	EndTs            time.Time
	DurationMs       int64
	Title            string
	Kill             bool
	ParticipantCount int
	Actors           map[string]*ActorAgg
	Spells           map[spellKey]*SpellAgg
	Events           []CombatEvent
	EnemyDamageTaken map[string]int64 // enemy name -> damage taken (for title)
	BossHealingTaken map[string]int64 // friendly/heal-boss name -> healing taken (Valithria)
	EnemyDied        map[string]bool
	EncounterSuccess bool // set by encounter-specific success spells (e.g. Valithria)
	EventCount       int
}

type spellKey struct {
	ActorGUID string
	SpellID   int
	Metric    Metric
}

type ParseResult struct {
	Actors    []*ActorInfo
	Abilities []*AbilityInfo
	Fights    []*FightResult
}

type Parser struct {
	actors     map[string]*ActorInfo // guid -> actor
	abilities  map[int]*AbilityInfo  // spell_id -> ability
	petOwner   map[string]string     // pet guid -> owner guid
	year       int
	cur        *FightResult
	fights     []*FightResult
	lastCombat time.Time
	hasCombat  bool
}

func New() *Parser {
	return &Parser{
		actors:    make(map[string]*ActorInfo),
		abilities: make(map[int]*AbilityInfo),
		petOwner:  make(map[string]string),
		year:      time.Now().Year(),
	}
}

func Parse(r io.Reader) (*ParseResult, error) {
	p := New()
	if err := p.Parse(r); err != nil {
		return nil, err
	}
	return p.Result(), nil
}

func (p *Parser) Result() *ParseResult {
	p.finalizeOwners()
	actors := make([]*ActorInfo, 0, len(p.actors))
	for _, a := range p.actors {
		actors = append(actors, a)
	}
	abilities := make([]*AbilityInfo, 0, len(p.abilities))
	for _, a := range p.abilities {
		abilities = append(abilities, a)
	}
	return &ParseResult{Actors: actors, Abilities: abilities, Fights: p.fights}
}

func (p *Parser) Parse(r io.Reader) error {
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			continue
		}
		if err := p.handleLine(line); err != nil {
			// skip malformed lines
			continue
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	p.closeFight()
	return nil
}

func (p *Parser) handleLine(line string) error {
	ts, event, fields, err := splitLine(line)
	if err != nil {
		return err
	}
	ts = time.Date(p.year, ts.Month(), ts.Day(), ts.Hour(), ts.Minute(), ts.Second(), ts.Nanosecond(), time.UTC)

	switch event {
	case "SPELL_SUMMON":
		p.handleSummon(ts, fields)
		return nil
	}

	// Aura / interrupt / dispel: record only inside an open fight (do not start fights).
	// Pet-owner link auras are still processed out of combat.
	if isSupportEvent(event) {
		switch event {
		case "SPELL_AURA_APPLIED", "SPELL_AURA_APPLIED_DOSE":
			p.handleAura(ts, fields, EventAuraApplied)
		case "SPELL_AURA_REMOVED", "SPELL_AURA_REMOVED_DOSE":
			p.handleAura(ts, fields, EventAuraRemoved)
		case "SPELL_AURA_REFRESH":
			p.handleAura(ts, fields, EventAuraRefresh)
		case "SPELL_INTERRUPT":
			if p.cur == nil {
				return nil
			}
			p.cur.EndTs = ts
			p.handleInterrupt(ts, fields)
		case "SPELL_DISPEL", "SPELL_STOLEN", "SPELL_DISPEL_FAILED":
			if p.cur == nil {
				return nil
			}
			p.cur.EndTs = ts
			p.handleDispel(ts, fields)
		}
		return nil
	}

	combat := isCombatEvent(event)
	if !combat {
		return nil
	}

	// Ambient NPC-vs-NPC (gunship ships keep fighting after the raid leaves) must not
	// open fights, bridge gaps, or pollute boss damage taken / titles.
	if isPureNPCCombat(event, fields) {
		return nil
	}

	hostile := isHostileEngagement(event, fields)

	gap := FightGap
	if p.cur != nil && fightHasGunship(p.cur) {
		gap = GunshipFightGap
	}
	if p.hasCombat && ts.Sub(p.lastCombat) >= gap {
		p.closeFight()
	}

	// Engaging a known boss closes trash or a different encounter.
	if hostile && p.cur != nil {
		if name := hostileBossName(event, fields); name != "" {
			p.maybeSplitForBoss(name)
		}
	}

	if p.cur == nil {
		// Friendly heals / OOC ticks must not open a fight.
		// Orphan UNIT_DIED/PARTY_KILL (e.g. after a boss already closed) must not
		// open a hollow segment that later absorbs the next pull.
		if !hostile || event == "UNIT_DIED" || event == "PARTY_KILL" {
			return nil
		}
		p.cur = newFight(ts)
	}

	if hostile {
		p.lastCombat = ts
		p.hasCombat = true
		p.cur.EndTs = ts
	} else if p.hasCombat && ts.Sub(p.lastCombat) < FightGap {
		// Heals during an active pull may extend the window; OOC ticks after the gap do not.
		p.cur.EndTs = ts
	}
	p.cur.EventCount++

	switch event {
	case "SWING_DAMAGE":
		p.handleSwingDamage(ts, fields, false)
	case "SPELL_DAMAGE", "RANGE_DAMAGE", "DAMAGE_SHIELD":
		p.handleSpellDamage(ts, fields, false)
	case "SPELL_PERIODIC_DAMAGE":
		p.handleSpellDamage(ts, fields, true)
	case "SWING_MISSED":
		p.handleSwingMissed(ts, fields)
	case "SPELL_MISSED", "RANGE_MISSED", "SPELL_PERIODIC_MISSED":
		p.handleSpellMissed(ts, fields)
	case "SPELL_HEAL":
		p.handleHeal(ts, fields, false)
	case "SPELL_PERIODIC_HEAL":
		p.handleHeal(ts, fields, true)
	case "UNIT_DIED", "PARTY_KILL":
		p.handleDeath(ts, fields)
		// End single-boss encounters on boss death so trailing OOC heals don't inflate duration.
		// Multi-NPC encounters (gunship, blood council, …) must not close on add/prince deaths.
		if name := hostileBossName(event, fields); name != "" && p.cur != nil && fightHasKnownBoss(p.cur) {
			if wow.DeathEndsEncounter(name) {
				p.closeFight()
			}
		}
	}
	return nil
}

func newFight(start time.Time) *FightResult {
	return &FightResult{
		ID:               uuid.New(),
		StartTs:          start,
		EndTs:            start,
		Actors:           make(map[string]*ActorAgg),
		Spells:           make(map[spellKey]*SpellAgg),
		Events:           make([]CombatEvent, 0, 256),
		EnemyDamageTaken: make(map[string]int64),
		BossHealingTaken: make(map[string]int64),
		EnemyDied:        make(map[string]bool),
	}
}

// maybeSplitForBoss closes the current segment when a new known boss from another
// encounter (or after trash) appears.
func (p *Parser) maybeSplitForBoss(bossName string) {
	if p.cur == nil || bossName == "" {
		return
	}
	if fightHasNPC(p.cur, bossName) {
		return
	}
	if !fightHasKnownBoss(p.cur) {
		p.closeFight()
		return
	}
	if !fightSharesEncounter(p.cur, bossName) {
		p.closeFight()
	}
}

func (p *Parser) closeFight() {
	if p.cur == nil {
		return
	}
	f := p.cur
	p.cur = nil

	f.DurationMs = f.EndTs.Sub(f.StartTs).Milliseconds()
	if f.DurationMs < MinFightDur.Milliseconds() || f.EventCount < MinFightEvents {
		return
	}

	f.Title = pickTitle(f)
	f.Kill = f.EnemyDied[f.Title] || f.EncounterSuccess
	// Multi-NPC encounters: constituent boss deaths may count as a kill (not gunship adds).
	if !f.Kill {
		for name := range f.EnemyDied {
			if wow.UnitDeathCountsAsKill(name) && wow.FightTitle(name) == f.Title {
				f.Kill = true
				break
			}
		}
	}

	count := 0
	for guid, agg := range f.Actors {
		if agg.DamageDone == 0 && agg.HealingDone == 0 && agg.DamageTaken == 0 {
			continue
		}
		info := p.actors[guid]
		if info == nil || !info.IsPlayer {
			continue
		}
		count++
	}
	f.ParticipantCount = count
	p.fights = append(p.fights, f)
}

func pickTitle(f *FightResult) string {
	var bestAny string
	var bestAnyDmg int64
	var bestBoss string
	var bestBossDmg int64
	for name, dmg := range f.EnemyDamageTaken {
		if dmg > bestAnyDmg {
			bestAnyDmg = dmg
			bestAny = name
		}
		if wow.IsKnownBoss(name) && dmg > bestBossDmg {
			bestBossDmg = dmg
			bestBoss = name
		}
	}

	var bestHealBoss string
	var bestHeal int64
	for name, heal := range f.BossHealingTaken {
		// Only Valithria-style heal encounters may claim the title via healing.
		if !wow.IsHealEncounterBoss(name) {
			continue
		}
		if heal > bestHeal {
			bestHeal = heal
			bestHealBoss = name
		}
	}
	// Heal encounters (Valithria): any meaningful healing onto the boss claims the title.
	// Late encounter spells can dump huge damage onto portals/adds and must not rename it.
	const minHealBossClaim int64 = 100000
	if bestHealBoss != "" && bestHeal >= minHealBossClaim {
		return wow.FightTitle(bestHealBoss)
	}
	// Prefer a known boss only when it took a meaningful share of damage.
	// Avoid leftover encounter NPCs renaming long trash segments.
	if bestBoss != "" && bestBossDmg >= bestAnyDmg/4 {
		return wow.FightTitle(bestBoss)
	}
	if bestAny != "" {
		return bestAny
	}
	return "Trash"
}

func (p *Parser) ensureActor(guid, name string, flags int64) *ActorInfo {
	if guid == "" || guid == "0x0000000000000000" || name == "" || name == "nil" {
		return nil
	}
	a, ok := p.actors[guid]
	if !ok {
		a = &ActorInfo{
			GUID:     guid,
			Name:     unquote(name),
			Flags:    flags,
			IsPlayer: (flags & FlagPlayer) != 0,
		}
		if owner, ok := p.petOwner[guid]; ok {
			a.OwnerGUID = owner
		}
		p.actors[guid] = a
	} else {
		if name != "" && name != "nil" {
			a.Name = unquote(name)
		}
		if flags != 0 {
			a.Flags = flags
			a.IsPlayer = (flags & FlagPlayer) != 0
		}
		if a.OwnerGUID == "" {
			if owner, ok := p.petOwner[guid]; ok {
				a.OwnerGUID = owner
			}
		}
	}
	return a
}

func isPlayerControlledPet(flags int64) bool {
	return (flags&FlagPet) != 0 || (flags&FlagGuardian) != 0
}

// notePetOwnerLink records owner when a known pet-bond spell fires between
// a player and a pet/guardian (covers pets summoned before the log starts).
func (p *Parser) notePetOwnerLink(spellID int, srcGUID string, srcFlags int64, dstGUID string, dstFlags int64) {
	if _, ok := petOwnerLinkSpells[spellID]; !ok {
		return
	}
	srcPet := isPlayerControlledPet(srcFlags)
	dstPet := isPlayerControlledPet(dstFlags)
	srcPlayer := (srcFlags & FlagPlayer) != 0
	dstPlayer := (dstFlags & FlagPlayer) != 0

	var petGUID, ownerGUID string
	switch {
	case srcPet && dstPlayer:
		petGUID, ownerGUID = srcGUID, dstGUID
	case dstPet && srcPlayer:
		petGUID, ownerGUID = dstGUID, srcGUID
	default:
		return
	}
	if petGUID == "" || ownerGUID == "" {
		return
	}
	if existing, ok := p.petOwner[petGUID]; ok && existing != "" {
		return
	}
	p.petOwner[petGUID] = ownerGUID
	if a := p.actors[petGUID]; a != nil && a.OwnerGUID == "" {
		a.OwnerGUID = ownerGUID
	}
}

func (p *Parser) resolveSource(guid, name string, flags int64) string {
	a := p.ensureActor(guid, name, flags)
	if a == nil {
		return ""
	}
	// Keep pet/guardian stats on the pet itself; UI rolls them into the
	// owning player. Owner mapping comes from SPELL_SUMMON + finalizeOwners.
	return guid
}

// ultimatePlayerOwner walks the summon chain (pet → totem → player) and
// returns the player GUID if one is found.
func (p *Parser) ultimatePlayerOwner(guid string) string {
	seen := map[string]bool{}
	cur := guid
	for i := 0; i < 8; i++ {
		if seen[cur] {
			return ""
		}
		seen[cur] = true
		owner := p.petOwner[cur]
		if owner == "" {
			if a := p.actors[cur]; a != nil {
				owner = a.OwnerGUID
			}
		}
		if owner == "" {
			return ""
		}
		if a := p.actors[owner]; a != nil && a.IsPlayer {
			return owner
		}
		cur = owner
	}
	return ""
}

// finalizeOwners rewrites OwnerGUID to the ultimate player owner when known,
// so nested summons (e.g. elemental → totem → shaman) attach to the player.
func (p *Parser) finalizeOwners() {
	for _, a := range p.actors {
		if a.IsPlayer {
			a.OwnerGUID = ""
			continue
		}
		if player := p.ultimatePlayerOwner(a.GUID); player != "" {
			a.OwnerGUID = player
		}
	}
}

func (p *Parser) fightActor(guid string, ts time.Time) *ActorAgg {
	if guid == "" || p.cur == nil {
		return nil
	}
	agg, ok := p.cur.Actors[guid]
	if !ok {
		agg = &ActorAgg{GUID: guid, FirstEvent: ts, LastEvent: ts, Seen: true}
		p.cur.Actors[guid] = agg
	} else {
		if !agg.Seen || ts.Before(agg.FirstEvent) {
			agg.FirstEvent = ts
		}
		if ts.After(agg.LastEvent) {
			agg.LastEvent = ts
		}
		agg.Seen = true
	}
	return agg
}

func (p *Parser) ensureSpell(actorGUID string, spellID int, spellName string, school int, metric Metric) *SpellAgg {
	if p.cur == nil || actorGUID == "" {
		return nil
	}
	p.noteAbility(spellID, spellName, school)
	key := spellKey{ActorGUID: actorGUID, SpellID: spellID, Metric: metric}
	sp, ok := p.cur.Spells[key]
	if !ok {
		sp = &SpellAgg{
			ActorGUID: actorGUID,
			SpellID:   spellID,
			SpellName: spellName,
			School:    school,
			Metric:    metric,
		}
		p.cur.Spells[key] = sp
	}
	return sp
}

func (p *Parser) noteAbility(spellID int, spellName string, school int) {
	if spellID <= 0 {
		return
	}
	if a, ok := p.abilities[spellID]; ok {
		if spellName != "" && a.Name == "" {
			a.Name = spellName
		}
		if school != 0 && a.School == 0 {
			a.School = school
		}
		return
	}
	name := spellName
	if name == "" {
		name = strconv.Itoa(spellID)
	}
	p.abilities[spellID] = &AbilityInfo{SpellID: spellID, Name: name, School: school}
}

func (p *Parser) appendEvent(ev CombatEvent) {
	if p.cur == nil {
		return
	}
	if ev.Ts.IsZero() {
		ev.Ts = p.cur.EndTs
	}
	ev.OffsetMs = int(ev.Ts.Sub(p.cur.StartTs).Milliseconds())
	if ev.OffsetMs < 0 {
		ev.OffsetMs = 0
	}
	p.cur.Events = append(p.cur.Events, ev)
}

func (p *Parser) addSpell(actorGUID string, spellID int, spellName string, school int, metric Metric, amount int64, crit bool, tick bool, glancing bool) {
	if amount < 0 {
		return
	}
	sp := p.ensureSpell(actorGUID, spellID, spellName, school, metric)
	if sp == nil {
		return
	}
	sp.Total += amount
	sp.Hits++
	switch {
	case crit:
		sp.Crits++
		sp.Crit.add(amount)
	case glancing:
		sp.Glancing++
		sp.Glance.add(amount)
	default:
		sp.Normal.add(amount)
	}
	if tick {
		sp.Ticks++
	}
}

func (p *Parser) addSpellMiss(actorGUID string, spellID int, spellName string, school int, metric Metric) {
	sp := p.ensureSpell(actorGUID, spellID, spellName, school, metric)
	if sp == nil {
		return
	}
	sp.Misses++
}

func (p *Parser) handleSummon(ts time.Time, fields []string) {
	if len(fields) < 6 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	p.ensureActor(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	if srcGUID != "" && dstGUID != "" && dstGUID != "0x0000000000000000" {
		p.petOwner[dstGUID] = srcGUID
		if a := p.actors[dstGUID]; a != nil {
			a.OwnerGUID = srcGUID
		}
	}
	if p.cur == nil {
		return
	}
	spellID := 0
	if len(fields) >= 9 {
		spellID = int(parseInt64(fields[6]))
		p.noteAbility(spellID, unquote(fields[7]), int(parseInt64(fields[8])))
	}
	p.cur.EndTs = ts
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventSummon,
		SourceGUID: srcGUID,
		TargetGUID: dstGUID,
		SpellID:    spellID,
	})
}

func (p *Parser) handleSwingDamage(ts time.Time, fields []string, _ bool) {
	if len(fields) < 7 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	amount := parseInt64(fields[6])
	// amount,overkill,school,resisted,blocked,absorbed,crit,glancing,crushing
	overkill := int64(0)
	resisted := int64(0)
	blocked := int64(0)
	absorbed := int64(0)
	if len(fields) > 7 {
		overkill = parseInt64(fields[7])
	}
	if len(fields) > 9 {
		resisted = parseInt64(fields[9])
	}
	if len(fields) > 10 {
		blocked = parseInt64(fields[10])
	}
	if len(fields) > 11 {
		absorbed = parseInt64(fields[11])
	}
	crit := len(fields) > 12 && isTruthy(fields[12])
	glancing := len(fields) > 13 && isTruthy(fields[13])
	crushing := len(fields) > 14 && isTruthy(fields[14])

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	p.noteAbility(SpellMeleeID, SpellMeleeName, 1)

	if agg := p.fightActor(src, ts); agg != nil {
		agg.DamageDone += amount
	}
	p.addSpell(src, SpellMeleeID, SpellMeleeName, 1, MetricDamage, amount, crit, false, glancing)

	if dest := p.fightActor(dstGUID, ts); dest != nil {
		dest.DamageTaken += amount
	}
	p.addSpell(dstGUID, SpellMeleeID, SpellMeleeName, 1, MetricDamageTaken, amount, crit, false, glancing)

	if (dstFlags&FlagPlayer) == 0 && (dstFlags&FlagNPC) != 0 {
		p.cur.EnemyDamageTaken[unquote(dstName)] += amount
	}

	flags := 0
	if crit {
		flags |= EventFlagCrit
	}
	if glancing {
		flags |= EventFlagGlancing
	}
	if crushing {
		flags |= EventFlagCrushing
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventDamage,
		SourceGUID: src,
		TargetGUID: dstGUID,
		SpellID:    SpellMeleeID,
		Amount:     int(amount),
		Overkill:   int(overkill),
		Absorbed:   int(absorbed),
		Resisted:   int(resisted),
		Blocked:    int(blocked),
		Flags:      flags,
	})
}

func (p *Parser) handleSpellDamage(ts time.Time, fields []string, periodic bool) {
	// prefix(6) + spellId,name,school + amount,...
	if len(fields) < 10 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	spellID := int(parseInt64(fields[6]))
	spellName := unquote(fields[7])
	school := int(parseInt64(fields[8]))
	amount := parseInt64(fields[9])
	// amount,overkill,school,resisted,blocked,absorbed,crit,glancing,crushing
	overkill := int64(0)
	resisted := int64(0)
	blocked := int64(0)
	absorbed := int64(0)
	if len(fields) > 10 {
		overkill = parseInt64(fields[10])
	}
	if len(fields) > 12 {
		resisted = parseInt64(fields[12])
	}
	if len(fields) > 13 {
		blocked = parseInt64(fields[13])
	}
	if len(fields) > 14 {
		absorbed = parseInt64(fields[14])
	}
	crit := len(fields) > 15 && isTruthy(fields[15])
	glancing := len(fields) > 16 && isTruthy(fields[16])
	crushing := len(fields) > 17 && isTruthy(fields[17])

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)

	if agg := p.fightActor(src, ts); agg != nil {
		agg.DamageDone += amount
	}
	p.addSpell(src, spellID, spellName, school, MetricDamage, amount, crit, periodic, glancing)

	if dest := p.fightActor(dstGUID, ts); dest != nil {
		dest.DamageTaken += amount
	}
	p.addSpell(dstGUID, spellID, spellName, school, MetricDamageTaken, amount, crit, periodic, glancing)

	if (dstFlags&FlagPlayer) == 0 && (dstFlags&FlagNPC) != 0 {
		p.cur.EnemyDamageTaken[unquote(dstName)] += amount
	}

	// Valithria success: Dreamwalker's Rage.
	if spellID == wow.ValithriaSuccessSpellID && wow.IsKnownBoss(unquote(srcName)) {
		p.cur.EncounterSuccess = true
	}

	flags := 0
	if crit {
		flags |= EventFlagCrit
	}
	if glancing {
		flags |= EventFlagGlancing
	}
	if crushing {
		flags |= EventFlagCrushing
	}
	if periodic {
		flags |= EventFlagPeriodic
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventDamage,
		SourceGUID: src,
		TargetGUID: dstGUID,
		SpellID:    spellID,
		Amount:     int(amount),
		Overkill:   int(overkill),
		Absorbed:   int(absorbed),
		Resisted:   int(resisted),
		Blocked:    int(blocked),
		Flags:      flags,
	})
}

func (p *Parser) handleSwingMissed(ts time.Time, fields []string) {
	if len(fields) < 7 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	missType := strings.ToUpper(unquote(fields[6]))

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	_ = p.fightActor(src, ts)
	p.noteAbility(SpellMeleeID, SpellMeleeName, 1)

	mt := parseMissType(missType)
	if missType == "MISS" {
		p.addSpellMiss(src, SpellMeleeID, SpellMeleeName, 1, MetricDamage)
	}
	var missPtr *int16
	if mt != 0 {
		missPtr = &mt
	}
	amount := 0
	if len(fields) > 7 {
		amount = int(parseInt64(fields[7]))
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventMiss,
		SourceGUID: src,
		TargetGUID: dstGUID,
		SpellID:    SpellMeleeID,
		Amount:     amount,
		MissType:   missPtr,
	})
}

func (p *Parser) handleSpellMissed(ts time.Time, fields []string) {
	// prefix(6) + spellId,name,school + missType[,amount]
	if len(fields) < 10 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, _ := fields[3], fields[4], parseFlags(fields[5])
	spellID := int(parseInt64(fields[6]))
	spellName := unquote(fields[7])
	school := int(parseInt64(fields[8]))
	missType := strings.ToUpper(unquote(fields[9]))

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, parseFlags(fields[5]))
	_ = p.fightActor(src, ts)
	p.noteAbility(spellID, spellName, school)

	mt := parseMissType(missType)
	if missType == "MISS" {
		p.addSpellMiss(src, spellID, spellName, school, MetricDamage)
	}
	var missPtr *int16
	if mt != 0 {
		missPtr = &mt
	}
	amount := 0
	if len(fields) > 10 {
		amount = int(parseInt64(fields[10]))
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventMiss,
		SourceGUID: src,
		TargetGUID: dstGUID,
		SpellID:    spellID,
		Amount:     amount,
		MissType:   missPtr,
	})
}

func (p *Parser) handleHeal(ts time.Time, fields []string, periodic bool) {
	// prefix(6) + spellId,name,school + amount,overheal,absorbed,crit
	if len(fields) < 10 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	spellID := int(parseInt64(fields[6]))
	spellName := unquote(fields[7])
	school := int(parseInt64(fields[8]))
	amount := parseInt64(fields[9])
	overheal := int64(0)
	absorbed := int64(0)
	if len(fields) > 10 {
		overheal = parseInt64(fields[10])
	}
	if len(fields) > 11 {
		absorbed = parseInt64(fields[11])
	}
	crit := len(fields) > 12 && isTruthy(fields[12])

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	p.notePetOwnerLink(spellID, srcGUID, srcFlags, dstGUID, dstFlags)

	dst := unquote(dstName)
	if p.cur != nil && (dstFlags&FlagPlayer) == 0 && (dstFlags&FlagNPC) != 0 && wow.IsKnownBoss(dst) {
		p.cur.BossHealingTaken[dst] += amount
	}

	if agg := p.fightActor(src, ts); agg != nil {
		agg.HealingDone += amount
		agg.Overheal += overheal
	}
	p.addSpell(src, spellID, spellName, school, MetricHealing, amount, crit, periodic, false)

	flags := 0
	if crit {
		flags |= EventFlagCrit
	}
	if periodic {
		flags |= EventFlagPeriodic
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventHeal,
		SourceGUID: src,
		TargetGUID: dstGUID,
		SpellID:    spellID,
		Amount:     int(amount),
		Overheal:   int(overheal),
		Absorbed:   int(absorbed),
		Flags:      flags,
	})
}

func (p *Parser) handleDeath(ts time.Time, fields []string) {
	if len(fields) < 6 || p.cur == nil {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	p.ensureActor(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	_ = p.fightActor(dstGUID, ts)

	if (dstFlags&FlagPlayer) == 0 && dstName != "" && dstName != "nil" {
		p.cur.EnemyDied[unquote(dstName)] = true
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventDeath,
		SourceGUID: srcGUID,
		TargetGUID: dstGUID,
	})
}

func (p *Parser) handleAura(ts time.Time, fields []string, eventType int16) {
	if len(fields) < 10 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	spellID := int(parseInt64(fields[6]))
	spellName := unquote(fields[7])
	school := int(parseInt64(fields[8]))
	auraType := strings.ToUpper(unquote(fields[9]))

	p.ensureActor(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	p.notePetOwnerLink(spellID, srcGUID, srcFlags, dstGUID, dstFlags)
	p.noteAbility(spellID, spellName, school)

	if p.cur == nil {
		return
	}
	p.cur.EndTs = ts

	flags := 0
	switch auraType {
	case "BUFF":
		flags |= EventFlagAuraBuff
	case "DEBUFF":
		flags |= EventFlagAuraDebuff
	}
	stacks := 0
	if len(fields) > 10 {
		stacks = int(parseInt64(fields[10]))
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  eventType,
		SourceGUID: srcGUID,
		TargetGUID: dstGUID,
		SpellID:    spellID,
		Flags:      flags,
		Extra:      stacks,
	})

	// Gunship victory: Teleport to Deathbringer's Rise.
	if spellID == wow.GunshipSuccessSpellID && fightHasGunship(p.cur) {
		p.cur.EncounterSuccess = true
		p.closeFight()
	}
}

func (p *Parser) handleInterrupt(ts time.Time, fields []string) {
	// prefix + spellId,name,school + extraSpellId,extraSpellName,extraSchool
	if len(fields) < 12 || p.cur == nil {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	spellID := int(parseInt64(fields[6]))
	spellName := unquote(fields[7])
	school := int(parseInt64(fields[8]))
	extraSpellID := int(parseInt64(fields[9]))
	extraName := unquote(fields[10])
	extraSchool := int(parseInt64(fields[11]))

	p.ensureActor(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	p.noteAbility(spellID, spellName, school)
	p.noteAbility(extraSpellID, extraName, extraSchool)

	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventInterrupt,
		SourceGUID: srcGUID,
		TargetGUID: dstGUID,
		SpellID:    spellID,
		Extra:      extraSpellID,
	})
}

func (p *Parser) handleDispel(ts time.Time, fields []string) {
	// prefix + spellId,name,school + extraSpellId,extraSpellName,extraSchool[,auraType]
	if len(fields) < 12 || p.cur == nil {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	spellID := int(parseInt64(fields[6]))
	spellName := unquote(fields[7])
	school := int(parseInt64(fields[8]))
	extraSpellID := int(parseInt64(fields[9]))
	extraName := unquote(fields[10])
	extraSchool := int(parseInt64(fields[11]))

	p.ensureActor(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)
	p.noteAbility(spellID, spellName, school)
	p.noteAbility(extraSpellID, extraName, extraSchool)

	flags := 0
	if len(fields) > 12 {
		switch strings.ToUpper(unquote(fields[12])) {
		case "BUFF":
			flags |= EventFlagAuraBuff
		case "DEBUFF":
			flags |= EventFlagAuraDebuff
		}
	}
	p.appendEvent(CombatEvent{
		Ts:         ts,
		EventType:  EventDispel,
		SourceGUID: srcGUID,
		TargetGUID: dstGUID,
		SpellID:    spellID,
		Flags:      flags,
		Extra:      extraSpellID,
	})
}

func isCombatEvent(event string) bool {
	switch event {
	case "SWING_DAMAGE", "SPELL_DAMAGE", "SPELL_PERIODIC_DAMAGE", "RANGE_DAMAGE",
		"DAMAGE_SHIELD", "SWING_MISSED", "SPELL_MISSED", "RANGE_MISSED", "SPELL_PERIODIC_MISSED",
		"SPELL_HEAL", "SPELL_PERIODIC_HEAL", "UNIT_DIED", "PARTY_KILL":
		return true
	default:
		return false
	}
}

// isHostileEngagement reports whether the event should start a fight or reset the
// out-of-combat gap timer. Friendly heals (including OOC HoTs like Fel Armor) do not.
func isHostileEngagement(event string, fields []string) bool {
	switch event {
	case "SPELL_HEAL", "SPELL_PERIODIC_HEAL":
		return false
	case "UNIT_DIED", "PARTY_KILL":
		if len(fields) < 6 {
			return false
		}
		return isNPCFlags(parseFlags(fields[5]))
	default:
		if len(fields) < 6 {
			return false
		}
		return isNPCFlags(parseFlags(fields[2])) || isNPCFlags(parseFlags(fields[5]))
	}
}

// isPureNPCCombat reports damage/miss events where both sides are NPCs (no player).
func isPureNPCCombat(event string, fields []string) bool {
	switch event {
	case "UNIT_DIED", "PARTY_KILL", "SPELL_HEAL", "SPELL_PERIODIC_HEAL":
		return false
	default:
		if len(fields) < 6 {
			return false
		}
		return isNPCFlags(parseFlags(fields[2])) && isNPCFlags(parseFlags(fields[5]))
	}
}

func isNPCFlags(flags int64) bool {
	return (flags&FlagNPC) != 0 && (flags&FlagPlayer) == 0
}

func hostileBossName(event string, fields []string) string {
	if len(fields) < 6 {
		return ""
	}
	check := func(name string, flags int64) string {
		name = unquote(name)
		if name == "" || name == "nil" || !isNPCFlags(flags) {
			return ""
		}
		if wow.IsKnownBoss(name) {
			return name
		}
		return ""
	}
	switch event {
	case "UNIT_DIED", "PARTY_KILL":
		return check(fields[4], parseFlags(fields[5]))
	default:
		if n := check(fields[1], parseFlags(fields[2])); n != "" {
			return n
		}
		return check(fields[4], parseFlags(fields[5]))
	}
}

func fightHasKnownBoss(f *FightResult) bool {
	for name := range f.EnemyDamageTaken {
		if wow.IsKnownBoss(name) {
			return true
		}
	}
	for name := range f.BossHealingTaken {
		if wow.IsKnownBoss(name) {
			return true
		}
	}
	return false
}

func fightHasNPC(f *FightResult, name string) bool {
	if _, ok := f.EnemyDamageTaken[name]; ok {
		return true
	}
	if _, ok := f.BossHealingTaken[name]; ok {
		return true
	}
	return false
}

func fightSharesEncounter(f *FightResult, bossName string) bool {
	enc := wow.EncounterID(bossName)
	if enc == "" {
		// Standalone bosses only share with themselves (handled by fightHasNPC).
		return false
	}
	for name := range f.EnemyDamageTaken {
		if wow.EncounterID(name) == enc {
			return true
		}
	}
	for name := range f.BossHealingTaken {
		if wow.EncounterID(name) == enc {
			return true
		}
	}
	return false
}

func fightHasGunship(f *FightResult) bool {
	for name := range f.EnemyDamageTaken {
		if wow.IsGunshipEncounter(name) {
			return true
		}
	}
	for name := range f.BossHealingTaken {
		if wow.IsGunshipEncounter(name) {
			return true
		}
	}
	return false
}

func isSupportEvent(event string) bool {
	switch event {
	case "SPELL_AURA_APPLIED", "SPELL_AURA_APPLIED_DOSE",
		"SPELL_AURA_REMOVED", "SPELL_AURA_REMOVED_DOSE",
		"SPELL_AURA_REFRESH",
		"SPELL_INTERRUPT",
		"SPELL_DISPEL", "SPELL_STOLEN", "SPELL_DISPEL_FAILED":
		return true
	default:
		return false
	}
}

func parseMissType(s string) int16 {
	switch strings.ToUpper(s) {
	case "MISS":
		return MissTypeMiss
	case "DODGE":
		return MissTypeDodge
	case "PARRY":
		return MissTypeParry
	case "BLOCK":
		return MissTypeBlock
	case "EVADE":
		return MissTypeEvade
	case "IMMUNE":
		return MissTypeImmune
	case "DEFLECT":
		return MissTypeDeflect
	case "ABSORB":
		return MissTypeAbsorb
	case "REFLECT":
		return MissTypeReflect
	case "RESIST":
		return MissTypeResist
	default:
		return 0
	}
}

func splitLine(line string) (time.Time, string, []string, error) {
	// "M/D HH:MM:SS.mmm  EVENT,fields..."
	idx := strings.Index(line, "  ")
	if idx < 0 {
		return time.Time{}, "", nil, fmt.Errorf("no timestamp separator")
	}
	tsPart := line[:idx]
	rest := strings.TrimLeft(line[idx+2:], " ")
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return time.Time{}, "", nil, fmt.Errorf("no event comma")
	}
	event := rest[:comma]
	fields := splitCSV(rest[comma+1:])

	ts, err := parseTimestamp(tsPart)
	if err != nil {
		return time.Time{}, "", nil, err
	}
	return ts, event, fields, nil
}

func parseTimestamp(s string) (time.Time, error) {
	// M/D HH:MM:SS.mmm
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("bad ts: %q", s)
	}
	md := strings.Split(parts[0], "/")
	if len(md) != 2 {
		return time.Time{}, fmt.Errorf("bad date: %q", parts[0])
	}
	month, err := strconv.Atoi(md[0])
	if err != nil {
		return time.Time{}, err
	}
	day, err := strconv.Atoi(md[1])
	if err != nil {
		return time.Time{}, err
	}
	hms := parts[1]
	var hour, min, sec, ms int
	if _, err := fmt.Sscanf(hms, "%d:%d:%d.%d", &hour, &min, &sec, &ms); err != nil {
		return time.Time{}, fmt.Errorf("bad time: %q", hms)
	}
	return time.Date(0, time.Month(month), day, hour, min, sec, ms*1_000_000, time.UTC), nil
}

func splitCSV(s string) []string {
	var fields []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"':
			inQuote = !inQuote
			b.WriteByte(c)
		case c == ',' && !inQuote:
			fields = append(fields, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	fields = append(fields, b.String())
	return fields
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func parseFlags(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "nil" {
		return 0
	}
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0
	}
	return v
}

func parseInt64(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "nil" {
		return 0
	}
	v, err := strconv.ParseInt(s, 0, 64)
	if err != nil {
		return 0
	}
	return v
}

func isTruthy(s string) bool {
	s = strings.TrimSpace(s)
	return s == "1" || strings.EqualFold(s, "true")
}

// ActiveTimeMs returns actor active span within the fight.
func (a *ActorAgg) ActiveTimeMs() int64 {
	if !a.Seen {
		return 0
	}
	ms := a.LastEvent.Sub(a.FirstEvent).Milliseconds()
	if ms < 0 {
		return 0
	}
	return ms
}
