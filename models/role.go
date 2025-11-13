package models

// Role represents a role in the system
// Đây là reference implementation của contracts.RoleInterface
// Ứng dụng bên ngoài có thể sử dụng model này hoặc implement RoleInterface với model của riêng họ
type Role struct {
	ID     uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name   string `gorm:"uniqueIndex;not null" json:"name"`
	System bool   `gorm:"column:is_system;default:false" json:"is_system"` // System roles cannot be deleted

	// Relationships
	Users []User `gorm:"many2many:user_roles;" json:"users,omitempty"`
}

// TableName specifies the table name
func (Role) TableName() string {
	return "roles"
}

// Implement contracts.RoleInterface

// GetID trả về ID của role
func (r *Role) GetID() uint {
	return r.ID
}

// GetName trả về tên của role
func (r *Role) GetName() string {
	return r.Name
}

// IsSystem trả về true nếu đây là system role (không thể xóa)
func (r *Role) IsSystem() bool {
	return r.System
}
