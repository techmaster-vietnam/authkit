package models

import (
	"time"

	"github.com/techmaster-vietnam/authkit/utils"
	"gorm.io/gorm"
)

// User represents a user in the system
type User struct {
	ID        string         `gorm:"type:varchar(12);primary_key" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"not null" json:"-"` // Hidden from JSON
	FullName  string         `json:"full_name"`
	Active    bool           `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Roles []Role `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// BeforeCreate hook to generate ID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		id, err := utils.GenerateID()
		if err != nil {
			return err
		}
		u.ID = id
	}
	return nil
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

// GetID trả về ID của user
func (u *User) GetID() string {
	return u.ID
}

// GetEmail trả về email của user
func (u *User) GetEmail() string {
	return u.Email
}

// GetPassword trả về password đã hash của user
func (u *User) GetPassword() string {
	return u.Password
}

// SetPassword set password đã hash cho user
func (u *User) SetPassword(password string) {
	u.Password = password
}

// IsActive trả về trạng thái active của user
func (u *User) IsActive() bool {
	return u.Active
}

// SetActive set trạng thái active cho user
func (u *User) SetActive(active bool) {
	u.Active = active
}

// GetRoles trả về danh sách roles của user
func (u *User) GetRoles() []Role {
	return u.Roles
}
