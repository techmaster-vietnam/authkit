package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Role represents a role in the system
// Đây là reference implementation của contracts.RoleInterface
// Ứng dụng bên ngoài có thể sử dụng model này hoặc implement RoleInterface với model của riêng họ
type Role struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Name        string         `gorm:"uniqueIndex;not null" json:"name"`
	Description string         `json:"description"`
	System      bool           `gorm:"column:is_system;default:false" json:"is_system"` // System roles cannot be deleted
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Users []User `gorm:"many2many:user_roles;" json:"users,omitempty"`
}

// BeforeCreate hook to generate UUID
func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Role) TableName() string {
	return "roles"
}

// Implement contracts.RoleInterface

// GetID trả về ID của role
func (r *Role) GetID() uuid.UUID {
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
