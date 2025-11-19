package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/router"
)

// setupRoutes sets up all routes với fluent API
// Sử dụng AuthKit instance với generic types - CustomUser thay vì BaseUser
func setupRoutes(
	app *fiber.App,
	ak *authkit.AuthKit[*CustomUser, *authkit.BaseRole],
	blogHandler *BlogHandler,
) {
	// Serve favicon (public, không cần đăng ký vào registry)
	app.Get("/favicon.ico", func(c *fiber.Ctx) error {
		return c.SendFile("favicon.png")
	})

	// Serve static HTML file (public)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("index.html")
	})

	// Tạo AuthRouter cho API routes
	// Sử dụng Group() để tự động tính prefix, không cần truyền prefix thủ công
	apiRouter := router.NewAuthRouter(app, ak.RouteRegistry, ak.AuthMiddleware, ak.AuthorizationMiddleware).Group("/api")

	// Auth routes (public)
	auth := apiRouter.Group("/auth")
	auth.Post("/login", ak.AuthHandler.Login).
		Public().
		Description("Đăng nhập người dùng").
		Register()
	auth.Post("/register", ak.AuthHandler.Register).
		Public().
		Description("Đăng ký người dùng mới").
		Register()
	auth.Post("/logout", ak.AuthHandler.Logout).
		Public().
		Description("Đăng xuất người dùng").
		Register()

	// Protected auth routes
	auth.Get("/profile", ak.AuthHandler.GetProfile).
		Allow().
		Description("Lấy thông tin profile").
		Register()
	auth.Put("/profile", ak.AuthHandler.UpdateProfile).
		Allow().
		Description("Cập nhật thông tin profile").
		Register()
	auth.Delete("/profile", ak.AuthHandler.DeleteProfile).
		Allow().
		Description("Xóa tài khoản").
		Register()
	auth.Post("/change-password", ak.AuthHandler.ChangePassword).
		Allow().
		Description("Đổi mật khẩu").
		Register()

	// Blog routes
	blogs := apiRouter.Group("/blogs")
	blogs.Get("/", blogHandler.List).
		Public().
		Description("Danh sách blog công khai").
		Register()

	blogs.Get("/:id", blogHandler.GetByID).
		Allow("reader", "author", "editor", "admin").
		Fixed().
		Description("Xem chi tiết blog").
		Register()
	blogs.Post("/", blogHandler.Create).
		Allow("author", "editor", "admin").
		Description("Tạo blog mới").
		Register()
	blogs.Put("/:id", blogHandler.Update).
		Allow("author", "editor", "admin").
		Description("Cập nhật blog").
		Register()
	blogs.Delete("/:id", blogHandler.Delete).
		Allow("author", "editor", "admin").
		Description("Xóa blog").
		Register()
	blogs.Get("/my", blogHandler.ListMyBlogs).
		Allow().
		Description("Danh sách blog của tôi").
		Register()

	// Role routes (admin only)
	roles := apiRouter.Group("/roles")
	roles.Get("/", ak.RoleHandler.ListRoles).
		Allow("admin").
		Fixed().
		Description("Danh sách roles").
		Register()
	roles.Post("/", ak.RoleHandler.AddRole).
		Allow("admin").
		Fixed().
		Description("Tạo role mới").
		Register()
	roles.Delete("/:id", ak.RoleHandler.RemoveRole).
		Allow("admin").
		Fixed().
		Description("Xóa role").
		Register()
	roles.Get("/:role_name/users", ak.RoleHandler.ListUsersHasRole).
		Allow("admin").
		Fixed().
		Description("Danh sách users có role").
		Register()

	// User role routes (admin only)
	users := apiRouter.Group("/users")
	users.Get("/:user_id/roles", ak.RoleHandler.ListRolesOfUser).
		Allow("admin").
		Fixed().
		Description("Danh sách roles của user").
		Register()
	users.Post("/:user_id/roles/:role_id", ak.RoleHandler.AddRoleToUser).
		Allow("admin").
		Fixed().
		Description("Thêm role cho user").
		Register()
	users.Delete("/:user_id/roles/:role_id", ak.RoleHandler.RemoveRoleFromUser).
		Allow("admin").
		Fixed().
		Description("Xóa role của user").
		Register()
	users.Get("/:user_id/roles/:role_name/check", ak.RoleHandler.CheckUserHasRole).
		Allow("admin").
		Fixed().
		Description("Kiểm tra user có role").
		Register()

	// Rule routes (admin only)
	rules := apiRouter.Group("/rules")
	rules.Get("/", ak.RuleHandler.ListRules).
		Allow("admin").
		Fixed().
		Description("Danh sách rules").
		Register()
	rules.Post("/", ak.RuleHandler.AddRule).
		Allow("admin").
		Fixed().
		Description("Tạo rule mới").
		Register()
	rules.Put("/:id", ak.RuleHandler.UpdateRule).
		Allow("admin").
		Fixed().
		Description("Cập nhật rule").
		Register()
	rules.Delete("/:id", ak.RuleHandler.RemoveRule).
		Allow("admin").
		Fixed().
		Description("Xóa rule").
		Register()
}
