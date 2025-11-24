package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/core"
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
			return goerrorkit.NewAuthError(401, "Token không hợp lệ").WithData(map[string]interface{}{
				"error": err.Error(),
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

// GetUserFromContextGeneric gets user from context với generic type
func GetUserFromContextGeneric[TUser core.UserInterface](c *fiber.Ctx) (TUser, bool) {
	var zero TUser
	user, ok := c.Locals("user").(TUser)
	if !ok {
		return zero, false
	}
	return user, true
}
