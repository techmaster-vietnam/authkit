package contracts

import "github.com/google/uuid"

// UserInterface định nghĩa interface cho User trong hệ thống authentication/authorization
// Ứng dụng bên ngoài cần implement interface này với model User của họ
type UserInterface interface {
	// GetID trả về ID của user
	GetID() uuid.UUID

	// GetEmail trả về email của user
	GetEmail() string

	// GetUsername trả về username của user
	GetUsername() string

	// GetPassword trả về password đã hash của user
	GetPassword() string

	// SetPassword set password đã hash cho user
	SetPassword(password string)

	// IsActive trả về trạng thái active của user
	IsActive() bool

	// SetActive set trạng thái active cho user
	SetActive(active bool)

	// GetRoles trả về danh sách roles của user
	// Ứng dụng có thể implement để trả về []RoleInterface hoặc []string tùy theo nhu cầu
	GetRoles() []RoleInterface
}
