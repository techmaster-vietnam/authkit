package service

import (
	"errors"

	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// BaseAuthService là generic auth service
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseAuthService[TUser core.UserInterface, TRole core.RoleInterface] struct {
	userRepo *repository.BaseUserRepository[TUser]
	roleRepo *repository.BaseRoleRepository[TRole]
	config   *config.Config
}

// NewBaseAuthService tạo mới BaseAuthService với generic types
func NewBaseAuthService[TUser core.UserInterface, TRole core.RoleInterface](
	userRepo *repository.BaseUserRepository[TUser],
	roleRepo *repository.BaseRoleRepository[TRole],
	cfg *config.Config,
) *BaseAuthService[TUser, TRole] {
	return &BaseAuthService[TUser, TRole]{
		userRepo: userRepo,
		roleRepo: roleRepo,
		config:   cfg,
	}
}

// BaseLoginRequest represents login request
type BaseLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// BaseLoginResponse represents login response với generic User type
type BaseLoginResponse[TUser any] struct {
	Token string `json:"token"`
	User  TUser  `json:"user"`
}

// Login authenticates a user and returns a JWT token
func (s *BaseAuthService[TUser, TRole]) Login(req BaseLoginRequest) (*BaseLoginResponse[TUser], error) {
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
			"user_id": user.GetID(),
		})
	}

	if !utils.CheckPasswordHash(req.Password, user.GetPassword()) {
		return nil, goerrorkit.NewAuthError(401, "Email hoặc mật khẩu không đúng")
	}

	// Get user roles
	userRoles := user.GetRoles()

	// Extract role IDs
	roleIDs := make([]uint, 0, len(userRoles))
	for _, role := range userRoles {
		roleIDs = append(roleIDs, role.GetID())
	}

	// Generate token with role IDs
	token, err := utils.GenerateToken(user.GetID(), user.GetEmail(), roleIDs, s.config.JWT.Secret, s.config.JWT.Expiration)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo token")
	}

	return &BaseLoginResponse[TUser]{
		Token: token,
		User:  user,
	}, nil
}

// BaseRegisterRequest represents registration request
type BaseRegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
}

// Register creates a new user account
// Note: Method này yêu cầu TUser có thể được tạo từ zero value và set các fields
// Với embedded structs, điều này sẽ hoạt động tốt
func (s *BaseAuthService[TUser, TRole]) Register(req BaseRegisterRequest) (TUser, error) {
	var zero TUser

	// Validate input
	if req.Email == "" {
		return zero, goerrorkit.NewValidationError("Email là bắt buộc", map[string]interface{}{
			"field": "email",
		})
	}
	if req.Password == "" || len(req.Password) < 6 {
		return zero, goerrorkit.NewValidationError("Mật khẩu phải có ít nhất 6 ký tự", map[string]interface{}{
			"field": "password",
			"min":   6,
		})
	}

	// Check if email already exists
	_, err := s.userRepo.GetByEmail(req.Email)
	if err == nil {
		return zero, goerrorkit.NewBusinessError(409, "Email đã tồn tại").WithData(map[string]interface{}{
			"email": req.Email,
		})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra email")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi hash mật khẩu")
	}

	// Create user - sử dụng interface methods
	// Với embedded structs, TUser sẽ có các methods từ BaseUser
	user := zero
	user.SetEmail(req.Email)
	user.SetPassword(hashedPassword)
	user.SetFullName(req.FullName)
	user.SetActive(true)

	if err := s.userRepo.Create(user); err != nil {
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo tài khoản")
	}

	return user, nil
}

// ChangePassword changes user password
func (s *BaseAuthService[TUser, TRole]) ChangePassword(userID string, oldPassword, newPassword string) error {
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

	if !utils.CheckPasswordHash(oldPassword, user.GetPassword()) {
		return goerrorkit.NewAuthError(401, "Mật khẩu cũ không đúng")
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi hash mật khẩu")
	}

	user.SetPassword(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật mật khẩu")
	}

	return nil
}

// UpdateProfile updates user profile
func (s *BaseAuthService[TUser, TRole]) UpdateProfile(userID string, fullName string) (TUser, error) {
	var zero TUser

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return zero, goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng").WithData(map[string]interface{}{
				"user_id": userID,
			})
		}
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	// Update full name sử dụng interface method
	user.SetFullName(fullName)

	if err := s.userRepo.Update(user); err != nil {
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật profile")
	}

	return user, nil
}

// DeleteProfile soft deletes a user profile
func (s *BaseAuthService[TUser, TRole]) DeleteProfile(userID string) error {
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

