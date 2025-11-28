package models

import (
	"time"
)

// PasswordResetToken model để lưu password reset tokens
// Token được tạo khi user yêu cầu reset password và được gửi qua email/tin nhắn
type PasswordResetToken struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Token     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"` // Token hash, không trả về JSON
	UserID    string    `gorm:"type:varchar(12);not null;index" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	Used      bool      `gorm:"default:false;index" json:"used"` // Đánh dấu token đã được sử dụng
}

// TableName specifies the table name
func (PasswordResetToken) TableName() string {
	return "password_reset_tokens"
}

// IsExpired kiểm tra xem token đã hết hạn chưa
func (prt *PasswordResetToken) IsExpired() bool {
	return time.Now().After(prt.ExpiresAt)
}

// IsValid kiểm tra token có hợp lệ không (chưa hết hạn và chưa được sử dụng)
func (prt *PasswordResetToken) IsValid() bool {
	return !prt.IsExpired() && !prt.Used
}
