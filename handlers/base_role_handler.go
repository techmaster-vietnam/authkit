package handlers

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/goerrorkit"
)

// BaseRoleHandler là generic role handler
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseRoleHandler[TUser core.UserInterface, TRole core.RoleInterface] struct {
	roleService *service.BaseRoleService[TUser, TRole]
}

// NewBaseRoleHandler tạo mới BaseRoleHandler với generic types
func NewBaseRoleHandler[TUser core.UserInterface, TRole core.RoleInterface](
	roleService *service.BaseRoleService[TUser, TRole],
) *BaseRoleHandler[TUser, TRole] {
	return &BaseRoleHandler[TUser, TRole]{roleService: roleService}
}

// AddRole handles add role request
// POST /api/roles
func (h *BaseRoleHandler[TUser, TRole]) AddRole(c *fiber.Ctx) error {
	var req service.BaseAddRoleRequest
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
		"data": role,
	})
}

// RemoveRole handles remove role request
// DELETE /api/roles/:id
func (h *BaseRoleHandler[TUser, TRole]) RemoveRole(c *fiber.Ctx) error {
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
		"message": "Xóa role thành công",
	})
}

// ListRoles handles list roles request
// GET /api/roles
func (h *BaseRoleHandler[TUser, TRole]) ListRoles(c *fiber.Ctx) error {
	roles, err := h.roleService.ListRoles()
	if err != nil {
		return err
	}

	// Đảm bảo luôn trả về empty array thay vì null trong JSON
	if roles == nil {
		roles = []TRole{}
	}

	return c.JSON(fiber.Map{
		"data": roles,
	})
}

// AddRoleToUser handles add role to user request
// POST /api/users/:user_id/roles/:role_id
func (h *BaseRoleHandler[TUser, TRole]) AddRoleToUser(c *fiber.Ctx) error {
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

	// Lấy role IDs của user đang login từ context
	currentUserRoleIDs, ok := middleware.GetRoleIDsFromContext(c)
	if !ok {
		currentUserRoleIDs = []uint{}
	}

	if err := h.roleService.AddRoleToUser(userID, uint(roleID), currentUserRoleIDs); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Thêm role cho user thành công",
	})
}

// RemoveRoleFromUser handles remove role from user request
// DELETE /api/users/:user_id/roles/:role_id
func (h *BaseRoleHandler[TUser, TRole]) RemoveRoleFromUser(c *fiber.Ctx) error {
	userID := c.Params("user_id")
	if userID == "" {
		return goerrorkit.NewValidationError("User ID không hợp lệ", map[string]interface{}{
			"user_id": c.Params("user_id"),
		})
	}

	roleID := c.Params("role_id")
	if roleID == "" {
		return goerrorkit.NewValidationError("Role ID không hợp lệ", map[string]interface{}{
			"role_id": c.Params("role_id"),
		})
	}

	if err := h.roleService.RemoveRoleFromUser(userID, roleID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Xóa role khỏi user thành công",
	})
}

// CheckUserHasRole handles check user has role request
// GET /api/users/:user_id/roles/:role_name/check
func (h *BaseRoleHandler[TUser, TRole]) CheckUserHasRole(c *fiber.Ctx) error {
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
		"data": fiber.Map{
			"has_role": hasRole,
		},
	})
}

// ListUsersHasRole handles list users has role request
// GET /api/roles/:role_id_name/users
// role_id_name có thể là số (role_id) hoặc chuỗi (role_name)
func (h *BaseRoleHandler[TUser, TRole]) ListUsersHasRole(c *fiber.Ctx) error {
	roleIdName := c.Params("role_id_name")
	if roleIdName == "" {
		return goerrorkit.NewValidationError("Role ID hoặc name là bắt buộc", map[string]interface{}{
			"field": "role_id_name",
		})
	}

	users, err := h.roleService.ListUsersHasRole(roleIdName)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"data": users,
	})
}

// UpdateUserRoles handles update user roles request
// PUT /api/users/:userId/roles
func (h *BaseRoleHandler[TUser, TRole]) UpdateUserRoles(c *fiber.Ctx) error {
	userID := c.Params("userId")
	if userID == "" {
		return goerrorkit.NewValidationError("User ID không hợp lệ", map[string]interface{}{
			"user_id": c.Params("userId"),
		})
	}

	var req service.BaseUpdateUserRolesRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Lấy role IDs của user đang login từ context
	currentUserRoleIDs, ok := middleware.GetRoleIDsFromContext(c)
	if !ok {
		currentUserRoleIDs = []uint{}
	}

	if err := h.roleService.UpdateUserRoles(userID, req, currentUserRoleIDs); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Cập nhật roles cho user thành công",
	})
}
