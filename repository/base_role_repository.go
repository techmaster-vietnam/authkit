package repository

import (
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// BaseRoleRepository là generic repository cho Role models
// T phải implement RoleInterface
type BaseRoleRepository[T core.RoleInterface] struct {
	db *gorm.DB
}

// NewBaseRoleRepository tạo mới BaseRoleRepository với generic type
func NewBaseRoleRepository[T core.RoleInterface](db *gorm.DB) *BaseRoleRepository[T] {
	return &BaseRoleRepository[T]{db: db}
}

// Create tạo mới role
func (r *BaseRoleRepository[T]) Create(role T) error {
	// Sử dụng Select để chỉ định rõ các fields cần insert, bao gồm cả ID
	// Điều này đảm bảo ID được insert vào database ngay cả khi không phải auto-increment
	return r.db.Select("id", "name", "is_system").Create(&role).Error
}

// GetByID lấy role theo ID
func (r *BaseRoleRepository[T]) GetByID(id uint) (T, error) {
	var role T
	err := r.db.Where("id = ?", id).First(&role).Error
	return role, err
}

// GetByName lấy role theo name
func (r *BaseRoleRepository[T]) GetByName(name string) (T, error) {
	var role T
	err := r.db.Where("name = ?", name).First(&role).Error
	return role, err
}

// GetByIDs lấy roles theo IDs (batch query)
func (r *BaseRoleRepository[T]) GetByIDs(ids []uint) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}
	var roles []T
	err := r.db.Where("id IN ?", ids).Find(&roles).Error
	return roles, err
}

// Update cập nhật role
func (r *BaseRoleRepository[T]) Update(role T) error {
	return r.db.Save(&role).Error
}

// Delete hard delete role (chỉ nếu không phải system role)
// Sử dụng stored procedure để đảm bảo tính nhất quán dữ liệu:
// 1. Xóa khỏi bảng user_roles
// 2. Xóa role_id khỏi mảng rules.roles
// 3. Xóa khỏi bảng roles
func (r *BaseRoleRepository[T]) Delete(id uint) error {
	var role T
	if err := r.db.First(&role, id).Error; err != nil {
		return err
	}
	if role.IsSystem() {
		return gorm.ErrRecordNotFound // Cannot delete system roles
	}

	// Gọi stored procedure để xóa role và dọn dẹp dữ liệu liên quan
	return r.db.Exec("SELECT delete_role(?)", id).Error
}

// List lấy danh sách tất cả roles
func (r *BaseRoleRepository[T]) List() ([]T, error) {
	var roles []T
	err := r.db.Find(&roles).Error
	return roles, err
}

// AddRoleToUser thêm role cho user
// Note: Method này vẫn sử dụng models.BaseUser vì cần làm việc với many2many relationship
func (r *BaseRoleRepository[T]) AddRoleToUser(userID string, roleID uint) error {
	var user models.BaseUser
	var role T

	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}

	return r.db.Model(&user).Association("Roles").Append(&role)
}

// RemoveRoleFromUser xóa role khỏi user
func (r *BaseRoleRepository[T]) RemoveRoleFromUser(userID string, roleID uint) error {
	var user models.BaseUser
	var role T

	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}

	return r.db.Model(&user).Association("Roles").Delete(&role)
}

// CheckUserHasRole kiểm tra user có role cụ thể không
func (r *BaseRoleRepository[T]) CheckUserHasRole(userID string, roleName string) (bool, error) {
	var count int64
	err := r.db.Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, roleName).
		Count(&count).Error
	return count > 0, err
}

// ListRolesOfUser lấy danh sách roles của user
func (r *BaseRoleRepository[T]) ListRolesOfUser(userID string) ([]T, error) {
	var user models.BaseUser
	if err := r.db.Where("id = ?", userID).Preload("Roles").First(&user).Error; err != nil {
		return nil, err
	}

	// Convert []models.BaseRole sang []T
	roles := make([]T, len(user.Roles))
	for i := range user.Roles {
		// Type assertion - cần cast từ BaseRole sang T
		// Điều này hoạt động vì T sẽ là BaseRole hoặc custom type embed BaseRole
		rolePtr := any(&user.Roles[i])
		if tRole, ok := rolePtr.(T); ok {
			roles[i] = tRole
		}
	}
	return roles, nil
}

// ListUsersHasRole lấy danh sách users có role cụ thể
// Trả về []models.BaseUser vì không biết custom User type
func (r *BaseRoleRepository[T]) ListUsersHasRole(roleName string) ([]models.BaseUser, error) {
	var users []models.BaseUser
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("roles.name = ?", roleName).
		Preload("Roles").
		Find(&users).Error
	return users, err
}

// GetIDsByNames lấy role IDs theo role names (batch query)
func (r *BaseRoleRepository[T]) GetIDsByNames(names []string) (map[string]uint, error) {
	if len(names) == 0 {
		return make(map[string]uint), nil
	}
	var roles []T
	err := r.db.Where("name IN ?", names).Find(&roles).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]uint, len(roles))
	for _, role := range roles {
		result[role.GetName()] = role.GetID()
	}
	return result, nil
}

// DB trả về *gorm.DB để cho phép extend với custom methods
func (r *BaseRoleRepository[T]) DB() *gorm.DB {
	return r.db
}
