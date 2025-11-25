# 8. Tích hợp và Sử dụng

Tài liệu này cung cấp hướng dẫn thực tế để tích hợp AuthKit vào ứng dụng Fiber, bao gồm quick start, các use case phổ biến và troubleshooting.

> 📖 **Lưu ý**: Để hiểu về kiến trúc và cơ chế hoạt động, xem các tài liệu khác:
> - [2. Kiến trúc tổng thể](./02-kien-truc-tong-the.md) - Dependency Injection và Route Registration Flow
> - [4. Hệ thống phân quyền](./04-he-thong-phan-quyen.md) - Sync Routes Flow chi tiết
> - [6. Generic Types và Extensibility](./06-generic-types-extensibility.md) - Builder Pattern

---

## 8.1. Quick Start

### 8.1.1. Ví dụ đầy đủ

```go
package main

import (
    "github.com/gofiber/fiber/v2"
    "github.com/techmaster-vietnam/authkit"
    "github.com/techmaster-vietnam/authkit/router"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
)

func main() {
    // 1. Load config
    cfg := authkit.LoadConfig()
    
    // 2. Connect database
    db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    
    // 3. Create Fiber app
    app := fiber.New()
    
    // 4. Initialize AuthKit
    ak, err := authkit.New[*authkit.BaseUser, *authkit.BaseRole](app, db).
        WithConfig(cfg).
        WithUserModel(&authkit.BaseUser{}).
        WithRoleModel(&authkit.BaseRole{}).
        Initialize()
    if err != nil {
        panic(err)
    }
    
    // 5. Setup routes
    apiRouter := router.NewAuthRouter(app, ak.RouteRegistry, 
        ak.AuthMiddleware, ak.AuthorizationMiddleware).Group("/api")
    
    apiRouter.Post("/auth/login", ak.AuthHandler.Login).
        Public().
        Register()
    
    apiRouter.Get("/users", userHandler.List).
        Allow("admin").
        Register()
    
    // 6. Sync routes to database
    ak.SyncRoutes()
    ak.InvalidateCache()
    
    // 7. Start server
    app.Listen(":8080")
}
```

### 8.1.2. Checklist tích hợp

- [ ] Cài đặt dependencies: `go get github.com/techmaster-vietnam/authkit`
- [ ] Tạo database PostgreSQL
- [ ] Cấu hình environment variables (JWT_SECRET, DB_*)
- [ ] Khởi tạo AuthKit với Builder Pattern
- [ ] Định nghĩa routes với Fluent API
- [ ] Gọi `SyncRoutes()` để đồng bộ rules vào database
- [ ] Gọi `InvalidateCache()` để refresh cache
- [ ] Seed initial data (roles, users) nếu cần

---

## 8.2. Common Use Cases

### 8.2.1. Sử dụng với Custom User Model

```go
// Define CustomUser
type CustomUser struct {
    authkit.BaseUser `gorm:"embedded"`
    Mobile  string `gorm:"type:varchar(15)"`
    Address string `gorm:"type:varchar(200)"`
}

// Implement UserInterface (delegate to BaseUser)
func (u *CustomUser) GetID() string { return u.BaseUser.GetID() }
// ... implement other methods

// Initialize với CustomUser
ak, err := authkit.New[*CustomUser, *authkit.BaseRole](app, db).
    WithUserModel(&CustomUser{}).
    Initialize()

// Sử dụng type-safe
user, _ := authkit.GetUserFromContextGeneric[*CustomUser](c)
mobile := user.Mobile  // ✅ Type-safe, không cần ép kiểu
```

### 8.2.2. Định nghĩa Routes với các Access Types

```go
apiRouter := router.NewAuthRouter(app, ak.RouteRegistry, 
    ak.AuthMiddleware, ak.AuthorizationMiddleware).Group("/api")

// PUBLIC - Không cần authentication
apiRouter.Get("/blogs", blogHandler.List).
    Public().
    Description("Danh sách blog công khai").
    Register()

// ALLOW - Yêu cầu authentication, cho phép mọi user đã đăng nhập
apiRouter.Get("/auth/profile", ak.AuthHandler.GetProfile).
    Allow().
    Register()

// ALLOW với roles cụ thể
apiRouter.Post("/blogs", blogHandler.Create).
    Allow("author", "editor", "admin").
    Description("Tạo blog mới").
    Register()

// FORBID - Cấm một số roles cụ thể
apiRouter.Delete("/blogs/:id", blogHandler.Delete).
    Forbid("guest").
    Description("Xóa blog (cấm guest)").
    Register()

// Fixed rule - Không thể sửa từ database
apiRouter.Get("/admin/users", adminHandler.List).
    Allow("admin").
    Fixed().
    Register()

// Override rule - Luôn ghi đè cấu hình từ code lên database
apiRouter.Put("/blogs/:id", blogHandler.Update).
    Allow("author", "editor", "admin").
    Override().  // Luôn update rule trong DB khi sync
    Description("Cập nhật blog").
    Register()
```

### 8.2.3. Lấy User từ Context

```go
// Với BaseUser
user, ok := authkit.GetUserFromContext(c)
if !ok {
    return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
}

// Với CustomUser (type-safe)
user, ok := authkit.GetUserFromContextGeneric[*CustomUser](c)
if !ok {
    return c.Status(401).JSON(fiber.Map{"error": "Unauthorized"})
}
mobile := user.Mobile  // Truy cập custom fields
```

### 8.2.4. Sử dụng AuthService trực tiếp

```go
// Login
loginReq := service.BaseLoginRequest{
    Email:    "user@example.com",
    Password: "password123",
}
response, err := ak.AuthService.Login(loginReq)
if err != nil {
    // Handle error
}
token := response.Token
user := response.User

// Register
registerReq := service.BaseRegisterRequest{
    Email:    "newuser@example.com",
    Password: "password123",
    FullName: "New User",
}
user, err := ak.AuthService.Register(registerReq)

// Change Password
err := ak.AuthService.ChangePassword(userID, oldPassword, newPassword)
```

---

## 8.3. Error Handling

AuthKit sử dụng `goerrorkit` để xử lý errors:

```go
import "github.com/techmaster-vietnam/goerrorkit"

// Authentication errors (401)
if err := ak.AuthService.Login(req); err != nil {
    if authErr, ok := err.(*goerrorkit.AuthError); ok {
        // authErr.Code = 401
        // authErr.Message = "Email hoặc mật khẩu không đúng"
    }
}

// Authorization errors (403)
// Tự động trả về bởi AuthorizationMiddleware

// Validation errors (400)
if err := ak.AuthService.Register(req); err != nil {
    if valErr, ok := err.(*goerrorkit.ValidationError); ok {
        // valErr.Fields = map[string]interface{}{"field": "email"}
    }
}
```

**Error Types:**
- `AuthError` (401): Authentication failures
- `BusinessError` (403): Authorization failures, business logic errors
- `ValidationError` (400): Input validation errors
- `SystemError` (500): System/internal errors

---

## 8.4. Troubleshooting

### 8.4.1. Routes không được sync vào database

**Vấn đề**: Sau khi định nghĩa routes, rules không xuất hiện trong database.

**Giải pháp**:
```go
// Đảm bảo gọi SyncRoutes() sau khi định nghĩa routes
ak.SyncRoutes()

// Refresh cache sau khi sync
ak.InvalidateCache()
```

### 8.4.2. Token không hợp lệ

**Vấn đề**: Token bị reject với lỗi "Token không hợp lệ".

**Kiểm tra**:
- JWT_SECRET có đúng không?
- Token đã hết hạn chưa? (check `exp` claim)
- Token có đúng format không? (Bearer token trong header)

### 8.4.3. User không có quyền truy cập

**Vấn đề**: User đã đăng nhập nhưng vẫn bị 403 Forbidden.

**Kiểm tra**:
- Routes đã được sync chưa? (`ak.SyncRoutes()`)
- Cache đã được refresh chưa? (`ak.InvalidateCache()`)
- User có đúng roles không? (check trong database `user_roles` table)
- Rule có đúng không? (check trong database `rules` table)

### 8.4.4. Custom fields không được lưu vào database

**Vấn đề**: CustomUser với Mobile và Address nhưng không có trong database.

**Giải pháp**:
```go
// Đảm bảo truyền model vào WithUserModel()
ak, err := authkit.New[*CustomUser, *authkit.BaseRole](app, db).
    WithUserModel(&CustomUser{}).  // ✅ Quan trọng!
    Initialize()

// Auto migrate sẽ tự động tạo các cột custom
```

### 8.4.5. Role names không được convert thành IDs

**Vấn đề**: Khi sync routes, role names không được convert thành role IDs.

**Kiểm tra**:
- Roles đã được seed vào database chưa?
- Role names trong code có khớp với names trong database không?
- Check logs để xem có lỗi khi convert không?

---

## 8.5. Best Practices

### ✅ Do's

1. **Luôn gọi SyncRoutes() sau khi thay đổi routes**
   ```go
   // Sau khi định nghĩa routes
   ak.SyncRoutes()
   ak.InvalidateCache()
   ```

2. **Sử dụng Fixed() cho critical endpoints**
   ```go
   apiRouter.Get("/admin/users", handler).
       Allow("admin").
       Fixed().  // Bảo vệ khỏi thay đổi từ database
       Register()
   ```

3. **Sử dụng Override() khi cần luôn đồng bộ từ code**
   ```go
   apiRouter.Put("/blogs/:id", handler).
       Allow("author", "editor").
       Override().  // Luôn ghi đè cấu hình từ code lên DB khi sync
       Register()
   ```
   - Override và Fixed loại trừ lẫn nhau, không thể dùng cùng lúc
   - Override=true: SyncRoutes() sẽ update rule nếu đã tồn tại trong DB
   - Fixed=true: SyncRoutes() chỉ tạo mới, không update

4. **Sử dụng Description() để mô tả routes**
   ```go
   apiRouter.Post("/blogs", handler).
       Allow("author").
       Description("Tạo blog mới").  // Hữu ích cho documentation
       Register()
   ```

5. **Seed roles trước khi sync routes**
   ```go
   // Seed roles trước
   SeedRoles(db)
   
   // Sau đó sync routes (cần roles để convert names → IDs)
   ak.SyncRoutes()
   ```

### ❌ Don'ts

1. **Không quên gọi InvalidateCache() sau SyncRoutes()**
   ```go
   ak.SyncRoutes()
   ak.InvalidateCache()  // ✅ Cần thiết!
   ```

2. **Không hard-code role IDs trong code**
   ```go
   // ❌ Sai: Hard-code role IDs
   Allow("1", "2", "3")
   
   // ✅ Đúng: Sử dụng role names
   Allow("admin", "editor")
   ```

3. **Không modify Fixed rules từ database**
   - Fixed rules được bảo vệ, không thể update/delete qua API
   - Nếu cần thay đổi, sửa trong code và sync lại

---

**Xem thêm:**
- [2. Kiến trúc tổng thể](./02-kien-truc-tong-the.md) - Dependency Injection và Route Registration Flow
- [4. Hệ thống phân quyền](./04-he-thong-phan-quyen.md) - Sync Routes Flow chi tiết
- [6. Generic Types và Extensibility](./06-generic-types-extensibility.md) - Custom Models
- [Mục lục](./README.md)
