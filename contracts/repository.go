package contracts

// UserRepositoryInterface định nghĩa interface cho User Repository
// Ứng dụng bên ngoài cần implement interface này với repository của họ
type UserRepositoryInterface interface {
	// Create tạo user mới
	Create(user UserInterface) error

	// GetByID lấy user theo ID
	GetByID(id string) (UserInterface, error)

	// GetByEmail lấy user theo email
	GetByEmail(email string) (UserInterface, error)

	// Update cập nhật user
	Update(user UserInterface) error

	// Delete xóa user (soft delete)
	Delete(id string) error
}

// RoleRepositoryInterface định nghĩa interface cho Role Repository
type RoleRepositoryInterface interface {
	// GetByID lấy role theo ID
	GetByID(id uint) (RoleInterface, error)

	// GetByName lấy role theo tên
	GetByName(name string) (RoleInterface, error)

	// ListRolesOfUser lấy danh sách roles của user
	ListRolesOfUser(userID string) ([]RoleInterface, error)

	// CheckUserHasRole kiểm tra user có role cụ thể không
	CheckUserHasRole(userID string, roleName string) (bool, error)

	// AddRoleToUser thêm role cho user
	AddRoleToUser(userID string, roleID uint) error

	// RemoveRoleFromUser xóa role khỏi user
	RemoveRoleFromUser(userID string, roleID uint) error
}

// RuleRepositoryInterface định nghĩa interface cho Rule Repository
type RuleRepositoryInterface interface {
	// GetAllRulesForCache lấy tất cả rules để cache (dùng cho middleware)
	GetAllRulesForCache() ([]RuleInterface, error)
}
