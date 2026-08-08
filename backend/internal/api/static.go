package api

import (
	"io/fs"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"rg-logs/web"
)

// mountSPA serves the embedded Vite build and falls back to index.html for
// client-side routes. Paths under /api are never handled here.
func mountSPA(app *fiber.App) {
	indexHTML, err := fs.ReadFile(web.FS, "index.html")
	hasIndex := err == nil && len(indexHTML) > 0

	app.Get("/*", static.New("", static.Config{
		FS: web.FS,
		Next: func(c fiber.Ctx) bool {
			return strings.HasPrefix(c.Path(), "/api")
		},
		NotFoundHandler: func(c fiber.Ctx) error {
			if !hasIndex {
				return fiber.ErrNotFound
			}
			c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
			return c.Status(fiber.StatusOK).Send(indexHTML)
		},
	}))
}
