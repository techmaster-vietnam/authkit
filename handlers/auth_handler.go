package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/goerrorkit"
)

// AuthHandler handles authentication endpoints
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// Login handles login request
// POST /api/auth/login
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req service.LoginRequest
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
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// In a stateless JWT system, logout is typically handled client-side
	// by removing the token. For server-side logout, you might want to
	// maintain a blacklist of tokens.
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Đăng xuất thành công",
	})
}

// Register handles registration request
// POST /api/auth/register
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req service.RegisterRequest
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
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
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
func (h *AuthHandler) UpdateProfile(c *fiber.Ctx) error {
	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}

	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	user, err := h.authService.UpdateProfile(userID, req.FirstName, req.LastName)
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
func (h *AuthHandler) DeleteProfile(c *fiber.Ctx) error {
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
func (h *AuthHandler) GetProfile(c *fiber.Ctx) error {
	user, ok := middleware.GetUserFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    user,
	})
}
