package repository

import (
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// UserRepository handles user database operations
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// GetByID gets a user by ID
func (r *UserRepository) GetByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Roles").Where("id = ?", id).First(&user).Error
	return &user, err
}

// GetByEmail gets a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Roles").Where("email = ?", email).First(&user).Error
	return &user, err
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete soft deletes a user
func (r *UserRepository) Delete(id string) error {
	return r.db.Delete(&models.User{}, id).Error
}

// List lists all users with pagination and filter
func (r *UserRepository) List(offset, limit int, filter interface{}) ([]models.User, int64, error) {
	var users []models.User
	var total int64

	// Build query
	query := r.db.Model(&models.User{})

	// Apply filter nếu có
	if filter != nil {
		if userFilter, ok := filter.(*UserFilter); ok {
			// Filter email
			if userFilter.Email != "" {
				query = query.Where("email LIKE ?", "%"+userFilter.Email+"%")
			}
			// Filter full_name
			if userFilter.FullName != "" {
				query = query.Where("full_name LIKE ?", "%"+userFilter.FullName+"%")
			}
			// Custom fields filter không áp dụng cho UserRepository (non-generic)
			// Vì UserRepository chỉ làm việc với models.User (không có custom fields)
		}
	}

	// Count total với filters
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get users với pagination
	err := query.Preload("Roles").Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}
