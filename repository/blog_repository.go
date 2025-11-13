package repository

import (
	"github.com/google/uuid"
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// BlogRepository handles blog database operations
type BlogRepository struct {
	db *gorm.DB
}

// NewBlogRepository creates a new blog repository
func NewBlogRepository(db *gorm.DB) *BlogRepository {
	return &BlogRepository{db: db}
}

// Create creates a new blog
func (r *BlogRepository) Create(blog *models.Blog) error {
	return r.db.Create(blog).Error
}

// GetByID gets a blog by ID
func (r *BlogRepository) GetByID(id uuid.UUID) (*models.Blog, error) {
	var blog models.Blog
	err := r.db.Preload("Author").Where("id = ?", id).First(&blog).Error
	return &blog, err
}

// Update updates a blog
func (r *BlogRepository) Update(blog *models.Blog) error {
	return r.db.Save(blog).Error
}

// Delete soft deletes a blog
func (r *BlogRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Blog{}, id).Error
}

// List lists all blogs with pagination
func (r *BlogRepository) List(offset, limit int) ([]models.Blog, int64, error) {
	var blogs []models.Blog
	var total int64

	if err := r.db.Model(&models.Blog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Preload("Author").Order("created_at DESC").Offset(offset).Limit(limit).Find(&blogs).Error
	return blogs, total, err
}

// ListByAuthor lists blogs by author ID
func (r *BlogRepository) ListByAuthor(authorID uuid.UUID, offset, limit int) ([]models.Blog, int64, error) {
	var blogs []models.Blog
	var total int64

	if err := r.db.Model(&models.Blog{}).Where("author_id = ?", authorID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Preload("Author").Where("author_id = ?", authorID).Order("created_at DESC").Offset(offset).Limit(limit).Find(&blogs).Error
	return blogs, total, err
}

