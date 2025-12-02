package service

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// JWTCustomizer là callback function để tùy chỉnh JWT claims
// Function này được gọi khi tạo JWT token trong quá trình login
// user: User object đang đăng nhập
// roleIDs: Danh sách role IDs của user
// Returns: ClaimsConfig với custom fields để thêm vào JWT token
type JWTCustomizer[TUser core.UserInterface] func(user TUser, roleIDs []uint) utils.ClaimsConfig

// BaseAuthService là generic auth service
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseAuthService[TUser core.UserInterface, TRole core.RoleInterface] struct {
	userRepo               *repository.BaseUserRepository[TUser]
	roleRepo               *repository.BaseRoleRepository[TRole]
	refreshTokenRepo       *repository.RefreshTokenRepository
	passwordResetTokenRepo *repository.PasswordResetTokenRepository
	config                 *config.Config
	jwtCustomizer          JWTCustomizer[TUser]
	notificationSender     core.NotificationSender // Callback để gửi email/tin nhắn
}

// NewBaseAuthService tạo mới BaseAuthService với generic types
func NewBaseAuthService[TUser core.UserInterface, TRole core.RoleInterface](
	userRepo *repository.BaseUserRepository[TUser],
	roleRepo *repository.BaseRoleRepository[TRole],
	refreshTokenRepo *repository.RefreshTokenRepository,
	passwordResetTokenRepo *repository.PasswordResetTokenRepository,
	cfg *config.Config,
) *BaseAuthService[TUser, TRole] {
	return &BaseAuthService[TUser, TRole]{
		userRepo:               userRepo,
		roleRepo:               roleRepo,
		refreshTokenRepo:       refreshTokenRepo,
		passwordResetTokenRepo: passwordResetTokenRepo,
		config:                 cfg,
		jwtCustomizer:          nil,
		notificationSender:     nil,
	}
}

// SetJWTCustomizer set JWT customizer callback để tùy chỉnh JWT claims
func (s *BaseAuthService[TUser, TRole]) SetJWTCustomizer(customizer JWTCustomizer[TUser]) {
	s.jwtCustomizer = customizer
}

// SetNotificationSender set notification sender callback để gửi email/tin nhắn
// Người dùng phải implement core.NotificationSender interface
func (s *BaseAuthService[TUser, TRole]) SetNotificationSender(sender core.NotificationSender) {
	s.notificationSender = sender
}

// GetConfig trả về config của service (để handler có thể truy cập)
func (s *BaseAuthService[TUser, TRole]) GetConfig() *config.Config {
	return s.config
}

// BaseLoginRequest represents login request
type BaseLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// BaseLoginResponse represents login response với generic User type
type BaseLoginResponse[TUser any] struct {
	Token        string `json:"token"` // Access token (JWT)
	RefreshToken string `json:"-"`     // Refresh token (không trả về JSON, chỉ set cookie)
	User         TUser  `json:"user"`
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

	// Generate access token với custom claims nếu có JWT customizer
	var token string
	if s.jwtCustomizer != nil {
		// Sử dụng JWT customizer để tạo claims config
		claimsConfig := s.jwtCustomizer(user, roleIDs)
		// Đảm bảo roleIDs được set trong claims config
		claimsConfig.RoleIDs = roleIDs
		claimsConfig.RoleFormat = "ids"

		// Generate token với flexible claims
		var err error
		token, err = utils.GenerateTokenFlexible(
			user.GetID(),
			user.GetEmail(),
			claimsConfig,
			s.config.JWT.Secret,
			s.config.JWT.Expiration,
		)
		if err != nil {
			return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo token")
		}
	} else {
		// Generate token với standard claims (backward compatible)
		var err error
		token, err = utils.GenerateToken(user.GetID(), user.GetEmail(), roleIDs, s.config.JWT.Secret, s.config.JWT.Expiration)
		if err != nil {
			return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo token")
		}
	}

	// Generate refresh token
	refreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo refresh token")
	}

	// Lưu refresh token vào database
	expiresAt := time.Now().Add(s.config.JWT.RefreshExpiration)
	_, err = s.refreshTokenRepo.Create(refreshToken, user.GetID(), expiresAt)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lưu refresh token")
	}

	return &BaseLoginResponse[TUser]{
		Token:        token,
		RefreshToken: refreshToken,
		User:         user,
	}, nil
}

// BaseRegisterRequest represents registration request
type BaseRegisterRequest struct {
	Email        string                 `json:"email"`
	Password     string                 `json:"password"`
	FullName     string                 `json:"full_name"`
	CustomFields map[string]interface{} `json:"-"` // Các trường custom sẽ được set thông qua reflection
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
	// Xử lý cả pointer types và value types
	var user TUser
	userType := reflect.TypeOf(zero)
	if userType.Kind() == reflect.Ptr {
		// Nếu là pointer type (ví dụ: *CustomUser), tạo instance mới
		elemType := userType.Elem()
		newValue := reflect.New(elemType)
		user = newValue.Interface().(TUser)
	} else {
		// Nếu là value type, dùng zero value
		user = zero
	}

	user.SetEmail(req.Email)
	user.SetPassword(hashedPassword)
	user.SetFullName(req.FullName)
	user.SetActive(true)

	// Set các trường custom bằng reflection nếu có
	if len(req.CustomFields) > 0 {
		if err := s.setCustomFields(&user, req.CustomFields); err != nil {
			return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi set các trường custom")
		}
	}

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
func (s *BaseAuthService[TUser, TRole]) UpdateProfile(userID string, fullName string, customFields map[string]interface{}) (TUser, error) {
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
	if fullName != "" {
		user.SetFullName(fullName)
	}

	// Set các trường custom bằng reflection nếu có
	if len(customFields) > 0 {
		if err := s.setCustomFields(&user, customFields); err != nil {
			return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi set các trường custom")
		}
	}

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

// DeleteUserByID xóa user theo ID với logic phân quyền
// - admin: chỉ xóa các user không có role "admin" và "super_admin". Xóa là soft delete
// - super_admin: xóa bất kỳ user nào không chứa role "super_admin". Xóa là hard delete
func (s *BaseAuthService[TUser, TRole]) DeleteUserByID(currentUserID string, targetUserID string) error {
	// Lấy thông tin current user
	currentUser, err := s.userRepo.GetByID(currentUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng hiện tại").WithData(map[string]interface{}{
				"user_id": currentUserID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng hiện tại")
	}

	// Lấy roles của current user
	currentUserRoles := currentUser.GetRoles()
	currentRoleIDs := make([]uint, 0, len(currentUserRoles))
	for _, role := range currentUserRoles {
		currentRoleIDs = append(currentRoleIDs, role.GetID())
	}

	// Lấy role names từ role IDs
	currentRoles, err := s.roleRepo.GetByIDs(currentRoleIDs)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin roles")
	}

	currentRoleNames := make(map[string]bool)
	for _, role := range currentRoles {
		currentRoleNames[role.GetName()] = true
	}

	isSuperAdmin := currentRoleNames["super_admin"]
	isAdmin := currentRoleNames["admin"]

	if !isSuperAdmin && !isAdmin {
		return goerrorkit.NewAuthError(403, "Chỉ admin và super_admin mới có quyền xóa user")
	}

	// Lấy thông tin target user
	targetUser, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng cần xóa").WithData(map[string]interface{}{
				"user_id": targetUserID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng cần xóa")
	}

	// Lấy roles của target user
	targetUserRoles := targetUser.GetRoles()
	targetRoleIDs := make([]uint, 0, len(targetUserRoles))
	for _, role := range targetUserRoles {
		targetRoleIDs = append(targetRoleIDs, role.GetID())
	}

	// Lấy role names từ role IDs
	targetRoles, err := s.roleRepo.GetByIDs(targetRoleIDs)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin roles của target user")
	}

	targetRoleNames := make(map[string]bool)
	for _, role := range targetRoles {
		targetRoleNames[role.GetName()] = true
	}

	// Kiểm tra phân quyền và thực hiện xóa
	if isSuperAdmin {
		// super_admin: xóa bất kỳ user nào không chứa role "super_admin"
		if targetRoleNames["super_admin"] {
			return goerrorkit.NewAuthError(403, "Super_admin không được phép xóa user có role 'super_admin'").WithData(map[string]interface{}{
				"target_user_id": targetUserID,
			})
		}
		// Hard delete
		if err := s.userRepo.HardDelete(targetUserID); err != nil {
			return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa user")
		}
	} else if isAdmin {
		// admin: chỉ xóa các user không có role "admin" và "super_admin"
		if targetRoleNames["admin"] || targetRoleNames["super_admin"] {
			return goerrorkit.NewAuthError(403, "Admin không được phép xóa user có role 'admin' hoặc 'super_admin'").WithData(map[string]interface{}{
				"target_user_id": targetUserID,
			})
		}
		// Soft delete
		if err := s.userRepo.Delete(targetUserID); err != nil {
			return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa user")
		}
	}

	return nil
}

// BaseUserDetailResponse represents user detail response với roles
type BaseUserDetailResponse[TUser core.UserInterface, TRole core.RoleInterface] struct {
	User  TUser   `json:"user"`
	Roles []TRole `json:"roles"`
}

// GetUserDetail lấy thông tin chi tiết người dùng theo ID hoặc email
// identifier có thể là ID hoặc email
func (s *BaseAuthService[TUser, TRole]) GetUserDetail(identifier string) (*BaseUserDetailResponse[TUser, TRole], error) {
	if identifier == "" {
		return nil, goerrorkit.NewValidationError("ID hoặc email là bắt buộc", map[string]interface{}{
			"field": "identifier",
		})
	}

	// Kiểm tra identifier là ID hay email dựa trên ký tự @
	// ID: chuỗi 12 ký tự chỉ gồm A-Za-z0-9 (không có @)
	// Email: chứa ký tự @
	var user TUser
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

	// Lấy roles của user từ roleRepo để đảm bảo type đúng
	userID := user.GetID()
	roles, err := s.roleRepo.ListRolesOfUser(userID)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách roles")
	}

	// Đảm bảo roles không nil
	if roles == nil {
		roles = []TRole{}
	}

	return &BaseUserDetailResponse[TUser, TRole]{
		User:  user,
		Roles: roles,
	}, nil
}

// GetUserByID lấy thông tin user theo ID
func (s *BaseAuthService[TUser, TRole]) GetUserByID(userID string) (TUser, error) {
	var zero TUser
	if userID == "" {
		return zero, goerrorkit.NewValidationError("ID người dùng là bắt buộc", map[string]interface{}{
			"field": "user_id",
		})
	}

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return zero, goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng").WithData(map[string]interface{}{
				"user_id": userID,
			})
		}
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	return user, nil
}

// GetUserByIdentifier lấy thông tin user theo identifier (id, email, hoặc mobile)
// Tự động phát hiện loại identifier dựa trên format:
// - Email: chứa ký tự @
// - Mobile: chỉ chứa số (0-9) và có độ dài từ 10-15 ký tự
// - ID: các trường hợp còn lại (thường là chuỗi 12 ký tự A-Za-z0-9)
func (s *BaseAuthService[TUser, TRole]) GetUserByIdentifier(identifier string) (TUser, error) {
	var zero TUser
	if identifier == "" {
		return zero, goerrorkit.NewValidationError("Identifier là bắt buộc (id, email, hoặc mobile)", map[string]interface{}{
			"field": "identifier",
		})
	}

	var user TUser
	var err error

	// Kiểm tra identifier là email (chứa ký tự @)
	if strings.Contains(identifier, "@") {
		user, err = s.userRepo.GetByEmail(identifier)
	} else if utils.IsMobile(identifier) {
		// Kiểm tra identifier là mobile (chỉ chứa số, độ dài 10-15)
		user, err = s.userRepo.GetByMobile(identifier)
	} else {
		// Các trường hợp còn lại coi như ID
		user, err = s.userRepo.GetByID(identifier)
	}

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return zero, goerrorkit.NewBusinessError(404, "Không tìm thấy người dùng").WithData(map[string]interface{}{
				"identifier": identifier,
			})
		}
		return zero, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	return user, nil
}

// BaseRefreshResponse represents refresh token response
type BaseRefreshResponse struct {
	Token        string `json:"token"` // New access token
	RefreshToken string `json:"-"`     // New refresh token (không trả về JSON, chỉ set cookie)
}

// Refresh tạo access token mới từ refresh token
func (s *BaseAuthService[TUser, TRole]) Refresh(refreshToken string) (*BaseRefreshResponse, error) {
	// Tìm refresh token trong database
	tokenRecord, err := s.refreshTokenRepo.GetByToken(refreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewAuthError(401, "Refresh token không hợp lệ")
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra refresh token")
	}

	// Kiểm tra token đã hết hạn chưa
	if tokenRecord.IsExpired() {
		// Xóa token đã hết hạn
		_ = s.refreshTokenRepo.DeleteByToken(refreshToken)
		return nil, goerrorkit.NewAuthError(401, "Refresh token đã hết hạn")
	}

	// Lấy user từ token
	user, err := s.userRepo.GetByID(tokenRecord.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewAuthError(401, "Người dùng không tồn tại")
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	// Kiểm tra user còn active không
	if !user.IsActive() {
		// Xóa tất cả refresh tokens của user này
		_ = s.refreshTokenRepo.DeleteByUserID(user.GetID())
		return nil, goerrorkit.NewAuthError(403, "Tài khoản đã bị vô hiệu hóa")
	}

	// Lấy roles của user
	userRoles := user.GetRoles()
	roleIDs := make([]uint, 0, len(userRoles))
	for _, role := range userRoles {
		roleIDs = append(roleIDs, role.GetID())
	}

	// Tạo access token mới
	var newAccessToken string
	if s.jwtCustomizer != nil {
		claimsConfig := s.jwtCustomizer(user, roleIDs)
		claimsConfig.RoleIDs = roleIDs
		claimsConfig.RoleFormat = "ids"

		newAccessToken, err = utils.GenerateTokenFlexible(
			user.GetID(),
			user.GetEmail(),
			claimsConfig,
			s.config.JWT.Secret,
			s.config.JWT.Expiration,
		)
		if err != nil {
			return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo access token")
		}
	} else {
		newAccessToken, err = utils.GenerateToken(user.GetID(), user.GetEmail(), roleIDs, s.config.JWT.Secret, s.config.JWT.Expiration)
		if err != nil {
			return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo access token")
		}
	}

	// Tạo refresh token mới (rotation - tăng cường bảo mật)
	newRefreshToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo refresh token mới")
	}

	// Xóa refresh token cũ và lưu token mới
	_ = s.refreshTokenRepo.DeleteByToken(refreshToken)
	expiresAt := time.Now().Add(s.config.JWT.RefreshExpiration)
	_, err = s.refreshTokenRepo.Create(newRefreshToken, user.GetID(), expiresAt)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lưu refresh token mới")
	}

	return &BaseRefreshResponse{
		Token:        newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// Logout xóa refresh token của user
func (s *BaseAuthService[TUser, TRole]) Logout(refreshToken string) error {
	if refreshToken == "" {
		return nil // Không có token thì không cần làm gì
	}
	return s.refreshTokenRepo.DeleteByToken(refreshToken)
}

// setCustomFields sử dụng reflection để set các trường custom vào user object
// Chỉ set các trường có thể set được (exported fields, không phải các trường đặc biệt)
func (s *BaseAuthService[TUser, TRole]) setCustomFields(user *TUser, customFields map[string]interface{}) error {
	userValue := reflect.ValueOf(user)

	// Unwrap pointer đệ quy cho đến khi gặp struct type
	// Xử lý trường hợp TUser là pointer type (ví dụ: *CustomUser)
	// Khi đó user là *TUser = **CustomUser, cần unwrap 2 lần
	for userValue.Kind() == reflect.Ptr {
		if userValue.IsNil() {
			return goerrorkit.NewSystemError(fmt.Errorf("user pointer is nil"))
		}
		userValue = userValue.Elem()
	}

	// Kiểm tra sau khi unwrap phải là struct
	if userValue.Kind() != reflect.Struct {
		return goerrorkit.NewSystemError(fmt.Errorf("user type is not a struct after unwrapping pointers: %v", userValue.Kind()))
	}

	userType := userValue.Type()

	// Danh sách các trường không được phép set trực tiếp
	protectedFields := map[string]bool{
		"ID":        true,
		"CreatedAt": true,
		"UpdatedAt": true,
		"DeletedAt": true,
		"Roles":     true,
		"Email":     true, // Email được set qua SetEmail()
		"Password":  true, // Password được set qua SetPassword()
		"FullName":  true, // FullName được set qua SetFullName()
		"Active":    true, // Active được set qua SetActive()
	}

	for fieldName, fieldValue := range customFields {
		// Tìm field trong struct
		// Hỗ trợ cả snake_case (mobile) và PascalCase (Mobile)
		var field reflect.StructField
		var found bool

		// Convert snake_case sang PascalCase
		// Ví dụ: "mobile" -> "Mobile", "full_name" -> "FullName"
		pascalCaseName := s.snakeToPascalCase(fieldName)

		// Tìm field với tên chính xác (case-sensitive) - thử cả tên gốc và PascalCase
		if f, ok := userType.FieldByName(fieldName); ok {
			field = f
			found = true
		} else if f, ok := userType.FieldByName(pascalCaseName); ok {
			field = f
			found = true
		} else {
			// Tìm field với tên không phân biệt hoa thường
			for i := 0; i < userType.NumField(); i++ {
				f := userType.Field(i)
				if strings.EqualFold(f.Name, fieldName) || strings.EqualFold(f.Name, pascalCaseName) {
					field = f
					found = true
					break
				}
			}
		}

		if !found {
			continue // Bỏ qua các trường không tồn tại trong struct
		}

		// Kiểm tra field có được phép set không
		if protectedFields[field.Name] {
			continue // Bỏ qua các trường được bảo vệ
		}

		// Kiểm tra field có thể set được không (exported và settable)
		fieldValueReflect := userValue.FieldByName(field.Name)
		if !fieldValueReflect.IsValid() || !fieldValueReflect.CanSet() {
			continue
		}

		// Convert giá trị từ interface{} sang type của field
		valueReflect := reflect.ValueOf(fieldValue)
		if !valueReflect.IsValid() {
			continue
		}

		// Kiểm tra type compatibility
		fieldType := fieldValueReflect.Type()
		valueType := valueReflect.Type()

		// Nếu type khớp trực tiếp, set ngay
		if valueType.AssignableTo(fieldType) {
			fieldValueReflect.Set(valueReflect)
			continue
		}

		// Nếu type không khớp, thử convert
		if valueReflect.CanConvert(fieldType) {
			convertedValue := valueReflect.Convert(fieldType)
			fieldValueReflect.Set(convertedValue)
			continue
		}

		// Xử lý trường hợp đặc biệt: convert từ float64 (JSON number) sang string
		if valueType.Kind() == reflect.Float64 && fieldType.Kind() == reflect.String {
			// JSON numbers được parse thành float64, convert sang string
			floatValue := valueReflect.Float()
			// Kiểm tra xem có phải là số nguyên không
			if floatValue == float64(int64(floatValue)) {
				fieldValueReflect.SetString(fmt.Sprintf("%.0f", floatValue))
			} else {
				fieldValueReflect.SetString(fmt.Sprintf("%g", floatValue))
			}
			continue
		}

		// Xử lý trường hợp string sang string (đảm bảo type safety)
		if valueType.Kind() == reflect.String && fieldType.Kind() == reflect.String {
			fieldValueReflect.SetString(valueReflect.String())
			continue
		}
	}

	return nil
}

// snakeToPascalCase converts snake_case to PascalCase
// Ví dụ: "mobile" -> "Mobile", "full_name" -> "FullName"
func (s *BaseAuthService[TUser, TRole]) snakeToPascalCase(snake string) string {
	if snake == "" {
		return snake
	}

	parts := strings.Split(snake, "_")
	result := ""
	for _, part := range parts {
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return result
}

// RequestPasswordReset tạo password reset token và gửi qua email/tin nhắn
// email: Email của user cần reset password
// Returns: error nếu không tìm thấy user hoặc không thể gửi notification
func (s *BaseAuthService[TUser, TRole]) RequestPasswordReset(email string) error {
	if email == "" {
		return goerrorkit.NewValidationError("Email là bắt buộc", map[string]interface{}{
			"field": "email",
		})
	}

	// Validate email format
	if err := utils.ValidateEmail(email); err != nil {
		return err
	}

	// Tìm user theo email
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Không trả về lỗi chi tiết để tránh email enumeration attack
			// Luôn trả về success message ngay cả khi email không tồn tại
			return nil
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi tìm kiếm người dùng")
	}

	// Kiểm tra user có active không
	if !user.IsActive() {
		// Không trả về lỗi chi tiết để tránh thông tin leak
		return nil
	}

	// Tạo reset token (tương tự refresh token)
	resetToken, err := utils.GenerateRefreshToken()
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi tạo reset token")
	}

	// Token hết hạn sau 1 giờ (có thể config)
	tokenExpiration := 1 * time.Hour
	expiresAt := time.Now().Add(tokenExpiration)

	// Lưu token vào database
	_, err = s.passwordResetTokenRepo.Create(resetToken, user.GetID(), expiresAt)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lưu reset token")
	}

	// Gửi token qua email/tin nhắn nếu có notification sender
	if s.notificationSender != nil {
		expiresIn := "1 giờ"
		if err := s.notificationSender.SendPasswordResetToken(email, resetToken, expiresIn); err != nil {
			// Log error nhưng không fail request (token đã được lưu)
			// Người dùng có thể request lại nếu cần
			return goerrorkit.WrapWithMessage(err, "Lỗi khi gửi email/tin nhắn reset password")
		}
	}

	return nil
}

// ResetPassword reset password của user bằng reset token
// token: Reset token nhận được từ email/tin nhắn
// newPassword: Mật khẩu mới
// Returns: error nếu token không hợp lệ hoặc password không đúng format
func (s *BaseAuthService[TUser, TRole]) ResetPassword(token string, newPassword string) error {
	if token == "" {
		return goerrorkit.NewValidationError("Reset token là bắt buộc", map[string]interface{}{
			"field": "token",
		})
	}

	if newPassword == "" || len(newPassword) < 6 {
		return goerrorkit.NewValidationError("Mật khẩu mới phải có ít nhất 6 ký tự", map[string]interface{}{
			"field": "new_password",
			"min":   6,
		})
	}

	// Validate password format theo config
	if err := utils.ValidatePassword(newPassword, s.config.Password); err != nil {
		return err
	}

	// Tìm token trong database
	tokenRecord, err := s.passwordResetTokenRepo.GetByToken(token)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewAuthError(401, "Reset token không hợp lệ hoặc đã hết hạn")
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra reset token")
	}

	// Kiểm tra token có hợp lệ không (chưa hết hạn và chưa được sử dụng)
	if !tokenRecord.IsValid() {
		if tokenRecord.IsExpired() {
			return goerrorkit.NewAuthError(401, "Reset token đã hết hạn")
		}
		if tokenRecord.Used {
			return goerrorkit.NewAuthError(401, "Reset token đã được sử dụng")
		}
		return goerrorkit.NewAuthError(401, "Reset token không hợp lệ")
	}

	// Lấy user từ token
	user, err := s.userRepo.GetByID(tokenRecord.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Người dùng không tồn tại").WithData(map[string]interface{}{
				"user_id": tokenRecord.UserID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin người dùng")
	}

	// Hash password mới
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi hash mật khẩu")
	}

	// Cập nhật password
	user.SetPassword(hashedPassword)
	if err := s.userRepo.Update(user); err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật mật khẩu")
	}

	// Đánh dấu token đã được sử dụng
	if err := s.passwordResetTokenRepo.MarkAsUsed(token); err != nil {
		// Log error nhưng không fail request (password đã được đổi)
		_ = err
	}

	// Invalidate tất cả reset tokens khác của user này (bảo mật)
	_ = s.passwordResetTokenRepo.InvalidateUserTokens(user.GetID())

	// Xóa tất cả refresh tokens của user (force logout tất cả sessions)
	_ = s.refreshTokenRepo.DeleteByUserID(user.GetID())

	return nil
}

// BaseListUsersResponse represents paginated list users response
// Hỗ trợ cả trường hợp có và không có pagination
type BaseListUsersResponse[TUser core.UserInterface] struct {
	Users             []TUser            `json:"users"`
	RoleNames         map[string][]string `json:"role_names,omitempty"` // Map userID -> []roleName
	Total             int64              `json:"total"`
	Page              *int               `json:"page,omitempty"`        // nil khi không dùng pagination
	PageSize          *int               `json:"page_size,omitempty"`   // nil khi không dùng pagination
	TotalPages        *int               `json:"total_pages,omitempty"` // nil khi không dùng pagination
	PaginationEnabled bool               `json:"pagination_enabled"`    // true nếu đang dùng pagination
}

// ListUsersOptions chứa các tùy chọn cho ListUsers
type ListUsersOptions struct {
	Page                int    // Số trang (bắt đầu từ 1)
	PageSize            int    // Số lượng items mỗi trang
	EnablePagination     *bool  // nil = auto, true = bật, false = tắt
	PaginationThreshold int    // Ngưỡng để tự động bật pagination (mặc định 100)
	MaxPageSize         int    // Giới hạn tối đa page_size khi có pagination (mặc định 100)
	SortBy              string // Trường để sort (email, full_name, hoặc custom field)
	Order               string // Thứ tự sort: "asc" hoặc "desc" (mặc định "asc")
}

// ListUsers lấy danh sách users với pagination và filter linh hoạt
// Hỗ trợ:
// - Tự động bật pagination khi số lượng user > threshold
// - Tự động tắt pagination khi số lượng user <= threshold
// - Cho phép client bật/tắt pagination thủ công
// - Tham số hóa được số bản ghi trả về trong một page
func (s *BaseAuthService[TUser, TRole]) ListUsers(page, pageSize int, filter interface{}) (*BaseListUsersResponse[TUser], error) {
	// Sử dụng options với giá trị mặc định
	options := ListUsersOptions{
		Page:                page,
		PageSize:            pageSize,
		EnablePagination:    nil, // nil = auto mode
		PaginationThreshold: 50,  // Mặc định: nếu total <= 100 thì không pagination
		MaxPageSize:         50,  // Mặc định: max 100 items/page khi có pagination
	}
	return s.ListUsersWithOptions(options, filter)
}

// ListUsersWithOptions lấy danh sách users với các tùy chọn chi tiết
func (s *BaseAuthService[TUser, TRole]) ListUsersWithOptions(options ListUsersOptions, filter interface{}) (*BaseListUsersResponse[TUser], error) {
	// Set giá trị mặc định cho options
	if options.PaginationThreshold <= 0 {
		options.PaginationThreshold = 50
	}
	if options.MaxPageSize <= 0 {
		options.MaxPageSize = 50
	}

	// Convert filter to *repository.UserFilter nếu cần
	var userFilter *repository.UserFilter
	if filter != nil {
		if f, ok := filter.(*repository.UserFilter); ok {
			userFilter = f
			// Thêm sort options vào filter nếu có (ưu tiên options nếu có)
			if options.SortBy != "" {
				userFilter.SortBy = options.SortBy
				userFilter.Order = options.Order
				// Validate order
				if userFilter.Order != "asc" && userFilter.Order != "desc" {
					userFilter.Order = "asc" // Fallback về "asc" nếu không hợp lệ
				}
			} else if userFilter.SortBy != "" {
				// Nếu filter đã có sort nhưng options không có, validate order trong filter
				if userFilter.Order != "asc" && userFilter.Order != "desc" {
					userFilter.Order = "asc" // Fallback về "asc" nếu không hợp lệ
				}
			}
		} else {
			userFilter = nil
		}
	} else if options.SortBy != "" {
		// Nếu không có filter nhưng có sort, tạo filter mới chỉ để sort
		userFilter = &repository.UserFilter{
			SortBy: options.SortBy,
			Order:  options.Order,
			Custom: make(map[string]string),
		}
		// Validate order
		if userFilter.Order != "asc" && userFilter.Order != "desc" {
			userFilter.Order = "asc" // Fallback về "asc" nếu không hợp lệ
		}
	}

	// Bước 1: Đếm tổng số users (cần thiết để quyết định có dùng pagination không)
	// Sử dụng List với limit = 1 để lấy total (repository đã có count query tối ưu)
	var total int64
	_, total, err := s.userRepo.List(0, 1, userFilter)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi đếm số lượng users")
	}

	// Bước 2: Quyết định có dùng pagination không
	usePagination := false
	if options.EnablePagination != nil {
		// Client chỉ định rõ ràng
		usePagination = *options.EnablePagination
	} else {
		// Auto mode: tự động quyết định dựa trên threshold
		usePagination = total > int64(options.PaginationThreshold)
	}

	// Bước 3: Validate và xử lý pagination params
	var page, pageSize int
	var offset int
	var totalPages int

	if usePagination {
		// Có pagination
		page = options.Page
		if page < 1 {
			page = 1
		}
		pageSize = options.PageSize
		if pageSize < 1 {
			pageSize = 10 // Default page size
		}
		if pageSize > options.MaxPageSize {
			pageSize = options.MaxPageSize // Giới hạn max page size
		}
		offset = (page - 1) * pageSize
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	} else {
		// Không pagination: lấy tất cả records
		page = 0
		pageSize = 0 // 0 = unlimited
		offset = 0
		totalPages = 0
	}

	// Bước 4: Lấy users từ repository
	var users []TUser
	if usePagination {
		users, _, err = s.userRepo.List(offset, pageSize, userFilter)
	} else {
		// Không pagination: lấy tất cả (sử dụng limit rất lớn)
		// Sử dụng total + 1 để đảm bảo lấy hết (hoặc có thể dùng limit = 0 nếu GORM hỗ trợ)
		limit := int(total) + 1000 // Thêm buffer để đảm bảo lấy hết trong trường hợp có thêm records mới
		if limit > 100000 {
			limit = 100000 // Giới hạn tối đa để tránh query quá lớn
		}
		users, _, err = s.userRepo.List(0, limit, userFilter)
	}
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách users")
	}

	// Bước 5: Lấy role names cho tất cả users cùng lúc (tối ưu với một query duy nhất)
	userIDs := make([]string, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.GetID())
	}

	roleNamesMap, err := s.roleRepo.GetRoleNamesByUserIDs(userIDs)
	if err != nil {
		// Log error nhưng không fail request (role names là optional)
		// Có thể trả về empty map nếu có lỗi
		roleNamesMap = make(map[string][]string)
	}

	// Bước 6: Tạo response
	response := &BaseListUsersResponse[TUser]{
		Users:             users,
		RoleNames:         roleNamesMap,
		Total:             total,
		PaginationEnabled: usePagination,
	}

	if usePagination {
		response.Page = &page
		response.PageSize = &pageSize
		response.TotalPages = &totalPages
	}

	return response, nil
}
