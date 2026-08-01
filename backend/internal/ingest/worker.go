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
	"rg-logs/internal/parser"
	"rg-logs/internal/storage"
	"rg-logs/internal/wow"
)

type Job struct {
	UploadID    uuid.UUID
	StoragePath string
}

type Worker struct {
	store   *db.Store
	storage *storage.Client
	queue   chan Job
	wg      sync.WaitGroup
}

func NewWorker(store *db.Store, storageClient *storage.Client, queueSize int) *Worker {
	if queueSize < 1 {
		queueSize = 8
	}
	return &Worker{
		store:   store,
		storage: storageClient,
		queue:   make(chan Job, queueSize),
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
	path := job.StoragePath
	if path == "" {
		upload, err := w.store.GetUpload(ctx, job.UploadID)
		if err != nil {
			return fmt.Errorf("load upload: %w", err)
		}
		path = upload.StoragePath
	}

	rc, err := w.storage.Download(ctx, path)
	if err != nil {
		return fmt.Errorf("download log: %w", err)
	}
	defer rc.Close()

	tmp, err := os.CreateTemp("", "rglogs-*.txt")
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

	f, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("reopen temp: %w", err)
	}
	defer f.Close()

	result, err := parser.Parse(f)
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
		cls := string(wow.DetectClass(spellTotalsByGUID[a.GUID]))
		if cls != "" {
			a.Class = &cls
		}
	}
	classByGUID := make(map[string]*string, len(actors))
	for i := range actors {
		if actors[i].IsPlayer && actors[i].Class != nil {
			classByGUID[actors[i].GUID] = actors[i].Class
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
	}

	fights := make([]db.PersistedFight, 0, len(result.Fights))
	for _, fr := range result.Fights {
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

		fights = append(fights, pf)
	}

	if err := w.store.PersistParseResult(ctx, job.UploadID, actors, fights); err != nil {
		return err
	}

	upload, err := w.store.GetUpload(ctx, job.UploadID)
	if err != nil {
		return fmt.Errorf("reload upload: %w", err)
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
