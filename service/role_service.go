package service

import (
	"errors"
	"strconv"

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

	// Bảo vệ: Không cho phép tạo role "super_admin" qua API
	if req.Name == "super_admin" {
		return nil, goerrorkit.NewBusinessError(403, "Không được phép tạo role 'super_admin' qua API. Role này chỉ có thể được tạo trong database").WithData(map[string]interface{}{
			"role_name": req.Name,
		})
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
	// Kiểm tra role trước khi xóa để có error message rõ ràng hơn
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy role").WithData(map[string]interface{}{
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin role")
	}

	// Bảo vệ: Không cho phép xóa system role (bao gồm super_admin)
	if role.IsSystem() {
		return goerrorkit.NewBusinessError(403, "Không được phép xóa system role").WithData(map[string]interface{}{
			"role_id":   roleID,
			"role_name": role.Name,
		})
	}

	if err := s.roleRepo.Delete(roleID); err != nil {
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
// currentUserRoleIDs: role IDs của user đang thực hiện thao tác (từ JWT token)
func (s *RoleService) AddRoleToUser(userID string, roleID uint, currentUserRoleIDs []uint) error {
	// Kiểm tra role trước khi gán
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy role").WithData(map[string]interface{}{
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin role")
	}

	roleName := role.Name

	// Bảo vệ: Không cho phép gán role "super_admin" qua REST API
	if roleName == "super_admin" {
		return goerrorkit.NewBusinessError(403, "Không được phép gán role 'super_admin' qua REST API. Role này chỉ có thể được gán trực tiếp trong database").WithData(map[string]interface{}{
			"role_id":   roleID,
			"role_name": roleName,
			"user_id":   userID,
		})
	}

	// Lấy role names của user đang login từ role IDs
	currentUserRoleNames := make(map[string]bool)
	if len(currentUserRoleIDs) > 0 {
		currentUserRoles, err := s.roleRepo.GetByIDs(currentUserRoleIDs)
		if err != nil {
			return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin roles của user đang login")
		}
		for _, r := range currentUserRoles {
			currentUserRoleNames[r.Name] = true
		}
	}

	// Kiểm tra quyền gán role:
	// - super-admin: được gán bất kỳ role nào (bao gồm admin)
	// - admin: chỉ được gán các role khác admin và super-admin
	// - các role khác: không được gán role nào
	isSuperAdmin := currentUserRoleNames["super_admin"]
	isAdmin := currentUserRoleNames["admin"]

	if !isSuperAdmin && !isAdmin {
		return goerrorkit.NewBusinessError(403, "Bạn không có quyền gán role cho user khác").WithData(map[string]interface{}{
			"user_id":   userID,
			"role_id":   roleID,
			"role_name": roleName,
		})
	}

	// Nếu là admin (nhưng không phải super-admin), không được gán role admin hoặc super-admin
	if isAdmin && !isSuperAdmin {
		if roleName == "admin" || roleName == "super_admin" {
			return goerrorkit.NewBusinessError(403, "Admin không được phép gán role 'admin' hoặc 'super_admin'. Chỉ super-admin mới có quyền này").WithData(map[string]interface{}{
				"user_id":   userID,
				"role_id":   roleID,
				"role_name": roleName,
			})
		}
	}

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
	// Kiểm tra role trước khi gỡ
	role, err := s.roleRepo.GetByID(roleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy role").WithData(map[string]interface{}{
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin role")
	}

	// Bảo vệ: Không cho phép gỡ role "super_admin" qua REST API
	if role.Name == "super_admin" {
		return goerrorkit.NewBusinessError(403, "Không được phép gỡ role 'super_admin' qua REST API. Role này chỉ có thể được gỡ trực tiếp trong database").WithData(map[string]interface{}{
			"role_id":   roleID,
			"role_name": role.Name,
			"user_id":   userID,
		})
	}

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
// role_id_name có thể là số (role_id) hoặc chuỗi (role_name)
func (s *RoleService) ListUsersHasRole(role_id_name string) ([]models.User, error) {
	var users []models.User
	var err error

	// Thử convert sang số, nếu thành công là ID, thất bại sẽ là string
	roleID, parseErr := strconv.ParseUint(role_id_name, 10, 32)
	if parseErr == nil {
		// role_id_name là số, dùng ListUsersHasRoleId
		users, err = s.roleRepo.ListUsersHasRoleId(uint(roleID))
	} else {
		// role_id_name là chuỗi, dùng ListUsersHasRoleName
		users, err = s.roleRepo.ListUsersHasRoleName(role_id_name)
	}

	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách user có role")
	}

	return users, nil
}
