package ingest

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"

	"rg-logs/internal/db"
	"rg-logs/internal/parser"
)

type Job struct {
	UploadID uuid.UUID
	Path     string
}

type Worker struct {
	store    *db.Store
	queue    chan Job
	wg       sync.WaitGroup
	uploadDir string
}

func NewWorker(store *db.Store, uploadDir string, queueSize int) *Worker {
	if queueSize < 1 {
		queueSize = 8
	}
	return &Worker{
		store:     store,
		queue:     make(chan Job, queueSize),
		uploadDir: uploadDir,
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
	path := job.Path
	if path == "" {
		path = filepath.Join(w.uploadDir, job.UploadID.String()+".txt")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
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
				ActorID:   actorID,
				SpellID:   sp.SpellID,
				SpellName: sp.SpellName,
				School:    sp.School,
				Metric:    string(sp.Metric),
				Total:     sp.Total,
				Hits:      sp.Hits,
				Crits:     sp.Crits,
				Ticks:     sp.Ticks,
			})
		}

		if pf.ParticipantCount == 0 {
			pf.ParticipantCount = len(pf.ActorIDs)
		}
		fights = append(fights, pf)
	}

	return w.store.PersistParseResult(ctx, job.UploadID, actors, fights)
}
