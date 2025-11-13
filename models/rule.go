package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/techmaster-vietnam/authkit/contracts"
	"gorm.io/gorm"
)

// RuleType represents the type of rule
// DEPRECATED: Sử dụng contracts.RuleType thay thế
type RuleType = contracts.RuleType

const (
	RuleTypePublic RuleType = contracts.RuleTypePublic        // Allow anyone, including anonymous
	RuleTypeAllow  RuleType = contracts.RuleTypeAllow         // Allow specific roles
	RuleTypeForbid RuleType = contracts.RuleTypeForbid        // Forbid specific roles
	RuleTypeAuth   RuleType = contracts.RuleTypeAuthenticated // Require authentication but any role
)

// Rule represents an authorization rule for HTTP endpoints
// Đây là reference implementation của contracts.RuleInterface
// Ứng dụng bên ngoài có thể sử dụng model này hoặc implement RuleInterface với model của riêng họ
type Rule struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Method    string         `gorm:"not null;index:idx_method_path" json:"method"` // GET, POST, PUT, DELETE, etc.
	Path      string         `gorm:"not null;index:idx_method_path" json:"path"`   // URL path pattern
	Type      RuleType       `gorm:"type:varchar(20);not null" json:"type"`        // PUBLIC, ALLOW, FORBIDE, AUTHENTICATED
	Roles     []string       `gorm:"-" json:"roles"`                               // Not stored directly, use RolesJSON
	Priority  int            `gorm:"default:0" json:"priority"`                    // Higher priority rules are checked first
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Helper fields for GORM (stored as JSON in database)
	RolesJSON string `gorm:"type:text" json:"-"` // Internal storage
}

// BeforeCreate hook to generate UUID and serialize roles
func (r *Rule) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return r.serializeRoles()
}

// BeforeUpdate hook to serialize roles
func (r *Rule) BeforeUpdate(tx *gorm.DB) error {
	return r.serializeRoles()
}

// AfterFind hook to deserialize roles
func (r *Rule) AfterFind(tx *gorm.DB) error {
	return r.deserializeRoles()
}

// serializeRoles converts Roles slice to JSON string
func (r *Rule) serializeRoles() error {
	data, err := json.Marshal(r.Roles)
	if err != nil {
		return err
	}
	r.RolesJSON = string(data)
	return nil
}

// deserializeRoles converts JSON string to Roles slice
func (r *Rule) deserializeRoles() error {
	if r.RolesJSON == "" {
		r.Roles = []string{}
		return nil
	}
	return json.Unmarshal([]byte(r.RolesJSON), &r.Roles)
}

// TableName specifies the table name
func (Rule) TableName() string {
	return "rules"
}

// Implement contracts.RuleInterface

// GetID trả về ID của rule
func (r *Rule) GetID() uuid.UUID {
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
func (r *Rule) GetType() contracts.RuleType {
	return contracts.RuleType(r.Type)
}

// GetRoles trả về danh sách roles được áp dụng cho rule này
func (r *Rule) GetRoles() []string {
	return r.Roles
}

// GetPriority trả về priority của rule (priority cao hơn được check trước)
func (r *Rule) GetPriority() int {
	return r.Priority
}
