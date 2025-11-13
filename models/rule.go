package models

import (
	"fmt"

	"github.com/techmaster-vietnam/authkit/contracts"
	"gorm.io/gorm"
)

// AccessType represents the type of rule
// DEPRECATED: Sử dụng contracts.AccessType thay thế
type AccessType = contracts.AccessType

const (
	AccessPublic AccessType = contracts.AccessPublic // Allow anyone, including anonymous
	AccessAllow  AccessType = contracts.AccessAllow  // Allow specific roles (empty roles = any authenticated user)
	AccessForbid AccessType = contracts.AccessForbid // Forbid specific roles
)

// Rule represents an authorization rule for HTTP endpoints
// Đây là reference implementation của contracts.RuleInterface
// Ứng dụng bên ngoài có thể sử dụng model này hoặc implement RuleInterface với model của riêng họ
// ID format: "METHOD|PATH" (ví dụ: "GET|/api/users")
type Rule struct {
	ID     string     `gorm:"primaryKey" json:"id"`                               // Format: "METHOD|PATH"
	Method string     `gorm:"not null;uniqueIndex:idx_method_path" json:"method"` // GET, POST, PUT, DELETE, etc.
	Path   string     `gorm:"not null;uniqueIndex:idx_method_path" json:"path"`   // URL path pattern
	Type   AccessType `gorm:"type:varchar(20);not null" json:"type"`              // PUBLIC, ALLOW, FORBIDE
	Roles  []string   `gorm:"type:text;serializer:json" json:"roles"`             // Stored as JSON in database
}

// BeforeCreate hook to generate ID from Method and Path
func (r *Rule) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = fmt.Sprintf("%s|%s", r.Method, r.Path)
	}
	return nil
}

// TableName specifies the table name
func (Rule) TableName() string {
	return "rules"
}

// Implement contracts.RuleInterface

// GetID trả về ID của rule
func (r *Rule) GetID() string {
	return r.ID
}

// GetMethod trả về HTTP method (GET, POST, PUT, DELETE, etc.)
func (r *Rule) GetMethod() string {
	return r.Method
}

// GetPath trả về URL path pattern
func (r *Rule) GetPath() string {
	return r.Path
}

// GetType trả về loại rule
func (r *Rule) GetType() contracts.AccessType {
	return contracts.AccessType(r.Type)
}

// GetRoles trả về danh sách roles được áp dụng cho rule này
func (r *Rule) GetRoles() []string {
	return r.Roles
}
