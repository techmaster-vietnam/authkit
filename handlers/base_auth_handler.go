package handlers

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/authkit/utils"
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
// Trả về access token trong JSON và refresh token trong cookie HttpOnly
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

	// Set refresh token vào cookie với các thuộc tính bảo mật
	// HttpOnly: JavaScript không thể truy cập (chống XSS)
	// Secure: Chỉ gửi qua HTTPS (production) - có thể config qua COOKIE_SECURE env
	// SameSite=Strict: Chống CSRF
	// Path: Chỉ gửi cookie khi request đến /api/auth/*
	cookieSecure := h.authService.GetConfig().Server.CookieSecure
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour), // 7 ngày
		HTTPOnly: true,
		Secure:   cookieSecure,
		SameSite: "Strict",
		Path:     "/api/auth",
	})

	// Chỉ trả về access token và user trong JSON response
	// Refresh token không được trả về trong JSON để bảo mật
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"token": resp.Token,
			"user":  resp.User,
		},
	})
}

// Refresh handles refresh token request
// POST /api/auth/refresh
// Lấy refresh token từ cookie và trả về access token mới
func (h *BaseAuthHandler[TUser, TRole]) Refresh(c *fiber.Ctx) error {
	// Lấy refresh token từ cookie
	refreshToken := c.Cookies("refresh_token")
	if refreshToken == "" {
		return goerrorkit.NewAuthError(401, "Refresh token không được cung cấp")
	}

	// Refresh access token
	cookieSecure := h.authService.GetConfig().Server.CookieSecure
	resp, err := h.authService.Refresh(refreshToken)
	if err != nil {
		// Nếu refresh token không hợp lệ, xóa cookie
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    "",
			Expires:  time.Now().Add(-1 * time.Hour), // Xóa cookie
			HTTPOnly: true,
			Secure:   cookieSecure,
			SameSite: "Strict",
			Path:     "/api/auth",
		})
		return err
	}

	// Set refresh token mới vào cookie (rotation)
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    resp.RefreshToken,
		Expires:  time.Now().Add(7 * 24 * time.Hour), // 7 ngày
		HTTPOnly: true,
		Secure:   cookieSecure,
		SameSite: "Strict",
		Path:     "/api/auth",
	})

	// Trả về access token mới
	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"token": resp.Token,
		},
	})
}

// Logout handles logout request
// POST /api/auth/logout
// Xóa refresh token từ database và cookie
func (h *BaseAuthHandler[TUser, TRole]) Logout(c *fiber.Ctx) error {
	// Lấy refresh token từ cookie
	refreshToken := c.Cookies("refresh_token")

	// Xóa refresh token từ database
	if refreshToken != "" {
		if err := h.authService.Logout(refreshToken); err != nil {
			// Log error nhưng vẫn tiếp tục xóa cookie
			_ = err
		}
	}

	// Xóa cookie
	cookieSecure := h.authService.GetConfig().Server.CookieSecure
	c.Cookie(&fiber.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour), // Xóa cookie
		HTTPOnly: true,
		Secure:   cookieSecure,
		SameSite: "Strict",
		Path:     "/api/auth",
	})

	return c.JSON(fiber.Map{
		"message": "Đăng xuất thành công",
	})
}

// Register handles registration request
// POST /api/auth/register
// Hỗ trợ các trường custom từ request body (ví dụ: mobile, address)
func (h *BaseAuthHandler[TUser, TRole]) Register(c *fiber.Ctx) error {
	// Parse request body vào map để lấy cả các trường custom
	var requestMap map[string]interface{}
	if err := c.BodyParser(&requestMap); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Extract các trường cơ bản
	req := service.BaseRegisterRequest{
		CustomFields: make(map[string]interface{}),
	}

	if email, ok := requestMap["email"].(string); ok {
		// Validate email format
		if err := utils.ValidateEmail(email); err != nil {
			return err
		}
		req.Email = email
	} else {
		// Email không tồn tại hoặc không phải string
		return goerrorkit.NewValidationError("Email là bắt buộc và phải là chuỗi", map[string]interface{}{
			"field": "email",
		})
	}
	if password, ok := requestMap["password"].(string); ok {
		// Validate password format theo config
		if err := utils.ValidatePassword(password, h.authService.GetConfig().Password); err != nil {
			return err
		}
		req.Password = password
	} else {
		// Password không tồn tại hoặc không phải string
		return goerrorkit.NewValidationError("Mật khẩu là bắt buộc và phải là chuỗi", map[string]interface{}{
			"field": "password",
		})
	}
	if fullName, ok := requestMap["full_name"].(string); ok {
		req.FullName = fullName
	}

	// Các trường còn lại (không phải email, password, full_name) là custom fields
	// Loại trừ các trường đặc biệt không nên được set trực tiếp
	excludedFields := map[string]bool{
		"email":      true,
		"password":   true,
		"full_name":  true,
		"id":         true, // Không cho phép set ID
		"created_at": true,
		"updated_at": true,
		"deleted_at": true,
		"roles":      true, // Roles được quản lý riêng
	}

	for key, value := range requestMap {
		if !excludedFields[key] {
			req.CustomFields[key] = value
		}
	}

	user, err := h.authService.Register(req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data": user,
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
		"data": user,
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
		"data": user,
	})
}

// GetUserDetail handles get user detail request
// GET /api/users/detail?identifier=id_or_email
// Chỉ dành cho admin và super_admin
func (h *BaseAuthHandler[TUser, TRole]) GetUserDetail(c *fiber.Ctx) error {
	identifier := c.Query("identifier")
	if identifier == "" {
		return goerrorkit.NewValidationError("ID hoặc email là bắt buộc", map[string]interface{}{
			"field": "identifier",
		})
	}

	response, err := h.authService.GetUserDetail(identifier)
	if err != nil {
		return err
	}

	// Format response với roles dạng [{role_id, role_name}]
	type RoleInfo struct {
		ID   uint   `json:"role_id"`
		Name string `json:"role_name"`
	}

	rolesInfo := make([]RoleInfo, 0, len(response.Roles))
	for _, role := range response.Roles {
		rolesInfo = append(rolesInfo, RoleInfo{
			ID:   role.GetID(),
			Name: role.GetName(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"user":  response.User,
			"roles": rolesInfo,
		},
	})
}
