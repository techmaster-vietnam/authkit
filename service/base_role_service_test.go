package service

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// MockRoleRepository là mock repository cho testing
type MockRoleRepository[TRole core.RoleInterface] struct {
	roles       map[uint]TRole
	rolesByName map[string]TRole
	userRoles   map[string][]uint           // userID -> roleIDs
	roleUsers   map[uint][]string           // roleID -> userIDs (để hỗ trợ ListUsersHasRoleId)
	users       map[string]*models.BaseUser // userID -> user (để hỗ trợ ListUsersHasRole)
}

func NewMockRoleRepository[TRole core.RoleInterface]() *MockRoleRepository[TRole] {
	return &MockRoleRepository[TRole]{
		roles:       make(map[uint]TRole),
		rolesByName: make(map[string]TRole),
		userRoles:   make(map[string][]uint),
		roleUsers:   make(map[uint][]string),
		users:       make(map[string]*models.BaseUser),
	}
}

func (m *MockRoleRepository[TRole]) GetByID(id uint) (TRole, error) {
	var zero TRole
	role, ok := m.roles[id]
	if !ok {
		return zero, gorm.ErrRecordNotFound
	}
	return role, nil
}

func (m *MockRoleRepository[TRole]) GetByName(name string) (TRole, error) {
	var zero TRole
	role, ok := m.rolesByName[name]
	if !ok {
		return zero, gorm.ErrRecordNotFound
	}
	return role, nil
}

func (m *MockRoleRepository[TRole]) GetByIDs(ids []uint) ([]TRole, error) {
	result := make([]TRole, 0, len(ids))
	for _, id := range ids {
		if role, ok := m.roles[id]; ok {
			result = append(result, role)
		}
	}
	return result, nil
}

func (m *MockRoleRepository[TRole]) AddRoleToUser(userID string, roleID uint) error {
	if _, ok := m.userRoles[userID]; !ok {
		m.userRoles[userID] = []uint{}
	}
	// Check if role already assigned
	for _, rid := range m.userRoles[userID] {
		if rid == roleID {
			return nil // Already assigned
		}
	}
	m.userRoles[userID] = append(m.userRoles[userID], roleID)

	// Update roleUsers map
	if _, ok := m.roleUsers[roleID]; !ok {
		m.roleUsers[roleID] = []string{}
	}
	// Check if user already in list
	found := false
	for _, uid := range m.roleUsers[roleID] {
		if uid == userID {
			found = true
			break
		}
	}
	if !found {
		m.roleUsers[roleID] = append(m.roleUsers[roleID], userID)
	}
	return nil
}

func (m *MockRoleRepository[TRole]) RemoveRoleFromUser(userID string, roleID uint) error {
	roleIDs, ok := m.userRoles[userID]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	for i, rid := range roleIDs {
		if rid == roleID {
			m.userRoles[userID] = append(roleIDs[:i], roleIDs[i+1:]...)
			// Update roleUsers map
			if userIDs, ok := m.roleUsers[roleID]; ok {
				for j, uid := range userIDs {
					if uid == userID {
						m.roleUsers[roleID] = append(userIDs[:j], userIDs[j+1:]...)
						break
					}
				}
			}
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}

func (m *MockRoleRepository[TRole]) CheckUserHasRole(userID string, roleName string) (bool, error) {
	roleIDs, ok := m.userRoles[userID]
	if !ok {
		return false, nil
	}
	for _, rid := range roleIDs {
		if role, ok := m.roles[rid]; ok {
			if role.GetName() == roleName {
				return true, nil
			}
		}
	}
	return false, nil
}

func (m *MockRoleRepository[TRole]) ListRolesOfUser(userID string) ([]TRole, error) {
	roleIDs, ok := m.userRoles[userID]
	if !ok {
		return []TRole{}, nil
	}
	result := make([]TRole, 0, len(roleIDs))
	for _, rid := range roleIDs {
		if role, ok := m.roles[rid]; ok {
			result = append(result, role)
		}
	}
	return result, nil
}

func (m *MockRoleRepository[TRole]) ListUsersHasRole(roleName string) ([]interface{}, error) {
	// Deprecated: sử dụng ListUsersHasRoleName thay thế
	return m.ListUsersHasRoleName(roleName)
}

func (m *MockRoleRepository[TRole]) ListUsersHasRoleId(roleID uint) ([]interface{}, error) {
	userIDs, ok := m.roleUsers[roleID]
	if !ok {
		return []interface{}{}, nil
	}
	result := make([]interface{}, 0, len(userIDs))
	for _, userID := range userIDs {
		if user, ok := m.users[userID]; ok {
			result = append(result, user)
		} else {
			// Tạo user mặc định nếu chưa có
			user := &models.BaseUser{
				ID:    userID,
				Email: userID + "@test.com",
			}
			m.users[userID] = user
			result = append(result, user)
		}
	}
	return result, nil
}

func (m *MockRoleRepository[TRole]) ListUsersHasRoleName(roleName string) ([]interface{}, error) {
	role, ok := m.rolesByName[roleName]
	if !ok {
		return []interface{}{}, nil
	}
	return m.ListUsersHasRoleId(role.GetID())
}

func (m *MockRoleRepository[TRole]) List() ([]TRole, error) {
	result := make([]TRole, 0, len(m.roles))
	for _, role := range m.roles {
		result = append(result, role)
	}
	return result, nil
}

func (m *MockRoleRepository[TRole]) Create(role TRole) error {
	m.roles[role.GetID()] = role
	m.rolesByName[role.GetName()] = role
	return nil
}

func (m *MockRoleRepository[TRole]) Update(role TRole) error {
	m.roles[role.GetID()] = role
	m.rolesByName[role.GetName()] = role
	return nil
}

func (m *MockRoleRepository[TRole]) Delete(id uint) error {
	role, ok := m.roles[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	if role.IsSystem() {
		return errors.New("cannot delete system role")
	}
	delete(m.roles, id)
	delete(m.rolesByName, role.GetName())
	return nil
}

func (m *MockRoleRepository[TRole]) GetIDsByNames(names []string) (map[string]uint, error) {
	result := make(map[string]uint)
	for _, name := range names {
		if role, ok := m.rolesByName[name]; ok {
			result[name] = role.GetID()
		}
	}
	return result, nil
}

func (m *MockRoleRepository[TRole]) DB() interface{} {
	return nil // Not needed for tests
}

// MockUserRepository là mock repository cho user testing
type MockUserRepository[TUser core.UserInterface] struct {
	users        map[string]TUser
	usersByEmail map[string]TUser
}

func NewMockUserRepository[TUser core.UserInterface]() *MockUserRepository[TUser] {
	return &MockUserRepository[TUser]{
		users:        make(map[string]TUser),
		usersByEmail: make(map[string]TUser),
	}
}

func (m *MockUserRepository[TUser]) GetByID(id string) (TUser, error) {
	var zero TUser
	user, ok := m.users[id]
	if !ok {
		return zero, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *MockUserRepository[TUser]) GetByEmail(email string) (TUser, error) {
	var zero TUser
	user, ok := m.usersByEmail[email]
	if !ok {
		return zero, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (m *MockUserRepository[TUser]) Create(user TUser) error {
	m.users[user.GetID()] = user
	m.usersByEmail[user.GetEmail()] = user
	return nil
}

func (m *MockUserRepository[TUser]) Update(user TUser) error {
	m.users[user.GetID()] = user
	m.usersByEmail[user.GetEmail()] = user
	return nil
}

func (m *MockUserRepository[TUser]) Delete(id string) error {
	user, ok := m.users[id]
	if !ok {
		return gorm.ErrRecordNotFound
	}
	delete(m.users, id)
	delete(m.usersByEmail, user.GetEmail())
	return nil
}

func (m *MockUserRepository[TUser]) List(offset, limit int) ([]TUser, int64, error) {
	result := make([]TUser, 0, len(m.users))
	for _, user := range m.users {
		result = append(result, user)
	}
	return result, int64(len(m.users)), nil
}

func (m *MockUserRepository[TUser]) DB() interface{} {
	return nil // Not needed for tests
}

// Helper function để tạo test roles
func createTestRole(id uint, name string, isSystem bool) *models.BaseRole {
	return &models.BaseRole{
		ID:     id,
		Name:   name,
		System: isSystem,
	}
}

// Helper function để tạo test users
func createTestUser(id, email string) *models.BaseUser {
	return &models.BaseUser{
		ID:    id,
		Email: email,
	}
}

func TestBaseRoleService_AddRoleToUser(t *testing.T) {
	// Setup test roles
	superAdminRole := createTestRole(1, "super_admin", true)
	adminRole := createTestRole(2, "admin", false)
	editorRole := createTestRole(3, "editor", false)
	readerRole := createTestRole(4, "reader", false)

	tests := []struct {
		name               string
		targetUserID       string
		roleID             uint
		currentUserRoleIDs []uint
		setupRepo          func() *MockRoleRepository[*models.BaseRole]
		expectedError      bool
		expectedErrorType  string // "business", "validation", "not_found", "none"
		expectedErrorMsg   string
	}{
		{
			name:               "Super-admin gán editor cho user khác - thành công",
			targetUserID:       "user123",
			roleID:             3,         // editor
			currentUserRoleIDs: []uint{1}, // super_admin
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(superAdminRole)
				repo.Create(editorRole)
				return repo
			},
			expectedError:     false,
			expectedErrorType: "none",
		},
		{
			name:               "Super-admin gán admin cho user khác - thành công",
			targetUserID:       "user123",
			roleID:             2,         // admin
			currentUserRoleIDs: []uint{1}, // super_admin
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(superAdminRole)
				repo.Create(adminRole)
				return repo
			},
			expectedError:     false,
			expectedErrorType: "none",
		},
		{
			name:               "Admin gán editor cho user khác - thành công",
			targetUserID:       "user123",
			roleID:             3,         // editor
			currentUserRoleIDs: []uint{2}, // admin
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				repo.Create(editorRole)
				return repo
			},
			expectedError:     false,
			expectedErrorType: "none",
		},
		{
			name:               "Admin gán admin cho user khác - lỗi nghiệp vụ",
			targetUserID:       "user123",
			roleID:             2,         // admin
			currentUserRoleIDs: []uint{2}, // admin
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Admin không được phép gán role 'admin'",
		},
		{
			name:               "Admin gán super_admin cho user khác - lỗi nghiệp vụ (bị chặn trước khi đến check quyền)",
			targetUserID:       "user123",
			roleID:             1,         // super_admin
			currentUserRoleIDs: []uint{2}, // admin
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				repo.Create(superAdminRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Không được phép gán role 'super_admin' qua REST API",
		},
		{
			name:               "Editor gán reader cho user khác - lỗi authorization",
			targetUserID:       "user123",
			roleID:             4,         // reader
			currentUserRoleIDs: []uint{3}, // editor
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(editorRole)
				repo.Create(readerRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Bạn không có quyền gán role cho user khác",
		},
		{
			name:               "User không có role gán role cho user khác - lỗi authorization",
			targetUserID:       "user123",
			roleID:             4,        // reader
			currentUserRoleIDs: []uint{}, // no roles
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(readerRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Bạn không có quyền gán role cho user khác",
		},
		{
			name:               "Role không tồn tại - lỗi not found",
			targetUserID:       "user123",
			roleID:             999,       // không tồn tại
			currentUserRoleIDs: []uint{2}, // admin
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "not_found",
			expectedErrorMsg:  "Không tìm thấy role",
		},
		{
			name:               "Super-admin gán super_admin - lỗi nghiệp vụ (không được phép qua API)",
			targetUserID:       "user123",
			roleID:             1,         // super_admin
			currentUserRoleIDs: []uint{1}, // super_admin
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(superAdminRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Không được phép gán role 'super_admin' qua REST API",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := tt.setupRepo()
			mockUserRepo := NewMockUserRepository[*models.BaseUser]()
			service := &BaseRoleService[*models.BaseUser, *models.BaseRole]{
				roleRepo: mockRepo,
				userRepo: mockUserRepo,
			}

			// Execute
			err := service.AddRoleToUser(tt.targetUserID, tt.roleID, tt.currentUserRoleIDs)

			// Assert
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}

				// Check error type
				var appErr *goerrorkit.AppError
				if errors.As(err, &appErr) {
					switch tt.expectedErrorType {
					case "business":
						if appErr.Type != goerrorkit.BusinessError {
							t.Errorf("Expected business error, got %v", appErr.Type)
						}
					case "not_found":
						if appErr.Type != goerrorkit.BusinessError {
							t.Errorf("Expected business error (not found), got %v", appErr.Type)
						}
					}

					// Check error message contains expected text
					if tt.expectedErrorMsg != "" {
						if !contains(appErr.Error(), tt.expectedErrorMsg) {
							t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, appErr.Error())
						}
					}
				} else {
					t.Errorf("Expected AppError, got %T: %v", err, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBaseRoleService_AddRole(t *testing.T) {
	tests := []struct {
		name              string
		req               BaseAddRoleRequest
		setupRepo         func() *MockRoleRepository[*models.BaseRole]
		expectedError     bool
		expectedErrorType string
		expectedErrorMsg  string
	}{
		{
			name: "Tạo role hợp lệ - thành công",
			req: BaseAddRoleRequest{
				ID:       5,
				Name:     "test_role",
				IsSystem: false,
			},
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				return NewMockRoleRepository[*models.BaseRole]()
			},
			expectedError: false,
		},
		{
			name: "Tạo role super_admin - lỗi nghiệp vụ",
			req: BaseAddRoleRequest{
				ID:       1,
				Name:     "super_admin",
				IsSystem: true,
			},
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				return NewMockRoleRepository[*models.BaseRole]()
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Không được phép tạo role 'super_admin' qua API",
		},
		{
			name: "Tạo role với ID đã tồn tại - lỗi conflict",
			req: BaseAddRoleRequest{
				ID:       2,
				Name:     "new_role",
				IsSystem: false,
			},
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				existingRole := createTestRole(2, "existing", false)
				repo.Create(existingRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Role với ID này đã tồn tại",
		},
		{
			name: "Tạo role với tên đã tồn tại - lỗi conflict",
			req: BaseAddRoleRequest{
				ID:       5,
				Name:     "existing",
				IsSystem: false,
			},
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				existingRole := createTestRole(2, "existing", false)
				repo.Create(existingRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Role với tên này đã tồn tại",
		},
		{
			name: "Tạo role với ID = 0 - lỗi validation",
			req: BaseAddRoleRequest{
				ID:       0,
				Name:     "test_role",
				IsSystem: false,
			},
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				return NewMockRoleRepository[*models.BaseRole]()
			},
			expectedError:     true,
			expectedErrorType: "validation",
			expectedErrorMsg:  "ID role là bắt buộc",
		},
		{
			name: "Tạo role với tên rỗng - lỗi validation",
			req: BaseAddRoleRequest{
				ID:       5,
				Name:     "",
				IsSystem: false,
			},
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				return NewMockRoleRepository[*models.BaseRole]()
			},
			expectedError:     true,
			expectedErrorType: "validation",
			expectedErrorMsg:  "Tên role là bắt buộc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := tt.setupRepo()
			mockUserRepo := NewMockUserRepository[*models.BaseUser]()
			service := &BaseRoleService[*models.BaseUser, *models.BaseRole]{
				roleRepo: mockRepo,
				userRepo: mockUserRepo,
			}

			// Execute
			role, err := service.AddRole(tt.req)

			// Assert
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}

				var appErr *goerrorkit.AppError
				if errors.As(err, &appErr) {
					switch tt.expectedErrorType {
					case "business":
						if appErr.Type != goerrorkit.BusinessError {
							t.Errorf("Expected business error, got %v", appErr.Type)
						}
					case "validation":
						if appErr.Type != goerrorkit.ValidationError {
							t.Errorf("Expected validation error, got %v", appErr.Type)
						}
					}

					if tt.expectedErrorMsg != "" {
						if !contains(appErr.Error(), tt.expectedErrorMsg) {
							t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, appErr.Error())
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if reflect.ValueOf(role).IsNil() {
					t.Errorf("Expected role but got nil")
				}
				if role.GetID() != tt.req.ID {
					t.Errorf("Expected role ID %d, got %d", tt.req.ID, role.GetID())
				}
				if role.GetName() != tt.req.Name {
					t.Errorf("Expected role name %s, got %s", tt.req.Name, role.GetName())
				}
			}
		})
	}
}

func TestBaseRoleService_RemoveRole(t *testing.T) {
	systemRole := createTestRole(1, "system_role", true)
	normalRole := createTestRole(2, "normal_role", false)

	tests := []struct {
		name              string
		roleID            uint
		setupRepo         func() *MockRoleRepository[*models.BaseRole]
		expectedError     bool
		expectedErrorType string
		expectedErrorMsg  string
	}{
		{
			name:   "Xóa role không phải system - thành công",
			roleID: 2,
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(normalRole)
				return repo
			},
			expectedError: false,
		},
		{
			name:   "Xóa system role - lỗi nghiệp vụ",
			roleID: 1,
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(systemRole)
				return repo
			},
			expectedError:     true,
			expectedErrorType: "business",
			expectedErrorMsg:  "Không được phép xóa system role",
		},
		{
			name:   "Xóa role không tồn tại - lỗi not found",
			roleID: 999,
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				return NewMockRoleRepository[*models.BaseRole]()
			},
			expectedError:     true,
			expectedErrorType: "not_found",
			expectedErrorMsg:  "Không tìm thấy role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := tt.setupRepo()
			mockUserRepo := NewMockUserRepository[*models.BaseUser]()
			service := &BaseRoleService[*models.BaseUser, *models.BaseRole]{
				roleRepo: mockRepo,
				userRepo: mockUserRepo,
			}

			// Execute
			err := service.RemoveRole(tt.roleID)

			// Assert
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}

				var appErr *goerrorkit.AppError
				if errors.As(err, &appErr) {
					if tt.expectedErrorMsg != "" {
						if !contains(appErr.Error(), tt.expectedErrorMsg) {
							t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedErrorMsg, appErr.Error())
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestBaseRoleService_ListUsersHasRole(t *testing.T) {
	// Setup test roles
	adminRole := createTestRole(2, "admin", false)
	editorRole := createTestRole(3, "editor", false)

	// Setup test users
	user1 := createTestUser("user1", "user1@test.com")
	user2 := createTestUser("user2", "user2@test.com")
	user3 := createTestUser("user3", "user3@test.com")

	tests := []struct {
		name            string
		roleIdName      string // có thể là số (ID) hoặc chuỗi (name)
		setupRepo       func() *MockRoleRepository[*models.BaseRole]
		expectedError   bool
		expectedCount   int
		expectedUserIDs []string
	}{
		{
			name:       "Lấy users có role bằng ID - thành công",
			roleIdName: "2", // admin role ID
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				repo.users[user1.ID] = user1
				repo.users[user2.ID] = user2
				repo.AddRoleToUser(user1.ID, 2)
				repo.AddRoleToUser(user2.ID, 2)
				return repo
			},
			expectedError:   false,
			expectedCount:   2,
			expectedUserIDs: []string{"user1", "user2"},
		},
		{
			name:       "Lấy users có role bằng name - thành công",
			roleIdName: "admin", // admin role name
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				repo.users[user1.ID] = user1
				repo.users[user2.ID] = user2
				repo.AddRoleToUser(user1.ID, 2)
				repo.AddRoleToUser(user2.ID, 2)
				return repo
			},
			expectedError:   false,
			expectedCount:   2,
			expectedUserIDs: []string{"user1", "user2"},
		},
		{
			name:       "Lấy users có role editor bằng ID - thành công",
			roleIdName: "3", // editor role ID
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(editorRole)
				repo.users[user3.ID] = user3
				repo.AddRoleToUser(user3.ID, 3)
				return repo
			},
			expectedError:   false,
			expectedCount:   1,
			expectedUserIDs: []string{"user3"},
		},
		{
			name:       "Lấy users có role editor bằng name - thành công",
			roleIdName: "editor", // editor role name
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(editorRole)
				repo.users[user3.ID] = user3
				repo.AddRoleToUser(user3.ID, 3)
				return repo
			},
			expectedError:   false,
			expectedCount:   1,
			expectedUserIDs: []string{"user3"},
		},
		{
			name:       "Lấy users có role không tồn tại bằng ID - trả về empty",
			roleIdName: "999", // role ID không tồn tại
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				return repo
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:       "Lấy users có role không tồn tại bằng name - trả về empty",
			roleIdName: "nonexistent", // role name không tồn tại
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				return repo
			},
			expectedError: false,
			expectedCount: 0,
		},
		{
			name:       "Lấy users có role nhưng không có user nào - trả về empty",
			roleIdName: "2", // admin role ID
			setupRepo: func() *MockRoleRepository[*models.BaseRole] {
				repo := NewMockRoleRepository[*models.BaseRole]()
				repo.Create(adminRole)
				return repo
			},
			expectedError: false,
			expectedCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := tt.setupRepo()
			mockUserRepo := NewMockUserRepository[*models.BaseUser]()
			service := &BaseRoleService[*models.BaseUser, *models.BaseRole]{
				roleRepo: mockRepo,
				userRepo: mockUserRepo,
			}

			// Execute
			users, err := service.ListUsersHasRole(tt.roleIdName)

			// Assert
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got nil")
					return
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
					return
				}

				if len(users) != tt.expectedCount {
					t.Errorf("Expected %d users, got %d", tt.expectedCount, len(users))
					return
				}

				// Kiểm tra user IDs nếu có
				if len(tt.expectedUserIDs) > 0 {
					userIDs := make(map[string]bool)
					for _, user := range users {
						if baseUser, ok := user.(*models.BaseUser); ok {
							userIDs[baseUser.ID] = true
						}
					}
					for _, expectedID := range tt.expectedUserIDs {
						if !userIDs[expectedID] {
							t.Errorf("Expected user %s not found in result", expectedID)
						}
					}
				}
			}
		})
	}
}

// Helper function để check string contains
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
