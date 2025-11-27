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
	List(offset, limit int) ([]TUser, int64, error)
	DB() interface{} // Trả về *gorm.DB nhưng dùng interface{} để tránh circular dependency
}
