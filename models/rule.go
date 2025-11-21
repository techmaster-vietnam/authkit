package models

import (
	"database/sql/driver"
	"fmt"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// AccessType represents the type of rule
type AccessType string

const (
	AccessPublic AccessType = "PUBLIC"  // Allow anyone, including anonymous
	AccessAllow  AccessType = "ALLOW"   // Allow specific roles (empty roles = any authenticated user)
	AccessForbid AccessType = "FORBIDE" // Forbid specific roles
)

// IntArray is a custom type for PostgreSQL integer[] array
// It implements driver.Valuer and sql.Scanner for GORM compatibility
// Simpler than SmallIntArray - no conversion needed, just use pq.Array directly
type IntArray []uint

// Value implements driver.Valuer interface for GORM
func (a IntArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	// Convert []uint to []int32 for PostgreSQL integer[]
	int32Array := make([]int32, len(a))
	for i, v := range a {
		int32Array[i] = int32(v)
	}
	return pq.Array(int32Array).Value()
}

// Scan implements sql.Scanner interface for GORM
func (a *IntArray) Scan(value interface{}) error {
	if value == nil {
		*a = IntArray{}
		return nil
	}

	// Use pq.Array to scan into []int32
	var int32Array []int32
	if err := pq.Array(&int32Array).Scan(value); err != nil {
		return err
	}

	// Convert []int32 to []uint
	result := make([]uint, len(int32Array))
	for i, v := range int32Array {
		result[i] = uint(v)
	}
	*a = IntArray(result)
	return nil
}

// ToUintSlice converts IntArray to []uint
func (a IntArray) ToUintSlice() []uint {
	return []uint(a)
}

// FromUintSlice creates IntArray from []uint
func FromUintSlice(slice []uint) IntArray {
	return IntArray(slice)
}

// Rule represents an authorization rule for HTTP endpoints
// ID format: "METHOD|PATH" (ví dụ: "GET|/api/users")
type Rule struct {
	ID          string     `gorm:"primaryKey" json:"id"`                               // Format: "METHOD|PATH"
	Method      string     `gorm:"not null" json:"method"`                              // GET, POST, PUT, DELETE, etc.
	Path        string     `gorm:"not null" json:"path"`                                // URL path pattern
	Type        AccessType `gorm:"type:varchar(20);not null" json:"type"`              // PUBLIC, ALLOW, FORBIDE
	Roles       IntArray   `gorm:"type:integer[]" json:"roles"`                        // Stored as PostgreSQL array of role IDs
	Fixed       bool       `gorm:"default:false" json:"fixed"`                         // Fixed=true: rule từ code, không thể sửa từ DB
	Description string     `gorm:"type:text" json:"description"`                       // Mô tả rule
	ServiceName string     `gorm:"type:varchar(20);index" json:"service_name"`        // Service name for microservice architecture (max 20 chars)
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

// GetType trả về loại rule
func (r *Rule) GetType() AccessType {
	return r.Type
}

// GetRoles trả về danh sách role IDs được áp dụng cho rule này
func (r *Rule) GetRoles() []uint {
	return r.Roles.ToUintSlice()
}
