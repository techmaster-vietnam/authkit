package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/techmaster-vietnam/authkit"
	"gorm.io/gorm"
)

// Blog represents a blog post
// Đây là model riêng của ứng dụng, không thuộc authkit core
type Blog struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Title     string         `gorm:"not null" json:"title"`
	Content   string         `gorm:"type:text;not null" json:"content"`
	AuthorID  uuid.UUID      `gorm:"type:uuid;not null;index" json:"author_id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Author *authkit.User `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
}

// BeforeCreate hook to generate UUID
func (b *Blog) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// TableName specifies the table name
func (Blog) TableName() string {
	return "blogs"
}
