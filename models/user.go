package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/techmaster-vietnam/authkit/contracts"
	"gorm.io/gorm"
)

// User represents a user in the system
// Đây là reference implementation của contracts.UserInterface
// Ứng dụng bên ngoài có thể sử dụng model này hoặc implement UserInterface với model của riêng họ
type User struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Username  string         `gorm:"uniqueIndex;not null" json:"username"`
	Password  string         `gorm:"not null" json:"-"` // Hidden from JSON
	FirstName string         `json:"first_name"`
	LastName  string         `json:"last_name"`
	Active    bool           `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Roles []Role `gorm:"many2many:user_roles;" json:"roles,omitempty"`
}

// BeforeCreate hook to generate UUID
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (User) TableName() string {
	return "users"
}

// Implement contracts.UserInterface

// GetID trả về ID của user
func (u *User) GetID() uuid.UUID {
	return u.ID
}

// GetEmail trả về email của user
func (u *User) GetEmail() string {
	return u.Email
}

// GetUsername trả về username của user
func (u *User) GetUsername() string {
	return u.Username
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
func (u *User) GetRoles() []contracts.RoleInterface {
	roles := make([]contracts.RoleInterface, len(u.Roles))
	for i := range u.Roles {
		roles[i] = &u.Roles[i]
	}
	return roles
}
