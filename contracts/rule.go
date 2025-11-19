package contracts

// AccessType định nghĩa các loại rule
type AccessType string

const (
	AccessPublic AccessType = "PUBLIC"  // Cho phép mọi người, kể cả anonymous
	AccessAllow  AccessType = "ALLOW"   // Cho phép các role cụ thể (roles rỗng = mọi user đã đăng nhập)
	AccessForbid AccessType = "FORBIDE" // Cấm các role cụ thể
)

// RuleInterface định nghĩa interface cho Rule trong hệ thống authorization
// Ứng dụng bên ngoài cần implement interface này với model Rule của họ
type RuleInterface interface {
	// GetID trả về ID của rule (format: "METHOD|PATH")
	GetID() string

	// GetMethod trả về HTTP method (GET, POST, PUT, DELETE, etc.)
	GetMethod() string

	// GetPath trả về URL path pattern
	GetPath() string

	// GetType trả về loại rule
	GetType() AccessType

	// GetRoles trả về danh sách role IDs được áp dụng cho rule này
	GetRoles() []uint
}
