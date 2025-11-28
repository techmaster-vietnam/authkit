package models

import (
	"time"
)

// RefreshToken model để lưu refresh tokens
// Refresh token được lưu trong database để có thể revoke và kiểm tra tính hợp lệ
type RefreshToken struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Token     string    `gorm:"type:varchar(255);uniqueIndex;not null" json:"-"` // Token hash, không trả về JSON
	UserID    string    `gorm:"type:varchar(12);not null;index" json:"user_id"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
}

// TableName specifies the table name
func (RefreshToken) TableName() string {
	return "refresh_tokens"
}

// IsExpired kiểm tra xem token đã hết hạn chưa
func (rt *RefreshToken) IsExpired() bool {
	return time.Now().After(rt.ExpiresAt)
}
