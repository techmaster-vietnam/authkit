package main

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/goerrorkit"
)

// BlogHandler handles blog endpoints
type BlogHandler struct {
	blogService *BlogService
	roleRepo    *authkit.RoleRepository
}

// NewBlogHandler creates a new blog handler
func NewBlogHandler(blogService *BlogService, roleRepo *authkit.RoleRepository) *BlogHandler {
	return &BlogHandler{
		blogService: blogService,
		roleRepo:    roleRepo,
	}
}

// Create handles create blog request
// POST /api/blogs
func (h *BlogHandler) Create(c *fiber.Ctx) error {
	userID, ok := authkit.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
	}

	var req CreateBlogRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	blog, err := h.blogService.Create(userID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data":    blog,
	})
}

// GetByID handles get blog by ID request
// GET /api/blogs/:id
func (h *BlogHandler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"id": c.Params("id"),
		})
	}

	blog, err := h.blogService.GetByID(id)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    blog,
	})
}

// Update handles update blog request
// PUT /api/blogs/:id
func (h *BlogHandler) Update(c *fiber.Ctx) error {
	userID, ok := authkit.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
	}

	id := c.Params("id")
	if id == "" {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"id": c.Params("id"),
		})
	}

	// Get user roles
	userRoles, err := h.roleRepo.ListRolesOfUser(userID)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy roles của user")
	}

	roleNames := make([]string, len(userRoles))
	for i, role := range userRoles {
		roleNames[i] = role.GetName()
	}

	var req UpdateBlogRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	blog, err := h.blogService.Update(id, userID, roleNames, req)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    blog,
	})
}

// Delete handles delete blog request
// DELETE /api/blogs/:id
func (h *BlogHandler) Delete(c *fiber.Ctx) error {
	userID, ok := authkit.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
	}

	id := c.Params("id")
	if id == "" {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"id": c.Params("id"),
		})
	}

	// Get user roles
	userRoles, err := h.roleRepo.ListRolesOfUser(userID)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy roles của user")
	}

	roleNames := make([]string, len(userRoles))
	for i, role := range userRoles {
		roleNames[i] = role.GetName()
	}

	if err := h.blogService.Delete(id, userID, roleNames); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Xóa blog thành công",
	})
}

// List handles list blogs request
// GET /api/blogs
func (h *BlogHandler) List(c *fiber.Ctx) error {
	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if limit > 100 {
		limit = 100
	}

	blogs, total, err := h.blogService.List(offset, limit)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    blogs,
		"total":   total,
		"offset":  offset,
		"limit":   limit,
	})
}

// ListMyBlogs handles list my blogs request
// GET /api/blogs/my
func (h *BlogHandler) ListMyBlogs(c *fiber.Ctx) error {
	userID, ok := authkit.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
	}

	offset, _ := strconv.Atoi(c.Query("offset", "0"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	if limit > 100 {
		limit = 100
	}

	blogs, total, err := h.blogService.ListByAuthor(userID, offset, limit)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    blogs,
		"total":   total,
		"offset":  offset,
		"limit":   limit,
	})
}
