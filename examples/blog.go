package main

import (
	"time"

	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/utils"
	"gorm.io/gorm"
)

// Blog represents a blog post
// Đây là model riêng của ứng dụng, không thuộc authkit core
type Blog struct {
	ID        string         `gorm:"type:varchar(12);primary_key" json:"id"`
	Title     string         `gorm:"not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	AuthorID  string         `gorm:"type:varchar(12);not null;index" json:"author_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Author *authkit.User `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
}

// BeforeCreate hook to generate ID
func (b *Blog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == "" {
		id, err := utils.GenerateID()
		if err != nil {
			return err
		}
		b.ID = id
	}
	return nil
}

// TableName specifies the table name
func (Blog) TableName() string {
	return "blogs"
}
