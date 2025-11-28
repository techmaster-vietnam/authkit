package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
)

// BaseAuthMiddleware là generic auth middleware
// TUser phải implement UserInterface
type BaseAuthMiddleware[TUser core.UserInterface] struct {
	config   *config.Config
	userRepo *repository.BaseUserRepository[TUser]
}

// NewBaseAuthMiddleware tạo mới BaseAuthMiddleware với generic type
func NewBaseAuthMiddleware[TUser core.UserInterface](
	cfg *config.Config,
	userRepo *repository.BaseUserRepository[TUser],
) *BaseAuthMiddleware[TUser] {
	return &BaseAuthMiddleware[TUser]{
		config:   cfg,
		userRepo: userRepo,
	}
}

// RequireAuth middleware requires authentication
func (m *BaseAuthMiddleware[TUser]) RequireAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractToken(c)
		if token == "" {
			return goerrorkit.NewAuthError(401, "Token không được cung cấp")
		}

		// Validate token and extract role IDs (supports both standard and flexible token formats)
		userID, _, roleIDs, err := utils.ValidateTokenAndExtractRoleIDs(token, m.config.JWT.Secret)
		if err != nil {
			errMsg := err.Error()
			isExpired := strings.Contains(strings.ToLower(errMsg), "expired")
			
			// Token hết hạn là trạng thái bình thường trong authentication flow
			// Client có thể refresh token để lấy token mới
			// Không log vào file errors.log vì đây là behavior bình thường
			if isExpired {
				// Return error trực tiếp từ Fiber để không trigger logging của goerrorkit
				// Format response nhất quán với goerrorkit
				return c.Status(401).JSON(fiber.Map{
					"error": "Token đã hết hạn",
					"type":  "AUTH",
					"data": map[string]interface{}{
						"error": errMsg,
					},
				})
			}
			
			// Các lỗi khác (invalid signature, malformed token) vẫn log như bình thường
			return goerrorkit.NewAuthError(401, "Token không hợp lệ").WithData(map[string]interface{}{
				"error": errMsg,
			})
		}

		user, err := m.userRepo.GetByID(userID)
		if err != nil {
			return goerrorkit.WrapWithMessage(err, "Người dùng không tồn tại")
		}

		if !user.IsActive() {
			return goerrorkit.NewAuthError(403, "Tài khoản đã bị vô hiệu hóa").WithData(map[string]interface{}{
				"user_id": user.GetID(),
			})
		}

		// Role IDs are safe because token signature has been verified
		// If hacker modified role_ids, ValidateTokenAndExtractRoleIDs would have failed
		if roleIDs == nil {
			roleIDs = []uint{} // Ensure non-nil slice
		}

		// Store user and role IDs in context
		c.Locals("user", user)
		c.Locals("userID", user.GetID())
		c.Locals("roleIDs", roleIDs) // Store role IDs from validated token

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

// GetUserFromContextGeneric gets user from context với generic type
func GetUserFromContextGeneric[TUser core.UserInterface](c *fiber.Ctx) (TUser, bool) {
	var zero TUser
	user, ok := c.Locals("user").(TUser)
	if !ok {
		return zero, false
	}
	return user, true
}

// GetUserFromContext gets user from context (non-generic version for backward compatibility)
func GetUserFromContext(c *fiber.Ctx) (*models.User, bool) {
	user, ok := c.Locals("user").(*models.User)
	return user, ok
}

// GetUserIDFromContext gets user ID from context
func GetUserIDFromContext(c *fiber.Ctx) (string, bool) {
	userID, ok := c.Locals("userID").(string)
	return userID, ok
}

// GetRoleIDsFromContext gets role IDs from context (from validated JWT token)
// Role IDs are safe because they come from a token that has been validated
// This function does NOT query database - it only reads from memory (context)
func GetRoleIDsFromContext(c *fiber.Ctx) ([]uint, bool) {
	roleIDs, ok := c.Locals("roleIDs").([]uint)
	if !ok {
		return nil, false
	}
	return roleIDs, true
}
