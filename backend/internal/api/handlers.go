package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"

	"rg-logs/internal/auth"
	"rg-logs/internal/db"
	"rg-logs/internal/ingest"
	"rg-logs/internal/logarchive"
	"rg-logs/internal/storage"
)

type Handler struct {
	Store   *db.Store
	Worker  *ingest.Worker
	Auth    *auth.Client
	Storage *storage.Client
	// Public frontend bootstrap (safe to expose; anon key is designed for browsers).
	SupabaseURL     string
	SupabaseAnonKey string
}

// DefaultCORSOrigins allows the Vite dev server during local development.
var DefaultCORSOrigins = []string{
	"http://localhost:5173",
	"http://127.0.0.1:5173",
}

func NewRouter(h *Handler, corsOrigins []string) *fiber.App {
	origins := DefaultCORSOrigins
	if len(corsOrigins) > 0 {
		origins = corsOrigins
	}

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
		AllowOrigins: origins,
		AllowMethods: []string{
			fiber.MethodGet, fiber.MethodPost, fiber.MethodPatch,
			fiber.MethodDelete, fiber.MethodOptions,
		},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization"},
	}))

	app.Get("/api/health", h.Health)
	app.Get("/api/config", h.PublicConfig)

	authed := app.Group("/api", OptionalAuth(h.Auth))
	authed.Get("/uploads", h.ListUploads)
	authed.Get("/uploads/:id", h.GetUpload)
	authed.Get("/fights/:id", h.GetFight)
	authed.Get("/fights/:id/spells", h.GetFightSpells)
	authed.Get("/fights/:id/auras", h.GetFightAuras)
	authed.Get("/fights/:id/interrupts", h.GetFightInterrupts)
	authed.Get("/fights/:id/dispels", h.GetFightDispels)
	authed.Get("/fights/:id/timeline", h.GetFightTimeline)
	authed.Get("/fights/:id/events", h.GetFightEvents)

	write := app.Group("/api", RequireAuth(h.Auth))
	write.Post("/uploads", h.CreateUpload)
	write.Patch("/uploads/:id", h.UpdateUpload)
	write.Delete("/uploads/:id", h.DeleteUpload)

	mountSPA(app)

	return app
}

func (h *Handler) Health(c fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// PublicConfig returns browser-safe runtime settings so the SPA does not need
// build-time VITE_SUPABASE_* baked into the Docker image.
func (h *Handler) PublicConfig(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"supabaseUrl":     h.SupabaseURL,
		"supabaseAnonKey": h.SupabaseAnonKey,
	})
}

func (h *Handler) CreateUpload(c fiber.Ctx) error {
	user := UserFromContext(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "multipart field 'file' required")
	}

	src, err := fileHeader.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "open upload")
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, 100*1024*1024+1))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "read upload")
	}
	if len(data) > 100*1024*1024 {
		return fiber.NewError(fiber.StatusRequestEntityTooLarge, "file too large (max 100MB)")
	}

	filename := fileHeader.Filename
	if filename == "" {
		filename = "combatlog.txt"
	}
	if logarchive.LooksLikeZip(filename, data) {
		if !logarchive.IsZip(data) {
			return fiber.NewError(fiber.StatusBadRequest, "file looks like a zip but is not a valid zip archive")
		}
		if err := logarchive.ValidateZip(data); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
	}

	sum := sha256.Sum256(data)
	contentHash := hex.EncodeToString(sum[:])

	if existing, err := h.Store.GetUploadByUserHash(c.Context(), user.ID, contentHash); err == nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error":  "duplicate upload",
			"upload": existing,
		})
	} else if !db.IsNoRows(err) {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("db: %v", err))
	}

	isPrivate := parseBoolForm(c.FormValue("is_private"), c.FormValue("isPrivate"))
	includeTrash := parseBoolForm(c.FormValue("include_trash"), c.FormValue("includeTrash"))
	name := strings.TrimSpace(firstNonEmpty(c.FormValue("name"), c.FormValue("Name")))

	id := uuid.New()
	ext, contentType := logarchive.StorageMeta(filename, data)
	storagePath := fmt.Sprintf("%s/%s%s", user.ID.String(), id.String(), ext)

	if err := h.Storage.Upload(c.Context(), storagePath, contentType, data); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("storage: %v", err))
	}

	upload, err := h.Store.CreateUpload(c.Context(), db.CreateUploadParams{
		ID:           id,
		UserID:       user.ID,
		Name:         name,
		Filename:     filename,
		SizeBytes:    int64(len(data)),
		ContentHash:  contentHash,
		IsPrivate:    isPrivate,
		IncludeTrash: includeTrash,
		StoragePath:  storagePath,
	})
	if err != nil {
		_ = h.Storage.Delete(c.Context(), storagePath)
		if db.IsUniqueViolation(err) {
			if existing, getErr := h.Store.GetUploadByUserHash(c.Context(), user.ID, contentHash); getErr == nil {
				return c.Status(fiber.StatusConflict).JSON(fiber.Map{
					"error":  "duplicate upload",
					"upload": existing,
				})
			}
			return fiber.NewError(fiber.StatusConflict, "duplicate upload")
		}
		return fiber.NewError(fiber.StatusInternalServerError, fmt.Sprintf("db: %v", err))
	}

	h.Worker.Enqueue(ingest.Job{UploadID: id, StoragePath: storagePath})
	return c.Status(fiber.StatusAccepted).JSON(upload)
}

func (h *Handler) ListUploads(c fiber.Ctx) error {
	user := UserFromContext(c)
	mineOnly := parseBoolQuery(c.Query("mine"))
	var viewerID *uuid.UUID
	if user != nil {
		viewerID = &user.ID
	}
	if mineOnly && viewerID == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
	}
	list, err := h.Store.ListUploads(c.Context(), viewerID, mineOnly)
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
	if !canAccessUpload(UserFromContext(c), upload.UserID, upload.IsPrivate) {
		return fiber.NewError(fiber.StatusNotFound, "upload not found")
	}
	fights, err := h.Store.ListFightsByUpload(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{
		"id":          upload.ID,
		"userId":      upload.UserID,
		"name":        upload.Name,
		"filename":    upload.Filename,
		"sizeBytes":   upload.SizeBytes,
		"status":      upload.Status,
		"error":       upload.Error,
		"contentHash": upload.ContentHash,
		"isPrivate":   upload.IsPrivate,
		"createdAt":   upload.CreatedAt,
		"processedAt": upload.ProcessedAt,
		"fights":      fights,
	})
}

func (h *Handler) UpdateUpload(c fiber.Ctx) error {
	user := UserFromContext(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
	}
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
	if !isOwner(user, upload.UserID) {
		return fiber.NewError(fiber.StatusForbidden, "not the upload owner")
	}

	var body struct {
		Name *string `json:"name"`
	}
	if err := c.Bind().Body(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json body")
	}
	if body.Name == nil {
		return fiber.NewError(fiber.StatusBadRequest, "name required")
	}
	name := strings.TrimSpace(*body.Name)
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name must not be empty")
	}
	if len(name) > 200 {
		return fiber.NewError(fiber.StatusBadRequest, "name too long")
	}
	if err := h.Store.UpdateUploadName(c.Context(), id, name); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	upload.Name = name
	return c.JSON(upload)
}

func (h *Handler) DeleteUpload(c fiber.Ctx) error {
	user := UserFromContext(c)
	if user == nil {
		return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
	}
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
	if !isOwner(user, upload.UserID) {
		return fiber.NewError(fiber.StatusForbidden, "not the upload owner")
	}
	if err := h.Store.DeleteUpload(c.Context(), id); err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "upload not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	_ = h.Storage.Delete(c.Context(), upload.StoragePath)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) GetFight(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid id")
	}
	if err := h.ensureFightAccess(c, id); err != nil {
		return err
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

	if err := h.ensureFightAccess(c, fightID); err != nil {
		return err
	}

	spells, err := h.Store.ListSpellStats(c.Context(), fightID, actorID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(spells)
}

func (h *Handler) GetFightAuras(c fiber.Ctx) error {
	fightID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid fight id")
	}
	if err := h.ensureFightAccess(c, fightID); err != nil {
		return err
	}

	kind := strings.ToLower(strings.TrimSpace(c.Query("kind", "buff")))
	var auraKind db.AuraKind
	switch kind {
	case "buff", "buffs":
		auraKind = db.AuraKindBuff
	case "debuff", "debuffs":
		auraKind = db.AuraKindDebuff
	default:
		return fiber.NewError(fiber.StatusBadRequest, "kind must be buff or debuff")
	}

	stats, err := h.Store.ListAuraStats(c.Context(), fightID, auraKind)
	if err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "fight not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(stats)
}

func (h *Handler) GetFightInterrupts(c fiber.Ctx) error {
	fightID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid fight id")
	}
	if err := h.ensureFightAccess(c, fightID); err != nil {
		return err
	}
	stats, err := h.Store.ListInterruptStats(c.Context(), fightID)
	if err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "fight not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(stats)
}

func (h *Handler) GetFightDispels(c fiber.Ctx) error {
	fightID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid fight id")
	}
	if err := h.ensureFightAccess(c, fightID); err != nil {
		return err
	}
	stats, err := h.Store.ListDispelStats(c.Context(), fightID)
	if err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "fight not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(stats)
}

func (h *Handler) GetFightTimeline(c fiber.Ctx) error {
	fightID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid fight id")
	}
	if err := h.ensureFightAccess(c, fightID); err != nil {
		return err
	}

	mode := strings.ToLower(strings.TrimSpace(c.Query("mode", "summary")))
	bucketMs, _ := strconv.Atoi(c.Query("bucketMs", "0"))
	side := db.ParseTimelineSide(c.Query("side", "players"))

	switch mode {
	case "summary":
		out, err := h.Store.GetTimelineSummary(c.Context(), fightID, bucketMs, side)
		if err != nil {
			if db.IsNoRows(err) {
				return fiber.NewError(fiber.StatusNotFound, "fight not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(out)
	case "damage", "healing", "taken":
		out, err := h.Store.GetTimelinePlayers(c.Context(), fightID, db.TimelineMode(mode), bucketMs, side)
		if err != nil {
			if db.IsNoRows(err) {
				return fiber.NewError(fiber.StatusNotFound, "fight not found")
			}
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(out)
	default:
		return fiber.NewError(fiber.StatusBadRequest, "mode must be summary, damage, healing, or taken")
	}
}

func (h *Handler) GetFightEvents(c fiber.Ctx) error {
	fightID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid fight id")
	}
	if err := h.ensureFightAccess(c, fightID); err != nil {
		return err
	}

	limit, _ := strconv.Atoi(c.Query("limit", "100"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	typeFilter := strings.ToLower(strings.TrimSpace(c.Query("type", "")))

	var actorID *uuid.UUID
	if actorStr := c.Query("actorId"); actorStr != "" {
		id, err := uuid.Parse(actorStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid actorId")
		}
		actorID = &id
	}

	out, err := h.Store.ListCombatEvents(c.Context(), fightID, limit, offset, typeFilter, actorID)
	if err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "fight not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(out)
}

func (h *Handler) ensureFightAccess(c fiber.Ctx, fightID uuid.UUID) error {
	upload, err := h.Store.GetUploadByFightID(c.Context(), fightID)
	if err != nil {
		if db.IsNoRows(err) {
			return fiber.NewError(fiber.StatusNotFound, "fight not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if !canAccessUpload(UserFromContext(c), upload.UserID, upload.IsPrivate) {
		return fiber.NewError(fiber.StatusNotFound, "fight not found")
	}
	return nil
}

func parseBoolForm(values ...string) bool {
	for _, v := range values {
		if parseBoolQuery(v) {
			return true
		}
	}
	return false
}

func parseBoolQuery(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err == nil {
		return b
	}
	return v == "1" || v == "yes" || v == "on"
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
