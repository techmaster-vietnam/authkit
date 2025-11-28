package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// RefreshTokenRepository handles refresh token database operations
type RefreshTokenRepository struct {
	db *gorm.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// HashToken tạo hash của token để lưu trong database
// Không lưu plain token để bảo mật
func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// Create tạo mới refresh token trong database
// token: plain refresh token (sẽ được hash trước khi lưu)
// userID: ID của user sở hữu token
// expiresAt: thời gian hết hạn của token
func (r *RefreshTokenRepository) Create(token string, userID string, expiresAt time.Time) (*models.RefreshToken, error) {
	tokenHash := HashToken(token)
	
	refreshToken := &models.RefreshToken{
		Token:     tokenHash,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
	
	if err := r.db.Create(refreshToken).Error; err != nil {
		return nil, err
	}
	
	return refreshToken, nil
}

// GetByTokenHash tìm refresh token theo token hash
func (r *RefreshTokenRepository) GetByTokenHash(tokenHash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.Where("token = ?", tokenHash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

// GetByToken tìm refresh token theo plain token (hash trước khi tìm)
func (r *RefreshTokenRepository) GetByToken(token string) (*models.RefreshToken, error) {
	tokenHash := HashToken(token)
	return r.GetByTokenHash(tokenHash)
}

// DeleteByToken xóa refresh token theo plain token
func (r *RefreshTokenRepository) DeleteByToken(token string) error {
	tokenHash := HashToken(token)
	return r.db.Where("token = ?", tokenHash).Delete(&models.RefreshToken{}).Error
}

// DeleteByUserID xóa tất cả refresh tokens của một user (khi logout hoặc đổi mật khẩu)
func (r *RefreshTokenRepository) DeleteByUserID(userID string) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error
}

// DeleteExpired xóa tất cả refresh tokens đã hết hạn (cleanup job)
func (r *RefreshTokenRepository) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error
}

// RevokeByTokenHash revoke một refresh token (soft delete)
func (r *RefreshTokenRepository) RevokeByTokenHash(tokenHash string) error {
	return r.db.Where("token = ?", tokenHash).Delete(&models.RefreshToken{}).Error
}

