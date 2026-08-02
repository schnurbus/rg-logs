package db

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"

	"rg-logs/internal/parser"
)

// AuraKind filters aura aggregations by buff or debuff flag.
type AuraKind string

const (
	AuraKindBuff   AuraKind = "buff"
	AuraKindDebuff AuraKind = "debuff"
)

// AuraStat is an aggregated aura uptime row for a fight.
type AuraStat struct {
	SpellID      int     `json:"spellId"`
	SpellName    string  `json:"spellName"`
	School       int     `json:"school"`
	Applications int     `json:"applications"`
	Refreshes    int     `json:"refreshes"`
	Targets      int     `json:"targets"`
	UptimeMs     int64   `json:"uptimeMs"`
	UptimePct    float64 `json:"uptimePct"`
}

// CastCountStat is an aggregated interrupt or dispel row.
type CastCountStat struct {
	ActorID        uuid.UUID `json:"actorId"`
	ActorName      string    `json:"actorName"`
	Class          *string   `json:"class,omitempty"`
	Spec           *string   `json:"spec,omitempty"`
	SpellID        int       `json:"spellId"`
	SpellName      string    `json:"spellName"`
	ExtraSpellID   int       `json:"extraSpellId"`
	ExtraSpellName string    `json:"extraSpellName"`
	Count          int       `json:"count"`
}

type auraEventRow struct {
	OffsetMs      int
	EventType     int16
	TargetActorID *uuid.UUID
	SpellID       int
	Flags         int
}

type openAura struct {
	startMs int
}

// ListAuraStats aggregates buff or debuff uptime for a fight from combat_events.
func (s *Store) ListAuraStats(ctx context.Context, fightID uuid.UUID, kind AuraKind) ([]AuraStat, error) {
	fight, err := s.GetFight(ctx, fightID)
	if err != nil {
		return nil, err
	}

	flagMask := parser.EventFlagAuraBuff
	if kind == AuraKindDebuff {
		flagMask = parser.EventFlagAuraDebuff
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT e.offset_ms, e.event_type, e.target_actor_id, e.spell_id, e.flags
		FROM combat_events e
		WHERE e.fight_id = $1
		  AND e.event_type IN ($2, $3, $4)
		  AND (e.flags & $5) <> 0
		ORDER BY e.offset_ms ASC, e.id ASC`,
		fightID,
		parser.EventAuraApplied,
		parser.EventAuraRemoved,
		parser.EventAuraRefresh,
		flagMask,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type targetKey struct {
		target uuid.UUID
		spell  int
	}

	open := make(map[targetKey]*openAura)
	applications := make(map[int]int)
	refreshes := make(map[int]int)
	uptimeBySpell := make(map[int]int64)
	targetsBySpell := make(map[int]map[uuid.UUID]struct{})
	spellIDs := make(map[int]struct{})

	addTarget := func(spellID int, target uuid.UUID) {
		m := targetsBySpell[spellID]
		if m == nil {
			m = make(map[uuid.UUID]struct{})
			targetsBySpell[spellID] = m
		}
		m[target] = struct{}{}
	}

	closeSegment := func(key targetKey, endMs int) {
		o := open[key]
		if o == nil {
			return
		}
		if endMs > o.startMs {
			uptimeBySpell[key.spell] += int64(endMs - o.startMs)
		}
		delete(open, key)
	}

	for rows.Next() {
		var r auraEventRow
		if err := rows.Scan(&r.OffsetMs, &r.EventType, &r.TargetActorID, &r.SpellID, &r.Flags); err != nil {
			return nil, err
		}
		if r.TargetActorID == nil || r.SpellID <= 0 {
			continue
		}
		key := targetKey{target: *r.TargetActorID, spell: r.SpellID}
		spellIDs[r.SpellID] = struct{}{}
		addTarget(r.SpellID, *r.TargetActorID)

		switch r.EventType {
		case parser.EventAuraApplied:
			applications[r.SpellID]++
			if open[key] != nil {
				closeSegment(key, r.OffsetMs)
			}
			open[key] = &openAura{startMs: r.OffsetMs}
		case parser.EventAuraRefresh:
			refreshes[r.SpellID]++
			if open[key] == nil {
				open[key] = &openAura{startMs: r.OffsetMs}
			}
			// refresh keeps the segment open; no duration change yet
		case parser.EventAuraRemoved:
			closeSegment(key, r.OffsetMs)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	endMs := int(fight.DurationMs)
	if endMs < 0 {
		endMs = 0
	}
	for key := range open {
		closeSegment(key, endMs)
	}

	abilityNames, abilitySchools, err := s.abilityLookup(ctx, fight.UploadID, spellIDs)
	if err != nil {
		return nil, err
	}

	out := make([]AuraStat, 0, len(spellIDs))
	for spellID := range spellIDs {
		targets := len(targetsBySpell[spellID])
		uptime := uptimeBySpell[spellID]
		var pct float64
		if fight.DurationMs > 0 && targets > 0 {
			denom := float64(fight.DurationMs) * float64(targets)
			pct = float64(uptime) / denom
			if pct > 1 {
				pct = 1
			}
		}
		name := abilityNames[spellID]
		if name == "" {
			name = fmt.Sprintf("Spell %d", spellID)
		}
		out = append(out, AuraStat{
			SpellID:      spellID,
			SpellName:    name,
			School:       abilitySchools[spellID],
			Applications: applications[spellID],
			Refreshes:    refreshes[spellID],
			Targets:      targets,
			UptimeMs:     uptime,
			UptimePct:    pct,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UptimeMs == out[j].UptimeMs {
			return out[i].Applications > out[j].Applications
		}
		return out[i].UptimeMs > out[j].UptimeMs
	})
	if out == nil {
		out = []AuraStat{}
	}
	return out, nil
}

// ListInterruptStats aggregates interrupt events for a fight.
func (s *Store) ListInterruptStats(ctx context.Context, fightID uuid.UUID) ([]CastCountStat, error) {
	return s.listCastCountStats(ctx, fightID, parser.EventInterrupt)
}

// ListDispelStats aggregates dispel events for a fight.
func (s *Store) ListDispelStats(ctx context.Context, fightID uuid.UUID) ([]CastCountStat, error) {
	return s.listCastCountStats(ctx, fightID, parser.EventDispel)
}

func (s *Store) listCastCountStats(ctx context.Context, fightID uuid.UUID, eventType int16) ([]CastCountStat, error) {
	fight, err := s.GetFight(ctx, fightID)
	if err != nil {
		return nil, err
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT
			e.source_actor_id,
			a.name,
			a.class,
			a.spec,
			e.spell_id,
			e.extra,
			COUNT(*)::int
		FROM combat_events e
		JOIN actors a ON a.id = e.source_actor_id
		WHERE e.fight_id = $1
		  AND e.event_type = $2
		  AND e.source_actor_id IS NOT NULL
		GROUP BY e.source_actor_id, a.name, a.class, a.spec, e.spell_id, e.extra
		ORDER BY COUNT(*) DESC, a.name ASC`,
		fightID, eventType,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spellIDs := make(map[int]struct{})
	type rawRow struct {
		ActorID      uuid.UUID
		ActorName    string
		Class        *string
		Spec         *string
		SpellID      int
		ExtraSpellID int
		Count        int
	}
	var raw []rawRow
	for rows.Next() {
		var r rawRow
		if err := rows.Scan(&r.ActorID, &r.ActorName, &r.Class, &r.Spec, &r.SpellID, &r.ExtraSpellID, &r.Count); err != nil {
			return nil, err
		}
		spellIDs[r.SpellID] = struct{}{}
		if r.ExtraSpellID > 0 {
			spellIDs[r.ExtraSpellID] = struct{}{}
		}
		raw = append(raw, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	names, _, err := s.abilityLookup(ctx, fight.UploadID, spellIDs)
	if err != nil {
		return nil, err
	}

	out := make([]CastCountStat, 0, len(raw))
	for _, r := range raw {
		spellName := names[r.SpellID]
		if spellName == "" {
			spellName = fmt.Sprintf("Spell %d", r.SpellID)
		}
		extraName := names[r.ExtraSpellID]
		if r.ExtraSpellID > 0 && extraName == "" {
			extraName = fmt.Sprintf("Spell %d", r.ExtraSpellID)
		}
		out = append(out, CastCountStat{
			ActorID:        r.ActorID,
			ActorName:      r.ActorName,
			Class:          r.Class,
			Spec:           r.Spec,
			SpellID:        r.SpellID,
			SpellName:      spellName,
			ExtraSpellID:   r.ExtraSpellID,
			ExtraSpellName: extraName,
			Count:          r.Count,
		})
	}
	if out == nil {
		out = []CastCountStat{}
	}
	return out, nil
}

func (s *Store) abilityLookup(ctx context.Context, uploadID uuid.UUID, spellIDs map[int]struct{}) (names map[int]string, schools map[int]int, err error) {
	names = make(map[int]string, len(spellIDs))
	schools = make(map[int]int, len(spellIDs))
	if len(spellIDs) == 0 {
		return names, schools, nil
	}
	ids := make([]int, 0, len(spellIDs))
	for id := range spellIDs {
		ids = append(ids, id)
	}
	rows, err := s.Pool.Query(ctx, `
		SELECT spell_id, name, school
		FROM abilities
		WHERE upload_id = $1 AND spell_id = ANY($2)`,
		uploadID, ids,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, school int
		var name string
		if err := rows.Scan(&id, &name, &school); err != nil {
			return nil, nil, err
		}
		names[id] = name
		schools[id] = school
	}
	return names, schools, rows.Err()
}
