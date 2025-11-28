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
	userRepo         *repository.BaseUserRepository[TUser]
	roleRepo         *repository.BaseRoleRepository[TRole]
	refreshTokenRepo *repository.RefreshTokenRepository
	config           *config.Config
	jwtCustomizer    JWTCustomizer[TUser]
}

// NewBaseAuthService tạo mới BaseAuthService với generic types
func NewBaseAuthService[TUser core.UserInterface, TRole core.RoleInterface](
	userRepo *repository.BaseUserRepository[TUser],
	roleRepo *repository.BaseRoleRepository[TRole],
	refreshTokenRepo *repository.RefreshTokenRepository,
	cfg *config.Config,
) *BaseAuthService[TUser, TRole] {
	return &BaseAuthService[TUser, TRole]{
		userRepo:         userRepo,
		roleRepo:         roleRepo,
		refreshTokenRepo: refreshTokenRepo,
		config:           cfg,
		jwtCustomizer:    nil,
	}
}

// SetJWTCustomizer set JWT customizer callback để tùy chỉnh JWT claims
func (s *BaseAuthService[TUser, TRole]) SetJWTCustomizer(customizer JWTCustomizer[TUser]) {
	s.jwtCustomizer = customizer
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
