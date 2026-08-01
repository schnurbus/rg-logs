package api

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"rg-logs/internal/auth"
)

const userLocalsKey = "authUser"

func UserFromContext(c fiber.Ctx) *auth.User {
	u, _ := c.Locals(userLocalsKey).(*auth.User)
	return u
}

func OptionalAuth(client *auth.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := bearerToken(c)
		if token == "" {
			return c.Next()
		}
		user, err := client.GetUser(c.Context(), token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}
		c.Locals(userLocalsKey, user)
		return c.Next()
	}
}

func RequireAuth(client *auth.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		token := bearerToken(c)
		if token == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "authentication required")
		}
		user, err := client.GetUser(c.Context(), token)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "invalid or expired token")
		}
		c.Locals(userLocalsKey, user)
		return c.Next()
	}
}

func bearerToken(c fiber.Ctx) string {
	h := c.Get("Authorization")
	if h == "" {
		return ""
	}
	const prefix = "Bearer "
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

func canAccessUpload(user *auth.User, ownerID uuid.UUID, isPrivate bool) bool {
	if !isPrivate {
		return true
	}
	if user == nil {
		return false
	}
	return user.ID == ownerID
}

func isOwner(user *auth.User, ownerID uuid.UUID) bool {
	return user != nil && user.ID == ownerID
}
