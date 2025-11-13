package main

import (
	"errors"

	"github.com/google/uuid"
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// BlogService handles blog business logic
type BlogService struct {
	blogRepo *BlogRepository
	userRepo *authkit.UserRepository
	roleRepo *authkit.RoleRepository
}

// NewBlogService creates a new blog service
func NewBlogService(blogRepo *BlogRepository, userRepo *authkit.UserRepository, roleRepo *authkit.RoleRepository) *BlogService {
	return &BlogService{
		blogRepo: blogRepo,
		userRepo: userRepo,
		roleRepo: roleRepo,
	}
}

// CreateBlogRequest represents create blog request
type CreateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// UpdateBlogRequest represents update blog request
type UpdateBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

// Create creates a new blog
func (s *BlogService) Create(authorID uuid.UUID, req CreateBlogRequest) (*Blog, error) {
	if req.Title == "" {
		return nil, goerrorkit.NewValidationError("Tiêu đề là bắt buộc", map[string]interface{}{
			"field": "title",
		})
	}
	if req.Content == "" {
		return nil, goerrorkit.NewValidationError("Nội dung là bắt buộc", map[string]interface{}{
			"field": "content",
		})
	}

	blog := &Blog{
		Title:    req.Title,
		Content:  req.Content,
		AuthorID: authorID,
	}

	if err := s.blogRepo.Create(blog); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo blog")
	}

	// Reload with author
	return s.blogRepo.GetByID(blog.ID)
}

// GetByID gets a blog by ID
func (s *BlogService) GetByID(id uuid.UUID) (*Blog, error) {
	blog, err := s.blogRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewBusinessError(404, "Không tìm thấy blog").WithData(map[string]interface{}{
				"blog_id": id,
			})
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy blog")
	}
	return blog, nil
}

// Update updates a blog (with permission check)
func (s *BlogService) Update(blogID uuid.UUID, userID uuid.UUID, userRoles []string, req UpdateBlogRequest) (*Blog, error) {
	blog, err := s.blogRepo.GetByID(blogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewBusinessError(404, "Không tìm thấy blog").WithData(map[string]interface{}{
				"blog_id": blogID,
			})
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy blog")
	}

	// Check permission: editor can edit any blog, author can only edit their own
	hasEditorRole := false
	for _, role := range userRoles {
		if role == "editor" || role == "admin" {
			hasEditorRole = true
			break
		}
	}

	if !hasEditorRole && blog.AuthorID != userID {
		return nil, goerrorkit.NewAuthError(403, "Bạn không có quyền sửa blog này").WithData(map[string]interface{}{
			"blog_id":   blogID,
			"author_id": blog.AuthorID,
			"user_id":   userID,
		})
	}

	if req.Title != "" {
		blog.Title = req.Title
	}
	if req.Content != "" {
		blog.Content = req.Content
	}

	if err := s.blogRepo.Update(blog); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật blog")
	}

	return s.blogRepo.GetByID(blog.ID)
}

// Delete soft deletes a blog (with permission check)
func (s *BlogService) Delete(blogID uuid.UUID, userID uuid.UUID, userRoles []string) error {
	blog, err := s.blogRepo.GetByID(blogID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy blog").WithData(map[string]interface{}{
				"blog_id": blogID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy blog")
	}

	// Check permission: editor can delete any blog, author can only delete their own
	hasEditorRole := false
	for _, role := range userRoles {
		if role == "editor" || role == "admin" {
			hasEditorRole = true
			break
		}
	}

	if !hasEditorRole && blog.AuthorID != userID {
		return goerrorkit.NewAuthError(403, "Bạn không có quyền xóa blog này").WithData(map[string]interface{}{
			"blog_id":   blogID,
			"author_id": blog.AuthorID,
			"user_id":   userID,
		})
	}

	if err := s.blogRepo.Delete(blogID); err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa blog")
	}

	return nil
}

// List lists all blogs with pagination
func (s *BlogService) List(offset, limit int) ([]Blog, int64, error) {
	blogs, total, err := s.blogRepo.List(offset, limit)
	if err != nil {
		return nil, 0, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách blog")
	}
	return blogs, total, nil
}

// ListByAuthor lists blogs by author ID
func (s *BlogService) ListByAuthor(authorID uuid.UUID, offset, limit int) ([]Blog, int64, error) {
	blogs, total, err := s.blogRepo.ListByAuthor(authorID, offset, limit)
	if err != nil {
		return nil, 0, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách blog của tác giả")
	}
	return blogs, total, nil
}
