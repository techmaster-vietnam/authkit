package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/goerrorkit"
)

// RoleHandler handles role endpoints
type RoleHandler struct {
	roleService *service.RoleService
}

// NewRoleHandler creates a new role handler
func NewRoleHandler(roleService *service.RoleService) *RoleHandler {
	return &RoleHandler{roleService: roleService}
}

// AddRole handles add role request
// POST /api/roles
func (h *RoleHandler) AddRole(c *fiber.Ctx) error {
	var req service.AddRoleRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	role, err := h.roleService.AddRole(req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    role,
	})
}

// RemoveRole handles remove role request
// DELETE /api/roles/:id
func (h *RoleHandler) RemoveRole(c *fiber.Ctx) error {
	roleID, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"id": c.Params("id"),
		})
	}

	if err := h.roleService.RemoveRole(uint(roleID)); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Xóa role thành công",
	})
}

// ListRoles handles list roles request
// GET /api/roles
func (h *RoleHandler) ListRoles(c *fiber.Ctx) error {
	roles, err := h.roleService.ListRoles()
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    roles,
	})
}

// AddRoleToUser handles add role to user request
// POST /api/users/:user_id/roles/:role_id
func (h *RoleHandler) AddRoleToUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if userID == "" {
		return goerrorkit.NewValidationError("User ID không hợp lệ", map[string]interface{}{
			"user_id": c.Params("user_id"),
		})
	}

	roleID, err := strconv.ParseUint(c.Params("role_id"), 10, 32)
	if err != nil {
		return goerrorkit.NewValidationError("Role ID không hợp lệ", map[string]interface{}{
			"role_id": c.Params("role_id"),
		})
	}

	if err := h.roleService.AddRoleToUser(userID, uint(roleID)); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Thêm role cho user thành công",
	})
}

// RemoveRoleFromUser handles remove role from user request
// DELETE /api/users/:user_id/roles/:role_id
func (h *RoleHandler) RemoveRoleFromUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if userID == "" {
		return goerrorkit.NewValidationError("User ID không hợp lệ", map[string]interface{}{
			"user_id": c.Params("user_id"),
		})
	}

	roleID, err := strconv.ParseUint(c.Params("role_id"), 10, 32)
	if err != nil {
		return goerrorkit.NewValidationError("Role ID không hợp lệ", map[string]interface{}{
			"role_id": c.Params("role_id"),
		})
	}

	if err := h.roleService.RemoveRoleFromUser(userID, uint(roleID)); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Xóa role khỏi user thành công",
	})
}

// CheckUserHasRole handles check user has role request
// GET /api/users/:user_id/roles/:role_name/check
func (h *RoleHandler) CheckUserHasRole(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if userID == "" {
		return goerrorkit.NewValidationError("User ID không hợp lệ", map[string]interface{}{
			"user_id": c.Params("user_id"),
		})
	}

	roleName := c.Params("role_name")
	if roleName == "" {
		return goerrorkit.NewValidationError("Role name là bắt buộc", map[string]interface{}{
			"field": "role_name",
		})
	}

	hasRole, err := h.roleService.CheckUserHasRole(userID, roleName)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"has_role": hasRole,
		},
	})
}

// ListRolesOfUser handles list roles of user request
// GET /api/users/:user_id/roles
func (h *RoleHandler) ListRolesOfUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if userID == "" {
		return goerrorkit.NewValidationError("User ID không hợp lệ", map[string]interface{}{
			"user_id": c.Params("user_id"),
		})
	}

	roles, err := h.roleService.ListRolesOfUser(userID)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    roles,
	})
}

// ListUsersHasRole handles list users has role request
// GET /api/roles/:role_name/users
func (h *RoleHandler) ListUsersHasRole(c *fiber.Ctx) error {
	roleName := c.Params("role_name")
	if roleName == "" {
		return goerrorkit.NewValidationError("Role name là bắt buộc", map[string]interface{}{
			"field": "role_name",
		})
	}

	users, err := h.roleService.ListUsersHasRole(roleName)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    users,
	})
}

