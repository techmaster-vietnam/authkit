package repository

import (
	"time"

	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// PasswordResetTokenRepository handles password reset token database operations
type PasswordResetTokenRepository struct {
	db *gorm.DB
}

// NewPasswordResetTokenRepository creates a new password reset token repository
func NewPasswordResetTokenRepository(db *gorm.DB) *PasswordResetTokenRepository {
	return &PasswordResetTokenRepository{db: db}
}

// Create tạo mới password reset token trong database
// token: plain reset token (sẽ được hash trước khi lưu)
// userID: ID của user yêu cầu reset password
// expiresAt: thời gian hết hạn của token (thường là 1 giờ)
func (r *PasswordResetTokenRepository) Create(token string, userID string, expiresAt time.Time) (*models.PasswordResetToken, error) {
	tokenHash := HashToken(token)

	passwordResetToken := &models.PasswordResetToken{
		Token:     tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Used:      false,
	}

	if err := r.db.Create(passwordResetToken).Error; err != nil {
		return nil, err
	}

	return passwordResetToken, nil
}

// GetByTokenHash tìm password reset token theo token hash
func (r *PasswordResetTokenRepository) GetByTokenHash(tokenHash string) (*models.PasswordResetToken, error) {
	var token models.PasswordResetToken
	err := r.db.Where("token = ?", tokenHash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// GetByToken tìm password reset token theo plain token (hash trước khi tìm)
func (r *PasswordResetTokenRepository) GetByToken(token string) (*models.PasswordResetToken, error) {
	tokenHash := HashToken(token)
	return r.GetByTokenHash(tokenHash)
}

// MarkAsUsed đánh dấu token đã được sử dụng (sau khi reset password thành công)
func (r *PasswordResetTokenRepository) MarkAsUsed(token string) error {
	tokenHash := HashToken(token)
	return r.db.Model(&models.PasswordResetToken{}).
		Where("token = ?", tokenHash).
		Update("used", true).Error
}

// DeleteByToken xóa password reset token theo plain token
func (r *PasswordResetTokenRepository) DeleteByToken(token string) error {
	tokenHash := HashToken(token)
	return r.db.Where("token = ?", tokenHash).Delete(&models.PasswordResetToken{}).Error
}

// DeleteByUserID xóa tất cả password reset tokens của một user
func (r *PasswordResetTokenRepository) DeleteByUserID(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.PasswordResetToken{}).Error
}

// DeleteExpired xóa tất cả password reset tokens đã hết hạn (cleanup job)
func (r *PasswordResetTokenRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&models.PasswordResetToken{}).Error
}

// InvalidateUserTokens đánh dấu tất cả tokens của user là đã sử dụng (khi reset password thành công)
func (r *PasswordResetTokenRepository) InvalidateUserTokens(userID string) error {
	return r.db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used = ?", userID, false).
		Update("used", true).Error
}

