package parser

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	FightGap        = 45 * time.Second
	MinFightDur     = 5 * time.Second
	MinFightEvents  = 10
	FlagPlayer      = 0x400
	FlagNPC         = 0x800
	FlagPet         = 0x1000
	FlagGuardian    = 0x2000
	SpellMeleeID    = 1
	SpellMeleeName  = "Nahkampf"
)

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
}

type ActorAgg struct {
	GUID         string
	DamageDone   int64
	HealingDone  int64
	Overheal     int64
	DamageTaken  int64
	FirstEvent   time.Time
	LastEvent    time.Time
	Seen         bool
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
	EnemyDamageTaken map[string]int64 // enemy name -> damage taken (for title)
	EnemyDied        map[string]bool
	EventCount       int
}

type spellKey struct {
	ActorGUID string
	SpellID   int
	Metric    Metric
}

type ParseResult struct {
	Actors []*ActorInfo
	Fights []*FightResult
}

type Parser struct {
	actors     map[string]*ActorInfo // guid -> actor
	petOwner   map[string]string     // pet guid -> owner guid
	year       int
	cur        *FightResult
	fights     []*FightResult
	lastCombat time.Time
	hasCombat  bool
}

func New() *Parser {
	return &Parser{
		actors:   make(map[string]*ActorInfo),
		petOwner: make(map[string]string),
		year:     time.Now().Year(),
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
	return &ParseResult{Actors: actors, Fights: p.fights}
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
		p.handleSummon(fields)
		return nil
	}

	combat := isCombatEvent(event)
	if !combat {
		return nil
	}

	if p.hasCombat && ts.Sub(p.lastCombat) >= FightGap {
		p.closeFight()
	}

	if p.cur == nil {
		p.cur = newFight(ts)
	}

	p.lastCombat = ts
	p.hasCombat = true
	p.cur.EndTs = ts
	p.cur.EventCount++

	switch event {
	case "SWING_DAMAGE":
		p.handleSwingDamage(ts, fields, false)
	case "SPELL_DAMAGE", "RANGE_DAMAGE", "DAMAGE_SHIELD":
		p.handleSpellDamage(ts, fields, false)
	case "SPELL_PERIODIC_DAMAGE":
		p.handleSpellDamage(ts, fields, true)
	case "SPELL_HEAL":
		p.handleHeal(ts, fields, false)
	case "SPELL_PERIODIC_HEAL":
		p.handleHeal(ts, fields, true)
	case "UNIT_DIED", "PARTY_KILL":
		p.handleDeath(fields)
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
		EnemyDamageTaken: make(map[string]int64),
		EnemyDied:        make(map[string]bool),
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
	f.Kill = f.EnemyDied[f.Title]

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
	bestName := "Trash"
	var bestDmg int64
	for name, dmg := range f.EnemyDamageTaken {
		if dmg > bestDmg {
			bestDmg = dmg
			bestName = name
		}
	}
	return bestName
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

func (p *Parser) addSpell(actorGUID string, spellID int, spellName string, school int, metric Metric, amount int64, crit bool, tick bool) {
	if p.cur == nil || actorGUID == "" || amount < 0 {
		return
	}
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
	sp.Total += amount
	sp.Hits++
	if crit {
		sp.Crits++
	}
	if tick {
		sp.Ticks++
	}
}

func (p *Parser) handleSummon(fields []string) {
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
}

func (p *Parser) handleSwingDamage(ts time.Time, fields []string, _ bool) {
	if len(fields) < 7 {
		return
	}
	srcGUID, srcName, srcFlags := fields[0], fields[1], parseFlags(fields[2])
	dstGUID, dstName, dstFlags := fields[3], fields[4], parseFlags(fields[5])
	amount := parseInt64(fields[6])
	crit := len(fields) > 12 && isTruthy(fields[12])

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)

	if agg := p.fightActor(src, ts); agg != nil {
		agg.DamageDone += amount
	}
	p.addSpell(src, SpellMeleeID, SpellMeleeName, 1, MetricDamage, amount, crit, false)

	if dest := p.fightActor(dstGUID, ts); dest != nil {
		dest.DamageTaken += amount
	}
	p.addSpell(dstGUID, SpellMeleeID, SpellMeleeName, 1, MetricDamageTaken, amount, crit, false)

	if (dstFlags&FlagPlayer) == 0 && (dstFlags&FlagNPC) != 0 {
		p.cur.EnemyDamageTaken[unquote(dstName)] += amount
	}
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
	// amount,overkill,school,resisted,blocked,absorbed,crit,...
	crit := len(fields) > 15 && isTruthy(fields[15])

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)

	if agg := p.fightActor(src, ts); agg != nil {
		agg.DamageDone += amount
	}
	p.addSpell(src, spellID, spellName, school, MetricDamage, amount, crit, periodic)

	if dest := p.fightActor(dstGUID, ts); dest != nil {
		dest.DamageTaken += amount
	}
	p.addSpell(dstGUID, spellID, spellName, school, MetricDamageTaken, amount, crit, periodic)

	if (dstFlags&FlagPlayer) == 0 && (dstFlags&FlagNPC) != 0 {
		p.cur.EnemyDamageTaken[unquote(dstName)] += amount
	}
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
	if len(fields) > 10 {
		overheal = parseInt64(fields[10])
	}
	crit := len(fields) > 12 && isTruthy(fields[12])

	src := p.resolveSource(srcGUID, srcName, srcFlags)
	p.ensureActor(dstGUID, dstName, dstFlags)

	if agg := p.fightActor(src, ts); agg != nil {
		agg.HealingDone += amount
		agg.Overheal += overheal
	}
	p.addSpell(src, spellID, spellName, school, MetricHealing, amount, crit, periodic)
}

func (p *Parser) handleDeath(fields []string) {
	if len(fields) < 6 || p.cur == nil {
		return
	}
	dstName := unquote(fields[4])
	dstFlags := parseFlags(fields[5])
	if (dstFlags&FlagPlayer) == 0 && dstName != "" && dstName != "nil" {
		p.cur.EnemyDied[dstName] = true
	}
}

func isCombatEvent(event string) bool {
	switch event {
	case "SWING_DAMAGE", "SPELL_DAMAGE", "SPELL_PERIODIC_DAMAGE", "RANGE_DAMAGE",
		"DAMAGE_SHIELD", "SPELL_HEAL", "SPELL_PERIODIC_HEAL", "UNIT_DIED", "PARTY_KILL":
		return true
	default:
		return false
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
