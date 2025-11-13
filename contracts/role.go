package contracts

import "github.com/google/uuid"

// RoleInterface định nghĩa interface cho Role trong hệ thống authorization
// Ứng dụng bên ngoài cần implement interface này với model Role của họ
type RoleInterface interface {
	// GetID trả về ID của role
	GetID() uuid.UUID

	// GetName trả về tên của role
	GetName() string

	// IsSystem trả về true nếu đây là system role (không thể xóa)
	IsSystem() bool
}
