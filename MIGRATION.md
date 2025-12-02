# Migration Guide - Tách User Management Handlers

## Tổng quan

Từ phiên bản này, các user management handlers đã được tách ra khỏi `BaseAuthHandler` và chuyển sang `BaseUserHandler` mới. Các routes user management đã được chuyển từ `/api/auth/*` sang `/api/user/*`.

## Breaking Changes

### 1. Routes đã thay đổi

| Route cũ | Route mới | Method | Mô tả |
|---------|-----------|--------|-------|
| `/api/auth/profile` | `/api/user/profile` | GET | Lấy profile của chính mình |
| `/api/auth/profile` | `/api/user/profile` | PUT | Cập nhật profile của chính mình |
| `/api/auth/profile` | `/api/user/profile` | DELETE | Xóa tài khoản |
| `/api/auth/profile/:id` | `/api/user/:id` | GET | Lấy profile theo ID (admin/super_admin) |
| `/api/auth/profile/:id` | `/api/user/:id` | PUT | Cập nhật profile theo ID (admin/super_admin) |
| `/api/auth/profile/:id` | `/api/user/:id` | DELETE | Xóa user theo ID (admin/super_admin) |
| `/api/users` | `/api/user` | GET | Danh sách users (admin/super_admin) |

**Lưu ý:** Route `/api/users/*` cho user role management (thêm/xóa roles) **KHÔNG thay đổi**.

### 2. Handler mới trong AuthKit

`AuthKit` struct giờ có thêm field `UserHandler`:

```go
type AuthKit[TUser UserInterface, TRole RoleInterface] struct {
    // ... các fields khác
    AuthHandler *BaseAuthHandler[TUser, TRole]
    UserHandler *BaseUserHandler[TUser, TRole]  // ← MỚI
    RoleHandler *BaseRoleHandler[TUser, TRole]
    RuleHandler *BaseRuleHandler[TUser, TRole]
}
```

## Hướng dẫn Migration

### Bước 1: Cập nhật Routes trong setupRoutes.go

**Trước:**
```go
auth := apiRouter.Group("/auth")

// User management routes (cũ)
auth.Get("/profile", ak.AuthHandler.GetProfile).
    Allow().
    Register()
auth.Put("/profile", ak.AuthHandler.UpdateProfile).
    Allow().
    Register()
auth.Delete("/profile", ak.AuthHandler.DeleteProfile).
    Allow().
    Register()
auth.Get("/profile/:id", ak.AuthHandler.GetProfileByID).
    Allow("admin", "super_admin").
    Register()
auth.Put("/profile/:id", ak.AuthHandler.UpdateProfileByID).
    Allow("admin", "super_admin").
    Register()
auth.Delete("/profile/:id", ak.AuthHandler.DeleteUserByID).
    Allow("admin", "super_admin").
    Register()

users := apiRouter.Group("/users")
users.Get("/", ak.AuthHandler.ListUsers).
    Allow("admin", "super_admin").
    Register()
```

**Sau:**
```go
auth := apiRouter.Group("/auth")
// Chỉ còn các auth routes: login, register, logout, refresh, change-password, etc.

// User management routes (mới)
users := apiRouter.Group("/user")
users.Get("/profile", ak.UserHandler.GetProfile).
    Allow().
    Register()
users.Put("/profile", ak.UserHandler.UpdateProfile).
    Allow().
    Register()
users.Delete("/profile", ak.UserHandler.DeleteProfile).
    Allow().
    Register()
users.Get("/:id", ak.UserHandler.GetProfileByID).
    Allow("admin", "super_admin").
    Register()
users.Put("/:id", ak.UserHandler.UpdateProfileByID).
    Allow("admin", "super_admin").
    Register()
users.Delete("/:id", ak.UserHandler.DeleteUserByID).
    Allow("admin", "super_admin").
    Register()
users.Get("/", ak.UserHandler.ListUsers).
    Allow("admin", "super_admin").
    Register()

// User role routes (KHÔNG đổi)
usersRoles := apiRouter.Group("/users")
usersRoles.Post("/:user_id/roles/:role_id", ak.RoleHandler.AddRoleToUser).
    Allow("admin").
    Register()
// ... các routes role khác
```

### Bước 2: Cập nhật Frontend/API Client

Tìm và thay thế tất cả các API calls:

**JavaScript/TypeScript:**
```javascript
// Trước
GET  /api/auth/profile
PUT  /api/auth/profile
DELETE /api/auth/profile
GET  /api/auth/profile/:id
PUT  /api/auth/profile/:id
DELETE /api/auth/profile/:id
GET  /api/users

// Sau
GET  /api/user/profile
PUT  /api/user/profile
DELETE /api/user/profile
GET  /api/user/:id
PUT  /api/user/:id
DELETE /api/user/:id
GET  /api/user
```

**Python:**
```python
# Trước
requests.get(f"{base_url}/api/auth/profile", ...)
requests.put(f"{base_url}/api/auth/profile", ...)
requests.delete(f"{base_url}/api/auth/profile", ...)
requests.get(f"{base_url}/api/auth/profile/{user_id}", ...)
requests.get(f"{base_url}/api/users", ...)

# Sau
requests.get(f"{base_url}/api/user/profile", ...)
requests.put(f"{base_url}/api/user/profile", ...)
requests.delete(f"{base_url}/api/user/profile", ...)
requests.get(f"{base_url}/api/user/{user_id}", ...)
requests.get(f"{base_url}/api/user", ...)
```

### Bước 3: Cập nhật Documentation

Cập nhật tất cả các tài liệu API, Postman collections, Swagger/OpenAPI specs để phản ánh các routes mới.

## Routes không thay đổi

Các routes sau **KHÔNG thay đổi**, tiếp tục hoạt động như cũ:

- `/api/auth/login`
- `/api/auth/register`
- `/api/auth/logout`
- `/api/auth/refresh`
- `/api/auth/change-password`
- `/api/auth/request-password-reset`
- `/api/auth/reset-password`
- `/api/users/:user_id/roles/:role_id` (POST/DELETE)
- `/api/users/:user_id/roles/:role_name/check` (GET)
- `/api/users/:userId/roles` (PUT)

## Checklist Migration

- [ ] Cập nhật `setupRoutes.go` - chuyển user management routes sang `/api/user` và sử dụng `ak.UserHandler`
- [ ] Cập nhật tất cả API calls trong frontend/client code
- [ ] Cập nhật test scripts và integration tests
- [ ] Cập nhật API documentation
- [ ] Cập nhật Postman collections / Swagger specs
- [ ] Test lại tất cả các endpoints user management

## Lợi ích của thay đổi

1. **Separation of Concerns**: Auth và User Management được tách biệt rõ ràng
2. **API Design tốt hơn**: Routes nhất quán và dễ hiểu hơn
3. **Dễ maintain**: Code được tổ chức tốt hơn, dễ mở rộng
4. **Scalability**: Dễ dàng tách thành microservice sau này

## Hỗ trợ

Nếu gặp vấn đề khi migration, vui lòng tạo issue trên repository hoặc liên hệ maintainer.

