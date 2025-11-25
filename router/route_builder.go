package router

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/models"
)

// RouteBuilder cung cấp fluent API để cấu hình route và phân quyền
type RouteBuilder struct {
	metadata *RouteMetadata
	router   fiber.Router
	registry *RouteRegistry
	authMw   AuthMiddlewareInterface
	authzMw  AuthorizationMiddlewareInterface
}

// Public đánh dấu route là public (không cần authentication)
func (rb *RouteBuilder) Public() *RouteBuilder {
	rb.metadata.AccessType = models.AccessPublic
	rb.metadata.Roles = []string{}
	return rb
}

// Allow cho phép các roles cụ thể truy cập (roles rỗng = mọi user đã đăng nhập)
func (rb *RouteBuilder) Allow(roles ...string) *RouteBuilder {
	rb.metadata.AccessType = models.AccessAllow
	rb.metadata.Roles = roles
	return rb
}

// Forbid cấm các roles cụ thể truy cập
func (rb *RouteBuilder) Forbid(roles ...string) *RouteBuilder {
	rb.metadata.AccessType = models.AccessForbid
	rb.metadata.Roles = roles
	return rb
}

// Fixed đánh dấu rule không thể thay đổi từ database
// Fixed và Override loại trừ lẫn nhau, không thể dùng cùng lúc
func (rb *RouteBuilder) Fixed() *RouteBuilder {
	rb.metadata.Fixed = true
	rb.metadata.Override = false // Loại trừ Override
	return rb
}

// Override đánh dấu luôn ghi đè cấu hình từ code lên database
// Fixed và Override loại trừ lẫn nhau, không thể dùng cùng lúc
func (rb *RouteBuilder) Override() *RouteBuilder {
	rb.metadata.Override = true
	rb.metadata.Fixed = false // Loại trừ Fixed
	return rb
}

// Description thêm mô tả cho rule
func (rb *RouteBuilder) Description(desc string) *RouteBuilder {
	rb.metadata.Description = desc
	return rb
}

// Register hoàn tất việc đăng ký route và áp dụng middleware phù hợp
func (rb *RouteBuilder) Register() {
	// Đăng ký route vào registry
	rb.registry.Register(rb.metadata)

	// Áp dụng middleware dựa trên AccessType
	if rb.metadata.AccessType == models.AccessPublic {
		// Public route: không cần authentication và authorization
		rb.router.Add(rb.metadata.Method, rb.metadata.Path, rb.metadata.Handler)
	} else {
		// Protected route: cần authentication và authorization
		rb.router.Add(
			rb.metadata.Method,
			rb.metadata.Path,
			rb.authMw.RequireAuth(),
			rb.authzMw.Authorize(),
			rb.metadata.Handler,
		)
	}
}
