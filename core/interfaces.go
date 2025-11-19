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

