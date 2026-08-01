package db

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"rg-logs/internal/wow"
)

type UploadStatus string

const (
	StatusPending    UploadStatus = "pending"
	StatusProcessing UploadStatus = "processing"
	StatusReady      UploadStatus = "ready"
	StatusFailed     UploadStatus = "failed"
)

type Upload struct {
	ID          uuid.UUID    `json:"id"`
	Filename    string       `json:"filename"`
	SizeBytes   int64        `json:"sizeBytes"`
	Status      UploadStatus `json:"status"`
	Error       *string      `json:"error"`
	CreatedAt   time.Time    `json:"createdAt"`
	ProcessedAt *time.Time   `json:"processedAt"`
}

type Fight struct {
	ID                uuid.UUID `json:"id"`
	UploadID          uuid.UUID `json:"uploadId"`
	StartTs           time.Time `json:"startTs"`
	EndTs             time.Time `json:"endTs"`
	DurationMs        int64     `json:"durationMs"`
	Title             string    `json:"title"`
	Kill              bool      `json:"kill"`
	ParticipantCount  int       `json:"participantCount"`
}

type Actor struct {
	ID        uuid.UUID `json:"id"`
	UploadID  uuid.UUID `json:"uploadId"`
	GUID      string    `json:"guid"`
	Name      string    `json:"name"`
	Flags     int64     `json:"flags"`
	IsPlayer  bool      `json:"isPlayer"`
	OwnerGUID *string   `json:"ownerGuid"`
	Class     *string   `json:"class,omitempty"`
}

type ActorStat struct {
	FightID      uuid.UUID `json:"fightId"`
	ActorID      uuid.UUID `json:"actorId"`
	Name         string    `json:"name"`
	GUID         string    `json:"guid"`
	IsPlayer     bool      `json:"isPlayer"`
	OwnerGUID    *string   `json:"ownerGuid,omitempty"`
	Class        *string   `json:"class,omitempty"`
	DamageDone   int64     `json:"damageDone"`
	HealingDone  int64     `json:"healingDone"`
	Overheal     int64     `json:"overheal"`
	DamageTaken  int64     `json:"damageTaken"`
	ActiveTimeMs int64     `json:"activeTimeMs"`
	DPS          float64   `json:"dps"`
	HPS          float64   `json:"hps"`
}

type SpellStat struct {
	SpellID        int    `json:"spellId"`
	SpellName      string `json:"spellName"`
	School         int    `json:"school"`
	Metric         string `json:"metric"`
	Total          int64  `json:"total"`
	Hits           int    `json:"hits"`
	Crits          int    `json:"crits"`
	Ticks          int    `json:"ticks"`
	Misses         int    `json:"misses"`
	Glancing       int    `json:"glancing"`
	NormalHits     int    `json:"normalHits"`
	NormalTotal    int64  `json:"normalTotal"`
	NormalMin      int64  `json:"normalMin"`
	NormalMax      int64  `json:"normalMax"`
	CritTotal      int64  `json:"critTotal"`
	CritMin        int64  `json:"critMin"`
	CritMax        int64  `json:"critMax"`
	GlancingTotal int64  `json:"glancingTotal"`
	GlancingMin   int64  `json:"glancingMin"`
	GlancingMax   int64  `json:"glancingMax"`
	// Pet is true when this row aggregates a pet/summon by name under its owner.
	Pet bool `json:"pet,omitempty"`
}

type Store struct {
	Pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{Pool: pool}
}

func (s *Store) CreateUpload(ctx context.Context, id uuid.UUID, filename string, sizeBytes int64) (*Upload, error) {
	u := &Upload{
		ID:        id,
		Filename:  filename,
		SizeBytes: sizeBytes,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	}
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO uploads (id, filename, size_bytes, status, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Filename, u.SizeBytes, u.Status, u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (s *Store) ListUploads(ctx context.Context) ([]Upload, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, filename, size_bytes, status, error, created_at, processed_at
		FROM uploads
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Upload
	for rows.Next() {
		var u Upload
		if err := rows.Scan(&u.ID, &u.Filename, &u.SizeBytes, &u.Status, &u.Error, &u.CreatedAt, &u.ProcessedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	if out == nil {
		out = []Upload{}
	}
	return out, rows.Err()
}

func (s *Store) GetUpload(ctx context.Context, id uuid.UUID) (*Upload, error) {
	var u Upload
	err := s.Pool.QueryRow(ctx, `
		SELECT id, filename, size_bytes, status, error, created_at, processed_at
		FROM uploads WHERE id=$1`, id,
	).Scan(&u.ID, &u.Filename, &u.SizeBytes, &u.Status, &u.Error, &u.CreatedAt, &u.ProcessedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) SetUploadStatus(ctx context.Context, id uuid.UUID, status UploadStatus, errMsg *string) error {
	if status == StatusReady || status == StatusFailed {
		_, err := s.Pool.Exec(ctx, `
			UPDATE uploads SET status=$2, error=$3, processed_at=NOW() WHERE id=$1`,
			id, status, errMsg,
		)
		return err
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE uploads SET status=$2, error=$3 WHERE id=$1`,
		id, status, errMsg,
	)
	return err
}

func (s *Store) ListFightsByUpload(ctx context.Context, uploadID uuid.UUID) ([]Fight, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT f.id, f.upload_id, f.start_ts, f.end_ts, f.duration_ms, f.title, f."kill",
		       (
		         SELECT COUNT(*)::int
		         FROM actor_stats s
		         JOIN actors a ON a.id = s.actor_id
		         WHERE s.fight_id = f.id
		           AND a.is_player
		           AND (s.damage_done > 0 OR s.healing_done > 0 OR s.damage_taken > 0)
		       ) AS participant_count
		FROM fights f
		WHERE f.upload_id=$1
		ORDER BY f.start_ts ASC`, uploadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Fight
	for rows.Next() {
		var f Fight
		if err := rows.Scan(&f.ID, &f.UploadID, &f.StartTs, &f.EndTs, &f.DurationMs, &f.Title, &f.Kill, &f.ParticipantCount); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	if out == nil {
		out = []Fight{}
	}
	return out, rows.Err()
}

func (s *Store) GetFight(ctx context.Context, id uuid.UUID) (*Fight, error) {
	var f Fight
	err := s.Pool.QueryRow(ctx, `
		SELECT f.id, f.upload_id, f.start_ts, f.end_ts, f.duration_ms, f.title, f."kill",
		       (
		         SELECT COUNT(*)::int
		         FROM actor_stats s
		         JOIN actors a ON a.id = s.actor_id
		         WHERE s.fight_id = f.id
		           AND a.is_player
		           AND (s.damage_done > 0 OR s.healing_done > 0 OR s.damage_taken > 0)
		       ) AS participant_count
		FROM fights f
		WHERE f.id=$1`, id,
	).Scan(&f.ID, &f.UploadID, &f.StartTs, &f.EndTs, &f.DurationMs, &f.Title, &f.Kill, &f.ParticipantCount)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (s *Store) ListActorStats(ctx context.Context, fightID uuid.UUID) ([]ActorStat, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT a.id, a.name, a.guid, a.is_player, a.owner_guid, a.class,
		       s.damage_done, s.healing_done, s.overheal, s.damage_taken, s.active_time_ms,
		       f.duration_ms
		FROM actor_stats s
		JOIN actors a ON a.id = s.actor_id
		JOIN fights f ON f.id = s.fight_id
		WHERE s.fight_id=$1
		ORDER BY s.damage_done DESC, s.healing_done DESC, s.damage_taken DESC`, fightID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ActorStat
	for rows.Next() {
		var st ActorStat
		var durationMs int64
		st.FightID = fightID
		if err := rows.Scan(
			&st.ActorID, &st.Name, &st.GUID, &st.IsPlayer, &st.OwnerGUID, &st.Class,
			&st.DamageDone, &st.HealingDone, &st.Overheal, &st.DamageTaken, &st.ActiveTimeMs,
			&durationMs,
		); err != nil {
			return nil, err
		}
		secs := float64(durationMs) / 1000.0
		if secs > 0 {
			st.DPS = float64(st.DamageDone) / secs
			effective := st.HealingDone - st.Overheal
			if effective < 0 {
				effective = 0
			}
			st.HPS = float64(effective) / secs
		}
		out = append(out, st)
	}
	if out == nil {
		out = []ActorStat{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := s.fillMissingClasses(ctx, fightID, out); err != nil {
		return nil, err
	}
	return out, nil
}

// fillMissingClasses derives class from this fight's spell_stats when actors.class is empty
// (e.g. uploads persisted before class detection existed). Pets inherit their owner's class.
func (s *Store) fillMissingClasses(ctx context.Context, fightID uuid.UUID, stats []ActorStat) error {
	needDetect := false
	for _, st := range stats {
		if st.IsPlayer && (st.Class == nil || *st.Class == "") {
			needDetect = true
			break
		}
	}
	if !needDetect {
		s.propagateOwnerClass(stats)
		return nil
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT actor_id, spell_id, SUM(total)::bigint
		FROM spell_stats
		WHERE fight_id=$1
		GROUP BY actor_id, spell_id`, fightID)
	if err != nil {
		return err
	}
	defer rows.Close()

	byActor := make(map[uuid.UUID]map[int]int64)
	for rows.Next() {
		var actorID uuid.UUID
		var spellID int
		var total int64
		if err := rows.Scan(&actorID, &spellID, &total); err != nil {
			return err
		}
		m := byActor[actorID]
		if m == nil {
			m = make(map[int]int64)
			byActor[actorID] = m
		}
		m[spellID] += total
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range stats {
		st := &stats[i]
		if !st.IsPlayer || (st.Class != nil && *st.Class != "") {
			continue
		}
		cls := string(wow.DetectClass(byActor[st.ActorID]))
		if cls == "" {
			continue
		}
		st.Class = &cls
	}

	s.propagateOwnerClass(stats)
	return nil
}

func (s *Store) propagateOwnerClass(stats []ActorStat) {
	byGUID := make(map[string]*string, len(stats))
	for i := range stats {
		if stats[i].IsPlayer && stats[i].Class != nil && *stats[i].Class != "" {
			byGUID[stats[i].GUID] = stats[i].Class
		}
	}
	for i := range stats {
		st := &stats[i]
		if st.IsPlayer || st.OwnerGUID == nil {
			continue
		}
		if st.Class != nil && *st.Class != "" {
			continue
		}
		if cls, ok := byGUID[*st.OwnerGUID]; ok {
			st.Class = cls
		}
	}
}

func scanSpellStat(scan func(dest ...any) error) (SpellStat, error) {
	var sp SpellStat
	err := scan(
		&sp.SpellID, &sp.SpellName, &sp.School, &sp.Metric,
		&sp.Total, &sp.Hits, &sp.Crits, &sp.Ticks, &sp.Misses, &sp.Glancing,
		&sp.NormalHits, &sp.NormalTotal, &sp.NormalMin, &sp.NormalMax,
		&sp.CritTotal, &sp.CritMin, &sp.CritMax,
		&sp.GlancingTotal, &sp.GlancingMin, &sp.GlancingMax,
	)
	return sp, err
}

const spellStatColumns = `
	spell_id, spell_name, school, metric, total, hits, crits, ticks, misses, glancing,
	normal_hits, normal_total, normal_min, normal_max,
	crit_total, crit_min, crit_max,
	glancing_total, glancing_min, glancing_max`

func (s *Store) ListSpellStats(ctx context.Context, fightID, actorID uuid.UUID) ([]SpellStat, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+spellStatColumns+`
		FROM spell_stats
		WHERE fight_id=$1 AND actor_id=$2`, fightID, actorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SpellStat
	for rows.Next() {
		sp, err := scanSpellStat(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, sp)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	pets, err := s.listPetSpellAggregates(ctx, fightID, actorID)
	if err != nil {
		return nil, err
	}
	out = append(out, pets...)

	sort.Slice(out, func(i, j int) bool { return out[i].Total > out[j].Total })
	if out == nil {
		out = []SpellStat{}
	}
	return out, nil
}

// listPetSpellAggregates rolls all spells of owned pets/summons into one row per
// pet name + metric (e.g. three Treants → one "Treant" damage line).
func (s *Store) listPetSpellAggregates(ctx context.Context, fightID, actorID uuid.UUID) ([]SpellStat, error) {
	var ownerGUID string
	var isPlayer bool
	err := s.Pool.QueryRow(ctx, `
		SELECT a.guid, a.is_player
		FROM actors a
		JOIN fight_actors fa ON fa.actor_id = a.id
		WHERE fa.fight_id=$1 AND a.id=$2`, fightID, actorID,
	).Scan(&ownerGUID, &isPlayer)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if !isPlayer || ownerGUID == "" {
		return nil, nil
	}

	rows, err := s.Pool.Query(ctx, `
		SELECT
			0 AS spell_id,
			p.name AS spell_name,
			0 AS school,
			ss.metric,
			COALESCE(SUM(ss.total), 0),
			COALESCE(SUM(ss.hits), 0)::int,
			COALESCE(SUM(ss.crits), 0)::int,
			COALESCE(SUM(ss.ticks), 0)::int,
			COALESCE(SUM(ss.misses), 0)::int,
			COALESCE(SUM(ss.glancing), 0)::int,
			COALESCE(SUM(ss.normal_hits), 0)::int,
			COALESCE(SUM(ss.normal_total), 0),
			COALESCE(MIN(ss.normal_min) FILTER (WHERE ss.normal_hits > 0), 0),
			COALESCE(MAX(ss.normal_max) FILTER (WHERE ss.normal_hits > 0), 0),
			COALESCE(SUM(ss.crit_total), 0),
			COALESCE(MIN(ss.crit_min) FILTER (WHERE ss.crits > 0), 0),
			COALESCE(MAX(ss.crit_max) FILTER (WHERE ss.crits > 0), 0),
			COALESCE(SUM(ss.glancing_total), 0),
			COALESCE(MIN(ss.glancing_min) FILTER (WHERE ss.glancing > 0), 0),
			COALESCE(MAX(ss.glancing_max) FILTER (WHERE ss.glancing > 0), 0)
		FROM spell_stats ss
		JOIN actors p ON p.id = ss.actor_id
		JOIN fights f ON f.id = ss.fight_id
		WHERE ss.fight_id = $1
		  AND p.upload_id = f.upload_id
		  AND p.owner_guid = $2
		  AND p.is_player = FALSE
		GROUP BY p.name, ss.metric
		HAVING SUM(ss.total) > 0 OR SUM(ss.hits) > 0 OR SUM(ss.misses) > 0`,
		fightID, ownerGUID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SpellStat
	for rows.Next() {
		sp, err := scanSpellStat(rows.Scan)
		if err != nil {
			return nil, err
		}
		sp.Pet = true
		out = append(out, sp)
	}
	return out, rows.Err()
}

// PersistParseResult writes actors, fights and stats for an upload in one transaction.
func (s *Store) PersistParseResult(ctx context.Context, uploadID uuid.UUID, actors []PersistedActor, fights []PersistedFight) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	for _, a := range actors {
		_, err := tx.Exec(ctx, `
			INSERT INTO actors (id, upload_id, guid, name, flags, is_player, owner_guid, class)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (upload_id, guid) DO UPDATE
			SET name=EXCLUDED.name, flags=EXCLUDED.flags, is_player=EXCLUDED.is_player,
			    owner_guid=COALESCE(EXCLUDED.owner_guid, actors.owner_guid),
			    class=COALESCE(EXCLUDED.class, actors.class)`,
			a.ID, uploadID, a.GUID, a.Name, a.Flags, a.IsPlayer, a.OwnerGUID, a.Class,
		)
		if err != nil {
			return fmt.Errorf("insert actor %s: %w", a.GUID, err)
		}
	}

	for _, f := range fights {
		_, err := tx.Exec(ctx, `
			INSERT INTO fights (id, upload_id, start_ts, end_ts, duration_ms, title, "kill", participant_count)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
			f.ID, uploadID, f.StartTs, f.EndTs, f.DurationMs, f.Title, f.Kill, f.ParticipantCount,
		)
		if err != nil {
			return fmt.Errorf("insert fight: %w", err)
		}

		for actorID := range f.ActorIDs {
			_, err := tx.Exec(ctx, `
				INSERT INTO fight_actors (fight_id, actor_id) VALUES ($1,$2)
				ON CONFLICT DO NOTHING`, f.ID, actorID)
			if err != nil {
				return fmt.Errorf("insert fight_actor: %w", err)
			}
		}

		for _, st := range f.Stats {
			_, err := tx.Exec(ctx, `
				INSERT INTO actor_stats (fight_id, actor_id, damage_done, healing_done, overheal, damage_taken, active_time_ms)
				VALUES ($1,$2,$3,$4,$5,$6,$7)`,
				f.ID, st.ActorID, st.DamageDone, st.HealingDone, st.Overheal, st.DamageTaken, st.ActiveTimeMs,
			)
			if err != nil {
				return fmt.Errorf("insert actor_stats: %w", err)
			}
		}

		for _, sp := range f.Spells {
			minAmount, maxAmount := overallMinMax(sp)
			_, err := tx.Exec(ctx, `
				INSERT INTO spell_stats (
					fight_id, actor_id, spell_id, spell_name, school, metric,
					total, hits, crits, ticks, min_amount, max_amount, misses, glancing,
					normal_hits, normal_total, normal_min, normal_max,
					crit_total, crit_min, crit_max,
					glancing_total, glancing_min, glancing_max
				)
				VALUES (
					$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,
					$15,$16,$17,$18,$19,$20,$21,$22,$23,$24
				)`,
				f.ID, sp.ActorID, sp.SpellID, sp.SpellName, sp.School, sp.Metric,
				sp.Total, sp.Hits, sp.Crits, sp.Ticks, minAmount, maxAmount, sp.Misses, sp.Glancing,
				sp.NormalHits, sp.NormalTotal, sp.NormalMin, sp.NormalMax,
				sp.CritTotal, sp.CritMin, sp.CritMax,
				sp.GlancingTotal, sp.GlancingMin, sp.GlancingMax,
			)
			if err != nil {
				return fmt.Errorf("insert spell_stats: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

type PersistedActor struct {
	ID        uuid.UUID
	GUID      string
	Name      string
	Flags     int64
	IsPlayer  bool
	OwnerGUID *string
	Class     *string
}

type PersistedFightStat struct {
	ActorID      uuid.UUID
	DamageDone   int64
	HealingDone  int64
	Overheal     int64
	DamageTaken  int64
	ActiveTimeMs int64
}

type PersistedSpellStat struct {
	ActorID       uuid.UUID
	SpellID       int
	SpellName     string
	School        int
	Metric        string
	Total         int64
	Hits          int
	Crits         int
	Ticks         int
	Misses        int
	Glancing      int
	NormalHits    int
	NormalTotal   int64
	NormalMin     int64
	NormalMax     int64
	CritTotal     int64
	CritMin       int64
	CritMax       int64
	GlancingTotal int64
	GlancingMin   int64
	GlancingMax   int64
}

func overallMinMax(sp PersistedSpellStat) (min, max int64) {
	first := true
	consider := func(hits int, lo, hi int64) {
		if hits <= 0 {
			return
		}
		if first {
			min, max = lo, hi
			first = false
			return
		}
		if lo < min {
			min = lo
		}
		if hi > max {
			max = hi
		}
	}
	consider(sp.NormalHits, sp.NormalMin, sp.NormalMax)
	consider(sp.Crits, sp.CritMin, sp.CritMax)
	consider(sp.Glancing, sp.GlancingMin, sp.GlancingMax)
	return min, max
}

type PersistedFight struct {
	ID               uuid.UUID
	StartTs          time.Time
	EndTs            time.Time
	DurationMs       int64
	Title            string
	Kill             bool
	ParticipantCount int
	ActorIDs         map[uuid.UUID]struct{}
	Stats            []PersistedFightStat
	Spells           []PersistedSpellStat
}

func IsNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
