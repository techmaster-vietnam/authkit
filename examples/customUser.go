package main

import (
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/core"
)

// CustomUser là custom User model với các trường bổ sung
// Embed BaseUser để kế thừa tất cả các trường và methods từ AuthKit
type CustomUser struct {
	authkit.BaseUser `gorm:"embedded"` // Embed BaseUser
	Mobile           string             `gorm:"type:varchar(15)" json:"mobile"`
	Address          string             `gorm:"type:varchar(200)" json:"address"`
}

// Implement UserInterface bằng cách delegate về BaseUser
// Các methods này đảm bảo CustomUser tương thích với AuthKit

func (u *CustomUser) GetID() string {
	return u.BaseUser.GetID()
}

func (u *CustomUser) GetEmail() string {
	return u.BaseUser.GetEmail()
}

func (u *CustomUser) SetEmail(email string) {
	u.BaseUser.SetEmail(email)
}

func (u *CustomUser) GetPassword() string {
	return u.BaseUser.GetPassword()
}

func (u *CustomUser) SetPassword(password string) {
	u.BaseUser.SetPassword(password)
}

func (u *CustomUser) IsActive() bool {
	return u.BaseUser.IsActive()
}

func (u *CustomUser) SetActive(active bool) {
	u.BaseUser.SetActive(active)
}

func (u *CustomUser) GetRoles() []core.RoleInterface {
	return u.BaseUser.GetRoles()
}

func (u *CustomUser) GetFullName() string {
	return u.BaseUser.GetFullName()
}

func (u *CustomUser) SetFullName(fullName string) {
	u.BaseUser.SetFullName(fullName)
}

// TableName specifies the table name (sử dụng cùng bảng users)
func (CustomUser) TableName() string {
	return "users"
}

