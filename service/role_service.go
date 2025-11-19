package service

import (
	"errors"

	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// RoleService handles role business logic
type RoleService struct {
	roleRepo *repository.RoleRepository
}

// NewRoleService creates a new role service
func NewRoleService(roleRepo *repository.RoleRepository) *RoleService {
	return &RoleService{roleRepo: roleRepo}
}

// AddRoleRequest represents add role request
type AddRoleRequest struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

// AddRole creates a new role
func (s *RoleService) AddRole(req AddRoleRequest) (*models.Role, error) {
	if req.ID == 0 {
		return nil, goerrorkit.NewValidationError("ID role là bắt buộc", map[string]interface{}{
			"field": "id",
		})
	}
	if req.Name == "" {
		return nil, goerrorkit.NewValidationError("Tên role là bắt buộc", map[string]interface{}{
			"field": "name",
		})
	}

	// Check if role ID already exists
	_, err := s.roleRepo.GetByID(req.ID)
	if err == nil {
		return nil, goerrorkit.NewBusinessError(409, "Role với ID này đã tồn tại").WithData(map[string]interface{}{
			"id": req.ID,
		})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role")
	}

	// Check if role name already exists
	_, err = s.roleRepo.GetByName(req.Name)
	if err == nil {
		return nil, goerrorkit.NewBusinessError(409, "Role với tên này đã tồn tại").WithData(map[string]interface{}{
			"name": req.Name,
		})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role")
	}

	role := &models.Role{
		ID:     req.ID,
		Name:   req.Name,
		System: false,
	}

	if err := s.roleRepo.Create(role); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo role")
	}

	return role, nil
}

// RemoveRole removes a role
func (s *RoleService) RemoveRole(roleID uint) error {
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
func (s *RoleService) ListRoles() ([]models.Role, error) {
	roles, err := s.roleRepo.List()
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách role")
	}
	return roles, nil
}

// AddRoleToUser adds a role to a user
func (s *RoleService) AddRoleToUser(userID string, roleID uint) error {
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
func (s *RoleService) RemoveRoleFromUser(userID string, roleID uint) error {
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
func (s *RoleService) CheckUserHasRole(userID string, roleName string) (bool, error) {
	hasRole, err := s.roleRepo.CheckUserHasRole(userID, roleName)
	if err != nil {
		return false, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role")
	}
	return hasRole, nil
}

// ListRolesOfUser lists all roles of a user
func (s *RoleService) ListRolesOfUser(userID string) ([]models.Role, error) {
	roles, err := s.roleRepo.ListRolesOfUser(userID)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách role của user")
	}
	return roles, nil
}

// ListUsersHasRole lists all users with a specific role
func (s *RoleService) ListUsersHasRole(roleName string) ([]models.User, error) {
	users, err := s.roleRepo.ListUsersHasRole(roleName)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách user có role")
	}
	return users, nil
}
