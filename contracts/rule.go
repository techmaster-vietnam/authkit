package contracts

import "github.com/google/uuid"

// RuleType định nghĩa các loại rule
type RuleType string

const (
	RuleTypePublic        RuleType = "PUBLIC"        // Cho phép mọi người, kể cả anonymous
	RuleTypeAllow         RuleType = "ALLOW"         // Cho phép các role cụ thể
	RuleTypeForbid        RuleType = "FORBIDE"       // Cấm các role cụ thể
	RuleTypeAuthenticated RuleType = "AUTHENTICATED" // Yêu cầu authentication nhưng bất kỳ role nào
)

// RuleInterface định nghĩa interface cho Rule trong hệ thống authorization
// Ứng dụng bên ngoài cần implement interface này với model Rule của họ
type RuleInterface interface {
	// GetID trả về ID của rule
	GetID() uuid.UUID

	// GetMethod trả về HTTP method (GET, POST, PUT, DELETE, etc.)
	GetMethod() string

	// GetPath trả về URL path pattern
	GetPath() string

	// GetType trả về loại rule
	GetType() RuleType

	// GetRoles trả về danh sách roles được áp dụng cho rule này
	GetRoles() []string

	// GetPriority trả về priority của rule (priority cao hơn được check trước)
	GetPriority() int
}
