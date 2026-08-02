package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"rg-logs/internal/parser"
)

const timelineTopPlayers = 12

// TimelineMode selects raid-wide summary or a per-player metric series.
type TimelineMode string

const (
	TimelineModeSummary TimelineMode = "summary"
	TimelineModeDamage  TimelineMode = "damage"
	TimelineModeHealing TimelineMode = "healing"
	TimelineModeTaken   TimelineMode = "taken"
)

// TimelineSide selects player raid or enemy NPC series.
type TimelineSide string

const (
	TimelineSidePlayers TimelineSide = "players"
	TimelineSideEnemies TimelineSide = "enemies"
)

// ParseTimelineSide returns players/enemies; unknown values default to players.
func ParseTimelineSide(s string) TimelineSide {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "enemies", "enemy", "npcs", "npc":
		return TimelineSideEnemies
	default:
		return TimelineSidePlayers
	}
}

// Enemy actor: not a player, not a player-controlled pet/guardian, and not owned by a player.
const sqlIsEnemyActor = `
	%[1]s.is_player = FALSE
	AND (%[1]s.flags & 4096) = 0
	AND (%[1]s.flags & 8192) = 0
	AND (
		%[1]s.owner_guid IS NULL
		OR NOT EXISTS (
			SELECT 1 FROM actors o
			WHERE o.upload_id = %[1]s.upload_id
			  AND o.guid = %[1]s.owner_guid
			  AND o.is_player
		)
	)`

func enemyActorSQL(alias string) string {
	return fmt.Sprintf(sqlIsEnemyActor, alias)
}

// TimelineSummaryPoint is one bucket of raid-wide damage/healing/taken.
type TimelineSummaryPoint struct {
	T       int   `json:"t"`
	Damage  int64 `json:"damage"`
	Healing int64 `json:"healing"`
	Taken   int64 `json:"taken"`
}

// TimelineSummary is the summary-mode timeline response.
type TimelineSummary struct {
	BucketMs int                    `json:"bucketMs"`
	Points   []TimelineSummaryPoint `json:"points"`
}

// TimelineSeriesPoint is one bucket amount for a player series.
type TimelineSeriesPoint struct {
	T      int   `json:"t"`
	Amount int64 `json:"amount"`
}

// TimelinePlayerSeries is one player's timeline.
type TimelinePlayerSeries struct {
	ActorID uuid.UUID             `json:"actorId"`
	Name    string                `json:"name"`
	Class   *string               `json:"class,omitempty"`
	Points  []TimelineSeriesPoint `json:"points"`
	Total   int64                 `json:"total"`
}

// TimelinePlayers is the per-player timeline response.
type TimelinePlayers struct {
	BucketMs int                    `json:"bucketMs"`
	Series   []TimelinePlayerSeries `json:"series"`
}

// ChooseTimelineBucketMs picks a bucket size aiming for ~180 points (min 1s).
func ChooseTimelineBucketMs(durationMs, override int64) int {
	if override >= 1000 {
		return int(override)
	}
	if durationMs <= 0 {
		return 1000
	}
	bucket := int(durationMs / 180)
	if bucket < 1000 {
		return 1000
	}
	// Snap to 1s / 2s / 5s / 10s steps.
	switch {
	case bucket <= 1000:
		return 1000
	case bucket <= 2000:
		return 2000
	case bucket <= 5000:
		return 5000
	case bucket <= 10000:
		return 10000
	default:
		return (bucket / 1000) * 1000
	}
}

func bucketCount(durationMs int64, bucketMs int) int {
	if bucketMs <= 0 {
		bucketMs = 1000
	}
	if durationMs <= 0 {
		return 1
	}
	n := int(durationMs/int64(bucketMs)) + 1
	if n < 1 {
		return 1
	}
	return n
}

// GetTimelineSummary returns raid-wide damage/healing/taken per time bucket.
func (s *Store) GetTimelineSummary(ctx context.Context, fightID uuid.UUID, bucketMs int, side TimelineSide) (*TimelineSummary, error) {
	fight, err := s.GetFight(ctx, fightID)
	if err != nil {
		return nil, err
	}
	bucketMs = ChooseTimelineBucketMs(fight.DurationMs, int64(bucketMs))
	n := bucketCount(fight.DurationMs, bucketMs)
	points := make([]TimelineSummaryPoint, n)
	for i := range points {
		points[i].T = i * bucketMs
	}

	var dmgHealQuery string
	var takenQuery string
	if side == TimelineSideEnemies {
		dmgHealQuery = `
			SELECT (e.offset_ms / $2) * $2 AS t,
			       SUM(CASE WHEN e.event_type = $3 THEN e.amount ELSE 0 END)::bigint,
			       SUM(CASE WHEN e.event_type = $4 THEN e.amount ELSE 0 END)::bigint
			FROM combat_events e
			JOIN actors src ON src.id = e.source_actor_id
			WHERE e.fight_id = $1
			  AND e.event_type IN ($3, $4)
			  AND ` + enemyActorSQL("src") + `
			GROUP BY 1`
		takenQuery = `
			SELECT (e.offset_ms / $2) * $2 AS t, SUM(e.amount)::bigint
			FROM combat_events e
			JOIN actors tgt ON tgt.id = e.target_actor_id
			WHERE e.fight_id = $1
			  AND e.event_type = $3
			  AND ` + enemyActorSQL("tgt") + `
			GROUP BY 1`
	} else {
		dmgHealQuery = `
			SELECT (e.offset_ms / $2) * $2 AS t,
			       SUM(CASE WHEN e.event_type = $3 THEN e.amount ELSE 0 END)::bigint,
			       SUM(CASE WHEN e.event_type = $4 THEN e.amount ELSE 0 END)::bigint
			FROM combat_events e
			WHERE e.fight_id = $1
			  AND e.event_type IN ($3, $4)
			GROUP BY 1`
		takenQuery = `
			SELECT (e.offset_ms / $2) * $2 AS t, SUM(e.amount)::bigint
			FROM combat_events e
			JOIN actors a ON a.id = e.target_actor_id AND a.is_player
			WHERE e.fight_id = $1
			  AND e.event_type = $3
			GROUP BY 1`
	}

	rows, err := s.Pool.Query(ctx, dmgHealQuery, fightID, bucketMs, parser.EventDamage, parser.EventHeal)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t int
		var dmg, heal int64
		if err := rows.Scan(&t, &dmg, &heal); err != nil {
			return nil, err
		}
		idx := t / bucketMs
		if idx >= 0 && idx < n {
			points[idx].Damage = dmg
			points[idx].Healing = heal
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	takenRows, err := s.Pool.Query(ctx, takenQuery, fightID, bucketMs, parser.EventDamage)
	if err != nil {
		return nil, err
	}
	defer takenRows.Close()
	for takenRows.Next() {
		var t int
		var taken int64
		if err := takenRows.Scan(&t, &taken); err != nil {
			return nil, err
		}
		idx := t / bucketMs
		if idx >= 0 && idx < n {
			points[idx].Taken = taken
		}
	}
	if err := takenRows.Err(); err != nil {
		return nil, err
	}

	return &TimelineSummary{BucketMs: bucketMs, Points: points}, nil
}

// GetTimelinePlayers returns top actor series for damage, healing, or taken.
func (s *Store) GetTimelinePlayers(ctx context.Context, fightID uuid.UUID, mode TimelineMode, bucketMs int, side TimelineSide) (*TimelinePlayers, error) {
	fight, err := s.GetFight(ctx, fightID)
	if err != nil {
		return nil, err
	}
	bucketMs = ChooseTimelineBucketMs(fight.DurationMs, int64(bucketMs))
	n := bucketCount(fight.DurationMs, bucketMs)

	type bucketKey struct {
		actor uuid.UUID
		idx   int
	}
	amounts := make(map[bucketKey]int64)
	totals := make(map[uuid.UUID]int64)
	meta := make(map[uuid.UUID]struct {
		Name  string
		Class *string
	})

	var rows pgxRows
	eventType := parser.EventDamage
	if mode == TimelineModeHealing {
		eventType = parser.EventHeal
	}

	switch {
	case side == TimelineSideEnemies && (mode == TimelineModeDamage || mode == TimelineModeHealing):
		rows, err = s.Pool.Query(ctx, `
			SELECT
				src.id AS actor_id,
				src.name AS actor_name,
				src.class AS actor_class,
				(e.offset_ms / $2) * $2 AS t,
				SUM(e.amount)::bigint
			FROM combat_events e
			JOIN actors src ON src.id = e.source_actor_id
			WHERE e.fight_id = $1
			  AND e.event_type = $3
			  AND `+enemyActorSQL("src")+`
			GROUP BY 1, 2, 3, 4`,
			fightID, bucketMs, eventType,
		)
	case side == TimelineSideEnemies && mode == TimelineModeTaken:
		rows, err = s.Pool.Query(ctx, `
			SELECT
				tgt.id AS actor_id,
				tgt.name AS actor_name,
				tgt.class AS actor_class,
				(e.offset_ms / $2) * $2 AS t,
				SUM(e.amount)::bigint
			FROM combat_events e
			JOIN actors tgt ON tgt.id = e.target_actor_id
			WHERE e.fight_id = $1
			  AND e.event_type = $3
			  AND `+enemyActorSQL("tgt")+`
			GROUP BY 1, 2, 3, 4`,
			fightID, bucketMs, parser.EventDamage,
		)
	case mode == TimelineModeDamage || mode == TimelineModeHealing:
		rows, err = s.Pool.Query(ctx, `
			SELECT
				CASE WHEN src.is_player THEN src.id ELSE owner.id END AS player_id,
				CASE WHEN src.is_player THEN src.name ELSE owner.name END AS player_name,
				CASE WHEN src.is_player THEN src.class ELSE owner.class END AS player_class,
				(e.offset_ms / $2) * $2 AS t,
				SUM(e.amount)::bigint
			FROM combat_events e
			JOIN actors src ON src.id = e.source_actor_id
			LEFT JOIN actors owner
			  ON owner.upload_id = src.upload_id
			 AND owner.guid = src.owner_guid
			 AND owner.is_player
			WHERE e.fight_id = $1
			  AND e.event_type = $3
			  AND (src.is_player OR owner.id IS NOT NULL)
			GROUP BY 1, 2, 3, 4`,
			fightID, bucketMs, eventType,
		)
	case mode == TimelineModeTaken:
		rows, err = s.Pool.Query(ctx, `
			SELECT
				tgt.id AS player_id,
				tgt.name AS player_name,
				tgt.class AS player_class,
				(e.offset_ms / $2) * $2 AS t,
				SUM(e.amount)::bigint
			FROM combat_events e
			JOIN actors tgt ON tgt.id = e.target_actor_id AND tgt.is_player
			WHERE e.fight_id = $1
			  AND e.event_type = $3
			GROUP BY 1, 2, 3, 4`,
			fightID, bucketMs, parser.EventDamage,
		)
	default:
		return nil, fmt.Errorf("unsupported timeline mode %q", mode)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var actorID uuid.UUID
		var name string
		var class *string
		var t int
		var amount int64
		if err := rows.Scan(&actorID, &name, &class, &t, &amount); err != nil {
			return nil, err
		}
		idx := t / bucketMs
		if idx < 0 || idx >= n {
			continue
		}
		amounts[bucketKey{actor: actorID, idx: idx}] += amount
		totals[actorID] += amount
		if _, ok := meta[actorID]; !ok {
			meta[actorID] = struct {
				Name  string
				Class *string
			}{Name: name, Class: class}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	type ranked struct {
		id    uuid.UUID
		total int64
	}
	rank := make([]ranked, 0, len(totals))
	for id, total := range totals {
		rank = append(rank, ranked{id: id, total: total})
	}
	sort.Slice(rank, func(i, j int) bool { return rank[i].total > rank[j].total })
	if len(rank) > timelineTopPlayers {
		rank = rank[:timelineTopPlayers]
	}

	series := make([]TimelinePlayerSeries, 0, len(rank))
	for _, r := range rank {
		m := meta[r.id]
		pts := make([]TimelineSeriesPoint, n)
		for i := 0; i < n; i++ {
			pts[i] = TimelineSeriesPoint{
				T:      i * bucketMs,
				Amount: amounts[bucketKey{actor: r.id, idx: i}],
			}
		}
		series = append(series, TimelinePlayerSeries{
			ActorID: r.id,
			Name:    m.Name,
			Class:   m.Class,
			Points:  pts,
			Total:   r.total,
		})
	}
	if series == nil {
		series = []TimelinePlayerSeries{}
	}
	return &TimelinePlayers{BucketMs: bucketMs, Series: series}, nil
}

// pgxRows is the subset of pgx.Rows used here (avoids importing pgx in signatures).
type pgxRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
	Close()
}

// CombatEventRow is one resolved event for the raw event list.
type CombatEventRow struct {
	ID             int64      `json:"id"`
	OffsetMs       int        `json:"offsetMs"`
	EventType      int16      `json:"eventType"`
	SourceID       *uuid.UUID `json:"sourceId,omitempty"`
	SourceName     string     `json:"sourceName,omitempty"`
	SourceClass    *string    `json:"sourceClass,omitempty"`
	SourceSpec     *string    `json:"sourceSpec,omitempty"`
	TargetID       *uuid.UUID `json:"targetId,omitempty"`
	TargetName     string     `json:"targetName,omitempty"`
	TargetClass    *string    `json:"targetClass,omitempty"`
	TargetSpec     *string    `json:"targetSpec,omitempty"`
	SpellID        int        `json:"spellId"`
	SpellName      string     `json:"spellName,omitempty"`
	Amount         int        `json:"amount"`
	Overkill       int        `json:"overkill"`
	Overheal       int        `json:"overheal"`
	Absorbed       int        `json:"absorbed"`
	Flags          int        `json:"flags"`
	MissType       *int16     `json:"missType,omitempty"`
	Extra          int        `json:"extra"`
	ExtraSpellName string     `json:"extraSpellName,omitempty"`
}

// CombatEventList is a paginated event list response.
type CombatEventList struct {
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Events []CombatEventRow `json:"events"`
}

// EventTypeFilter maps API filter strings to event_type values.
func EventTypeFilter(filter string) []int16 {
	switch filter {
	case "damage":
		return []int16{parser.EventDamage}
	case "heal", "healing":
		return []int16{parser.EventHeal}
	case "miss":
		return []int16{parser.EventMiss}
	case "death":
		return []int16{parser.EventDeath}
	case "summon":
		return []int16{parser.EventSummon}
	case "aura", "buffs", "debuffs":
		return []int16{parser.EventAuraApplied, parser.EventAuraRemoved, parser.EventAuraRefresh}
	case "interrupt", "interrupts":
		return []int16{parser.EventInterrupt}
	case "dispel", "dispels":
		return []int16{parser.EventDispel}
	default:
		return nil
	}
}

// ListCombatEvents returns a paginated, name-resolved event list for a fight.
func (s *Store) ListCombatEvents(ctx context.Context, fightID uuid.UUID, limit, offset int, typeFilter string, actorID *uuid.UUID) (*CombatEventList, error) {
	fight, err := s.GetFight(ctx, fightID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	types := EventTypeFilter(typeFilter)

	countArgs := []any{fightID}
	countWhere := "e.fight_id = $1"
	cn := 2
	if len(types) > 0 {
		countWhere += fmt.Sprintf(" AND e.event_type = ANY($%d)", cn)
		countArgs = append(countArgs, types)
		cn++
	}
	if actorID != nil {
		countWhere += fmt.Sprintf(" AND (e.source_actor_id = $%d OR e.target_actor_id = $%d)", cn, cn)
		countArgs = append(countArgs, *actorID)
	}

	var total int
	if err := s.Pool.QueryRow(ctx, "SELECT COUNT(*)::int FROM combat_events e WHERE "+countWhere, countArgs...).Scan(&total); err != nil {
		return nil, err
	}

	args := []any{fightID, fight.UploadID}
	where := "e.fight_id = $1"
	n := 3
	if len(types) > 0 {
		where += fmt.Sprintf(" AND e.event_type = ANY($%d)", n)
		args = append(args, types)
		n++
	}
	if actorID != nil {
		where += fmt.Sprintf(" AND (e.source_actor_id = $%d OR e.target_actor_id = $%d)", n, n)
		args = append(args, *actorID)
		n++
	}
	limitPos := n
	offsetPos := n + 1
	args = append(args, limit, offset)

	q := fmt.Sprintf(`
		SELECT e.id, e.offset_ms, e.event_type,
		       e.source_actor_id, COALESCE(src.name, ''), src.class, src.spec,
		       e.target_actor_id, COALESCE(tgt.name, ''), tgt.class, tgt.spec,
		       e.spell_id, COALESCE(ab.name, ''),
		       e.amount, e.overkill, e.overheal, e.absorbed, e.flags, e.miss_type, e.extra,
		       COALESCE(ex.name, '')
		FROM combat_events e
		LEFT JOIN actors src ON src.id = e.source_actor_id
		LEFT JOIN actors tgt ON tgt.id = e.target_actor_id
		LEFT JOIN abilities ab ON ab.upload_id = $2 AND ab.spell_id = e.spell_id
		LEFT JOIN abilities ex ON ex.upload_id = $2 AND ex.spell_id = e.extra AND e.extra > 0
		WHERE %s
		ORDER BY e.offset_ms ASC, e.id ASC
		LIMIT $%d OFFSET $%d`, where, limitPos, offsetPos)

	eventRows, err := s.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer eventRows.Close()

	out := make([]CombatEventRow, 0, limit)
	for eventRows.Next() {
		var r CombatEventRow
		var srcName, tgtName, spellName, extraName string
		if err := eventRows.Scan(
			&r.ID, &r.OffsetMs, &r.EventType,
			&r.SourceID, &srcName, &r.SourceClass, &r.SourceSpec,
			&r.TargetID, &tgtName, &r.TargetClass, &r.TargetSpec,
			&r.SpellID, &spellName,
			&r.Amount, &r.Overkill, &r.Overheal, &r.Absorbed, &r.Flags, &r.MissType, &r.Extra,
			&extraName,
		); err != nil {
			return nil, err
		}
		r.SourceName = srcName
		r.TargetName = tgtName
		r.SpellName = spellName
		r.ExtraSpellName = extraName
		out = append(out, r)
	}
	if err := eventRows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []CombatEventRow{}
	}
	return &CombatEventList{Total: total, Limit: limit, Offset: offset, Events: out}, nil
}
