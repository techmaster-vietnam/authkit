package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
)

// AuthMiddleware handles JWT authentication
type AuthMiddleware struct {
	config   *config.Config
	userRepo *repository.UserRepository
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(cfg *config.Config, userRepo *repository.UserRepository) *AuthMiddleware {
	return &AuthMiddleware{
		config:   cfg,
		userRepo: userRepo,
	}
}

// RequireAuth middleware requires authentication
func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractToken(c)
		if token == "" {
			return goerrorkit.NewAuthError(401, "Token không được cung cấp")
		}

		claims, err := utils.ValidateToken(token, m.config.JWT.Secret)
		if err != nil {
			return goerrorkit.NewAuthError(401, "Token không hợp lệ").WithData(map[string]interface{}{
				"error": err.Error(),
			})
		}

		user, err := m.userRepo.GetByID(claims.UserID)
		if err != nil {
			return goerrorkit.WrapWithMessage(err, "Người dùng không tồn tại")
		}

		if !user.IsActive {
			return goerrorkit.NewAuthError(403, "Tài khoản đã bị vô hiệu hóa").WithData(map[string]interface{}{
				"user_id": user.ID,
			})
		}

		// Store user in context
		c.Locals("user", user)
		c.Locals("userID", user.ID)

		return c.Next()
	}
}

// extractToken extracts token from Authorization header or cookie
func extractToken(c *fiber.Ctx) string {
	// Try Authorization header first
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}

	// Try cookie
	return c.Cookies("token")
}

// GetUserFromContext gets user from context
func GetUserFromContext(c *fiber.Ctx) (*models.User, bool) {
	user, ok := c.Locals("user").(*models.User)
	return user, ok
}

// GetUserIDFromContext gets user ID from context
func GetUserIDFromContext(c *fiber.Ctx) (uuid.UUID, bool) {
	userID, ok := c.Locals("userID").(uuid.UUID)
	return userID, ok
}

