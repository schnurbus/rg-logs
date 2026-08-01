package api

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"

	"rg-logs/internal/db"
	"rg-logs/internal/ingest"
)

type Handler struct {
	Store     *db.Store
	Worker    *ingest.Worker
	UploadDir string
}

func NewRouter(h *Handler) *fiber.App {
	app := fiber.New(fiber.Config{
		BodyLimit: 100 * 1024 * 1024,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodOptions},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
	}))

	app.Get("/api/health", h.Health)
	app.Post("/api/uploads", h.CreateUpload)
	app.Get("/api/uploads", h.ListUploads)
	app.Get("/api/uploads/:id", h.GetUpload)
	app.Get("/api/fights/:id", h.GetFight)
	app.Get("/api/fights/:id/spells", h.GetFightSpells)

	return app
}

func (h *Handler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func (h *Handler) CreateUpload(c fiber.Ctx) error {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "multipart field 'file' required")
	}

	id := uuid.New()
	if err := os.MkdirAll(h.UploadDir, 0o755); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "upload dir")
	}

	destPath := filepath.Join(h.UploadDir, id.String()+".txt")
	src, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "open upload")
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "create upload file")
	}
	written, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(destPath)
		return fiber.NewError(fiber.StatusInternalServerError, "save upload")
	}

	filename := fileHeader.Filename
	if filename == "" {
		filename = "combatlog.txt"
	}

	upload, err := h.Store.CreateUpload(c.Context(), id, filename, written)
	if err != nil {
		_ = os.Remove(destPath)
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("db: %v", err))
	}

	h.Worker.Enqueue(ingest.Job{UploadID: id, Path: destPath})
	return c.Status(fiber.StatusAccepted).JSON(upload)
}

func (h *Handler) ListUploads(c fiber.Ctx) error {
	list, err := h.Store.ListUploads(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(list)
}

func (h *Handler) GetUpload(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	upload, err := h.Store.GetUpload(c.Context(), id)
	if err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "upload not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	fights, err := h.Store.ListFightsByUpload(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"id":          upload.ID,
		"filename":    upload.Filename,
		"sizeBytes":   upload.SizeBytes,
		"status":      upload.Status,
		"error":       upload.Error,
		"createdAt":   upload.CreatedAt,
		"processedAt": upload.ProcessedAt,
		"fights":      fights,
	})
}

func (h *Handler) GetFight(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	fight, err := h.Store.GetFight(c.Context(), id)
	if err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "fight not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	participants, err := h.Store.ListActorStats(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	switch c.Query("sort", "damage") {
	case "healing", "heal":
		sort.SliceStable(participants, func(i, j int) bool {
			return participants[i].HealingDone > participants[j].HealingDone
		})
	case "taken":
		sort.SliceStable(participants, func(i, j int) bool {
			return participants[i].DamageTaken > participants[j].DamageTaken
		})
	default:
		sort.SliceStable(participants, func(i, j int) bool {
			return participants[i].DamageDone > participants[j].DamageDone
		})
	}

	return c.JSON(fiber.Map{
		"id":               fight.ID,
		"uploadId":         fight.UploadID,
		"startTs":          fight.StartTs,
		"endTs":            fight.EndTs,
		"durationMs":       fight.DurationMs,
		"title":            fight.Title,
		"kill":             fight.Kill,
		"participantCount": fight.ParticipantCount,
		"participants":     participants,
	})
}

func (h *Handler) GetFightSpells(c fiber.Ctx) error {
	fightID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid fight id")
	}
	actorStr := c.Query("actorId")
	if actorStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "actorId query required")
	}
	actorID, err := uuid.Parse(actorStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid actorId")
	}

	if _, err := h.Store.GetFight(c.Context(), fightID); err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "fight not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	spells, err := h.Store.ListSpellStats(c.Context(), fightID, actorID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(spells)
}
