package ingest

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/google/uuid"

	"rg-logs/internal/db"
	"rg-logs/internal/logarchive"
	"rg-logs/internal/parser"
	"rg-logs/internal/storage"
	"rg-logs/internal/wow"
	"rg-logs/internal/wow/rgprofile"
)

type Job struct {
	UploadID    uuid.UUID
	StoragePath string
}

type Worker struct {
	store     *db.Store
	storage   *storage.Client
	queue     chan Job
	wg        sync.WaitGroup
	rgProfile *rgprofile.Client
}

func NewWorker(store *db.Store, storageClient *storage.Client, queueSize int) *Worker {
	if queueSize < 1 {
		queueSize = 8
	}
	return &Worker{
		store:     store,
		storage:   storageClient,
		queue:     make(chan Job, queueSize),
		rgProfile: rgprofile.NewClient(),
	}
}

func (w *Worker) Start(workers int) {
	if workers < 1 {
		workers = 1
	}
	for i := 0; i < workers; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for job := range w.queue {
				w.process(job)
			}
		}()
	}
}

func (w *Worker) Stop() {
	close(w.queue)
	w.wg.Wait()
}

func (w *Worker) Enqueue(job Job) {
	w.queue <- job
}

func (w *Worker) process(job Job) {
	ctx := context.Background()
	log.Printf("ingest: processing upload %s", job.UploadID)

	if err := w.store.SetUploadStatus(ctx, job.UploadID, db.StatusProcessing, nil); err != nil {
		log.Printf("ingest: set processing: %v", err)
		return
	}

	if err := w.runParse(ctx, job); err != nil {
		msg := err.Error()
		log.Printf("ingest: upload %s failed: %v", job.UploadID, err)
		_ = w.store.SetUploadStatus(ctx, job.UploadID, db.StatusFailed, &msg)
		return
	}

	if err := w.store.SetUploadStatus(ctx, job.UploadID, db.StatusReady, nil); err != nil {
		log.Printf("ingest: set ready: %v", err)
		return
	}
	log.Printf("ingest: upload %s ready", job.UploadID)
}

func (w *Worker) runParse(ctx context.Context, job Job) error {
	upload, err := w.store.GetUpload(ctx, job.UploadID)
	if err != nil {
		return fmt.Errorf("load upload: %w", err)
	}
	path := job.StoragePath
	if path == "" {
		path = upload.StoragePath
	}
	includeTrash := upload.IncludeTrash

	rc, err := w.storage.Download(ctx, path)
	if err != nil {
		return fmt.Errorf("download log: %w", err)
	}
	defer rc.Close()

	tmpPattern := "rglogs-*.txt"
	if strings.HasSuffix(strings.ToLower(path), ".zip") {
		tmpPattern = "rglogs-*.zip"
	}
	tmp, err := os.CreateTemp("", tmpPattern)
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmp, rc); err != nil {
		tmp.Close()
		return fmt.Errorf("buffer log: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}

	logReader, err := logarchive.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer logReader.Close()

	result, err := parser.Parse(logReader)
	if err != nil {
		return fmt.Errorf("parse: %w", err)
	}

	actors := make([]db.PersistedActor, 0, len(result.Actors))
	guidToID := make(map[string]uuid.UUID, len(result.Actors))
	for _, a := range result.Actors {
		id := uuid.New()
		guidToID[a.GUID] = id
		var owner *string
		if a.OwnerGUID != "" {
			o := a.OwnerGUID
			owner = &o
		}
		actors = append(actors, db.PersistedActor{
			ID:        id,
			GUID:      a.GUID,
			Name:      a.Name,
			Flags:     a.Flags,
			IsPlayer:  a.IsPlayer,
			OwnerGUID: owner,
		})
	}

	spellTotalsByGUID := make(map[string]map[int]int64)
	for _, fr := range result.Fights {
		for _, sp := range fr.Spells {
			m := spellTotalsByGUID[sp.ActorGUID]
			if m == nil {
				m = make(map[int]int64)
				spellTotalsByGUID[sp.ActorGUID] = m
			}
			m[sp.SpellID] += sp.Total
		}
	}
	for i := range actors {
		a := &actors[i]
		if !a.IsPlayer {
			continue
		}
		totals := spellTotalsByGUID[a.GUID]
		cls := string(wow.DetectClass(totals))
		if cls != "" {
			a.Class = &cls
		}
		spec := string(wow.DetectSpec(totals))
		if spec != "" {
			a.Spec = &spec
		}
	}
	classByGUID := make(map[string]*string, len(actors))
	specByGUID := make(map[string]*string, len(actors))
	for i := range actors {
		if !actors[i].IsPlayer {
			continue
		}
		if actors[i].Class != nil {
			classByGUID[actors[i].GUID] = actors[i].Class
		}
		if actors[i].Spec != nil {
			specByGUID[actors[i].GUID] = actors[i].Spec
		}
	}
	for i := range actors {
		a := &actors[i]
		if a.IsPlayer || a.OwnerGUID == nil {
			continue
		}
		if cls, ok := classByGUID[*a.OwnerGUID]; ok {
			a.Class = cls
		}
		if spec, ok := specByGUID[*a.OwnerGUID]; ok {
			a.Spec = spec
		}
	}

	w.enrichGearScores(ctx, actors)

	abilities := make([]db.PersistedAbility, 0, len(result.Abilities))
	for _, ab := range result.Abilities {
		abilities = append(abilities, db.PersistedAbility{
			SpellID: ab.SpellID,
			Name:    ab.Name,
			School:  ab.School,
		})
	}

	actorIDPtr := func(guid string) *uuid.UUID {
		if guid == "" {
			return nil
		}
		id, ok := guidToID[guid]
		if !ok {
			return nil
		}
		return &id
	}

	fights := make([]db.PersistedFight, 0, len(result.Fights))
	for _, fr := range result.Fights {
		if !includeTrash && !wow.IsKnownBoss(fr.Title) {
			continue
		}

		pf := db.PersistedFight{
			ID:               fr.ID,
			StartTs:          fr.StartTs,
			EndTs:            fr.EndTs,
			DurationMs:       fr.DurationMs,
			Title:            fr.Title,
			Kill:             fr.Kill,
			ParticipantCount: fr.ParticipantCount,
			ActorIDs:         make(map[uuid.UUID]struct{}),
		}

		for guid, agg := range fr.Actors {
			actorID, ok := guidToID[guid]
			if !ok {
				continue
			}
			if agg.DamageDone == 0 && agg.HealingDone == 0 && agg.DamageTaken == 0 {
				continue
			}
			pf.ActorIDs[actorID] = struct{}{}
			pf.Stats = append(pf.Stats, db.PersistedFightStat{
				ActorID:      actorID,
				DamageDone:   agg.DamageDone,
				HealingDone:  agg.HealingDone,
				Overheal:     agg.Overheal,
				DamageTaken:  agg.DamageTaken,
				ActiveTimeMs: agg.ActiveTimeMs(),
			})
		}

		for _, sp := range fr.Spells {
			actorID, ok := guidToID[sp.ActorGUID]
			if !ok {
				continue
			}
			pf.ActorIDs[actorID] = struct{}{}
			pf.Spells = append(pf.Spells, db.PersistedSpellStat{
				ActorID:       actorID,
				SpellID:       sp.SpellID,
				SpellName:     sp.SpellName,
				School:        sp.School,
				Metric:        string(sp.Metric),
				Total:         sp.Total,
				Hits:          sp.Hits,
				Crits:         sp.Crits,
				Ticks:         sp.Ticks,
				Misses:        sp.Misses,
				Glancing:      sp.Glancing,
				NormalHits:    sp.Normal.Hits,
				NormalTotal:   sp.Normal.Total,
				NormalMin:     sp.Normal.Min,
				NormalMax:     sp.Normal.Max,
				CritTotal:     sp.Crit.Total,
				CritMin:       sp.Crit.Min,
				CritMax:       sp.Crit.Max,
				GlancingTotal: sp.Glance.Total,
				GlancingMin:   sp.Glance.Min,
				GlancingMax:   sp.Glance.Max,
			})
		}

		pf.Events = make([]db.PersistedCombatEvent, 0, len(fr.Events))
		for _, ev := range fr.Events {
			src := actorIDPtr(ev.SourceGUID)
			tgt := actorIDPtr(ev.TargetGUID)
			if src != nil {
				pf.ActorIDs[*src] = struct{}{}
			}
			if tgt != nil {
				pf.ActorIDs[*tgt] = struct{}{}
			}
			pf.Events = append(pf.Events, db.PersistedCombatEvent{
				Ts:            ev.Ts,
				OffsetMs:      ev.OffsetMs,
				EventType:     ev.EventType,
				SourceActorID: src,
				TargetActorID: tgt,
				SpellID:       ev.SpellID,
				Amount:        ev.Amount,
				Overkill:      ev.Overkill,
				Overheal:      ev.Overheal,
				Absorbed:      ev.Absorbed,
				Resisted:      ev.Resisted,
				Blocked:       ev.Blocked,
				Flags:         ev.Flags,
				MissType:      ev.MissType,
				Extra:         ev.Extra,
			})
		}

		fights = append(fights, pf)
	}

	if err := w.store.PersistParseResult(ctx, job.UploadID, actors, abilities, fights); err != nil {
		return err
	}

	if strings.TrimSpace(upload.Name) == "" {
		titles := make([]string, 0, len(fights))
		for _, f := range fights {
			titles = append(titles, f.Title)
		}
		name := wow.DetectInstance(titles)
		if name == "" {
			name = upload.Filename
		}
		if err := w.store.SetUploadNameIfEmpty(ctx, job.UploadID, name); err != nil {
			return fmt.Errorf("set name: %w", err)
		}
	}
	return nil
}

// enrichGearScores fetches Rising Gods profiles and sets GearScoreLite on player actors.
// Failures are logged; missing profiles leave gear_score unset.
func (w *Worker) enrichGearScores(ctx context.Context, actors []db.PersistedActor) {
	if w.rgProfile == nil {
		return
	}

	type job struct{ idx int }
	var players []job
	for i := range actors {
		if actors[i].IsPlayer && strings.TrimSpace(actors[i].Name) != "" {
			players = append(players, job{idx: i})
		}
	}
	if len(players) == 0 {
		return
	}

	const workers = 3
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, p := range players {
		p := p
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			name := actors[p.idx].Name
			ch, err := w.rgProfile.Fetch(ctx, name)
			if err != nil {
				log.Printf("ingest: gearscore %q: %v", name, err)
				return
			}
			if ch == nil || ch.GearScore <= 0 {
				return
			}
			gs := ch.GearScore
			mu.Lock()
			actors[p.idx].GearScore = &gs
			// Prefer armory class when combat-log inference missed.
			if actors[p.idx].Class == nil && ch.Class != "" {
				cls := string(ch.Class)
				actors[p.idx].Class = &cls
			}
			mu.Unlock()
		}()
	}
	wg.Wait()
}
