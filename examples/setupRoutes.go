package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/router"
)

// setupRoutes sets up all routes với fluent API
func setupRoutes(
	app *fiber.App,
	registry *router.RouteRegistry,
	authHandler *authkit.AuthHandler,
	roleHandler *authkit.RoleHandler,
	ruleHandler *authkit.RuleHandler,
	blogHandler *BlogHandler,
	authMiddleware *authkit.AuthMiddleware,
	authzMiddleware *authkit.AuthorizationMiddleware,
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
	apiRouter := router.NewAuthRouter(app, registry, authMiddleware, authzMiddleware).Group("/api")

	// Auth routes (public)
	auth := apiRouter.Group("/auth")
	auth.Post("/login", authHandler.Login).
		Public().
		Description("Đăng nhập người dùng").
		Register()
	auth.Post("/register", authHandler.Register).
		Public().
		Description("Đăng ký người dùng mới").
		Register()
	auth.Post("/logout", authHandler.Logout).
		Public().
		Description("Đăng xuất người dùng").
		Register()

	// Protected auth routes
	auth.Get("/profile", authHandler.GetProfile).
		Allow().
		Description("Lấy thông tin profile").
		Register()
	auth.Put("/profile", authHandler.UpdateProfile).
		Allow().
		Description("Cập nhật thông tin profile").
		Register()
	auth.Delete("/profile", authHandler.DeleteProfile).
		Allow().
		Description("Xóa tài khoản").
		Register()
	auth.Post("/change-password", authHandler.ChangePassword).
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
	roles.Get("/", roleHandler.ListRoles).
		Allow("admin").
		Fixed().
		Description("Danh sách roles").
		Register()
	roles.Post("/", roleHandler.AddRole).
		Allow("admin").
		Fixed().
		Description("Tạo role mới").
		Register()
	roles.Delete("/:id", roleHandler.RemoveRole).
		Allow("admin").
		Fixed().
		Description("Xóa role").
		Register()
	roles.Get("/:role_name/users", roleHandler.ListUsersHasRole).
		Allow("admin").
		Fixed().
		Description("Danh sách users có role").
		Register()

	// User role routes (admin only)
	users := apiRouter.Group("/users")
	users.Get("/:user_id/roles", roleHandler.ListRolesOfUser).
		Allow("admin").
		Fixed().
		Description("Danh sách roles của user").
		Register()
	users.Post("/:user_id/roles/:role_id", roleHandler.AddRoleToUser).
		Allow("admin").
		Fixed().
		Description("Thêm role cho user").
		Register()
	users.Delete("/:user_id/roles/:role_id", roleHandler.RemoveRoleFromUser).
		Allow("admin").
		Fixed().
		Description("Xóa role của user").
		Register()
	users.Get("/:user_id/roles/:role_name/check", roleHandler.CheckUserHasRole).
		Allow("admin").
		Fixed().
		Description("Kiểm tra user có role").
		Register()

	// Rule routes (admin only)
	rules := apiRouter.Group("/rules")
	rules.Get("/", ruleHandler.ListRules).
		Allow("admin").
		Fixed().
		Description("Danh sách rules").
		Register()
	rules.Post("/", ruleHandler.AddRule).
		Allow("admin").
		Fixed().
		Description("Tạo rule mới").
		Register()
	rules.Put("/:id", ruleHandler.UpdateRule).
		Allow("admin").
		Fixed().
		Description("Cập nhật rule").
		Register()
	rules.Delete("/:id", ruleHandler.RemoveRule).
		Allow("admin").
		Fixed().
		Description("Xóa rule").
		Register()
}

