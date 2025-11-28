package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/goerrorkit"
)

// EmailNotificationSender là ví dụ implementation của NotificationSender interface
// Sử dụng để gửi email reset password
type EmailNotificationSender struct {
	// Có thể inject email service (SMTP, SendGrid, AWS SES, v.v.)
	// Ví dụ: smtpClient *smtp.Client
}

// NewEmailNotificationSender tạo mới EmailNotificationSender
func NewEmailNotificationSender() *EmailNotificationSender {
	return &EmailNotificationSender{}
}

// SendPasswordResetToken implement core.NotificationSender interface
// Gửi password reset token đến user qua email
func (e *EmailNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
	// TODO: Implement logic gửi email thực tế
	// Ví dụ với SMTP, SendGrid, AWS SES, v.v.

	// Ví dụ với SMTP:
	// subject := "Yêu cầu đặt lại mật khẩu"
	// body := fmt.Sprintf(`
	//     Xin chào,
	//
	//     Bạn đã yêu cầu đặt lại mật khẩu cho tài khoản của mình.
	//
	//     Reset token: %s
	//     Token này sẽ hết hạn sau: %s
	//
	//     Vui lòng sử dụng token này để đặt lại mật khẩu tại: https://your-app.com/reset-password
	//
	//     Nếu bạn không yêu cầu đặt lại mật khẩu, vui lòng bỏ qua email này.
	//
	//     Trân trọng,
	//     Đội ngũ hỗ trợ
	// `, token, expiresIn)
	//
	// return e.sendEmail(email, subject, body)

	// Tạm thời log ra console để demo
	log.Printf("[EmailNotificationSender] Gửi reset token đến %s: %s (hết hạn sau %s)", email, token, expiresIn)
	fmt.Printf("\n=== EMAIL RESET PASSWORD ===\n")
	fmt.Printf("To: %s\n", email)
	fmt.Printf("Subject: Yêu cầu đặt lại mật khẩu\n")
	fmt.Printf("Body:\n")
	fmt.Printf("Reset token: %s\n", token)
	fmt.Printf("Token này sẽ hết hạn sau: %s\n", expiresIn)
	fmt.Printf("URL: http://localhost:3000/reset-password?token=%s\n", token)
	fmt.Printf("===========================\n\n")

	return nil
}

// SMSNotificationSender là ví dụ implementation khác cho SMS
type SMSNotificationSender struct {
	// Có thể inject SMS service (Twilio, AWS SNS, v.v.)
}

// NewSMSNotificationSender tạo mới SMSNotificationSender
func NewSMSNotificationSender() *SMSNotificationSender {
	return &SMSNotificationSender{}
}

// SendPasswordResetToken implement core.NotificationSender interface
// Gửi password reset token đến user qua SMS
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

// CombinedNotificationSender gửi cả email và SMS
type CombinedNotificationSender struct {
	emailSender *EmailNotificationSender
	smsSender   *SMSNotificationSender
}

// NewCombinedNotificationSender tạo mới CombinedNotificationSender
func NewCombinedNotificationSender() *CombinedNotificationSender {
	return &CombinedNotificationSender{
		emailSender: NewEmailNotificationSender(),
		smsSender:   NewSMSNotificationSender(),
	}
}

// SendPasswordResetToken implement core.NotificationSender interface
// Gửi password reset token qua cả email và SMS
func (c *CombinedNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
	// Gửi email
	if err := c.emailSender.SendPasswordResetToken(email, token, expiresIn); err != nil {
		appErr := goerrorkit.WrapWithMessage(err, "lỗi khi gửi email")
		return appErr
	}

	// Gửi SMS
	if err := c.smsSender.SendPasswordResetToken(email, token, expiresIn); err != nil {
		appErr := goerrorkit.WrapWithMessage(err, "lỗi khi gửi SMS")
		return appErr
	}

	return nil
}

// TestNotificationSender lưu reset token vào file JSON để Python script dễ đọc
// Dùng cho môi trường test/development
type TestNotificationSender struct {
	filePath string // Đường dẫn file JSON để lưu token
}

// NewTestNotificationSender tạo mới TestNotificationSender
// filePath: Đường dẫn file JSON để lưu token (mặc định: "testscript/reset_tokens.json")
func NewTestNotificationSender(filePath string) *TestNotificationSender {
	if filePath == "" {
		filePath = "testscript/reset_tokens.json"
	}
	return &TestNotificationSender{
		filePath: filePath,
	}
}

// SendPasswordResetToken implement core.NotificationSender interface
// Lưu reset token vào file JSON để Python script có thể đọc
func (t *TestNotificationSender) SendPasswordResetToken(email string, token string, expiresIn string) error {
	// Tạo thư mục nếu chưa tồn tại
	dir := filepath.Dir(t.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		appErr := goerrorkit.WrapWithMessage(err, "lỗi khi tạo thư mục")
		return appErr
	}

	// Đọc file hiện tại nếu có
	var tokens map[string]interface{}
	if data, err := os.ReadFile(t.filePath); err == nil {
		if err := json.Unmarshal(data, &tokens); err != nil {
			tokens = make(map[string]interface{})
		}
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
	data, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		appErr := goerrorkit.WrapWithMessage(err, "lỗi khi encode JSON")
		return appErr
	}

	if err := os.WriteFile(t.filePath, data, 0644); err != nil {
		appErr := goerrorkit.WrapWithMessage(err, "lỗi khi ghi file")
		return appErr
	}

	fmt.Printf("\n=== TEST RESET TOKEN SAVED ===\n")
	fmt.Printf("Email: %s\n", email)
	fmt.Printf("Token: %s\n", token)
	fmt.Printf("Expires in: %s\n", expiresIn)
	fmt.Printf("Saved to: %s\n", t.filePath)
	fmt.Printf("==============================\n\n")

	return nil
}

// Đảm bảo các struct implement core.NotificationSender interface
var _ core.NotificationSender = (*EmailNotificationSender)(nil)
var _ core.NotificationSender = (*SMSNotificationSender)(nil)
var _ core.NotificationSender = (*CombinedNotificationSender)(nil)
var _ core.NotificationSender = (*TestNotificationSender)(nil)
