# Authentication và Authorization Flow

Tài liệu này mô tả chi tiết quá trình Authentication (Xác thực) và Authorization (Phân quyền) trong AuthKit, từ lúc người dùng đăng nhập cho đến khi truy cập các endpoint được bảo vệ.

## Mục lục

1. [Quá trình Login](#quá-trình-login)
2. [Cấu trúc JWT Token](#cấu-trúc-jwt-token)
3. [Lưu trữ và Lấy Role IDs](#lưu-trữ-và-lấy-role-ids)
4. [Quá trình Authentication Middleware](#quá-trình-authentication-middleware)
5. [Quá trình Authorization Middleware](#quá-trình-authorization-middleware)
6. [Matching Rules và Roles](#matching-rules-và-roles)
7. [Luồng hoàn chỉnh](#luồng-hoàn-chỉnh)

---

## Quá trình Login

### 1. Người dùng gửi request đăng nhập

**Endpoint:** `POST /api/auth/login`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

### 2. Xử lý trong AuthService.Login()

Quá trình xử lý diễn ra trong `service/auth_service.go`:

#### Bước 1: Validate Input
- Kiểm tra email không được rỗng
- Nếu thiếu email, trả về lỗi validation

#### Bước 2: Tìm User theo Email
```go
user, err := s.userRepo.GetByEmail(req.Email)
```
- Tìm user trong database theo email
- **Lưu ý:** `GetByEmail()` sử dụng `Preload("Roles")` nên `user.Roles` đã được load sẵn
- Nếu không tìm thấy, trả về lỗi: "Email hoặc mật khẩu không đúng" (không tiết lộ email có tồn tại hay không)

#### Bước 3: Kiểm tra Trạng thái Tài khoản
```go
if !user.IsActive() {
    return nil, goerrorkit.NewAuthError(403, "Tài khoản đã bị vô hiệu hóa")
}
```
- Kiểm tra tài khoản có đang active không
- Nếu không active, từ chối đăng nhập

#### Bước 4: Xác thực Mật khẩu
```go
if !utils.CheckPasswordHash(req.Password, user.Password) {
    return nil, goerrorkit.NewAuthError(401, "Email hoặc mật khẩu không đúng")
}
```
- So sánh mật khẩu người dùng nhập với hash trong database
- Sử dụng `bcrypt` để so sánh
- Nếu không khớp, trả về lỗi xác thực

#### Bước 5: Lấy Roles của User
```go
userRoles := user.Roles
```
- **Tối ưu:** Sử dụng `user.Roles` đã được Preload sẵn từ `GetByEmail()`
- Không cần query database thêm vì roles đã được load trong bước 2
- Trả về danh sách `[]models.Role` từ relationship đã có sẵn

#### Bước 6: Extract Role IDs
```go
roleIDs := make([]uint, 0, len(userRoles))
for _, role := range userRoles {
    roleIDs = append(roleIDs, role.ID)
}
```
- Trích xuất mảng các Role ID từ danh sách roles
- Ví dụ: `[1, 3, 5]` nếu user có 3 roles với ID là 1, 3, và 5

#### Bước 7: Tạo JWT Token
```go
token, err := utils.GenerateToken(
    user.ID, 
    user.Email, 
    roleIDs, 
    s.config.JWT.Secret, 
    s.config.JWT.Expiration
)
```

### 3. Response khi Login thành công

**Response Body:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "abc123xyz",
      "email": "user@example.com",
      "full_name": "Nguyễn Văn A",
      "is_active": true,
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-01T00:00:00Z"
    }
  }
}
```

**Lưu ý:** 
- Token được trả về trong response body
- Client cần lưu token này để sử dụng cho các request tiếp theo
- Token có thể được gửi qua:
  - Header: `Authorization: Bearer <token>`
  - Cookie: `token=<token>`

---

## Cấu trúc JWT Token

### Claims trong JWT Token

JWT token chứa các thông tin sau (được định nghĩa trong `utils/jwt.go`):

```go
type JWTClaims struct {
    UserID  string `json:"user_id"`   // ID của user
    Email   string `json:"email"`      // Email của user
    RoleIDs []uint `json:"role_ids"`   // Mảng các Role ID
    jwt.RegisteredClaims               // Standard JWT claims
}
```

### RegisteredClaims bao gồm:
- `ExpiresAt`: Thời gian hết hạn (mặc định: 24 giờ)
- `IssuedAt`: Thời gian token được tạo
- `NotBefore`: Thời gian token bắt đầu có hiệu lực
- `Issuer`: "authkit"

### Mã hóa JWT Token

#### Thuật toán ký: HMAC-SHA256
```go
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
tokenString, err := token.SignedString([]byte(secret))
```

#### Quá trình mã hóa:
1. **Header**: Chứa thuật toán ký (HS256) và loại token (JWT)
   ```json
   {
     "alg": "HS256",
     "typ": "JWT"
   }
   ```

2. **Payload**: Chứa claims (user_id, email, role_ids, exp, iat, nbf, iss)
   ```json
   {
     "user_id": "abc123xyz",
     "email": "user@example.com",
     "role_ids": [1, 3, 5],
     "exp": 1704067200,
     "iat": 1703980800,
     "nbf": 1703980800,
     "iss": "authkit"
   }
   ```

3. **Signature**: Được tạo bằng cách:
   ```
   signature = HMAC-SHA256(
     base64UrlEncode(header) + "." + base64UrlEncode(payload),
     secret
   )
   ```

4. **Token cuối cùng**: 
   ```
   token = base64UrlEncode(header) + "." + base64UrlEncode(payload) + "." + signature
   ```

### Bảo mật Role IDs trong Token

**Quan trọng:** Role IDs được bảo vệ bởi JWT signature:
- Nếu hacker cố gắng sửa đổi `role_ids` trong token, signature sẽ không khớp
- Khi validate token, hệ thống sẽ kiểm tra signature trước khi tin tưởng dữ liệu trong token
- Nếu signature không hợp lệ, token sẽ bị từ chối hoàn toàn

---

## Lưu trữ và Lấy Role IDs

### 1. Lưu trữ Role IDs trong JWT Token

Khi login thành công, mảng `role_ids` được **nhúng trực tiếp vào JWT token**:

```go
// Trong service/auth_service.go
roleIDs := make([]uint, 0, len(userRoles))
for _, role := range userRoles {
    roleIDs = append(roleIDs, role.ID)
}

token, err := utils.GenerateToken(user.ID, user.Email, roleIDs, ...)
```

**Ví dụ:** Nếu user có 3 roles với ID là 1, 3, 5:
- Token sẽ chứa: `"role_ids": [1, 3, 5]`
- Dữ liệu này được mã hóa trong payload của JWT

### 2. Lấy Role IDs từ Token (Authentication Middleware)

Trong `middleware/auth_middleware.go`, khi request đến:

```go
// Bước 1: Validate token và extract claims
claims, err := utils.ValidateToken(token, m.config.JWT.Secret)
// claims.RoleIDs chứa mảng role IDs từ token đã được validate

// Bước 2: Lưu vào context để tái sử dụng
roleIDs := claims.RoleIDs
if roleIDs == nil {
    roleIDs = []uint{} // Đảm bảo không nil
}

c.Locals("roleIDs", roleIDs) // Lưu vào context
```

**Lưu ý quan trọng:**
- Role IDs được lấy từ **token đã được validate** (signature đã được kiểm tra)
- Không cần query database để lấy role IDs
- Dữ liệu an toàn vì đã được bảo vệ bởi JWT signature

### 3. Sử dụng lại Role IDs trong Authorization

Trong `middleware/authorization_middleware.go`:

```go
// Lấy role IDs từ context (không cần query DB)
roleIDs, ok := GetRoleIDsFromContext(c)
if !ok {
    // Fallback: nếu không có trong context, query từ DB
    userRoles, err := m.roleRepo.ListRolesOfUser(user.ID)
    // ...
}

// Convert role IDs thành role names (cần query DB một lần)
if len(roleIDs) > 0 {
    roles, err := m.roleRepo.GetByIDs(roleIDs)
    // Chỉ query role names theo IDs, không query toàn bộ roles của user
    for _, role := range roles {
        userRoleNames[role.Name] = true
    }
}
```

**Tối ưu hóa:**
- Role IDs được lấy từ token (đã validate) → không cần query DB
- Chỉ query role names một lần bằng `GetByIDs(roleIDs)` → query nhẹ, nhanh
- Tránh query toàn bộ roles của user mỗi lần authorize

### 4. So sánh với cách truyền thống

**Cách truyền thống (không tối ưu):**
```
Mỗi request → Query DB để lấy roles của user → Chậm, tốn tài nguyên
```

**Cách của AuthKit (tối ưu):**
```
Login → Lưu role_ids vào JWT token
Mỗi request → Lấy role_ids từ token (đã validate) → Chỉ query role names một lần → Nhanh, hiệu quả
```

---

## Quá trình Authentication Middleware

Authentication middleware (`middleware/auth_middleware.go`) được áp dụng trước khi xử lý request.

### 1. Extract Token

```go
func extractToken(c *fiber.Ctx) string {
    // Ưu tiên: Lấy từ Authorization header
    authHeader := c.Get("Authorization")
    if authHeader != "" {
        parts := strings.Split(authHeader, " ")
        if len(parts) == 2 && parts[0] == "Bearer" {
            return parts[1]
        }
    }
    
    // Fallback: Lấy từ cookie
    return c.Cookies("token")
}
```

**Hỗ trợ 2 cách:**
1. **Authorization Header**: `Authorization: Bearer <token>`
2. **Cookie**: `token=<token>`

### 2. Validate Token

```go
claims, err := utils.ValidateToken(token, m.config.JWT.Secret)
```

**Quá trình validate (`utils/jwt.go`):**

1. **Parse token** với claims structure
2. **Kiểm tra signing method**: Phải là HMAC-SHA256
   ```go
   if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
       return nil, fmt.Errorf("unexpected signing method")
   }
   ```
3. **Verify signature**: Sử dụng secret key để verify
4. **Kiểm tra token hợp lệ**: `token.Valid == true`
5. **Trả về claims** nếu tất cả đều hợp lệ

**Nếu token không hợp lệ:**
- Signature không khớp → Từ chối
- Token hết hạn → Từ chối
- Token bị sửa đổi → Từ chối (signature không khớp)

### 3. Lấy User từ Database

```go
user, err := m.userRepo.GetByID(claims.UserID)
```

- Sử dụng `user_id` từ token (đã được validate)
- Query database để lấy thông tin user đầy đủ
- Kiểm tra user có tồn tại không

### 4. Kiểm tra Trạng thái User

```go
if !user.IsActive() {
    return goerrorkit.NewAuthError(403, "Tài khoản đã bị vô hiệu hóa")
}
```

- Kiểm tra user có đang active không
- Nếu không active, từ chối request

### 5. Extract và Lưu Role IDs

```go
roleIDs := claims.RoleIDs
if roleIDs == nil {
    roleIDs = []uint{}
}

c.Locals("user", user)
c.Locals("userID", user.ID)
c.Locals("roleIDs", roleIDs) // Lưu vào context để tái sử dụng
```

**Lưu vào context:**
- `user`: Object user đầy đủ
- `userID`: ID của user
- `roleIDs`: Mảng role IDs từ token (đã validate)

### 6. Tiếp tục Request

```go
return c.Next()
```

- Nếu tất cả đều hợp lệ, cho phép request tiếp tục
- Context đã chứa thông tin user và role IDs

---

## Quá trình Authorization Middleware

Authorization middleware (`middleware/authorization_middleware.go`) kiểm tra quyền truy cập endpoint dựa trên Rules.

### 1. Refresh Cache Rules (nếu cần)

```go
m.refreshCacheIfNeeded()
```

**Caching Rules:**
- Rules được cache trong memory để tăng hiệu suất
- Cache TTL: 5 phút
- Cấu trúc cache:
  - `exactRulesMap`: Map `"METHOD|PATH"` → `[]Rule` (O(1) lookup)
  - `patternRules`: Danh sách rules có wildcard pattern

### 2. Tìm Matching Rules

```go
matchingRules := m.findMatchingRules(method, path)
```

**Quá trình tìm rules:**

1. **Exact Match (O(1)):**
   ```go
   key := fmt.Sprintf("%s|%s", method, path)
   exactMatches, hasExactMatch := m.exactRulesMap[key]
   ```
   - Tìm rule khớp chính xác `METHOD|PATH`
   - Ví dụ: `"GET|/api/users"`

2. **Pattern Match (nếu không có exact match):**
   ```go
   for _, rule := range m.patternRules {
       if rule.Method == method && m.matchPath(rule.Path, path) {
           patternMatches = append(patternMatches, rule)
       }
   }
   ```
   - Hỗ trợ wildcard `*` trong path
   - Ví dụ: `"/api/users/*"` khớp với `"/api/users/123"`

**Nếu không có rule nào:**
```go
if len(matchingRules) == 0 {
    return goerrorkit.NewAuthError(403, "Không có quyền truy cập endpoint này")
}
```
- **Mặc định: FORBIDE** (từ chối) nếu không có rule

### 3. Kiểm tra PUBLIC Rule

```go
for _, rule := range matchingRules {
    if rule.Type == models.AccessPublic {
        return c.Next() // Cho phép truy cập ngay lập tức
    }
}
```

- Nếu có rule `PUBLIC`, cho phép truy cập mà không cần authentication
- Không cần kiểm tra user hay roles

### 4. Kiểm tra Authentication

```go
user, ok := GetUserFromContext(c)
if !ok {
    return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
}
```

- Tất cả rule khác (ALLOW, FORBIDE) đều yêu cầu authentication
- Nếu không có user trong context, từ chối

### 5. Lấy Role IDs và Convert thành Role Names

```go
// Lấy role IDs từ context (từ token đã validate)
roleIDs, ok := GetRoleIDsFromContext(c)
if !ok {
    // Fallback: query từ DB
    userRoles, err := m.roleRepo.ListRolesOfUser(user.ID)
    // ...
}

// Convert role IDs thành role names
userRoleNames := make(map[string]bool)
if len(roleIDs) > 0 {
    roles, err := m.roleRepo.GetByIDs(roleIDs)
    for _, role := range roles {
        userRoleNames[role.Name] = true
    }
}
```

**Tối ưu:**
- Lấy role IDs từ context (không cần query DB)
- Chỉ query role names một lần bằng `GetByIDs(roleIDs)`
- Tạo map `roleName → true` để lookup nhanh

### 6. Xử lý X-Role-Context Header (Optional)

```go
roleContext := c.Get("X-Role-Context")
if roleContext != "" {
    if !userRoleNames[roleContext] {
        return goerrorkit.NewAuthError(403, "Không có quyền sử dụng role context này")
    }
    // Chỉ sử dụng role context được chỉ định
    userRoleNames = map[string]bool{roleContext: true}
}
```

**Tính năng Role Context:**
- Cho phép client chỉ định role cụ thể để sử dụng
- Hữu ích khi user có nhiều roles nhưng chỉ muốn dùng một role
- Ví dụ: User có roles `["admin", "editor"]`, nhưng request chỉ muốn dùng `"editor"`

### 7. Kiểm tra Super Admin

```go
if userRoleNames["super_admin"] {
    return c.Next() // Bypass tất cả rules
}
```

- Role `super_admin` có quyền truy cập tất cả endpoints
- Bỏ qua tất cả rules

### 8. Xử lý Rules: FORBIDE và ALLOW

```go
hasForbidMatch := false
hasAllowMatch := false

for _, rule := range matchingRules {
    switch rule.Type {
    case models.AccessForbid:
        // FORBIDE rules: kiểm tra user có roles bị cấm không
        if len(rule.Roles) == 0 {
            hasForbidMatch = true // Cấm tất cả
        } else {
            for _, roleName := range rule.Roles {
                if userRoleNames[roleName] {
                    hasForbidMatch = true
                    break
                }
            }
        }
        
    case models.AccessAllow:
        // ALLOW rules: kiểm tra user có roles được phép không
        if len(rule.Roles) == 0 {
            hasAllowMatch = true // Cho phép tất cả authenticated users
        } else {
            for _, roleName := range rule.Roles {
                if userRoleNames[roleName] {
                    hasAllowMatch = true
                    break
                }
            }
        }
    }
}
```

**Logic xử lý:**

1. **FORBIDE Rule:**
   - Nếu `rule.Roles` rỗng → Cấm tất cả users
   - Nếu user có bất kỳ role nào trong `rule.Roles` → Cấm
   - Ví dụ: Rule `FORBIDE ["guest"]` → User có role `guest` bị cấm

2. **ALLOW Rule:**
   - Nếu `rule.Roles` rỗng → Cho phép tất cả authenticated users
   - Nếu user có bất kỳ role nào trong `rule.Roles` → Cho phép
   - Ví dụ: Rule `ALLOW ["admin", "editor"]` → User có role `admin` hoặc `editor` được phép

### 9. Áp dụng Priority: FORBIDE > ALLOW

```go
// FORBIDE có priority cao hơn ALLOW
if hasForbidMatch {
    return goerrorkit.NewAuthError(403, "Không có quyền truy cập")
}

if hasAllowMatch {
    return c.Next() // Cho phép truy cập
}

// Mặc định: từ chối nếu không có rule nào match
return goerrorkit.NewAuthError(403, "Không có quyền truy cập")
```

**Priority:**
1. **FORBIDE** → Từ chối (nếu có match)
2. **ALLOW** → Cho phép (nếu có match và không có FORBIDE)
3. **Default** → Từ chối (nếu không có rule nào match)

---

## Matching Rules và Roles

### Cấu trúc Rule

```go
type Rule struct {
    ID     string     `json:"id"`     // Format: "METHOD|PATH"
    Method string     `json:"method"` // GET, POST, PUT, DELETE, etc.
    Path   string     `json:"path"`   // URL path pattern (hỗ trợ wildcard *)
    Type   AccessType `json:"type"`   // PUBLIC, ALLOW, FORBIDE
    Roles  []string   `json:"roles"`  // Danh sách role names
}
```

### Các loại AccessType

1. **PUBLIC**: Cho phép truy cập công khai (không cần authentication)
2. **ALLOW**: Cho phép các roles cụ thể (hoặc tất cả authenticated users nếu roles rỗng)
3. **FORBIDE**: Cấm các roles cụ thể (hoặc cấm tất cả nếu roles rỗng)

### Ví dụ Rules

#### Ví dụ 1: Public Endpoint
```json
{
  "id": "GET|/api/public/posts",
  "method": "GET",
  "path": "/api/public/posts",
  "type": "PUBLIC",
  "roles": []
}
```
- Bất kỳ ai cũng có thể truy cập, không cần đăng nhập

#### Ví dụ 2: Allow Specific Roles
```json
{
  "id": "POST|/api/admin/users",
  "method": "POST",
  "path": "/api/admin/users",
  "type": "ALLOW",
  "roles": ["admin", "super_admin"]
}
```
- Chỉ users có role `admin` hoặc `super_admin` mới được phép

#### Ví dụ 3: Allow All Authenticated Users
```json
{
  "id": "GET|/api/profile",
  "method": "GET",
  "path": "/api/profile",
  "type": "ALLOW",
  "roles": []
}
```
- Tất cả users đã đăng nhập đều được phép (roles rỗng)

#### Ví dụ 4: Forbid Specific Roles
```json
{
  "id": "POST|/api/admin/settings",
  "method": "POST",
  "path": "/api/admin/settings",
  "type": "FORBIDE",
  "roles": ["guest"]
}
```
- Users có role `guest` bị cấm truy cập

#### Ví dụ 5: Wildcard Pattern
```json
{
  "id": "GET|/api/users/*",
  "method": "GET",
  "path": "/api/users/*",
  "type": "ALLOW",
  "roles": ["admin", "user"]
}
```
- Khớp với `/api/users/123`, `/api/users/abc`, etc.
- Chỉ users có role `admin` hoặc `user` mới được phép

### Matching Logic

**Quy trình matching:**

1. **Tìm exact match trước:**
   - Key: `"GET|/api/users/123"`
   - Nếu có rule khớp chính xác → Sử dụng rule đó

2. **Nếu không có exact match, tìm pattern match:**
   - Pattern: `"/api/users/*"`
   - Path: `"/api/users/123"`
   - So sánh từng segment:
     - `"api" == "api"` ✓
     - `"users" == "users"` ✓
     - `"*"` khớp với `"123"` ✓
   - → Match!

3. **Nếu có nhiều rules match:**
   - Tất cả rules đều được xem xét
   - FORBIDE có priority cao hơn ALLOW

---

## Luồng hoàn chỉnh

### Scenario: User đăng nhập và truy cập endpoint được bảo vệ

#### Bước 1: Login
```
Client → POST /api/auth/login
         { "email": "user@example.com", "password": "..." }
         
AuthService.Login():
  1. Validate email/password
  2. Get user from DB (với Preload Roles)
  3. Check user.IsActive()
  4. Verify password hash
  5. Get user roles từ user.Roles (đã Preload) → [Role{ID:1, Name:"admin"}, Role{ID:3, Name:"editor"}]
  6. Extract role IDs → [1, 3]
  7. Generate JWT token với role_ids: [1, 3]
  
Client ← { "token": "eyJhbGc...", "user": {...} }
```

#### Bước 2: Request với Token
```
Client → GET /api/admin/users
         Authorization: Bearer eyJhbGc...
```

#### Bước 3: Authentication Middleware
```
AuthMiddleware.RequireAuth():
  1. Extract token từ header
  2. ValidateToken() → Verify signature
  3. Extract claims → { user_id, email, role_ids: [1, 3] }
  4. Get user from DB by user_id
  5. Check user.IsActive()
  6. Store in context:
     - c.Locals("user", user)
     - c.Locals("userID", user.ID)
     - c.Locals("roleIDs", [1, 3])
  7. Continue → c.Next()
```

#### Bước 4: Authorization Middleware
```
AuthorizationMiddleware.Authorize():
  1. Refresh cache if needed
  2. Find matching rules:
     - Method: "GET"
     - Path: "/api/admin/users"
     - Find rule: { type: "ALLOW", roles: ["admin", "super_admin"] }
  
  3. Check PUBLIC? → No
  
  4. Get user from context → ✓
  
  5. Get role IDs from context → [1, 3]
  
  6. Convert role IDs to names:
     - GetByIDs([1, 3]) → [Role{ID:1, Name:"admin"}, Role{ID:3, Name:"editor"}]
     - userRoleNames = {"admin": true, "editor": true}
  
  7. Check X-Role-Context? → No
  
  8. Check super_admin? → No
  
  9. Process rules:
     - Rule type: ALLOW
     - Rule roles: ["admin", "super_admin"]
     - User roles: {"admin": true, "editor": true}
     - Match? → Yes (user has "admin")
     - hasAllowMatch = true
  
  10. Check FORBIDE? → No
  
  11. hasAllowMatch = true → Allow access
  
  12. Continue → c.Next()
```

#### Bước 5: Handler xử lý Request
```
Handler:
  - Get user from context
  - Process business logic
  - Return response
```

### Diagram tổng quan

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ 1. POST /api/auth/login
       │    { email, password }
       ▼
┌─────────────────────┐
│  AuthHandler.Login  │
└──────┬──────────────┘
       │
       ▼
┌─────────────────────┐
│ AuthService.Login() │
│  - Validate input   │
│  - Get user from DB │
│    (với Preload Roles)│
│  - Check password   │
│  - Get roles từ      │
│    user.Roles        │
│  - Extract role IDs │
│  - Generate JWT     │
└──────┬──────────────┘
       │ 2. Return { token, user }
       ▼
┌─────────────┐
│   Client    │
│  Store token│
└──────┬──────┘
       │ 3. GET /api/admin/users
       │    Authorization: Bearer <token>
       ▼
┌──────────────────────────────┐
│ AuthMiddleware.RequireAuth() │
│  - Extract token             │
│  - Validate token signature  │
│  - Get user from DB          │
│  - Extract role_ids from     │
│    validated token           │
│  - Store in context          │
└──────┬───────────────────────┘
       │
       ▼
┌───────────────────────────────┐
│ AuthorizationMiddleware.      │
│ Authorize()                   │
│  - Find matching rules        │
│  - Get role_ids from context  │
│  - Convert to role names      │
│  - Check rules (ALLOW/FORBIDE)│
│  - Allow/Deny                 │
└──────┬────────────────────────┘
       │
       ▼
┌─────────────────────┐
│   Handler           │
│  Process request    │
└──────┬──────────────┘
       │ 4. Return response
       ▼
┌─────────────┐
│   Client    │
└─────────────┘
```

---

## Tóm tắt

### Điểm quan trọng

1. **Role IDs được lưu trong JWT Token:**
   - Khi login, role IDs được nhúng vào JWT token
   - Token được ký bằng HMAC-SHA256, đảm bảo không thể sửa đổi

2. **Role IDs được lấy từ Token đã Validate:**
   - Authentication middleware validate token và extract role IDs
   - Không cần query database để lấy role IDs
   - Role IDs được lưu vào context để tái sử dụng

3. **Authorization sử dụng Role IDs từ Context:**
   - Lấy role IDs từ context (không query DB)
   - Chỉ query role names một lần bằng `GetByIDs(roleIDs)`
   - Tối ưu hiệu suất, giảm số lượng query DB

4. **Rules được Cache:**
   - Rules được cache trong memory với TTL 5 phút
   - Exact match lookup O(1) bằng map
   - Pattern matching chỉ khi không có exact match

5. **Priority: FORBIDE > ALLOW:**
   - FORBIDE rules có priority cao hơn ALLOW
   - Super admin bypass tất cả rules
   - PUBLIC rules cho phép truy cập không cần authentication

### Lợi ích của cách tiếp cận này

- **Hiệu suất cao:** 
  - Giảm số lượng query DB bằng cách lưu role IDs trong token
  - Sử dụng Preload để load user và roles trong cùng một lần query (GORM batch queries)
  - Không cần query roles riêng biệt khi login
- **Bảo mật:** Role IDs được bảo vệ bởi JWT signature
- **Scalable:** Rules được cache, lookup nhanh
- **Linh hoạt:** Hỗ trợ PUBLIC, ALLOW, FORBIDE rules với wildcard patterns
- **Tối ưu:** 
  - Login: Chỉ 1 lần query với Preload (user + roles)
  - Authorization: Chỉ query role names một lần bằng `GetByIDs(roleIDs)`, không query toàn bộ roles mỗi request

