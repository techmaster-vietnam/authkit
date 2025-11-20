# AuthKit - Module Authentication & Authorization cho Fiber

Module Go tái sử dụng cao cho ứng dụng Fiber REST API với authentication và authorization sử dụng GORM, PostgreSQL và goerrorkit.

## Mục lục

1. [Cài đặt và Tích hợp](#1-cài-đặt-và-tích-hợp)
2. [Định nghĩa Roles](#2-định-nghĩa-roles)
3. [Viết Route-Handler với Phân quyền](#3-viết-route-handler-với-phân-quyền)
4. [Custom User Model](#4-custom-user-model)
5. [Kỹ thuật Nâng cao](#5-kỹ-thuật-nâng-cao)

---

## 1. Cài đặt và Tích hợp

### 1.1. Tải về AuthKit

```bash
go get github.com/techmaster-vietnam/authkit
```

### 1.2. Cấu hình Environment Variables

Tạo file `.env` trong thư mục dự án của bạn:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=authkit
DB_SSLMODE=disable

# JWT
JWT_SECRET=your-secret-key-change-in-production
JWT_EXPIRATION_HOURS=24

# Server
PORT=3000
READ_TIMEOUT_SECONDS=10
WRITE_TIMEOUT_SECONDS=10
```

### 1.3. Tích hợp vào Ứng dụng (Bước đơn giản nhất)

Đây là cách tích hợp AuthKit vào ứng dụng Fiber của bạn với các bước tối thiểu:

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/gofiber/fiber/v2/middleware/requestid"
    "github.com/joho/godotenv"
    "github.com/techmaster-vietnam/authkit"
    "github.com/techmaster-vietnam/goerrorkit"
    fiberadapter "github.com/techmaster-vietnam/goerrorkit/adapters/fiber"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // 1. Load .env file (optional)
    _ = godotenv.Load()

    // 2. Khởi tạo goerrorkit logger (nếu bạn sử dụng goerrorkit)
    goerrorkit.InitLogger(goerrorkit.LoggerOptions{
        ConsoleOutput: true,
        FileOutput:    true,
        FilePath:      "logs/errors.log",
        JSONFormat:    true,
        MaxFileSize:   10,
        MaxBackups:    5,
        MaxAge:        30,
        LogLevel:      "info",
    })
    goerrorkit.ConfigureForApplication("main")

    // 3. Load config từ environment variables
    cfg := authkit.LoadConfig()

    // 4. Kết nối database
    dsn := fmt.Sprintf(
        "host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
        getEnv("DB_HOST", "localhost"),
        getEnv("DB_USER", "postgres"),
        getEnv("DB_PASSWORD", "postgres"),
        getEnv("DB_NAME", "authkit"),
        getEnv("DB_PORT", "5432"),
        getEnv("DB_SSLMODE", "disable"),
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // 5. Tạo Fiber app
    app := fiber.New(fiber.Config{
        AppName: "My App",
    })

    // 6. Cấu hình middleware
    app.Use(requestid.New())
    app.Use(logger.New())
    app.Use(fiberadapter.ErrorHandler()) // goerrorkit error handler
    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowHeaders: "Origin, Content-Type, Accept, Authorization",
        AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
    }))

    // 7. Khởi tạo AuthKit với BaseUser và BaseRole (mặc định)
    ak, err := authkit.New[*authkit.BaseUser, *authkit.BaseRole](app, db).
        WithConfig(cfg).
        WithUserModel(&authkit.BaseUser{}).
        WithRoleModel(&authkit.BaseRole{}).
        Initialize()

    if err != nil {
        log.Fatal("Failed to initialize AuthKit:", err)
    }

    // 8. Setup routes của bạn (xem phần 3)
    setupRoutes(app, ak)

    // 9. Sync routes vào database (quan trọng!)
    if err := ak.SyncRoutes(); err != nil {
        log.Fatal("Failed to sync routes:", err)
    }

    // 10. Refresh cache sau khi sync routes
    ak.InvalidateCache()

    // 11. Start server
    log.Printf("Server starting on port %s", cfg.Server.Port)
    if err := app.Listen(":" + cfg.Server.Port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// setupRoutes sẽ được định nghĩa ở phần 3
func setupRoutes(app *fiber.App, ak *authkit.AuthKit[*authkit.BaseUser, *authkit.BaseRole]) {
    // Xem phần 3 để biết cách viết routes
}
```

**Lưu ý quan trọng:**
- AuthKit sẽ tự động migrate database khi bạn gọi `Initialize()`
- Bạn **phải** gọi `ak.SyncRoutes()` sau khi setup tất cả routes để đồng bộ rules vào database
- Bạn **phải** gọi `ak.InvalidateCache()` sau khi sync routes để refresh cache

---

## 2. Định nghĩa Roles

### 2.1. Tạo Roles trong Database

Roles được lưu trong bảng `roles`. Bạn có thể tạo roles bằng cách:

**Cách 1: Tạo trực tiếp trong database**

```sql
INSERT INTO roles (id, name, created_at, updated_at) VALUES
(1, 'admin', NOW(), NOW()),
(2, 'editor', NOW(), NOW()),
(3, 'author', NOW(), NOW()),
(4, 'reader', NOW(), NOW());
```

**Cách 2: Tạo bằng code (khuyến nghị)**

```go
func initRoles(db *gorm.DB) error {
    roles := []*authkit.Role{
        {ID: 1, Name: "admin"},
        {ID: 2, Name: "editor"},
        {ID: 3, Name: "author"},
        {ID: 4, Name: "reader"},
    }

    for _, role := range roles {
        // FirstOrCreate: tìm theo Name, nếu không có thì tạo mới
        result := db.Where("name = ?", role.Name).FirstOrCreate(role)
        if result.Error != nil {
            return fmt.Errorf("failed to create role %s: %w", role.Name, result.Error)
        }
    }

    return nil
}

// Gọi trong main() sau khi kết nối database
func main() {
    // ... kết nối database ...
    
    if err := initRoles(db); err != nil {
        log.Fatal("Failed to init roles:", err)
    }
    
    // ... tiếp tục ...
}
```

### 2.2. Gán Roles cho User

```go
func assignRoleToUser(db *gorm.DB, userEmail string, roleName string) error {
    // Tìm user
    var user authkit.BaseUser
    if err := db.Where("email = ?", userEmail).First(&user).Error; err != nil {
        return err
    }

    // Tìm role
    var role authkit.Role
    if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
        return err
    }

    // Gán role cho user
    return db.Model(&user).Association("Roles").Append(&role)
}

// Gán nhiều roles cùng lúc
func assignRolesToUser(db *gorm.DB, userEmail string, roleNames []string) error {
    var user authkit.BaseUser
    if err := db.Where("email = ?", userEmail).First(&user).Error; err != nil {
        return err
    }

    var roles []authkit.Role
    for _, roleName := range roleNames {
        var role authkit.Role
        if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
            return err
        }
        roles = append(roles, role)
    }

    return db.Model(&user).Association("Roles").Replace(roles)
}
```

### 2.3. System Roles

AuthKit hỗ trợ **system roles** - các roles không thể xóa. Để tạo system role:

```go
role := &authkit.Role{
    ID:   1,
    Name: "super_admin",
    // System role được đánh dấu trong database
}
// System role sẽ được xử lý tự động bởi AuthKit
```

**Lưu ý:** Role `super_admin` có quyền bypass mọi rule authorization.

---

## 3. Viết Route-Handler với Phân quyền

AuthKit cung cấp **Fluent API** để định nghĩa routes với phân quyền một cách dễ dàng.

### 3.1. Import cần thiết

```go
import (
    "github.com/gofiber/fiber/v2"
    "github.com/techmaster-vietnam/authkit"
    "github.com/techmaster-vietnam/authkit/router"
)
```

### 3.2. Tạo AuthRouter

```go
func setupRoutes(
    app *fiber.App,
    ak *authkit.AuthKit[*authkit.BaseUser, *authkit.BaseRole],
) {
    // Tạo AuthRouter với group "/api"
    apiRouter := router.NewAuthRouter(
        app,
        ak.RouteRegistry,
        ak.AuthMiddleware,
        ak.AuthorizationMiddleware,
    ).Group("/api")

    // Bây giờ bạn có thể định nghĩa routes với phân quyền
}
```

### 3.3. Các loại Phân quyền

#### 3.3.1. Public - Route công khai (không cần đăng nhập)

```go
apiRouter.Get("/public/data", myHandler.GetPublicData).
    Public().
    Description("Lấy dữ liệu công khai").
    Register()
```

**Đặc điểm:**
- Không cần JWT token
- Bất kỳ ai cũng có thể truy cập
- Không áp dụng authentication middleware

#### 3.3.2. Allow - Cho phép các roles cụ thể

**Cho phép mọi user đã đăng nhập:**

```go
apiRouter.Get("/profile", authHandler.GetProfile).
    Allow().  // Không truyền roles = mọi user đã đăng nhập đều được
    Description("Lấy thông tin profile").
    Register()
```

**Cho phép các roles cụ thể:**

```go
apiRouter.Post("/blogs", blogHandler.Create).
    Allow("author", "editor", "admin").  // Chỉ các roles này được phép
    Description("Tạo blog mới").
    Register()
```

**Đặc điểm:**
- Yêu cầu JWT token (phải đăng nhập)
- Nếu không truyền roles: mọi user đã đăng nhập đều được
- Nếu truyền roles: chỉ các roles được chỉ định mới được phép

#### 3.3.3. Forbid - Cấm các roles cụ thể

```go
apiRouter.Delete("/blogs/:id", blogHandler.Delete).
    Forbid("reader").  // Cấm role "reader"
    Description("Xóa blog").
    Register()
```

**Đặc điểm:**
- Yêu cầu JWT token (phải đăng nhập)
- Nếu không truyền roles: cấm mọi user đã đăng nhập
- Nếu truyền roles: chỉ cấm các roles được chỉ định
- **Lưu ý:** Forbid có ưu tiên cao hơn Allow. Nếu user có nhiều roles và một role bị Forbid → bị từ chối

#### 3.3.4. Fixed - Rule không thể thay đổi từ database

```go
apiRouter.Get("/admin/users", adminHandler.ListUsers).
    Allow("admin").
    Fixed().  // Rule này không thể thay đổi từ API
    Description("Danh sách users (chỉ admin)").
    Register()
```

**Đặc điểm:**
- Rule được đánh dấu là "fixed" trong database
- Không thể cập nhật hoặc xóa rule này thông qua API `/api/rules`
- Hữu ích cho các routes quan trọng cần bảo vệ

### 3.4. Cú pháp đầy đủ

```go
apiRouter.<METHOD>(<PATH>, <HANDLER>).
    <ACCESS_TYPE>(<ROLES...>).  // Public(), Allow(), hoặc Forbid(roles...)
    Fixed().                     // Optional: đánh dấu rule không thể thay đổi
    Description("<MÔ_TẢ>").      // Optional: mô tả route
    Register()                    // Bắt buộc: đăng ký route
```

### 3.5. Ví dụ đầy đủ

```go
func setupRoutes(
    app *fiber.App,
    ak *authkit.AuthKit[*authkit.BaseUser, *authkit.BaseRole],
    blogHandler *BlogHandler,
) {
    // Tạo AuthRouter
    apiRouter := router.NewAuthRouter(
        app,
        ak.RouteRegistry,
        ak.AuthMiddleware,
        ak.AuthorizationMiddleware,
    ).Group("/api")

    // ===== AUTH ROUTES =====
    auth := apiRouter.Group("/auth")
    
    // Public routes
    auth.Post("/login", ak.AuthHandler.Login).
        Public().
        Description("Đăng nhập").
        Register()
    
    auth.Post("/register", ak.AuthHandler.Register).
        Public().
        Description("Đăng ký").
        Register()
    
    // Protected routes (mọi user đã đăng nhập)
    auth.Get("/profile", ak.AuthHandler.GetProfile).
        Allow().
        Description("Lấy profile").
        Register()
    
    auth.Put("/profile", ak.AuthHandler.UpdateProfile).
        Allow().
        Description("Cập nhật profile").
        Register()

    // ===== BLOG ROUTES =====
    blogs := apiRouter.Group("/blogs")
    
    // Public: ai cũng xem được
    blogs.Get("/", blogHandler.List).
        Public().
        Description("Danh sách blog công khai").
        Register()
    
    // Allow: chỉ các roles được chỉ định
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
        Allow("editor", "admin").
        Description("Xóa blog").
        Register()
    
    // Allow: mọi user đã đăng nhập
    blogs.Get("/my", blogHandler.ListMyBlogs).
        Allow().
        Description("Danh sách blog của tôi").
        Register()

    // ===== ADMIN ROUTES =====
    admin := apiRouter.Group("/admin")
    
    admin.Get("/users", adminHandler.ListUsers).
        Allow("admin").
        Fixed().
        Description("Danh sách users (chỉ admin)").
        Register()
    
    admin.Delete("/users/:id", adminHandler.DeleteUser).
        Allow("admin").
        Fixed().
        Description("Xóa user (chỉ admin)").
        Register()
}
```

### 3.6. Viết Handler

Handler là các hàm xử lý request. Ví dụ:

```go
type BlogHandler struct{}

func NewBlogHandler() *BlogHandler {
    return &BlogHandler{}
}

// GET /api/blogs
func (h *BlogHandler) List(c *fiber.Ctx) error {
    // Logic xử lý
    return c.JSON(fiber.Map{
        "success": true,
        "data": []string{"blog1", "blog2"},
    })
}

// GET /api/blogs/:id
func (h *BlogHandler) GetByID(c *fiber.Ctx) error {
    id := c.Params("id")
    
    // Lấy user từ context (nếu route yêu cầu auth)
    user, ok := authkit.GetUserFromContextGeneric[*authkit.BaseUser](c)
    if ok {
        // User đã đăng nhập
        fmt.Printf("User ID: %s\n", user.GetID())
    }
    
    return c.JSON(fiber.Map{
        "success": true,
        "id": id,
    })
}

// POST /api/blogs
func (h *BlogHandler) Create(c *fiber.Ctx) error {
    // Lấy user từ context
    user, ok := authkit.GetUserFromContextGeneric[*authkit.BaseUser](c)
    if !ok {
        return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
    }
    
    // Logic tạo blog
    return c.JSON(fiber.Map{
        "success": true,
        "message": "Blog created",
        "author_id": user.GetID(),
    })
}
```

### 3.7. Lấy User từ Context

Khi route yêu cầu authentication, bạn có thể lấy user từ context:

```go
// Với BaseUser
user, ok := authkit.GetUserFromContextGeneric[*authkit.BaseUser](c)
if ok {
    userID := user.GetID()
    userEmail := user.GetEmail()
    // ...
}

// Với CustomUser (xem phần 4)
user, ok := authkit.GetUserFromContextGeneric[*CustomUser](c)
if ok {
    userID := user.GetID()
    userMobile := user.Mobile  // Custom field
    // ...
}

// Chỉ lấy UserID (nhanh hơn)
userID, ok := authkit.GetUserIDFromContext(c)
if ok {
    // Sử dụng userID
}
```

---

## 4. Custom User Model

Nếu bạn cần thêm các trường bổ sung vào User model (ví dụ: `mobile`, `address`, `company_id`), bạn có thể tạo Custom User model.

### 4.1. Tạo Custom User Model

```go
package main

import (
    "github.com/techmaster-vietnam/authkit"
    "github.com/techmaster-vietnam/authkit/core"
)

// CustomUser là User model với các trường bổ sung
type CustomUser struct {
    authkit.BaseUser `gorm:"embedded"` // Embed BaseUser để kế thừa tất cả trường
    
    // Các trường bổ sung
    Mobile  string `gorm:"type:varchar(15)" json:"mobile"`
    Address string `gorm:"type:varchar(200)" json:"address"`
    // Thêm các trường khác nếu cần
}

// Implement UserInterface bằng cách delegate về BaseUser
func (u *CustomUser) GetID() string {
    return u.BaseUser.GetID()
}

func (u *CustomUser) GetEmail() string {
    return u.BaseUser.GetEmail()
}

func (u *CustomUser) SetEmail(email string) {
    u.BaseUser.SetEmail(email)
}

func (u *CustomUser) GetPassword() string {
    return u.BaseUser.GetPassword()
}

func (u *CustomUser) SetPassword(password string) {
    u.BaseUser.SetPassword(password)
}

func (u *CustomUser) IsActive() bool {
    return u.BaseUser.IsActive()
}

func (u *CustomUser) SetActive(active bool) {
    u.BaseUser.SetActive(active)
}

func (u *CustomUser) GetRoles() []core.RoleInterface {
    return u.BaseUser.GetRoles()
}

func (u *CustomUser) GetFullName() string {
    return u.BaseUser.GetFullName()
}

func (u *CustomUser) SetFullName(fullName string) {
    u.BaseUser.SetFullName(fullName)
}

// TableName: sử dụng cùng bảng "users"
func (CustomUser) TableName() string {
    return "users"
}
```

### 4.2. Sử dụng Custom User trong AuthKit

```go
func main() {
    // ... kết nối database ...
    
    // Khởi tạo AuthKit với CustomUser
    ak, err := authkit.New[*CustomUser, *authkit.BaseRole](app, db).
        WithConfig(cfg).
        WithUserModel(&CustomUser{}).  // Sử dụng CustomUser
        WithRoleModel(&authkit.BaseRole{}).
        Initialize()
    
    if err != nil {
        log.Fatal("Failed to initialize AuthKit:", err)
    }
    
    // ... setup routes ...
}
```

### 4.3. Sử dụng Custom User trong Handler

```go
func (h *BlogHandler) Create(c *fiber.Ctx) error {
    // Lấy CustomUser từ context
    user, ok := authkit.GetUserFromContextGeneric[*CustomUser](c)
    if !ok {
        return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
    }
    
    // Sử dụng các trường custom
    fmt.Printf("User Mobile: %s\n", user.Mobile)
    fmt.Printf("User Address: %s\n", user.Address)
    
    // Vẫn có thể sử dụng các methods từ BaseUser
    fmt.Printf("User Email: %s\n", user.GetEmail())
    fmt.Printf("User ID: %s\n", user.GetID())
    
    return c.JSON(fiber.Map{
        "success": true,
        "user_mobile": user.Mobile,
    })
}
```

### 4.4. Tạo User với Custom Fields

```go
import (
    "github.com/techmaster-vietnam/authkit/utils"
    "golang.org/x/crypto/bcrypt"
)

func createUserWithCustomFields(db *gorm.DB) error {
    // Hash password - Cách 1: Sử dụng utils từ AuthKit (nếu có quyền truy cập)
    hashedPassword, err := utils.HashPassword("123456")
    if err != nil {
        return err
    }
    
    // Hoặc Cách 2: Sử dụng bcrypt trực tiếp
    // bytes, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
    // hashedPassword := string(bytes)
    
    // Tạo CustomUser
    user := &CustomUser{
        BaseUser: authkit.BaseUser{
            Email:    "user@example.com",
            Password: hashedPassword,
            FullName: "John Doe",
            Active:   true,
        },
        Mobile:  "0901234567",
        Address: "123 Main Street",
    }
    
    // Lưu vào database
    return db.Create(user).Error
}
```

**Lưu ý:**
- CustomUser phải embed `authkit.BaseUser` với tag `gorm:"embedded"`
- Phải implement tất cả methods của `core.UserInterface`
- Sử dụng cùng bảng `users` (hoặc chỉ định bảng khác nếu cần)
- AuthKit sẽ tự động migrate các trường custom khi bạn gọi `Initialize()`

---

## 5. Kỹ thuật Nâng cao

### 5.1. Sync Routes vào Database

Sau khi định nghĩa tất cả routes, bạn **phải** sync vào database:

```go
func main() {
    // ... setup routes ...
    
    // Sync routes vào database
    if err := ak.SyncRoutes(); err != nil {
        log.Fatal("Failed to sync routes:", err)
    }
    
    // Refresh cache sau khi sync
    ak.InvalidateCache()
}
```

**Lưu ý:**
- `SyncRoutes()` sẽ tạo/update các rules trong bảng `rules` dựa trên routes bạn đã định nghĩa
- Nếu route đã có trong database và không phải `Fixed`, nó sẽ được cập nhật
- Nếu route là `Fixed`, nó sẽ không bị thay đổi từ database

### 5.2. Quản lý Rules từ API

AuthKit cung cấp API để quản lý rules:

```bash
# Liệt kê tất cả rules
GET /api/rules

# Tạo rule mới
POST /api/rules
{
  "method": "GET",
  "path": "/api/custom/endpoint",
  "type": "ALLOW",
  "roles": ["admin"],
  "description": "Custom endpoint"
}

# Cập nhật rule
PUT /api/rules/:id
{
  "type": "FORBID",
  "roles": ["guest"]
}

# Xóa rule
DELETE /api/rules/:id
```

**Lưu ý:** Rules có `Fixed = true` không thể cập nhật hoặc xóa từ API.

### 5.3. Refresh Cache

Khi bạn thay đổi rules từ database (qua API hoặc trực tiếp), bạn cần refresh cache:

```go
// Refresh cache
ak.InvalidateCache()
```

Hoặc trong handler:

```go
func (h *AdminHandler) UpdateRule(c *fiber.Ctx) error {
    // ... cập nhật rule ...
    
    // Refresh cache
    ak.InvalidateCache()
    
    return c.JSON(fiber.Map{"success": true})
}
```

### 5.4. Sử dụng với Database Connection có sẵn

Nếu bạn đã có database connection từ dự án khác:

```go
func main() {
    // Giả sử bạn đã có db connection
    var existingDB *gorm.DB // = your existing connection
    
    // Chỉ cần truyền vào AuthKit
    ak, err := authkit.New[*authkit.BaseUser, *authkit.BaseRole](app, existingDB).
        WithConfig(cfg).
        WithUserModel(&authkit.BaseUser{}).
        WithRoleModel(&authkit.BaseRole{}).
        Initialize()
    
    // ... tiếp tục ...
}
```

### 5.5. Xử lý Lỗi với goerrorkit

Nếu bạn sử dụng goerrorkit:

```go
import (
    "github.com/techmaster-vietnam/goerrorkit"
    fiberadapter "github.com/techmaster-vietnam/goerrorkit/adapters/fiber"
)

func main() {
    // Khởi tạo logger
    goerrorkit.InitLogger(goerrorkit.LoggerOptions{
        ConsoleOutput: true,
        FileOutput:    true,
        FilePath:      "logs/errors.log",
        JSONFormat:    true,
        MaxFileSize:   10,
        MaxBackups:    5,
        MaxAge:        30,
        LogLevel:      "info",
    })
    goerrorkit.ConfigureForApplication("main")
    
    // Thêm error handler middleware
    app.Use(fiberadapter.ErrorHandler())
    
    // Trong handler, bạn có thể throw error
    func (h *BlogHandler) Create(c *fiber.Ctx) error {
        if someCondition {
            return goerrorkit.NewBusinessError("Cannot create blog").
                WithCode("BLOG_CREATE_FAILED").
                WithData(map[string]interface{}{
                    "reason": "Invalid data",
                })
        }
        return c.JSON(fiber.Map{"success": true})
    }
}
```

### 5.6. Best Practices

1. **Luôn gọi `SyncRoutes()` sau khi setup routes**
   ```go
   setupRoutes(app, ak)
   ak.SyncRoutes()
   ak.InvalidateCache()
   ```

2. **Sử dụng `Fixed()` cho các routes quan trọng**
   ```go
   apiRouter.Delete("/admin/users/:id", handler).
       Allow("admin").
       Fixed().  // Bảo vệ route quan trọng
       Register()
   ```

3. **Sử dụng `Description()` để mô tả route**
   ```go
   apiRouter.Get("/blogs", handler).
       Public().
       Description("Lấy danh sách blog công khai").
       Register()
   ```

4. **Refresh cache sau khi thay đổi rules**
   ```go
   // Sau khi update rule từ API
   ak.InvalidateCache()
   ```

5. **Sử dụng Custom User khi cần mở rộng**
   - Embed `BaseUser` thay vì copy code
   - Implement đầy đủ `UserInterface`
   - Sử dụng cùng bảng `users` hoặc chỉ định bảng riêng

6. **Kiểm tra user trong handler**
   ```go
   user, ok := authkit.GetUserFromContextGeneric[*CustomUser](c)
   if !ok {
       return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized")
   }
   ```

### 5.7. Troubleshooting

**Vấn đề: Route không được authorize đúng**

- Kiểm tra đã gọi `SyncRoutes()` chưa
- Kiểm tra đã gọi `InvalidateCache()` sau khi sync chưa
- Kiểm tra rule trong database có đúng không
- Kiểm tra user có đúng roles không

**Vấn đề: Custom User không hoạt động**

- Kiểm tra đã implement đầy đủ `UserInterface` chưa
- Kiểm tra đã embed `BaseUser` với tag `gorm:"embedded"` chưa
- Kiểm tra đã truyền đúng type vào `New()` chưa: `authkit.New[*CustomUser, *authkit.BaseRole]`

**Vấn đề: Database migration lỗi**

- Kiểm tra database connection
- Kiểm tra quyền của database user
- Kiểm tra các trường custom có conflict với BaseUser không

---

## Tổng kết

AuthKit cung cấp một cách đơn giản và mạnh mẽ để tích hợp authentication và authorization vào ứng dụng Fiber của bạn:

1. ✅ **Dễ tích hợp**: Chỉ cần vài dòng code
2. ✅ **Fluent API**: Định nghĩa routes với phân quyền dễ dàng
3. ✅ **Linh hoạt**: Hỗ trợ Custom User model
4. ✅ **Mạnh mẽ**: Hỗ trợ Public, Allow, Forbid, Fixed rules
5. ✅ **Tự động**: Tự động migrate database và sync routes

Chúc bạn code vui vẻ! 🚀
