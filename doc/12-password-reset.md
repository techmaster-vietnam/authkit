# Password Reset - Đặt lại mật khẩu không cần password cũ

## Tổng quan

Tính năng Password Reset cho phép người dùng đặt lại mật khẩu mà không cần biết mật khẩu cũ. Hệ thống sẽ gửi một reset token qua email hoặc tin nhắn để xác thực danh tính người dùng.

## Luồng hoạt động

```
1. User yêu cầu reset password (POST /api/auth/request-password-reset)
   └─> Nhập email
   └─> Hệ thống tạo reset token (hết hạn sau 1 giờ)
   └─> Gửi token qua email/tin nhắn (nếu có NotificationSender)

2. User nhận token từ email/tin nhắn

3. User đặt lại password (POST /api/auth/reset-password)
   └─> Gửi token + password mới
   └─> Hệ thống xác thực token
   └─> Đổi password thành công
   └─> Invalidate token và tất cả refresh tokens (force logout)
```

## Cài đặt

### 1. Chạy Migration

Migration đã được tạo sẵn trong `examples/migrations/000008_create_password_reset_tokens_table.up.sql`. Chạy migration để tạo bảng `password_reset_tokens`:

```bash
# Migration sẽ tự động chạy khi khởi động ứng dụng
# Hoặc chạy thủ công:
migrate -path examples/migrations -database "postgres://user:pass@localhost/dbname?sslmode=disable" up
```

### 2. Implement NotificationSender Interface

Bạn cần implement interface `core.NotificationSender` để gửi email/tin nhắn. Xem ví dụ trong `examples/notification_sender_example.go`:

```go
package main

import (
    "github.com/techmaster-vietnam/authkit/core"
)

// EmailNotificationSender implement core.NotificationSender
type EmailNotificationSender struct {
    // Inject email service của bạn (SMTP, SendGrid, AWS SES, v.v.)
}

func (e *EmailNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
    // Implement logic gửi email
    // Ví dụ: gửi email với reset token và link reset password
    return nil
}

// Đảm bảo implement interface
var _ core.NotificationSender = (*EmailNotificationSender)(nil)
```

### 3. Đăng ký NotificationSender

Sau khi khởi tạo AuthKit, set notification sender:

```go
ak, err := authkit.New[*CustomUser, *authkit.BaseRole](app, db).
    WithConfig(cfg).
    WithUserModel(&CustomUser{}).
    WithRoleModel(&authkit.BaseRole{}).
    Initialize()

if err != nil {
    panic(err)
}

// Set notification sender để gửi email/tin nhắn
ak.AuthService.SetNotificationSender(NewEmailNotificationSender())
```

### 4. Đăng ký Routes

Routes đã được thêm vào `setupRoutes.go`:

```go
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
```

## API Endpoints

### 1. Request Password Reset

**POST** `/api/auth/request-password-reset`

Yêu cầu reset password. Hệ thống sẽ tạo reset token và gửi qua email/tin nhắn (nếu có NotificationSender).

**Request Body:**
```json
{
  "email": "user@example.com"
}
```

**Response:**
```json
{
  "message": "Nếu email tồn tại, bạn sẽ nhận được hướng dẫn reset mật khẩu"
}
```

**Lưu ý:** Luôn trả về success message để tránh email enumeration attack (không leak thông tin email có tồn tại hay không).

### 2. Reset Password

**POST** `/api/auth/reset-password`

Đặt lại mật khẩu bằng reset token nhận được từ email/tin nhắn.

**Request Body:**
```json
{
  "token": "reset_token_from_email",
  "new_password": "newSecurePassword123"
}
```

**Response:**
```json
{
  "message": "Đặt lại mật khẩu thành công"
}
```

**Lỗi có thể xảy ra:**
- `401`: Reset token không hợp lệ hoặc đã hết hạn
- `401`: Reset token đã được sử dụng
- `400`: Password không đúng format (theo PasswordConfig)

## Bảo mật

### 1. Token Expiration
- Reset token hết hạn sau **1 giờ** (có thể config)
- Token chỉ được sử dụng **1 lần** (sau khi reset thành công sẽ bị đánh dấu `used = true`)

### 2. Token Invalidation
- Sau khi reset password thành công:
  - Token hiện tại được đánh dấu `used = true`
  - Tất cả reset tokens khác của user bị invalidate
  - Tất cả refresh tokens bị xóa (force logout tất cả sessions)

### 3. Email Enumeration Protection
- Luôn trả về success message ngay cả khi email không tồn tại
- Không leak thông tin về trạng thái tài khoản (active/inactive)

### 4. Token Storage
- Token được hash (SHA-256) trước khi lưu vào database
- Plain token chỉ tồn tại trong memory và được gửi qua email/tin nhắn

## Ví dụ sử dụng

### Request Password Reset

```bash
curl -X POST http://localhost:3000/api/auth/request-password-reset \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com"
  }'
```

### Reset Password

```bash
curl -X POST http://localhost:3000/api/auth/reset-password \
  -H "Content-Type: application/json" \
  -d '{
    "token": "reset_token_from_email",
    "new_password": "newSecurePassword123"
  }'
```

## Database Schema

Bảng `password_reset_tokens`:

```sql
CREATE TABLE password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    token VARCHAR(255) NOT NULL UNIQUE,  -- Token hash (SHA-256)
    user_id VARCHAR(12) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used BOOLEAN DEFAULT FALSE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);
```

## Cleanup Job (Optional)

Có thể tạo cleanup job để xóa các tokens đã hết hạn:

```go
// Xóa tất cả tokens đã hết hạn
passwordResetTokenRepo.DeleteExpired()
```

Chạy định kỳ (ví dụ: mỗi ngày) để giữ database sạch sẽ.

## So sánh với ChangePassword

| Tính năng | ChangePassword | ResetPassword |
|-----------|----------------|----------------|
| Yêu cầu password cũ | ✅ Có | ❌ Không |
| Yêu cầu đăng nhập | ✅ Có | ❌ Không |
| Xác thực | Password cũ | Reset token từ email/tin nhắn |
| Use case | User đổi password khi đã đăng nhập | User quên password |

## Troubleshooting

### Token không được gửi

- Kiểm tra xem đã set `NotificationSender` chưa:
  ```go
  ak.AuthService.SetNotificationSender(NewEmailNotificationSender())
  ```
- Kiểm tra logs để xem có lỗi khi gửi email/tin nhắn không
- Token vẫn được tạo và lưu vào database ngay cả khi không có NotificationSender

### Token không hợp lệ

- Kiểm tra token đã hết hạn chưa (mặc định: 1 giờ)
- Kiểm tra token đã được sử dụng chưa (`used = true`)
- Đảm bảo token được copy đầy đủ từ email/tin nhắn (không bị cắt)

### Email không nhận được

- Kiểm tra spam folder
- Kiểm tra email service configuration (SMTP, API keys, v.v.)
- Xem logs trong `notification_sender_example.go` để debug

