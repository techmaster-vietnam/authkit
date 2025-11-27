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
// Sử dụng PostgreSQL UPSERT (ON CONFLICT DO NOTHING) để tối ưu hiệu suất
func (r *BaseRoleRepository[T]) AddRoleToUser(userID string, roleID uint) error {
	var user models.BaseUser
	var role T

	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}

	// Sử dụng PostgreSQL UPSERT: nếu (user_id, role_id) đã tồn tại thì không làm gì
	// PRIMARY KEY constraint trên (user_id, role_id) sẽ tự động xử lý conflict
	return r.db.Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT (user_id, role_id) DO NOTHING",
		userID, roleID,
	).Error
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

	// Kiểm tra user có role đó hay không trước khi xóa
	var count int64
	err := r.db.Table("user_roles").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
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
// Query trực tiếp roles từ database thông qua JOIN với user_roles
// để đảm bảo GORM tự động scan vào đúng type T
func (r *BaseRoleRepository[T]) ListRolesOfUser(userID string) ([]T, error) {
	// Kiểm tra user có tồn tại không
	var user models.BaseUser
	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	// Query trực tiếp roles từ bảng roles thông qua JOIN với user_roles
	// Sử dụng type []T để GORM tự động scan vào đúng type
	var roles []T
	err := r.db.Table("roles").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}

	// Đảm bảo luôn trả về empty slice thay vì nil
	if roles == nil {
		return []T{}, nil
	}

	return roles, nil
}

// ListUsersHasRole lấy danh sách users có role cụ thể
// Trả về []interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) ListUsersHasRole(roleName string) ([]interface{}, error) {
	var users []models.BaseUser
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("roles.name = ?", roleName).
		Preload("Roles").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	// Convert []models.BaseUser sang []interface{}
	result := make([]interface{}, len(users))
	for i := range users {
		result[i] = users[i]
	}
	return result, nil
}

// ListUsersHasRoleId lấy danh sách users có role theo ID
// Trả về []interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) ListUsersHasRoleId(roleID uint) ([]interface{}, error) {
	var users []models.BaseUser
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Where("user_roles.role_id = ?", roleID).
		Preload("Roles").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	// Convert []models.BaseUser sang []interface{}
	result := make([]interface{}, len(users))
	for i := range users {
		result[i] = users[i]
	}
	return result, nil
}

// ListUsersHasRoleName lấy danh sách users có role theo tên
// Trả về []interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) ListUsersHasRoleName(roleName string) ([]interface{}, error) {
	var users []models.BaseUser
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("roles.name = ?", roleName).
		Preload("Roles").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	// Convert []models.BaseUser sang []interface{}
	result := make([]interface{}, len(users))
	for i := range users {
		result[i] = users[i]
	}
	return result, nil
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

// DB trả về interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) DB() interface{} {
	return r.db
}
