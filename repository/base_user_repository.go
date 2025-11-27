package repository

import (
	"github.com/techmaster-vietnam/authkit/core"
	"gorm.io/gorm"
)

// BaseUserRepository là generic repository cho User models
// T phải implement UserInterface
type BaseUserRepository[T core.UserInterface] struct {
	db *gorm.DB
}

// NewBaseUserRepository tạo mới BaseUserRepository với generic type
func NewBaseUserRepository[T core.UserInterface](db *gorm.DB) *BaseUserRepository[T] {
	return &BaseUserRepository[T]{db: db}
}

// Create tạo mới user
func (r *BaseUserRepository[T]) Create(user T) error {
	return r.db.Create(&user).Error
}

// GetByID lấy user theo ID
func (r *BaseUserRepository[T]) GetByID(id string) (T, error) {
	var user T
	err := r.db.Preload("Roles").Where("id = ?", id).First(&user).Error
	return user, err
}

// GetByEmail lấy user theo email
func (r *BaseUserRepository[T]) GetByEmail(email string) (T, error) {
	var user T
	err := r.db.Preload("Roles").Where("email = ?", email).First(&user).Error
	return user, err
}

// Update cập nhật user
func (r *BaseUserRepository[T]) Update(user T) error {
	return r.db.Save(&user).Error
}

// Delete soft delete user
func (r *BaseUserRepository[T]) Delete(id string) error {
	var user T
	return r.db.Where("id = ?", id).Delete(&user).Error
}

// List lấy danh sách users với pagination
func (r *BaseUserRepository[T]) List(offset, limit int) ([]T, int64, error) {
	var users []T
	var total int64
	var zero T

	if err := r.db.Model(&zero).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Preload("Roles").Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

// DB trả về *gorm.DB nhưng dùng interface{} để match với UserRepositoryInterface
func (r *BaseUserRepository[T]) DB() interface{} {
	return r.db
}

