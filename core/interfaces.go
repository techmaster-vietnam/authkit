package core

// UserInterface định nghĩa interface cho User model
// Các custom User models phải implement interface này
type UserInterface interface {
	GetID() string
	GetEmail() string
	SetEmail(email string)
	GetPassword() string
	SetPassword(password string)
	IsActive() bool
	SetActive(active bool)
	GetRoles() []RoleInterface
	// Methods để hỗ trợ Register và UpdateProfile
	GetFullName() string
	SetFullName(fullName string)
}

// RoleInterface định nghĩa interface cho Role model
// Các custom Role models phải implement interface này
type RoleInterface interface {
	GetID() uint
	GetName() string
	IsSystem() bool
}

// RoleRepositoryInterface định nghĩa interface cho Role Repository
// Cho phép mock repository trong tests
type RoleRepositoryInterface[TRole RoleInterface] interface {
	GetByID(id uint) (TRole, error)
	GetByName(name string) (TRole, error)
	GetByIDs(ids []uint) ([]TRole, error)
	AddRoleToUser(userID string, roleID uint) error
	RemoveRoleFromUser(userID string, roleID uint) error
	CheckUserHasRole(userID string, roleName string) (bool, error)
	ListRolesOfUser(userID string) ([]TRole, error)
	ListUsersHasRole(roleName string) ([]interface{}, error)
	ListUsersHasRoleId(roleID uint) ([]interface{}, error)
	ListUsersHasRoleName(roleName string) ([]interface{}, error)
	List() ([]TRole, error)
	Create(role TRole) error
	Update(role TRole) error
	Delete(id uint) error
	GetIDsByNames(names []string) (map[string]uint, error)
	UpdateUserRoles(userID string, roleIDs []uint) error
	// Cache methods để tối ưu chuyển đổi giữa role_id và role_name
	GetRoleNameByID(id uint) (string, bool)
	GetRoleIDByName(name string) (uint, bool)
	GetNamesByIDs(ids []uint) map[uint]string
	RefreshRoleCache() error
	DB() interface{} // Trả về *gorm.DB nhưng dùng interface{} để tránh circular dependency
}

// UserRepositoryInterface định nghĩa interface cho User Repository
// Cho phép mock repository trong tests
type UserRepositoryInterface[TUser UserInterface] interface {
	GetByID(id string) (TUser, error)
	GetByEmail(email string) (TUser, error)
	Create(user TUser) error
	Update(user TUser) error
	Delete(id string) error
	List(offset, limit int, filter interface{}) ([]TUser, int64, error) // filter có thể là *repository.UserFilter hoặc nil
	DB() interface{} // Trả về *gorm.DB nhưng dùng interface{} để tránh circular dependency
}

// NotificationSender là interface để gửi email hoặc tin nhắn xác thực
// Người dùng có thể implement interface này để tích hợp với hệ thống email/SMS của họ
type NotificationSender interface {
	// SendPasswordResetToken gửi password reset token đến user qua email hoặc tin nhắn
	// email: Email của user cần reset password
	// token: Reset token (plain text, sẽ được hash khi lưu vào DB)
	// expiresIn: Thời gian token còn hiệu lực (ví dụ: "1 giờ")
	// Returns: error nếu gửi thất bại
	SendPasswordResetToken(email string, token string, expiresIn string) error
}
