package service

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// BaseRoleService là generic role service
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseRoleService[TUser core.UserInterface, TRole core.RoleInterface] struct {
	roleRepo core.RoleRepositoryInterface[TRole]
	userRepo core.UserRepositoryInterface[TUser]
}

// NewBaseRoleService tạo mới BaseRoleService với generic types
func NewBaseRoleService[TUser core.UserInterface, TRole core.RoleInterface](
	roleRepo core.RoleRepositoryInterface[TRole],
	userRepo core.UserRepositoryInterface[TUser],
) *BaseRoleService[TUser, TRole] {
	return &BaseRoleService[TUser, TRole]{
		roleRepo: roleRepo,
		userRepo: userRepo,
	}
}

// BaseAddRoleRequest represents add role request
type BaseAddRoleRequest struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	IsSystem bool   `json:"is_system"`
}

// AddRole creates a new role
func (s *BaseRoleService[TUser, TRole]) AddRole(req BaseAddRoleRequest) (TRole, error) {
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

	// Bảo vệ: Không cho phép tạo role "super_admin" qua API
	if req.Name == "super_admin" {
		return zero, goerrorkit.NewBusinessError(403, "Không được phép tạo role 'super_admin' qua API. Role này chỉ có thể được tạo trong database").WithData(map[string]interface{}{
			"role_name": req.Name,
		})
	}

	// Create role using reflection to set fields
	// This works because TRole must embed BaseRole or be BaseRole itself
	// Sử dụng helper function tổng quát để tạo instance, hỗ trợ cả pointer và value types
	role, rv := utils.NewGenericInstance[TRole]()

	// Set các fields
	if rv.Kind() == reflect.Struct {
		// Set ID field
		if idField := rv.FieldByName("ID"); idField.IsValid() && idField.CanSet() {
			switch idField.Kind() {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				idField.SetUint(uint64(req.ID))
			}
		}
		// Set Name field
		if nameField := rv.FieldByName("Name"); nameField.IsValid() && nameField.CanSet() {
			if nameField.Kind() == reflect.String {
				nameField.SetString(req.Name)
			}
		}
		// Set System field
		if systemField := rv.FieldByName("System"); systemField.IsValid() && systemField.CanSet() {
			if systemField.Kind() == reflect.Bool {
				systemField.SetBool(req.IsSystem)
			}
		}
	}

	if err := s.roleRepo.Create(role); err != nil {
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo role")
	}

	return role, nil
}

// RemoveRole removes a role
func (s *BaseRoleService[TUser, TRole]) RemoveRole(roleID uint) error {
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
			"role_name": role.GetName(),
		})
	}

	if err := s.roleRepo.Delete(roleID); err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa role")
	}
	return nil
}

// ListRoles lists all roles
func (s *BaseRoleService[TUser, TRole]) ListRoles() ([]TRole, error) {
	roles, err := s.roleRepo.List()
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách role")
	}
	// Đảm bảo luôn trả về empty slice thay vì nil
	if roles == nil {
		return []TRole{}, nil
	}
	return roles, nil
}

// AddRoleToUser adds a role to a user
// currentUserRoleIDs: role IDs của user đang thực hiện thao tác (từ JWT token)
func (s *BaseRoleService[TUser, TRole]) AddRoleToUser(userID string, roleID uint, currentUserRoleIDs []uint) error {
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

	roleName := role.GetName()

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
			currentUserRoleNames[r.GetName()] = true
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
// userID có thể là ID (chuỗi 12 ký tự) hoặc email (chứa ký tự "@")
// roleID có thể là số (1,2,3,4) hoặc tên role ("editor", "reader", ...)
func (s *BaseRoleService[TUser, TRole]) RemoveRoleFromUser(userID string, roleID string) error {
	// Resolve roleID: kiểm tra là số hay tên role
	var role TRole
	var err error
	var actualRoleID uint

	// Thử parse roleID thành số
	parsedRoleID, parseErr := strconv.ParseUint(roleID, 10, 32)
	if parseErr == nil {
		// roleID là số, dùng GetByID
		actualRoleID = uint(parsedRoleID)
		role, err = s.roleRepo.GetByID(actualRoleID)
	} else {
		// roleID là tên role, dùng GetByName
		role, err = s.roleRepo.GetByName(roleID)
		if err == nil {
			actualRoleID = role.GetID()
		}
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy role").WithData(map[string]interface{}{
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin role")
	}

	// Bảo vệ: Không cho phép gỡ role "super_admin" qua REST API
	if role.GetName() == "super_admin" {
		return goerrorkit.NewBusinessError(403, "Không được phép gỡ role 'super_admin' qua REST API. Role này chỉ có thể được gỡ trực tiếp trong database").WithData(map[string]interface{}{
			"role_id":   roleID,
			"role_name": role.GetName(),
			"user_id":   userID,
		})
	}

	// Resolve userID: nếu là email thì cần lấy ID thực tế từ DB
	actualUserID := userID
	if strings.Contains(userID, "@") {
		// userID là email, sử dụng userRepo.GetByEmail để lấy user và ID thực tế
		user, err := s.userRepo.GetByEmail(userID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return goerrorkit.NewBusinessError(404, "Không tìm thấy user").WithData(map[string]interface{}{
					"user_id": userID,
				})
			}
			return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin user")
		}
		actualUserID = user.GetID()
	}

	if err := s.roleRepo.RemoveRoleFromUser(actualUserID, actualRoleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "User không có role này").WithData(map[string]interface{}{
				"user_id": userID,
				"role_id": roleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa role khỏi user")
	}
	return nil
}

// CheckUserHasRole checks if a user has a specific role
func (s *BaseRoleService[TUser, TRole]) CheckUserHasRole(userID string, roleName string) (bool, error) {
	hasRole, err := s.roleRepo.CheckUserHasRole(userID, roleName)
	if err != nil {
		return false, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role")
	}
	return hasRole, nil
}

// ListUsersHasRole lists all users with a specific role
// role_id_name có thể là số (role_id) hoặc chuỗi (role_name)
// Trả về []interface{} vì không biết custom User type
func (s *BaseRoleService[TUser, TRole]) ListUsersHasRole(role_id_name string) ([]interface{}, error) {
	var users []interface{}
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
