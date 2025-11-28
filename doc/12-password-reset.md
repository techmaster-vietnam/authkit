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

Bạn cần implement interface `core.NotificationSender` để gửi email/tin nhắn. AuthKit cung cấp 4 implementation mẫu trong `examples/notification_sender.go`:

#### 2.1. EmailNotificationSender

Gửi reset token qua email (hiện tại log ra console để demo):

```go
type EmailNotificationSender struct {
    // Có thể inject email service (SMTP, SendGrid, AWS SES, v.v.)
}

func (e *EmailNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
    // TODO: Implement logic gửi email thực tế
    // Ví dụ với SMTP, SendGrid, AWS SES, v.v.
    
    // Tạm thời log ra console để demo
    log.Printf("[EmailNotificationSender] Gửi reset token đến %s: %s (hết hạn sau %s)", email, token, expiresIn)
    fmt.Printf("\n=== EMAIL RESET PASSWORD ===\n")
    fmt.Printf("To: %s\n", email)
    fmt.Printf("Subject: Yêu cầu đặt lại mật khẩu\n")
    fmt.Printf("Reset token: %s\n", token)
    fmt.Printf("URL: http://localhost:3000/reset-password?token=%s\n", token)
    fmt.Printf("===========================\n\n")
    
    return nil
}
```

#### 2.2. SMSNotificationSender

Gửi reset token qua SMS (hiện tại log ra console để demo):

```go
type SMSNotificationSender struct {
    // Có thể inject SMS service (Twilio, AWS SNS, v.v.)
}

func (s *SMSNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
    // TODO: Implement logic gửi SMS thực tế
    // Ví dụ với Twilio, AWS SNS, v.v.
    
    // Tạm thời log ra console để demo
    log.Printf("[SMSNotificationSender] Gửi reset token đến %s: %s (hết hạn sau %s)", email, token, expiresIn)
    fmt.Printf("\n=== SMS RESET PASSWORD ===\n")
    fmt.Printf("To: %s\n", email)
    fmt.Printf("Message: Mã đặt lại mật khẩu của bạn là: %s. Mã này sẽ hết hạn sau %s.\n", token, expiresIn)
    fmt.Printf("===========================\n\n")
    
    return nil
}
```

#### 2.3. CombinedNotificationSender

Gửi reset token qua cả email và SMS:

```go
type CombinedNotificationSender struct {
    emailSender *EmailNotificationSender
    smsSender   *SMSNotificationSender
}

func (c *CombinedNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
    // Gửi email
    if err := c.emailSender.SendPasswordResetToken(email, token, expiresIn); err != nil {
        return goerrorkit.WrapWithMessage(err, "lỗi khi gửi email")
    }
    
    // Gửi SMS
    if err := c.smsSender.SendPasswordResetToken(email, token, expiresIn); err != nil {
        return goerrorkit.WrapWithMessage(err, "lỗi khi gửi SMS")
    }
    
    return nil
}
```

#### 2.4. TestNotificationSender (Cho Development/Testing)

Lưu reset token vào file JSON để Python script hoặc các công cụ test có thể đọc:

```go
type TestNotificationSender struct {
    filePath string // Đường dẫn file JSON để lưu token
}

func NewTestNotificationSender(filePath string) *TestNotificationSender {
    if filePath == "" {
        filePath = "testscript/reset_tokens.json"
    }
    return &TestNotificationSender{filePath: filePath}
}

func (t *TestNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
    // Đọc file hiện tại nếu có
    var tokens map[string]interface{}
    if data, err := os.ReadFile(t.filePath); err == nil {
        json.Unmarshal(data, &tokens)
    } else {
        tokens = make(map[string]interface{})
    }
    
    // Thêm token mới
    tokens[email] = map[string]interface{}{
        "token":      token,
        "expires_in": expiresIn,
        "created_at": time.Now().Format(time.RFC3339),
    }
    
    // Ghi vào file
    data, _ := json.MarshalIndent(tokens, "", "  ")
    os.WriteFile(t.filePath, data, 0644)
    
    fmt.Printf("\n=== TEST RESET TOKEN SAVED ===\n")
    fmt.Printf("Email: %s\n", email)
    fmt.Printf("Token: %s\n", token)
    fmt.Printf("Saved to: %s\n", t.filePath)
    fmt.Printf("==============================\n\n")
    
    return nil
}
```

**Đảm bảo implement interface:**

```go
var _ core.NotificationSender = (*EmailNotificationSender)(nil)
var _ core.NotificationSender = (*SMSNotificationSender)(nil)
var _ core.NotificationSender = (*CombinedNotificationSender)(nil)
var _ core.NotificationSender = (*TestNotificationSender)(nil)
```

#### 2.5. Implement cho Production

Để implement cho production, bạn cần tích hợp với email/SMS service thực tế. Dưới đây là ví dụ với SMTP:

```go
import (
    "net/smtp"
    "fmt"
)

type ProductionEmailSender struct {
    smtpHost     string
    smtpPort     string
    smtpUser     string
    smtpPassword string
    fromEmail    string
}

func NewProductionEmailSender(smtpHost, smtpPort, smtpUser, smtpPassword, fromEmail string) *ProductionEmailSender {
    return &ProductionEmailSender{
        smtpHost:     smtpHost,
        smtpPort:     smtpPort,
        smtpUser:     smtpUser,
        smtpPassword: smtpPassword,
        fromEmail:    fromEmail,
    }
}

func (p *ProductionEmailSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
    subject := "Yêu cầu đặt lại mật khẩu"
    resetURL := fmt.Sprintf("https://your-app.com/reset-password?token=%s", token)
    
    body := fmt.Sprintf(`
Xin chào,

Bạn đã yêu cầu đặt lại mật khẩu cho tài khoản của mình.

Reset token: %s
Token này sẽ hết hạn sau: %s

Vui lòng click vào link sau để đặt lại mật khẩu:
%s

Nếu bạn không yêu cầu đặt lại mật khẩu, vui lòng bỏ qua email này.

Trân trọng,
Đội ngũ hỗ trợ
`, token, expiresIn, resetURL)
    
    // Setup SMTP authentication
    auth := smtp.PlainAuth("", p.smtpUser, p.smtpPassword, p.smtpHost)
    
    // Email headers
    msg := []byte(fmt.Sprintf("To: %s\r\n", email) +
        fmt.Sprintf("Subject: %s\r\n", subject) +
        "\r\n" +
        body + "\r\n")
    
    // Send email
    addr := fmt.Sprintf("%s:%s", p.smtpHost, p.smtpPort)
    err := smtp.SendMail(addr, auth, p.fromEmail, []string{email}, msg)
    if err != nil {
        return goerrorkit.WrapWithMessage(err, "Lỗi khi gửi email")
    }
    
    return nil
}

var _ core.NotificationSender = (*ProductionEmailSender)(nil)
```

**Tương tự cho SMS service (ví dụ với Twilio):**

```go
import (
    "github.com/twilio/twilio-go"
    "github.com/twilio/twilio-go/rest/api/v2010"
)

type TwilioSMSSender struct {
    client      *twilio.RestClient
    fromNumber  string
}

func NewTwilioSMSSender(accountSID, authToken, fromNumber string) *TwilioSMSSender {
    client := twilio.NewRestClientWithParams(twilio.ClientParams{
        Username: accountSID,
        Password: authToken,
    })
    
    return &TwilioSMSSender{
        client:     client,
        fromNumber: fromNumber,
    }
}

func (t *TwilioSMSSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
    // Lấy số điện thoại từ email hoặc database
    phoneNumber := getPhoneNumberFromEmail(email)
    
    message := fmt.Sprintf("Mã đặt lại mật khẩu của bạn là: %s. Mã này sẽ hết hạn sau %s.", token, expiresIn)
    
    params := &api.CreateMessageParams{}
    params.SetTo(phoneNumber)
    params.SetFrom(t.fromNumber)
    params.SetBody(message)
    
    _, err := t.client.Api.CreateMessage(params)
    if err != nil {
        return goerrorkit.WrapWithMessage(err, "Lỗi khi gửi SMS")
    }
    
    return nil
}

var _ core.NotificationSender = (*TwilioSMSSender)(nil)
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

// Cho môi trường test/development: sử dụng TestNotificationSender
// Token sẽ được lưu vào file JSON để Python script có thể đọc
ak.AuthService.SetNotificationSender(NewTestNotificationSender("testscript/reset_tokens.json"))

// Cho production: sử dụng EmailNotificationSender hoặc SMSNotificationSender
// ak.AuthService.SetNotificationSender(NewEmailNotificationSender())
// ak.AuthService.SetNotificationSender(NewSMSNotificationSender())
// ak.AuthService.SetNotificationSender(NewCombinedNotificationSender())
```

**Lưu ý:** 
- Token vẫn được tạo và lưu vào database ngay cả khi không có NotificationSender
- Nếu NotificationSender trả về error, request sẽ fail (token đã được lưu, user có thể request lại)

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

## Debug trong Development

### Sử dụng TestNotificationSender

Trong môi trường development, bạn có thể sử dụng `TestNotificationSender` để lưu reset token vào file JSON thay vì gửi email/SMS thực tế:

```go
// Trong main.go
ak.AuthService.SetNotificationSender(NewTestNotificationSender("testscript/reset_tokens.json"))
```

**Cách hoạt động:**
1. Khi user request password reset, token sẽ được lưu vào file `testscript/reset_tokens.json`
2. File JSON có format:
```json
{
  "user@example.com": {
    "token": "bgWEOTjfeOrdDfhFLWyN7bkx1hfMbvvBj7soUnUwH_I=",
    "expires_in": "1 giờ",
    "created_at": "2025-11-28T23:26:21+07:00"
  }
}
```
3. Python script (`examples/testscript/reset_pass.py`) có thể đọc token từ file này để test tự động
4. Console sẽ hiển thị thông tin token để bạn có thể copy và test thủ công

### Console Logging

Các NotificationSender mẫu đều log ra console để dễ debug:

- **EmailNotificationSender**: Hiển thị email format với token và URL reset
- **SMSNotificationSender**: Hiển thị SMS message format
- **TestNotificationSender**: Hiển thị thông tin token đã được lưu vào file

### Test với Python Script

Script `examples/testscript/reset_pass.py` tự động test flow password reset:

```bash
cd examples/testscript
python3 reset_pass.py
```

Script sẽ:
1. Gửi request password reset
2. Đọc token từ `reset_tokens.json`
3. Reset password với token
4. Login với password mới
5. Change password
6. Login lại với password sau khi change

## Troubleshooting

### Token không được gửi

- Kiểm tra xem đã set `NotificationSender` chưa:
  ```go
  ak.AuthService.SetNotificationSender(NewEmailNotificationSender())
  ```
- Kiểm tra logs để xem có lỗi khi gửi email/tin nhắn không
- Token vẫn được tạo và lưu vào database ngay cả khi không có NotificationSender
- Nếu dùng `TestNotificationSender`, kiểm tra file `testscript/reset_tokens.json` có được tạo không

### Token không hợp lệ

- Kiểm tra token đã hết hạn chưa (mặc định: 1 giờ)
- Kiểm tra token đã được sử dụng chưa (`used = true`)
- Đảm bảo token được copy đầy đủ từ email/tin nhắn (không bị cắt)
- Nếu dùng `TestNotificationSender`, đảm bảo đọc đúng email key trong JSON file

### Email không nhận được

- Kiểm tra spam folder
- Kiểm tra email service configuration (SMTP, API keys, v.v.)
- Xem logs trong console (nếu dùng EmailNotificationSender mẫu)
- Trong development, sử dụng `TestNotificationSender` để tránh phụ thuộc vào email service

### File JSON không được tạo (TestNotificationSender)

- Kiểm tra quyền ghi file trong thư mục `testscript/`
- Kiểm tra đường dẫn file có đúng không
- Xem console logs để biết đường dẫn file đã được lưu

