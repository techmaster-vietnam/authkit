# Fiber Auth Module

Module Go tái sử dụng cao cho ứng dụng Fiber REST API với authentication và authorization sử dụng GORM, PostgreSQL và goerrorkit.

## Tính năng

### Authentication
- ✅ Đăng nhập (Login)
- ✅ Đăng xuất (Logout)
- ✅ Đăng ký (Register)
- ✅ Đổi mật khẩu (Change Password)
- ✅ Cập nhật profile (Update Profile)
- ✅ Xóa profile (Delete Profile)

### Quản lý Role
- ✅ Thêm role (Add Role)
- ✅ Xóa role (Remove Role)
- ✅ Liệt kê roles (List Roles)

### Quản lý Role User
- ✅ Thêm role cho user (Add Role to User)
- ✅ Xóa role khỏi user (Remove Role from User)
- ✅ Kiểm tra user có role (Check User Has Role)
- ✅ Liệt kê roles của user (List Roles of User)
- ✅ Liệt kê users có role (List Users Has Role)

### Quản lý Rule (Authorization Rules)
- ✅ Thêm rule (Add Rule)
- ✅ Cập nhật rule (Update Rule)
- ✅ Xóa rule (Remove Rule)
- ✅ Liệt kê rules (List Rules)

## Cài đặt

```bash
go get github.com/techmaster-vietnam/authkit
```

## Cấu hình

Tạo file `.env` hoặc set các biến môi trường:

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

## Sử dụng

### Nguyên tắc thiết kế

Module `authkit` được thiết kế để **tái sử dụng cao** và **độc lập**:
- ✅ **Tất cả dependencies được truyền từ bên ngoài**: GORM DB, goerrorkit, Fiber app
- ✅ **Module không tự khởi tạo dependencies**: Đảm bảo tính linh hoạt và dễ test
- ✅ **AuthKit chỉ tập trung vào authentication và authorization**: Không tự tạo database connection
- ✅ **Database connection phải được tạo từ bên ngoài**: Bạn tự tạo `*gorm.DB` và inject vào `router.SetupRoutes()`

### Khởi tạo ứng dụng cơ bản

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
    "github.com/techmaster-vietnam/authkit"
    "github.com/techmaster-vietnam/goerrorkit"
    fiberadapter "github.com/techmaster-vietnam/goerrorkit/adapters/fiber"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // 1. Khởi tạo goerrorkit (từ bên ngoài module)
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
    goerrorkit.ConfigureForApplication("your-app-module-path")

    // 2. Load config (có thể load từ env hoặc truyền từ bên ngoài)
    cfg := authkit.LoadConfig()

    // 3. Tạo database connection (từ bên ngoài module)
    // ⚠️ LƯU Ý: AuthKit chỉ nhận *gorm.DB đã được tạo sẵn
    // Bạn phải tự tạo database connection, AuthKit không cung cấp hàm Connect()
    dbHost := getEnv("DB_HOST", "localhost")
    dbPort := getEnv("DB_PORT", "5432")
    dbUser := getEnv("DB_USER", "postgres")
    dbPassword := getEnv("DB_PASSWORD", "postgres")
    dbName := getEnv("DB_NAME", "authkit")
    dbSSLMode := getEnv("DB_SSLMODE", "disable")

    dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
        dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("Failed to connect to database:", err)
    }

    // 4. Chạy migrations (từ bên ngoài module)
    if err := authkit.Migrate(db); err != nil {
        log.Fatal("Failed to run migrations:", err)
    }

    // 5. Tạo Fiber app (từ bên ngoài module)
    app := fiber.New(fiber.Config{
        AppName: "My App",
    })

    // 6. Cấu hình middleware (từ bên ngoài module)
    app.Use(requestid.New())
    app.Use(logger.New())
    app.Use(fiberadapter.ErrorHandler()) // goerrorkit error handler
    app.Use(cors.New(cors.Config{
        AllowOrigins: "*",
        AllowHeaders: "Origin, Content-Type, Accept, Authorization",
        AllowMethods: "GET, POST, PUT, DELETE, OPTIONS",
    }))

    // 7. Setup routes (module nhận tất cả dependencies từ bên ngoài)
    if err := authkit.SetupRoutes(app, db, cfg); err != nil {
        log.Fatal("Failed to setup routes:", err)
    }

    // 8. Start server
    log.Printf("Server starting on port %s", cfg.Server.Port)
    if err := app.Listen(":" + cfg.Server.Port); err != nil {
        log.Fatal("Failed to start server:", err)
    }
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

### Sử dụng với database connection có sẵn

Nếu bạn đã có database connection từ dự án khác, bạn có thể truyền trực tiếp:

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/techmaster-vietnam/authkit"
    "gorm.io/gorm"
)

func main() {
    // Giả sử bạn đã có DB connection từ nơi khác
    var existingDB *gorm.DB // = your existing connection
    
    // Chỉ cần chạy migrations
    authkit.Migrate(existingDB)
    
    // Load config
    cfg := authkit.LoadConfig()
    
    // Tạo app và setup routes
    app := fiber.New()
    authkit.SetupRoutes(app, existingDB, cfg)
    app.Listen(":3000")
}
```

### Xem thêm ví dụ

- `examples/demo/main.go`  - Ví dụ đầy đủ với goerrorkit (khuyến nghị)
- `examples/basic/main.go` - Ví dụ cơ bản
- `examples/init_rules.go` - Ví dụ khởi tạo rules

Để chạy ví dụ:
```bash
# Chạy demo với goerrorkit
cd examples/demo && go run main.go

# Chạy ví dụ cơ bản
cd examples/basic && go run main.go
```

## Quy tắc Authorization

### Rule Types

1. **PUBLIC**: Cho phép bất kỳ ai truy cập, kể cả anonymous (duy nhất cho phép anonymous)
2. **ALLOW**: Cho phép các role cụ thể (yêu cầu authentication)
   - Nếu `roles` rỗng: Mọi user đã đăng nhập đều được truy cập
   - Nếu `roles` không rỗng: Chỉ các role trong mảng mới được truy cập
3. **FORBIDE**: Cấm các role cụ thể (yêu cầu authentication)
   - Nếu `roles` rỗng: Cấm mọi user đã đăng nhập
   - Nếu `roles` không rỗng: Chỉ cấm các role trong mảng
4. **AUTHENTICATED**: Yêu cầu đăng nhập nhưng không cần role cụ thể

### Quy tắc mặc định

- **Endpoint mới không có trong rules**: Mặc định là **FORBIDE** (cấm)
- **Chỉ PUBLIC cho phép anonymous**: Tất cả rule khác đều yêu cầu authentication
- **Super Admin**: Role `super_admin` có thể bypass mọi rule
- **Xử lý xung đột**: FORBIDE có ưu tiên cao hơn ALLOW khi có nhiều rule match

### Quy tắc xử lý xung đột

Khi một endpoint có nhiều rule match:
1. **PUBLIC** → Cho phép ngay (bao gồm anonymous)
2. **FORBIDE** → Nếu user có role bị cấm → Từ chối (ưu tiên cao nhất)
3. **ALLOW** → Nếu user có role được phép → Cho phép
4. **AUTHENTICATED** → Nếu đã đăng nhập → Cho phép
5. Mặc định → Từ chối

**Nguyên tắc**: FORBIDE luôn ưu tiên hơn ALLOW. Nếu user có nhiều role và một role bị FORBIDE, một role được ALLOW → Kết quả là FORBIDE (từ chối).

### Optional Role Context

Bạn có thể chỉ định role context cụ thể thông qua header `X-Role-Context`:
- Header này là **optional**, không bắt buộc
- Nếu được cung cấp, hệ thống sẽ chỉ sử dụng role đó để kiểm tra authorization
- User phải có role được chỉ định, nếu không sẽ bị từ chối
- Hữu ích cho audit trail hoặc khi cần xác định rõ role được sử dụng

**Ví dụ**:
```bash
curl -H "Authorization: Bearer <token>" \
     -H "X-Role-Context: admin" \
     https://api.example.com/api/admin/users
```

### Ví dụ Rule

```json
{
  "method": "GET",
  "path": "/api/public/data",
  "type": "PUBLIC",
  "roles": [],
  "priority": 0
}
```

```json
{
  "method": "POST",
  "path": "/api/admin/users",
  "type": "ALLOW",
  "roles": ["admin", "super_admin"],
  "priority": 10
}
```

```json
{
  "method": "DELETE",
  "path": "/api/users/*",
  "type": "FORBIDE",
  "roles": ["guest"],
  "priority": 5
}
```

## API Endpoints

### Authentication

- `POST /api/auth/login` - Đăng nhập
- `POST /api/auth/register` - Đăng ký
- `POST /api/auth/logout` - Đăng xuất
- `GET /api/auth/profile` - Lấy thông tin profile (yêu cầu auth)
- `PUT /api/auth/profile` - Cập nhật profile (yêu cầu auth)
- `DELETE /api/auth/profile` - Xóa profile (yêu cầu auth)
- `POST /api/auth/change-password` - Đổi mật khẩu (yêu cầu auth)

### Roles

- `GET /api/roles` - Liệt kê roles (yêu cầu auth)
- `POST /api/roles` - Thêm role (yêu cầu auth)
- `DELETE /api/roles/:id` - Xóa role (yêu cầu auth)
- `GET /api/roles/:role_name/users` - Liệt kê users có role (yêu cầu auth)

### User Roles

- `GET /api/users/:user_id/roles` - Liệt kê roles của user (yêu cầu auth)
- `POST /api/users/:user_id/roles/:role_id` - Thêm role cho user (yêu cầu auth)
- `DELETE /api/users/:user_id/roles/:role_id` - Xóa role khỏi user (yêu cầu auth)
- `GET /api/users/:user_id/roles/:role_name/check` - Kiểm tra user có role (yêu cầu auth)

### Rules

- `GET /api/rules` - Liệt kê rules (yêu cầu auth)
- `POST /api/rules` - Thêm rule (yêu cầu auth)
- `PUT /api/rules/:id` - Cập nhật rule (yêu cầu auth)
- `DELETE /api/rules/:id` - Xóa rule (yêu cầu auth)

## Cải tiến Bảo mật

### Đã triển khai

1. ✅ **Default Deny**: Endpoint mới mặc định bị cấm
2. ✅ **Super Admin Bypass**: Role `super_admin` bypass mọi rule
3. ✅ **Rule Caching**: Cache rules để tăng hiệu suất
4. ✅ **JWT Authentication**: Sử dụng JWT cho authentication
5. ✅ **Password Hashing**: Sử dụng bcrypt để hash mật khẩu
6. ✅ **Soft Delete**: Soft delete cho users và roles
7. ✅ **System Roles**: System roles không thể xóa

### Đề xuất thêm

1. **Rate Limiting**: Thêm rate limiting cho login/register
2. **Audit Logging**: Log các request bị từ chối
3. **Token Blacklist**: Blacklist token khi logout (nếu cần)
4. **2FA**: Thêm two-factor authentication
5. **Password Policy**: Thêm policy cho mật khẩu mạnh hơn

## Cấu trúc Module

```
authkit/
├── authkit.go       # Package chính - export tất cả API công khai (LoadConfig, Migrate, SetupRoutes)
├── config/          # Cấu hình (internal)
├── database/        # Database migration (internal)
├── handlers/        # HTTP handlers (internal)
├── middleware/      # Authentication và authorization middleware (internal)
├── models/          # GORM models (export User, Role, Rule)
├── repository/      # Database repository layer (internal)
├── router/          # Route setup (internal)
├── service/         # Business logic layer (internal)
├── utils/           # Utilities (JWT, password hashing) (internal)
└── examples/         # Ví dụ sử dụng module
    ├── demo/        # Ví dụ đầy đủ với goerrorkit
    ├── basic/       # Ví dụ cơ bản
    └── init_rules.go # Ví dụ khởi tạo rules
```

### Nguyên tắc thiết kế module

- ✅ **Dependency Injection**: Tất cả dependencies (DB, config, app) được truyền từ bên ngoài
- ✅ **Không có side effects**: Module không tự khởi tạo global state
- ✅ **Export functions**: Chỉ export các constructor và utility functions
- ✅ **Tái sử dụng cao**: Có thể sử dụng với database connection có sẵn

## License

MIT

