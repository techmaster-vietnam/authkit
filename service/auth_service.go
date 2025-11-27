package service

import (
	"errors"
	"strings"

	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo *repository.UserRepository
	roleRepo *repository.RoleRepository
	config   *config.Config
}

// NewAuthService creates a new auth service
func NewAuthService(userRepo *repository.UserRepository, roleRepo *repository.RoleRepository, cfg *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		roleRepo: roleRepo,
		config:   cfg,
	}
}

// LoginRequest represents login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

// RegisterRequest represents registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// Login authenticates a user and returns a JWT token
func (s *AuthService) Login(req LoginRequest) (*LoginResponse, error) {
	if req.Email == "" {
		return nil, goerrorkit.NewValidationError("email là bắt buộc", map[string]interface{}{
			"field": "email",
		})
	}

	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewAuthError(401, "Email hoặc mật khẩu không đúng")
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi đăng nhập")
	}

	if !user.IsActive() {
		return nil, goerrorkit.NewAuthError(403, "Tài khoản đã bị vô hiệu hóa").WithData(map[string]interface{}{
			"user_id": user.ID,
		})
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, goerrorkit.NewAuthError(401, "Email hoặc mật khẩu không đúng")
	}

	// Get user roles - already loaded via Preload("Roles") in GetByEmail()
	// Roles will be protected by JWT signature - cannot be tampered
	userRoles := user.Roles

	// Extract role IDs
	roleIDs := make([]uint, 0, len(userRoles))
	for _, role := range userRoles {
		roleIDs = append(roleIDs, role.ID)
	}

	// Generate token with role IDs - protected by HMAC-SHA256 signature
	token, err := utils.GenerateToken(user.ID, user.Email, roleIDs, s.config.JWT.Secret, s.config.JWT.Expiration)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo token")
	}

	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

// Register creates a new user account
func (s *AuthService) Register(req RegisterRequest) (*models.User, error) {
	// Validate input
	if req.Email == "" {
		return nil, goerrorkit.NewValidationError("Email là bắt buộc", map[string]interface{}{
			"field": "email",
		})
	}
	if req.Password == "" || len(req.Password) < 6 {
		return nil, goerrorkit.NewValidationError("Mật khẩu phải có ít nhất 6 ký tự", map[string]interface{}{
			"field": "password",
			"min":   6,
		})
	}

	// Check if email already exists
	_, err := s.userRepo.GetByEmail(req.Email)
	if err == nil {
		return nil, goerrorkit.NewBusinessError(409, "Email đã tồn tại").WithData(map[string]interface{}{
			"email": req.Email,
		})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra email")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi hash mật khẩu")
	}

	// Create user
	user := &models.User{
		Email:    req.Email,
		Password: hashedPassword,
		FullName: req.FullName,
		Active:   true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo tài khoản")
	}

	return user, nil
}

// ChangePassword changes user password
func (s *AuthService) ChangePassword(userID string, oldPassword, newPassword string) error {
	if newPassword == "" || len(newPassword) < 6 {
		return goerrorkit.NewValidationError("Mật khẩu mới phải có ít nhất 6 ký tự", map[string]interface{}{
			"field": "new_password",
			"min":   6,
		})
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng").WithData(map[string]interface{}{
				"user_id": userID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	if !utils.CheckPasswordHash(oldPassword, user.Password) {
		return goerrorkit.NewAuthError(401, "Mật khẩu cũ không đúng")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi hash mật khẩu")
	}

	user.Password = hashedPassword
	if err := s.userRepo.Update(user); err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật mật khẩu")
	}

	return nil
}

// UpdateProfile updates user profile
func (s *AuthService) UpdateProfile(userID string, fullName string) (*models.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng").WithData(map[string]interface{}{
				"user_id": userID,
			})
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	user.FullName = fullName

	if err := s.userRepo.Update(user); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật profile")
	}

	return user, nil
}

// DeleteProfile soft deletes a user profile
func (s *AuthService) DeleteProfile(userID string) error {
	if err := s.userRepo.Delete(userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng").WithData(map[string]interface{}{
				"user_id": userID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa profile")
	}
	return nil
}

// UserDetailResponse represents user detail response với roles
type UserDetailResponse struct {
	User  *models.User  `json:"user"`
	Roles []models.Role `json:"roles"`
}

// GetUserDetail lấy thông tin chi tiết người dùng theo ID hoặc email
// identifier có thể là ID hoặc email
func (s *AuthService) GetUserDetail(identifier string) (*UserDetailResponse, error) {
	if identifier == "" {
		return nil, goerrorkit.NewValidationError("ID hoặc email là bắt buộc", map[string]interface{}{
			"field": "identifier",
		})
	}

	// Kiểm tra identifier là ID hay email dựa trên ký tự @
	// ID: chuỗi 12 ký tự chỉ gồm A-Za-z0-9 (không có @)
	// Email: chứa ký tự @
	var user *models.User
	var err error

	if strings.Contains(identifier, "@") {
		// Đây là email
		user, err = s.userRepo.GetByEmail(identifier)
	} else {
		// Đây là ID
		user, err = s.userRepo.GetByID(identifier)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng").WithData(map[string]interface{}{
				"identifier": identifier,
			})
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	// Lấy roles của user từ roleRepo
	roles, err := s.roleRepo.ListRolesOfUser(user.ID)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách roles")
	}

	// Đảm bảo roles không nil
	if roles == nil {
		roles = []models.Role{}
	}

	return &UserDetailResponse{
		User:  user,
		Roles: roles,
	}, nil
}
