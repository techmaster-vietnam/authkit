package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/router"
)

func setupRoutes(
	app *fiber.App,
	ak *authkit.AuthKit[*CustomUser, *authkit.BaseRole],
	blogHandler *BlogHandler,
	demoHandler *DemoHandler,
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
		Fixed().
		Description("Đăng nhập người dùng").
		Register()
	auth.Post("/register", ak.AuthHandler.Register).
		Public().
		Fixed().
		Description("Đăng ký người dùng mới").
		Register()
	auth.Post("/logout", ak.AuthHandler.Logout).
		Public().
		Fixed().
		Description("Đăng xuất người dùng").
		Register()
	auth.Post("/refresh", ak.AuthHandler.Refresh).
		Public().
		Fixed().
		Description("Làm mới access token bằng refresh token từ cookie").
		Register()
	auth.Post("/request-password-reset", ak.AuthHandler.RequestPasswordReset).
		Public().
		Fixed().
		Description("Yêu cầu reset password - gửi reset token qua email/tin nhắn").
		Register()
	auth.Post("/reset-password", ak.AuthHandler.ResetPassword).
		Public().
		Fixed().
		Description("Đặt lại mật khẩu bằng reset token từ email/tin nhắn").
		Register()

	// Protected auth routes
	auth.Post("/change-password", ak.AuthHandler.ChangePassword).
		Allow().
		Fixed().
		Description("Đổi mật khẩu").
		Register()

	// User management routes
	users := apiRouter.Group("/user")
	users.Get("/profile", ak.UserHandler.GetProfile).
		Allow().
		Fixed().
		Description("Lấy thông tin profile của chính mình").
		Register()
	users.Put("/profile", ak.UserHandler.UpdateProfile).
		Allow().
		Fixed().
		Description("Cập nhật thông tin profile của chính mình (chỉ user đang đăng nhập mới có thể cập nhật profile của chính mình)").
		Register()
	users.Delete("/profile", ak.UserHandler.DeleteProfile).
		Allow().
		Fixed().
		Description("Xóa tài khoản").
		Register()
	users.Get("/:id", ak.UserHandler.GetProfileByID).
		Allow("admin", "super_admin").
		Fixed().
		Description("Lấy thông tin profile theo identifier (id, email, hoặc mobile) kèm danh sách roles - chỉ dành cho admin và super_admin").
		Register()
	users.Put("/:id", ak.UserHandler.UpdateProfileByID).
		Allow("admin", "super_admin").
		Fixed().
		Description("Cập nhật thông tin profile theo user ID - chỉ dành cho admin và super_admin. Admin chỉ được cập nhật profile của chính mình hoặc profile có role khác 'admin'. Super_admin được phép cập nhật mọi profile").
		Register()
	users.Delete("/:id", ak.UserHandler.DeleteUserByID).
		Allow("admin", "super_admin").
		Fixed().
		Description("Xóa user theo ID - chỉ dành cho admin và super_admin. Admin chỉ được xóa các user không có role 'admin' và 'super_admin' (soft delete). Super_admin được phép xóa bất kỳ user nào không chứa role 'super_admin' (hard delete)").
		Register()
	users.Get("/", ak.UserHandler.ListUsers).
		Allow("admin", "super_admin").
		Fixed().
		Description("Danh sách users với pagination (trả về id, email, full_name, mobile, address)").
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
		Forbid("reader", "author").
		Description("Xóa blog").
		Register()
	blogs.Get("/my", blogHandler.ListMyBlogs).
		Forbid("editor", "admin").
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

	roles.Get("/:role_id_name/users", ak.RoleHandler.ListUsersHasRole).
		Allow("admin").
		Fixed().
		Description("Danh sách users có role (role_id_name có thể là số hoặc chuỗi)").
		Register()

	// User role routes (admin only)
	usersRoles := apiRouter.Group("/users")
	usersRoles.Post("/:user_id/roles/:role_id", ak.RoleHandler.AddRoleToUser).
		Allow("admin").
		Fixed().
		Description("Thêm role cho user").
		Register()

	usersRoles.Delete("/:user_id/roles/:role_id", ak.RoleHandler.RemoveRoleFromUser).
		Allow("admin").
		Fixed().
		Description("Xóa role của user").
		Register()

	usersRoles.Get("/:user_id/roles/:role_name/check", ak.RoleHandler.CheckUserHasRole).
		Allow("admin").
		Fixed().
		Description("Kiểm tra user có role").
		Register()

	usersRoles.Put("/:userId/roles", ak.RoleHandler.UpdateUserRoles).
		Allow("admin", "super_admin").
		Fixed().
		Description("Cập nhật danh sách roles cho user (chỉ admin và super_admin được phép)").
		Register()

	// Rule routes (admin only)
	// Lưu ý: Rules được đồng bộ trực tiếp từ cấu hình routes trong code (setupRoutes.go)
	// qua hàm SyncRoutesToDatabase(). Do đó không cần API để tạo/xóa rule.
	rules := apiRouter.Group("/rules")
	rules.Get("/", ak.RuleHandler.ListRules).
		Allow("admin").
		Fixed().
		Description("Danh sách rules").
		Register()
	rules.Get("/:id", ak.RuleHandler.GetByID).
		Allow("admin").
		Fixed().
		Description("Lấy thông tin rule theo ID (ID có dạng: GET|/api/blogs/*, cần URL encode khi gọi REST API)").
		Register()
	rules.Put("/", ak.RuleHandler.UpdateRule).
		Allow("admin").
		Fixed().
		Description("Cập nhật rule (chỉ cho phép cập nhật Type và Roles, không thể tạo/xóa)").
		Register()

	// Demo routes - demonstrate new features
	demo := apiRouter.Group("/demo")
	demo.Post("/login-with-username", demoHandler.LoginWithUsername).
		Public().
		Description("Đăng nhập với username và custom fields trong JWT token").
		Register()
	demo.Get("/token-info", demoHandler.GetTokenInfo).
		Allow().
		Description("Lấy thông tin từ flexible JWT token (username, custom fields, role conversion)").
		Register()

	// Route registry endpoint - trả về tất cả routes đã đăng ký
	apiRouter.Get("/routeregistry", demoHandler.GetRouteRegistry).
		Public().
		Fixed().
		Description("Lấy danh sách tất cả routes đã đăng ký trong RouteRegistry").
		Register()

	// Foo route - chỉ cho phép non-logged in users
	apiRouter.Get("/foo", demoHandler.Foo).
		Forbid().
		Description("Foo endpoint - cấm tất cả logged in users").
		Register()

	// Bar route - cấm users có role "reader" hoặc "admin"
	apiRouter.Get("/bar", demoHandler.Bar).
		Forbid("reader", "editor", "admin").
		//Override().
		Description("Bar dùng Override() để ghi đè rule code từ database").
		Register()
}
