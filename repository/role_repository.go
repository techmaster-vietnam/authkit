package repository

import (
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// RoleRepository handles role database operations
type RoleRepository struct {
	db *gorm.DB
}

// NewRoleRepository creates a new role repository
func NewRoleRepository(db *gorm.DB) *RoleRepository {
	return &RoleRepository{db: db}
}

// Create creates a new role
func (r *RoleRepository) Create(role *models.Role) error {
	return r.db.Create(role).Error
}

// GetByID gets a role by ID
func (r *RoleRepository) GetByID(id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.Where("id = ?", id).First(&role).Error
	return &role, err
}

// GetByName gets a role by name
func (r *RoleRepository) GetByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Where("name = ?", name).First(&role).Error
	return &role, err
}

// GetByIDs gets roles by IDs (used to get role names from role IDs in JWT token)
func (r *RoleRepository) GetByIDs(ids []uint) ([]models.Role, error) {
	if len(ids) == 0 {
		return []models.Role{}, nil
	}
	var roles []models.Role
	err := r.db.Where("id IN ?", ids).Find(&roles).Error
	return roles, err
}

// Update updates a role
func (r *RoleRepository) Update(role *models.Role) error {
	return r.db.Save(role).Error
}

// Delete hard deletes a role (only if not system role)
func (r *RoleRepository) Delete(id uint) error {
	var role models.Role
	if err := r.db.First(&role, id).Error; err != nil {
		return err
	}
	if role.IsSystem() {
		return gorm.ErrRecordNotFound // Cannot delete system roles
	}
	return r.db.Unscoped().Delete(&role).Error
}

// List lists all roles
func (r *RoleRepository) List() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Find(&roles).Error
	return roles, err
}

// AddRoleToUser adds a role to a user
func (r *RoleRepository) AddRoleToUser(userID string, roleID uint) error {
	var user models.User
	var role models.Role

	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}

	return r.db.Model(&user).Association("Roles").Append(&role)
}

// RemoveRoleFromUser removes a role from a user
func (r *RoleRepository) RemoveRoleFromUser(userID string, roleID uint) error {
	var user models.User
	var role models.Role

	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}

	return r.db.Model(&user).Association("Roles").Delete(&role)
}

// CheckUserHasRole checks if a user has a specific role
func (r *RoleRepository) CheckUserHasRole(userID string, roleName string) (bool, error) {
	var count int64
	err := r.db.Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, roleName).
		Count(&count).Error
	return count > 0, err
}

// ListRolesOfUser lists all roles of a user
func (r *RoleRepository) ListRolesOfUser(userID string) ([]models.Role, error) {
	var user models.User
	if err := r.db.Where("id = ?", userID).Preload("Roles").First(&user).Error; err != nil {
		return nil, err
	}
	return user.Roles, nil
}

// ListUsersHasRole lists all users with a specific role
func (r *RoleRepository) ListUsersHasRole(roleName string) ([]models.User, error) {
	var users []models.User
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("roles.name = ?", roleName).
		Preload("Roles").
		Find(&users).Error
	return users, err
}
