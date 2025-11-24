# Đề xuất cải tiến Authkit cho nhiều ứng dụng Golang

## 1. JWT Claims linh hoạt và có thể mở rộng

### Vấn đề hiện tại:
- Authkit chỉ có `JWTClaims` với `roleIDs []uint` và không có `username`
- Nhiều ứng dụng cần `roles []string` (role names) thay vì IDs
- Không thể thêm custom fields vào claims

### Đề xuất:

```go
// utils/jwt.go

// BaseClaims chứa các fields cơ bản
type BaseClaims struct {
    UserID  string `json:"user_id"`
    Email   string `json:"email"`
    jwt.RegisteredClaims
}

// ClaimsConfig cho phép customize claims
type ClaimsConfig struct {
    // Role representation: "ids" ([]uint) hoặc "names" ([]string)
    RoleFormat string // "ids" | "names"
    
    // Custom fields
    CustomFields map[string]interface{}
    
    // Include username
    IncludeUsername bool
    
    // Role IDs hoặc Role Names
    RoleIDs  []uint   // khi RoleFormat = "ids"
    RoleNames []string // khi RoleFormat = "names"
}

// GenerateTokenFlexible tạo token với config linh hoạt
func GenerateTokenFlexible(
    userID string,
    email string,
    config ClaimsConfig,
    secret string,
    expiration time.Duration,
) (string, error) {
    claims := jwt.MapClaims{
        "user_id": userID,
        "email":   email,
        "exp":     time.Now().Add(expiration).Unix(),
        "iat":     time.Now().Unix(),
        "nbf":     time.Now().Unix(),
    }
    
    // Thêm username nếu cần
    if config.IncludeUsername {
        if username, ok := config.CustomFields["username"].(string); ok {
            claims["username"] = username
        }
    }
    
    // Thêm roles theo format
    if config.RoleFormat == "names" {
        claims["roles"] = config.RoleNames
    } else {
        claims["role_ids"] = config.RoleIDs
    }
    
    // Thêm custom fields
    for k, v := range config.CustomFields {
        if k != "username" || !config.IncludeUsername {
            claims[k] = v
        }
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}

// ValidateTokenFlexible parse token và trả về MapClaims để dễ extract
func ValidateTokenFlexible(tokenString, secret string) (jwt.MapClaims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, jwt.ErrSignatureInvalid
}
```

## 2. Hỗ trợ nhiều Role Representation

### Đề xuất:

```go
// utils/jwt.go

// RoleConverter interface để convert giữa role IDs và names
type RoleConverter interface {
    RoleIDsToNames(roleIDs []uint) ([]string, error)
    RoleNamesToIDs(roleNames []string) ([]uint, error)
}

// GenerateTokenWithConverter tạo token với converter
func GenerateTokenWithConverter(
    userID string,
    email string,
    roles interface{}, // có thể là []uint hoặc []string
    converter RoleConverter,
    secret string,
    expiration time.Duration,
) (string, error) {
    // Auto-detect type và convert
    var roleIDs []uint
    var roleNames []string
    
    switch v := roles.(type) {
    case []uint:
        roleIDs = v
        if converter != nil {
            roleNames, _ = converter.RoleIDsToNames(v)
        }
    case []string:
        roleNames = v
        if converter != nil {
            roleIDs, _ = converter.RoleNamesToIDs(v)
        }
    }
    
    // Tạo claims với cả hai format để tương thích
    claims := jwt.MapClaims{
        "user_id":  userID,
        "email":    email,
        "role_ids": roleIDs,
        "roles":    roleNames,
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

## 3. Middleware cho các framework phổ biến

### Đề xuất:

```go
// middleware/fiber.go

package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/techmaster-vietnam/authkit/utils"
)

// JWTAuthMiddleware cho Fiber
func JWTAuthMiddleware(secret string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        token := c.Get("Authorization")
        if token == "" {
            // Thử lấy từ cookie
            token = c.Cookies("token")
        }
        
        if token == "" {
            return c.Status(401).JSON(fiber.Map{
                "error": "Unauthorized",
            })
        }
        
        // Remove "Bearer " prefix nếu có
        if len(token) > 7 && token[:7] == "Bearer " {
            token = token[7:]
        }
        
        claims, err := utils.ValidateTokenFlexible(token, secret)
        if err != nil {
            return c.Status(401).JSON(fiber.Map{
                "error": "Invalid token",
            })
        }
        
        // Set claims vào context
        c.Locals("userID", claims["user_id"])
        c.Locals("email", claims["email"])
        
        // Extract roles (hỗ trợ cả IDs và names)
        if roles, ok := claims["roles"].([]interface{}); ok {
            roleNames := make([]string, len(roles))
            for i, r := range roles {
                roleNames[i] = r.(string)
            }
            c.Locals("roles", roleNames)
        }
        
        if roleIDs, ok := claims["role_ids"].([]interface{}); ok {
            ids := make([]uint, len(roleIDs))
            for i, id := range roleIDs {
                ids[i] = uint(id.(float64))
            }
            c.Locals("roleIDs", ids)
        }
        
        return c.Next()
    }
}

// RequireRole middleware kiểm tra role
func RequireRole(allowedRoles ...string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        roles, ok := c.Locals("roles").([]string)
        if !ok {
            return c.Status(403).JSON(fiber.Map{
                "error": "Forbidden",
            })
        }
        
        for _, role := range roles {
            for _, allowed := range allowedRoles {
                if role == allowed {
                    return c.Next()
                }
            }
        }
        
        return c.Status(403).JSON(fiber.Map{
            "error": "Forbidden",
        })
    }
}
```

## 4. Repository Pattern và Interface

### Đề xuất:

```go
// repository/user_repository.go

package repository

// UserRepository interface
type UserRepository interface {
    FindByID(id string) (*models.User, error)
    FindByEmail(email string) (*models.User, error)
    Create(user *models.User) error
    Update(user *models.User) error
    Delete(id string) error
}

// RoleRepository interface
type RoleRepository interface {
    FindByID(id string) (*models.Role, error)
    FindByName(name string) (*models.Role, error)
    FindByIDs(ids []uint) ([]*models.Role, error)
    FindByNames(names []string) ([]*models.Role, error)
    Create(role *models.Role) error
}

// GORMUserRepository implementation
type GORMUserRepository struct {
    db *gorm.DB
}

func NewGORMUserRepository(db *gorm.DB) UserRepository {
    return &GORMUserRepository{db: db}
}

func (r *GORMUserRepository) FindByID(id string) (*models.User, error) {
    var user models.User
    err := r.db.Where("id = ?", id).First(&user).Error
    return &user, err
}

// ... các methods khác
```

## 5. Configuration linh hoạt

### Đề xuất:

```go
// config/jwt_config.go

package config

type JWTConfig struct {
    Secret     string
    Expiration time.Duration
    
    // Claims configuration
    ClaimsConfig ClaimsConfig
    
    // Role format preference
    RoleFormat string // "ids" | "names" | "both"
    
    // Custom validators
    Validators []TokenValidator
}

type ClaimsConfig struct {
    IncludeUsername bool
    CustomFields    map[string]interface{}
}

type TokenValidator interface {
    Validate(claims jwt.MapClaims) error
}

// LoadConfigFromEnv load config từ environment variables
func LoadJWTConfigFromEnv() *JWTConfig {
    return &JWTConfig{
        Secret:     getEnv("JWT_SECRET", ""),
        Expiration: getEnvAsDuration("JWT_EXPIRATION", 24*time.Hour),
        ClaimsConfig: ClaimsConfig{
            IncludeUsername: getEnvAsBool("JWT_INCLUDE_USERNAME", true),
        },
        RoleFormat: getEnv("JWT_ROLE_FORMAT", "both"), // "ids", "names", "both"
    }
}
```

## 6. Helper Functions cho Role Conversion

### Đề xuất:

```go
// utils/role_converter.go

package utils

// RoleConverter helper
type RoleConverter struct {
    // Function để lookup role name từ ID
    GetRoleName func(roleID uint) (string, error)
    // Function để lookup role ID từ name
    GetRoleID func(roleName string) (uint, error)
}

// ConvertRoleIDsToNames convert role IDs sang names
func (c *RoleConverter) RoleIDsToNames(roleIDs []uint) ([]string, error) {
    names := make([]string, 0, len(roleIDs))
    for _, id := range roleIDs {
        name, err := c.GetRoleName(id)
        if err != nil {
            return nil, err
        }
        names = append(names, name)
    }
    return names, nil
}

// ConvertRoleNamesToIDs convert role names sang IDs
func (c *RoleConverter) RoleNamesToIDs(roleNames []string) ([]uint, error) {
    ids := make([]uint, 0, len(roleNames))
    for _, name := range roleNames {
        id, err := c.GetRoleID(name)
        if err != nil {
            return nil, err
        }
        ids = append(ids, id)
    }
    return ids, nil
}
```

## 7. Builder Pattern cho Token Generation

### Đề xuất:

```go
// utils/token_builder.go

package utils

type TokenBuilder struct {
    userID     string
    email      string
    username   string
    roleIDs    []uint
    roleNames  []string
    customFields map[string]interface{}
    secret     string
    expiration time.Duration
}

func NewTokenBuilder(userID, email, secret string) *TokenBuilder {
    return &TokenBuilder{
        userID:       userID,
        email:        email,
        secret:        secret,
        expiration:   24 * time.Hour,
        customFields: make(map[string]interface{}),
    }
}

func (tb *TokenBuilder) WithUsername(username string) *TokenBuilder {
    tb.username = username
    return tb
}

func (tb *TokenBuilder) WithRoleIDs(roleIDs []uint) *TokenBuilder {
    tb.roleIDs = roleIDs
    return tb
}

func (tb *TokenBuilder) WithRoleNames(roleNames []string) *TokenBuilder {
    tb.roleNames = roleNames
    return tb
}

func (tb *TokenBuilder) WithCustomField(key string, value interface{}) *TokenBuilder {
    tb.customFields[key] = value
    return tb
}

func (tb *TokenBuilder) WithExpiration(expiration time.Duration) *TokenBuilder {
    tb.expiration = expiration
    return tb
}

func (tb *TokenBuilder) Build() (string, error) {
    config := ClaimsConfig{
        IncludeUsername: tb.username != "",
        CustomFields:    tb.customFields,
        RoleFormat:      "both",
        RoleIDs:          tb.roleIDs,
        RoleNames:        tb.roleNames,
    }
    
    if tb.username != "" {
        config.CustomFields["username"] = tb.username
    }
    
    return GenerateTokenFlexible(tb.userID, tb.email, config, tb.secret, tb.expiration)
}
```

## 8. Testing Utilities

### Đề xuất:

```go
// testing/test_helpers.go

package testing

// MockTokenGenerator để test
func MockTokenGenerator(claims jwt.MapClaims, secret string) string {
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    tokenString, _ := token.SignedString([]byte(secret))
    return tokenString
}

// CreateTestToken helper
func CreateTestToken(userID, email string, roles []string, secret string) string {
    claims := jwt.MapClaims{
        "user_id": userID,
        "email":   email,
        "roles":   roles,
        "exp":     time.Now().Add(24 * time.Hour).Unix(),
    }
    return MockTokenGenerator(claims, secret)
}
```

## Tóm tắt các cải tiến

1. ✅ **JWT Claims linh hoạt**: Hỗ trợ cả role IDs và role names, có thể thêm custom fields
2. ✅ **Middleware sẵn có**: Cho Fiber, Gin, Echo
3. ✅ **Repository Pattern**: Dễ test và mock
4. ✅ **Configuration linh hoạt**: Từ env vars hoặc code
5. ✅ **Role Converter**: Chuyển đổi giữa IDs và names
6. ✅ **Builder Pattern**: Dễ sử dụng và đọc code
7. ✅ **Testing Utilities**: Hỗ trợ viết test dễ dàng

## Migration Path

Để backward compatible, giữ các hàm cũ và thêm các hàm mới:

```go
// Giữ hàm cũ để backward compatible
func GenerateToken(userID string, email string, roleIDs []uint, secret string, expiration time.Duration) (string, error) {
    // Implementation cũ
}

// Thêm hàm mới với nhiều options
func GenerateTokenFlexible(...) (string, error) {
    // Implementation mới
}
```

