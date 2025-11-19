package service

import (
	"errors"

	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// BaseRoleService là generic role service
// TRole phải implement RoleInterface
type BaseRoleService[TRole core.RoleInterface] struct {
	roleRepo *repository.BaseRoleRepository[TRole]
}

// NewBaseRoleService tạo mới BaseRoleService với generic type
func NewBaseRoleService[TRole core.RoleInterface](
	roleRepo *repository.BaseRoleRepository[TRole],
) *BaseRoleService[TRole] {
	return &BaseRoleService[TRole]{
		roleRepo: roleRepo,
	}
}

// BaseAddRoleRequest represents add role request
type BaseAddRoleRequest struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// AddRole creates a new role
func (s *BaseRoleService[TRole]) AddRole(req BaseAddRoleRequest) (TRole, error) {
	var zero TRole

	if req.ID == 0 {
		return zero, goerrorkit.NewValidationError("ID role là bắt buộc", map[string]interface{}{
			"field": "id",
		})
	}
	if req.Name == "" {
		return zero, goerrorkit.NewValidationError("Tên role là bắt buộc", map[string]interface{}{
			"field": "name",
		})
	}

	// Check if role ID already exists
	_, err := s.roleRepo.GetByID(req.ID)
	if err == nil {
		return zero, goerrorkit.NewBusinessError(409, "Role với ID này đã tồn tại").WithData(map[string]interface{}{
			"id": req.ID,
		})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role")
	}

	// Check if role name already exists
	_, err = s.roleRepo.GetByName(req.Name)
	if err == nil {
		return zero, goerrorkit.NewBusinessError(409, "Role với tên này đã tồn tại").WithData(map[string]interface{}{
			"name": req.Name,
		})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role")
	}

	// Create role - cần type assertion hoặc helper function
	// Với embedded structs, ta cần tạo TRole từ zero value và set fields
	// Tạm thời return error để lập trình viên override
	return zero, goerrorkit.NewSystemError(errors.New("AddRole method cần được override trong custom service để tạo role với ID và Name"))
}

// RemoveRole removes a role
func (s *BaseRoleService[TRole]) RemoveRole(roleID uint) error {
	if err := s.roleRepo.Delete(roleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy role").WithData(map[string]interface{}{
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa role")
	}
	return nil
}

// ListRoles lists all roles
func (s *BaseRoleService[TRole]) ListRoles() ([]TRole, error) {
	roles, err := s.roleRepo.List()
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách role")
	}
	return roles, err
}

// AddRoleToUser adds a role to a user
func (s *BaseRoleService[TRole]) AddRoleToUser(userID string, roleID uint) error {
	if err := s.roleRepo.AddRoleToUser(userID, roleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy user hoặc role").WithData(map[string]interface{}{
				"user_id": userID,
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi thêm role cho user")
	}
	return nil
}

// RemoveRoleFromUser removes a role from a user
func (s *BaseRoleService[TRole]) RemoveRoleFromUser(userID string, roleID uint) error {
	if err := s.roleRepo.RemoveRoleFromUser(userID, roleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy user hoặc role").WithData(map[string]interface{}{
				"user_id": userID,
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa role khỏi user")
	}
	return nil
}

// CheckUserHasRole checks if a user has a specific role
func (s *BaseRoleService[TRole]) CheckUserHasRole(userID string, roleName string) (bool, error) {
	hasRole, err := s.roleRepo.CheckUserHasRole(userID, roleName)
	if err != nil {
		return false, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role")
	}
	return hasRole, nil
}

// ListRolesOfUser lists all roles of a user
func (s *BaseRoleService[TRole]) ListRolesOfUser(userID string) ([]TRole, error) {
	roles, err := s.roleRepo.ListRolesOfUser(userID)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách role của user")
	}
	return roles, err
}

// ListUsersHasRole lists all users with a specific role
// Trả về []models.BaseUser vì không biết custom User type
func (s *BaseRoleService[TRole]) ListUsersHasRole(roleName string) ([]interface{}, error) {
	users, err := s.roleRepo.ListUsersHasRole(roleName)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách user có role")
	}
	
	// Convert []models.BaseUser sang []interface{}
	result := make([]interface{}, len(users))
	for i := range users {
		result[i] = users[i]
	}
	return result, nil
}

