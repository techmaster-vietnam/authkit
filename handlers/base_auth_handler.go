package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/goerrorkit"
)

// BaseAuthHandler là generic auth handler
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseAuthHandler[TUser core.UserInterface, TRole core.RoleInterface] struct {
	authService *service.BaseAuthService[TUser, TRole]
}

// NewBaseAuthHandler tạo mới BaseAuthHandler với generic types
func NewBaseAuthHandler[TUser core.UserInterface, TRole core.RoleInterface](
	authService *service.BaseAuthService[TUser, TRole],
) *BaseAuthHandler[TUser, TRole] {
	return &BaseAuthHandler[TUser, TRole]{authService: authService}
}

// Login handles login request
// POST /api/auth/login
func (h *BaseAuthHandler[TUser, TRole]) Login(c *fiber.Ctx) error {
	var req service.BaseLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	resp, err := h.authService.Login(req)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    resp,
	})
}

// Logout handles logout request
// POST /api/auth/logout
func (h *BaseAuthHandler[TUser, TRole]) Logout(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Đăng xuất thành công",
	})
}

// Register handles registration request
// POST /api/auth/register
func (h *BaseAuthHandler[TUser, TRole]) Register(c *fiber.Ctx) error {
	var req service.BaseRegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	user, err := h.authService.Register(req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// ChangePassword handles change password request
// POST /api/auth/change-password
func (h *BaseAuthHandler[TUser, TRole]) ChangePassword(c *fiber.Ctx) error {
	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	if err := h.authService.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Đổi mật khẩu thành công",
	})
}

// UpdateProfile handles update profile request
// PUT /api/auth/profile
func (h *BaseAuthHandler[TUser, TRole]) UpdateProfile(c *fiber.Ctx) error {
	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	var req struct {
		FullName string `json:"full_name"`
	}

	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	user, err := h.authService.UpdateProfile(userID, req.FullName)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

// DeleteProfile handles delete profile request
// DELETE /api/auth/profile
func (h *BaseAuthHandler[TUser, TRole]) DeleteProfile(c *fiber.Ctx) error {
	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	if err := h.authService.DeleteProfile(userID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Xóa tài khoản thành công",
	})
}

// GetProfile handles get profile request
// GET /api/auth/profile
func (h *BaseAuthHandler[TUser, TRole]) GetProfile(c *fiber.Ctx) error {
	user, ok := middleware.GetUserFromContextGeneric[TUser](c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}

