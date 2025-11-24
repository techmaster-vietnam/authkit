package models

import (
	"time"

	"github.com/techmaster-vietnam/authkit/core"
	"gorm.io/gorm"
)

// BaseUser là base model cho User, có thể được embed vào custom User models
type BaseUser struct {
	ID        string         `gorm:"type:varchar(12);primary_key" json:"id"`
	Email     string         `gorm:"uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"not null" json:"-"` // Hidden from JSON
	FullName  string         `json:"full_name"`
	Active    bool           `gorm:"column:is_active;default:true" json:"is_active"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships - sử dụng BaseRole để có thể embed
	Roles []BaseRole `gorm:"many2many:user_roles;foreignKey:ID;joinForeignKey:user_id;References:ID;joinReferences:role_id" json:"roles,omitempty"`
}

// BeforeCreate hook to generate ID
func (u *BaseUser) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		id, err := core.GenerateID()
		if err != nil {
			return err
		}
		u.ID = id
	}
	return nil
}

// TableName specifies the table name
func (BaseUser) TableName() string {
	return "users"
}

// GetID trả về ID của user
func (u *BaseUser) GetID() string {
	return u.ID
}

// GetEmail trả về email của user
func (u *BaseUser) GetEmail() string {
	return u.Email
}

// SetEmail set email cho user
func (u *BaseUser) SetEmail(email string) {
	u.Email = email
}

// GetPassword trả về password đã hash của user
func (u *BaseUser) GetPassword() string {
	return u.Password
}

// SetPassword set password đã hash cho user
func (u *BaseUser) SetPassword(password string) {
	u.Password = password
}

// IsActive trả về trạng thái active của user
func (u *BaseUser) IsActive() bool {
	return u.Active
}

// SetActive set trạng thái active cho user
func (u *BaseUser) SetActive(active bool) {
	u.Active = active
}

// GetRoles trả về danh sách roles của user dưới dạng RoleInterface
func (u *BaseUser) GetRoles() []core.RoleInterface {
	roles := make([]core.RoleInterface, len(u.Roles))
	for i := range u.Roles {
		roles[i] = &u.Roles[i]
	}
	return roles
}

// GetFullName trả về full name của user
func (u *BaseUser) GetFullName() string {
	return u.FullName
}

// SetFullName set full name cho user
func (u *BaseUser) SetFullName(fullName string) {
	u.FullName = fullName
}

// User là alias cho BaseUser để backward compatibility
// Trong tương lai có thể deprecated
type User = BaseUser
