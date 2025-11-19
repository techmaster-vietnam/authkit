package models

// BaseRole là base model cho Role, có thể được embed vào custom Role models
type BaseRole struct {
	ID     uint   `gorm:"primaryKey" json:"id"`
	Name   string `gorm:"uniqueIndex;not null" json:"name"`
	System bool   `gorm:"column:is_system;default:false" json:"is_system"` // System roles cannot be deleted

	// Relationships - sử dụng BaseUser để có thể embed
	Users []BaseUser `gorm:"many2many:user_roles;foreignKey:ID;joinForeignKey:role_id;References:ID;joinReferences:user_id" json:"users,omitempty"`
}

// TableName specifies the table name
func (BaseRole) TableName() string {
	return "roles"
}

// GetID trả về ID của role
func (r *BaseRole) GetID() uint {
	return r.ID
}

// GetName trả về tên của role
func (r *BaseRole) GetName() string {
	return r.Name
}

// IsSystem trả về true nếu đây là system role (không thể xóa)
func (r *BaseRole) IsSystem() bool {
	return r.System
}

// Role là alias cho BaseRole để backward compatibility
// Trong tương lai có thể deprecated
type Role = BaseRole
