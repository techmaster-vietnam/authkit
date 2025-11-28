# 7. Cơ chế hoạt động chi tiết

Tài liệu này mô tả chi tiết **implementation** của các cơ chế bên trong AuthKit ở mức code, bao gồm cấu trúc dữ liệu, thuật toán và các chi tiết kỹ thuật.

> 📖 **Lưu ý**: Tài liệu này tập trung vào **implementation details** và **code-level explanations**. Để hiểu về **luồng xử lý** và **cách sử dụng**, xem các tài liệu khác:
> - [3. Middleware và Security](./03-middleware-security.md) - Luồng authentication và authorization
> - [4. Hệ thống phân quyền](./04-he-thong-phan-quyen.md) - Rule matching và evaluation

---

## 7.1. JWT Token Implementation

### 7.1.1. JWT Token là gì?

**JWT (JSON Web Token)** là một chuẩn mở (RFC 7519) để truyền thông tin an toàn giữa các parties dưới dạng JSON object. Token được ký bằng secret key hoặc public/private key pair.

**Trong AuthKit:**
- Sử dụng thuật toán **HMAC-SHA256** (HS256) với symmetric key để ký token
- Chi tiết về thuật toán và thư viện xem tại [7.1.5.1](#7151-implementation-details---thuật-toán-và-thư-viện)

**Đặc điểm chính:**
- **Stateless**: Server không cần lưu session, mỗi request tự chứa đủ thông tin
- **Self-contained**: Claims được embed trong token, không cần query database
- **Signed**: Được ký bằng secret key để đảm bảo tính toàn vẹn

### 7.1.2. Claims để làm gì?

**Claims** là các thông tin được lưu trong JWT token, bao gồm:

1. **Authentication**: Xác định user đã đăng nhập (user_id, email)
2. **Authorization**: Xác định quyền của user (role_ids, roles)
3. **Metadata**: Thông tin bổ sung (username, custom fields)
4. **Expiration**: Thời gian hết hạn của token

**Lợi ích:**
- Không cần query database mỗi request để lấy user info
- Tăng hiệu suất (giảm database load)
- Dễ scale (không cần shared session storage)

### 7.1.3. Cấu trúc JWT Token

JWT token có **3 phần**, ngăn cách bởi dấu chấm (`.`):

```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWJjMTIzIiwiZW1haWwiOiJ1c2VyQGV4YW1wbGUuY29tIn0.signature
│─────────────────────│ │──────────────────────────────────────────│ │──────────│
      Header                    Payload (Claims)                      Signature
```

**1. Header** - Metadata về token:
```json
{
  "alg": "HS256",  // Algorithm để sign (HMAC-SHA256)
  "typ": "JWT"     // Type của token
}
```

**2. Payload (Claims)** - Thông tin được lưu:
```json
{
  "user_id": "abc123",
  "email": "user@example.com",
  "role_ids": [1, 2],
  "exp": 1234567890,
  "iat": 1234567890
}
```

**3. Signature** - Chữ ký để verify tính toàn vẹn:
```
HMAC-SHA256(
  base64UrlEncode(header) + "." + base64UrlEncode(payload),
  secret_key
)
```

### 7.1.4. Encode vs Encrypt - Bảo mật Claims

**⚠️ Quan trọng:** Header và Payload chỉ là **Base64-encoded** (không mã hóa), không phải **encrypt**.

**Bất kỳ ai cũng có thể decode và đọc:**

```javascript
// Ví dụ decode token (bất kỳ ai cũng làm được)
const token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiYWJjMTIzIiwiZW1haWwiOiJ1c2VyQGV4YW1wbGUuY29tIn0.signature";

// Tách 3 phần
const [headerB64, payloadB64, signature] = token.split('.');

// Decode Base64 (KHÔNG CẦN SECRET)
const header = JSON.parse(atob(headerB64));
// Kết quả: { "alg": "HS256", "typ": "JWT" }

const payload = JSON.parse(atob(payloadB64));
// Kết quả: { "user_id": "abc123", "email": "user@example.com", "role_ids": [1,2] }
```

**Bạn có thể test tại:** https://jwt.io

**Signature bảo vệ:**
- ✅ **Không thể sửa đổi**: Nếu hacker sửa claims → signature không match → token invalid
- ❌ **Không bảo vệ tính bảo mật**: Claims vẫn có thể đọc được

**→ Không nên lưu thông tin nhạy cảm (password, credit card, SSN) trong claims**

### 7.1.5. Luồng hoạt động

**Login Flow:**

```
1. Client → POST /api/auth/login
   Body: { "email": "user@example.com", "password": "123456" }

2. Server → Validate credentials
   → Generate access token (JWT) với claims
   → Generate refresh token (random string)
   → Lưu refresh token hash vào database
   → Response: { "token": "eyJhbGciOiJIUzI1NiIs...", "user": {...} }
   → Set cookie: refresh_token (HttpOnly, Secure, SameSite=Strict)

3. Client → Lưu access token vào memory/localStorage
   → Refresh token tự động lưu trong cookie (browser quản lý)

4. Client → GET /api/blogs
   Header: Authorization: Bearer eyJhbGciOiJIUzI1NiIs...

5. Server → Extract token từ header
   → Validate token (không lưu token)
   → Extract claims từ token
   → Lưu vào context (chỉ trong request này)
   → Xử lý request

6. Request xong → Context bị xóa (token không được lưu)
```

**Refresh Flow (khi access token hết hạn):**

```
1. Client → POST /api/auth/refresh
   Cookie: refresh_token (tự động gửi)

2. Server → Extract refresh token từ cookie
   → Validate refresh token trong database
   → Generate access token mới
   → Generate refresh token mới (rotation)
   → Xóa refresh token cũ, lưu token mới
   → Response: { "token": "new_access_token" }
   → Update cookie: refresh_token (mới)

3. Client → Sử dụng access token mới cho các request tiếp theo
```

**Đặc điểm:**
- Access token: Ngắn hạn, gửi trong `Authorization` header
- Refresh token: Dài hạn, lưu trong cookie HttpOnly
- Token rotation: Mỗi lần refresh tạo token mới, xóa token cũ
- Stateless: Server không lưu session, chỉ validate token mỗi request

**Làm sao Client biết được khi nào access token hết hạn?**

JWT token chứa thông tin expiration time (`exp`) trong payload. Client có thể **decode token** (không cần secret key) để đọc thông tin này:

**Cách 1: Decode JWT token (Khuyến nghị)**

Vì JWT token chỉ là Base64-encoded (không phải encrypted), client có thể decode để lấy expiration time:

```javascript
// Ví dụ JavaScript/TypeScript
function getTokenExpiration(token) {
  try {
    // Tách 3 phần của JWT
    const parts = token.split('.');
    if (parts.length !== 3) {
      return null;
    }
    
    // Decode payload (phần thứ 2)
    const payload = JSON.parse(atob(parts[1]));
    
    // Lấy expiration time (Unix timestamp)
    const exp = payload.exp;
    
    // Chuyển sang Date object
    return new Date(exp * 1000);
  } catch (error) {
    return null;
  }
}

// Sử dụng
const token = "eyJhbGciOiJIUzI1NiIs..."; // Token từ login response
const expirationDate = getTokenExpiration(token);
const now = new Date();
const timeUntilExpiry = expirationDate - now; // milliseconds

console.log(`Token hết hạn vào: ${expirationDate}`);
console.log(`Còn lại: ${Math.floor(timeUntilExpiry / 1000 / 60)} phút`);
```

**Cách 2: Sử dụng thư viện JWT**

```javascript
// Sử dụng thư viện jsonwebtoken hoặc jose
import jwt from 'jsonwebtoken';

const token = "eyJhbGciOiJIUzI1NiIs...";
const decoded = jwt.decode(token); // Decode không verify signature
const expirationDate = new Date(decoded.exp * 1000);
```

**Cách 3: Tự động refresh trước khi hết hạn**

Client nên refresh token trước khi hết hạn một khoảng thời gian (ví dụ: 5 phút):

```javascript
function shouldRefreshToken(token) {
  const expirationDate = getTokenExpiration(token);
  if (!expirationDate) return false;
  
  const now = new Date();
  const timeUntilExpiry = expirationDate - now;
  const fiveMinutes = 5 * 60 * 1000; // 5 phút
  
  // Refresh nếu còn lại < 5 phút
  return timeUntilExpiry < fiveMinutes;
}

// Kiểm tra trước mỗi API call
if (shouldRefreshToken(currentToken)) {
  await refreshAccessToken();
}
```

**Lưu ý:**
- Client **KHÔNG CẦN** secret key để decode token và đọc `exp`
- Chỉ cần secret key khi **verify signature** (server làm việc này)
- Field `exp` trong token là Unix timestamp (seconds since epoch)
- Mặc định access token hết hạn sau `JWT_EXPIRATION_HOURS` giờ (thường là 24h)

### 7.1.5.1. Implementation Details - Thuật toán và Thư viện

**Thuật toán ký (Signing Algorithm):**
- **HMAC-SHA256** (HS256): Sử dụng symmetric key (secret key) để tạo chữ ký
- HMAC (Hash-based Message Authentication Code) là một cơ chế để xác thực tính toàn vẹn của message
- SHA-256 là hàm hash một chiều, kết hợp với secret key để tạo chữ ký không thể giả mạo

**Thư viện sử dụng:**
- **`github.com/golang-jwt/jwt/v5`** (phiên bản 5.3.0)
- Đây là thư viện chính thức của Go community cho JWT, được maintain tích cực và có security updates thường xuyên

**Code Implementation:**

```34:35:utils/jwt.go
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
```

**Giải thích:**
1. `jwt.SigningMethodHS256`: Chỉ định thuật toán HMAC-SHA256
2. `token.SignedString([]byte(secret))`: Tạo chữ ký bằng cách:
   - Encode header và payload thành Base64URL
   - Tính toán HMAC-SHA256 của `base64UrlEncode(header) + "." + base64UrlEncode(payload)` với secret key
   - Encode signature thành Base64URL
   - Kết hợp: `header.payload.signature`

**Validation và Bảo mật:**

```41:46:utils/jwt.go
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
```

**Bảo mật:**
- ✅ **Kiểm tra signing method**: Chỉ chấp nhận HMAC, từ chối các thuật toán khác (RSA, ECDSA) để tránh algorithm confusion attacks
- ✅ **Signature verification**: Token chỉ hợp lệ nếu signature khớp với secret key
- ✅ **Tamper detection**: Nếu hacker sửa claims → signature không match → `token.Valid = false`

**Lý do chọn HMAC-SHA256:**
- ✅ Đơn giản: Chỉ cần một secret key (không cần public/private key pair)
- ✅ Hiệu suất cao: HMAC nhanh hơn RSA/ECDSA
- ✅ Phù hợp cho single-server hoặc shared secret trong microservices
- ⚠️ Lưu ý: Secret key phải được bảo mật tuyệt đối, không commit vào git

### 7.1.6. Claims Structure trong AuthKit

JWT token trong AuthKit sử dụng custom claims structure:

```go
type JWTClaims struct {
    UserID  string `json:"user_id"`
    Email   string `json:"email"`
    RoleIDs []uint `json:"role_ids"`  // Protected by signature
    jwt.RegisteredClaims
}
```

**RegisteredClaims** bao gồm:
- `ExpiresAt`: Thời gian hết hạn (từ `JWT_EXPIRATION_HOURS`)
- `IssuedAt`: Thời gian phát hành
- `NotBefore`: Không hợp lệ trước thời điểm này
- `Issuer`: "authkit"

### 7.1.2. Token Generation Process

```go
func GenerateToken(userID, email string, roleIDs []uint, secret string, expiration time.Duration) (string, error) {
    claims := JWTClaims{
        UserID:  userID,
        Email:   email,
        RoleIDs: roleIDs,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
            Issuer:    "authkit",
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

**Chi tiết kỹ thuật:**
- **Signing Method**: `HS256` (HMAC-SHA256) - chỉ method này được chấp nhận
- **Secret Key**: Từ config `JWT_SECRET` (phải đủ mạnh, tối thiểu 32 bytes)
- **Role IDs Protection**: Role IDs được embed trong claims và được bảo vệ bởi signature

### 7.1.3. Token Validation Process

```go
func ValidateToken(tokenString, secret string) (*JWTClaims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
        // Algorithm confusion prevention
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
        }
        return []byte(secret), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, jwt.ErrSignatureInvalid
}
```

**Security Checks:**
1. **Algorithm Verification**: Chỉ chấp nhận `HS256`, reject các algorithm khác
2. **Signature Verification**: Verify signature với secret key
3. **Expiration Check**: `token.Valid` tự động check `ExpiresAt`
4. **Claims Extraction**: Chỉ return claims nếu token hợp lệ

**Vì sao an toàn:**
- Nếu hacker modify `role_ids` trong token → signature không match → `token.Valid = false`
- Algorithm confusion attack bị ngăn chặn bởi explicit method check

### 7.1.7. Cấu hình Custom Claims

AuthKit hỗ trợ thêm custom fields vào claims thông qua `ClaimsConfig`:

```go
type ClaimsConfig struct {
    // Username to include in token (optional)
    Username string
    
    // Custom fields to add to token claims
    CustomFields map[string]interface{}
    
    // Role format: "ids" ([]uint), "names" ([]string), or "both"
    RoleFormat string // "ids" | "names" | "both"
    
    // Role IDs (when RoleFormat is "ids" or "both")
    RoleIDs []uint
    
    // Role Names (when RoleFormat is "names" or "both")
    RoleNames []string
}
```

#### 7.1.7.1. Tạo Token với Custom Claims

**Ví dụ: Thêm username và custom fields vào token**

```go
import "github.com/techmaster-vietnam/authkit/utils"

// Lấy thông tin user và roles
user := getUserFromDB()
userRoles := user.GetRoles()
roleIDs := utils.ExtractRoleIDsFromRoleInterfaces(userRoles)
roleNames := utils.ExtractRoleNamesFromRoleInterfaces(userRoles)

// Cấu hình claims với custom fields
config := utils.ClaimsConfig{
    Username:   user.GetFullName(), // Thêm username
    RoleFormat: "both",             // Bao gồm cả IDs và names
    RoleIDs:    roleIDs,
    RoleNames:  roleNames,
    CustomFields: map[string]interface{}{
        "mobile":  user.Mobile,      // Custom field: mobile
        "address": user.Address,     // Custom field: address
        "company_id": 123,           // Custom field: company_id
        "department": "IT",          // Custom field: department
    },
}

// Tạo token với flexible claims
token, err := utils.GenerateTokenFlexible(
    user.GetID(),
    user.GetEmail(),
    config,
    secret,
    expiration,
)
```

**Token được tạo sẽ chứa:**
```json
{
  "user_id": "abc123xyz",
  "email": "user@example.com",
  "username": "John Doe",
  "role_ids": [1, 2],
  "roles": ["admin", "editor"],
  "mobile": "0901234567",
  "address": "123 Main St",
  "company_id": 123,
  "department": "IT",
  "exp": 1234567890,
  "iat": 1234567890,
  "iss": "authkit"
}
```

#### 7.1.7.2. Implementation của GenerateTokenFlexible

```go
func GenerateTokenFlexible(
    userID string,
    email string,
    config ClaimsConfig,
    secret string,
    expiration time.Duration,
) (string, error) {
    // Tạo base claims
    claims := jwt.MapClaims{
        "user_id": userID,
        "email":   email,
        "exp":     time.Now().Add(expiration).Unix(),
        "iat":     time.Now().Unix(),
        "nbf":     time.Now().Unix(),
        "iss":     "authkit",
    }
    
    // Thêm username nếu có
    if config.Username != "" {
        claims["username"] = config.Username
    }
    
    // Thêm roles theo format
    if config.RoleFormat == "names" || config.RoleFormat == "both" {
        if len(config.RoleNames) > 0 {
            claims["roles"] = config.RoleNames
        }
    }
    if config.RoleFormat == "ids" || config.RoleFormat == "both" {
        if len(config.RoleIDs) > 0 {
            claims["role_ids"] = config.RoleIDs
        }
    }
    
    // Thêm custom fields
    for k, v := range config.CustomFields {
        if k != "username" || config.Username == "" {
            claims[k] = v
        }
    }
    
    // Ký token với secret key
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(secret))
}
```

**Chi tiết:**
- Sử dụng `jwt.MapClaims` để linh hoạt thêm custom fields
- Custom fields được thêm vào claims trước khi ký
- Signature bảo vệ tất cả claims (kể cả custom fields)

#### 7.1.7.3. Extract Claims từ Token

**Sử dụng ValidateTokenFlexible để extract claims:**

```go
import "github.com/techmaster-vietnam/authkit/utils"

// Validate và extract claims
claims, err := utils.ValidateTokenFlexible(tokenString, secret)
if err != nil {
    return err
}

// Extract các fields cơ bản
userID := claims["user_id"].(string)
email := claims["email"].(string)

// Extract username (nếu có)
if username, ok := claims["username"].(string); ok {
    fmt.Printf("Username: %s\n", username)
}

// Extract role IDs
if roleIDs, ok := claims["role_ids"].([]interface{}); ok {
    ids := make([]uint, len(roleIDs))
    for i, id := range roleIDs {
        ids[i] = uint(id.(float64))
    }
}

// Extract role names
if roleNames, ok := claims["roles"].([]interface{}); ok {
    names := make([]string, len(roleNames))
    for i, name := range roleNames {
        names[i] = name.(string)
    }
}

// Extract custom fields
if mobile, ok := claims["mobile"].(string); ok {
    fmt.Printf("Mobile: %s\n", mobile)
}
if companyID, ok := claims["company_id"].(float64); ok {
    fmt.Printf("Company ID: %.0f\n", companyID)
}
```

#### 7.1.7.4. Ví dụ Sử dụng trong Login Handler

**Tạo custom login handler với username và custom fields:**

```go
func (h *AuthHandler) LoginWithCustomClaims(c *fiber.Ctx) error {
    var req struct {
        Email    string `json:"email"`
        Password string `json:"password"`
    }
    c.BodyParser(&req)
    
    // Validate credentials
    user, err := h.userRepo.GetByEmail(req.Email)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
    }
    
    if !utils.CheckPasswordHash(req.Password, user.Password) {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
    }
    
    // Lấy roles của user
    userRoles := user.GetRoles()
    roleIDs := utils.ExtractRoleIDsFromRoleInterfaces(userRoles)
    roleNames := utils.ExtractRoleNamesFromRoleInterfaces(userRoles)
    
    // Cấu hình claims với thông tin user
    config := utils.ClaimsConfig{
        Username:   user.GetFullName(),
        RoleFormat: "both",
        RoleIDs:    roleIDs,
        RoleNames:  roleNames,
        CustomFields: map[string]interface{}{
            "mobile":  getCustomUserMobile(user),
            "address": getCustomUserAddress(user),
        },
    }
    
    // Tạo token
    token, err := utils.GenerateTokenFlexible(
        user.GetID(),
        user.GetEmail(),
        config,
        h.config.JWT.Secret,
        h.config.JWT.Expiration,
    )
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "Failed to generate token"})
    }
    
    return c.JSON(fiber.Map{
        "token": token,
        "user":  user,
    })
}
```

#### 7.1.7.5. Lưu ý Bảo mật

**✅ Nên thêm vào claims:**
- User ID, Email, Username
- Role IDs, Role Names
- Custom fields không nhạy cảm (mobile, address, department, etc.)

**❌ KHÔNG NÊN thêm vào claims:**
- Password (dù đã hash)
- Credit card numbers
- Social Security Numbers (SSN)
- API keys hoặc secrets
- Thông tin nhạy cảm khác

**Lý do:** Claims chỉ được **encode** (Base64), không phải **encrypt**. Signature chỉ bảo vệ tính toàn vẹn (không thể sửa), không bảo vệ tính bảo mật (không thể đọc).

#### 7.1.7.6. So sánh GenerateToken vs GenerateTokenFlexible

| Tính năng | GenerateToken | GenerateTokenFlexible |
|-----------|---------------|----------------------|
| Username | ❌ Không hỗ trợ | ✅ Hỗ trợ |
| Custom Fields | ❌ Không hỗ trợ | ✅ Hỗ trợ |
| Role Names | ❌ Chỉ IDs | ✅ IDs, Names, hoặc cả hai |
| Backward Compatible | ✅ Có | ✅ Có (hàm riêng) |
| Use Case | Đơn giản, nhanh | Linh hoạt, nhiều thông tin |

**Khi nào dùng:**
- `GenerateToken()`: Khi chỉ cần user_id, email, role_ids (đơn giản)
- `GenerateTokenFlexible()`: Khi cần username, custom fields, hoặc role names

---

## 7.2. Refresh Token Implementation

### 7.2.1. Tổng quan

AuthKit sử dụng **dual-token strategy** để cân bằng giữa bảo mật và trải nghiệm người dùng:

- **Access Token (JWT)**: Ngắn hạn (mặc định 24h), chứa user info và roles, gửi trong `Authorization` header
- **Refresh Token**: Dài hạn (mặc định 7 ngày), lưu trong cookie HttpOnly, chỉ dùng để refresh access token

**Luồng hoạt động:**

```mermaid
sequenceDiagram
    participant C as Client
    participant H as Handler
    participant S as AuthService
    participant DB as Database
    
    C->>H: POST /api/auth/login<br/>{email, password}
    H->>S: Login(req)
    S->>DB: Validate credentials
    S->>S: Generate access token (JWT)
    S->>S: Generate refresh token (random)
    S->>DB: Save refresh token (hashed)
    S-->>H: {token, refreshToken, user}
    H->>H: Set cookie: refresh_token
    H-->>C: 200 OK {token, user}<br/>Cookie: refresh_token
    
    Note over C: Access token hết hạn
    
    C->>H: POST /api/auth/refresh<br/>Cookie: refresh_token
    H->>S: Refresh(refreshToken)
    S->>DB: Validate refresh token
    S->>S: Generate new access token
    S->>S: Generate new refresh token (rotation)
    S->>DB: Delete old, save new token
    S-->>H: {token, refreshToken}
    H->>H: Update cookie
    H-->>C: 200 OK {token}<br/>Cookie: refresh_token (new)
```

### 7.2.2. Refresh Token Model

```go
type RefreshToken struct {
    ID        uint           `gorm:"primaryKey;autoIncrement"`
    Token     string         `gorm:"type:varchar(255);uniqueIndex;not null"` // SHA256 hash
    UserID    string         `gorm:"type:varchar(12);not null;index"`
    ExpiresAt time.Time      `gorm:"not null;index"`
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"` // Soft delete để revoke
}
```

**Đặc điểm:**
- Token được **hash (SHA256)** trước khi lưu vào database
- Index trên `user_id` và `expires_at` để tối ưu queries
- Soft delete để có thể revoke và audit

### 7.2.3. Token Generation và Storage

**Generate Refresh Token:**

```go
func GenerateRefreshToken() (string, error) {
    b := make([]byte, 32) // 32 bytes = 256 bits
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}
```

**Hash Token trước khi lưu:**

```go
func HashToken(token string) string {
    hash := sha256.Sum256([]byte(token))
    return hex.EncodeToString(hash[:])
}
```

**Lý do hash:**
- ✅ Bảo mật: Nếu database bị leak, attacker không thể dùng plain tokens
- ✅ Privacy: Không lưu plain token trong database
- ✅ One-way: Không thể reverse hash về token gốc

### 7.2.4. Cookie Configuration

Refresh token được set vào cookie với các thuộc tính bảo mật:

```go
c.Cookie(&fiber.Cookie{
    Name:     "refresh_token",
    Value:    refreshToken,
    Expires:  time.Now().Add(7 * 24 * time.Hour),
    HTTPOnly: true,  // JavaScript không thể truy cập (chống XSS)
    Secure:   true,  // Chỉ gửi qua HTTPS (production)
    SameSite: "Strict", // Chống CSRF
    Path:     "/api/auth", // Chỉ gửi khi request đến /api/auth/*
})
```

**Bảo mật:**
- **HttpOnly**: Ngăn JavaScript truy cập (chống XSS attacks)
- **Secure**: Chỉ gửi qua HTTPS (production)
- **SameSite=Strict**: Chống CSRF attacks
- **Path**: Giới hạn phạm vi gửi cookie

### 7.2.5. Refresh Token Rotation

Mỗi lần refresh, hệ thống tạo refresh token mới và xóa token cũ:

```go
func (s *BaseAuthService) Refresh(refreshToken string) (*BaseRefreshResponse, error) {
    // 1. Validate refresh token
    tokenRecord, err := s.refreshTokenRepo.GetByToken(refreshToken)
    if err != nil || tokenRecord.IsExpired() {
        return nil, errors.New("invalid refresh token")
    }
    
    // 2. Generate new access token
    newAccessToken := generateAccessToken(user)
    
    // 3. Generate new refresh token (rotation)
    newRefreshToken, _ := utils.GenerateRefreshToken()
    
    // 4. Delete old token, save new token
    s.refreshTokenRepo.DeleteByToken(refreshToken)
    s.refreshTokenRepo.Create(newRefreshToken, userID, expiresAt)
    
    return &BaseRefreshResponse{
        Token:        newAccessToken,
        RefreshToken: newRefreshToken,
    }, nil
}
```

**Lợi ích rotation:**
- ✅ Giảm rủi ro nếu refresh token bị lộ (token cũ bị vô hiệu hóa ngay)
- ✅ Phát hiện token reuse (nếu token cũ được dùng lại → có thể bị compromise)
- ✅ Tự động cleanup tokens không dùng

### 7.2.6. Logout và Revocation

**Logout xóa refresh token:**

```go
func (h *BaseAuthHandler) Logout(c *fiber.Ctx) error {
    refreshToken := c.Cookies("refresh_token")
    
    // Xóa từ database
    if refreshToken != "" {
        h.authService.Logout(refreshToken)
    }
    
    // Xóa cookie
    c.Cookie(&fiber.Cookie{
        Name:     "refresh_token",
        Value:    "",
        Expires:  time.Now().Add(-1 * time.Hour),
        HTTPOnly: true,
        Secure:   true,
        SameSite: "Strict",
        Path:     "/api/auth",
    })
    
    return c.JSON(fiber.Map{"message": "Đăng xuất thành công"})
}
```

**Revocation scenarios:**
- Logout: Xóa refresh token hiện tại
- Change password: Xóa tất cả refresh tokens của user
- Account deactivation: Xóa tất cả refresh tokens
- Security breach: Có thể revoke tất cả tokens của user

### 7.2.7. Configuration

**Environment Variables:**

```bash
# Access token expiration (mặc định: 24 giờ)
JWT_EXPIRATION_HOURS=24

# Refresh token expiration (mặc định: 7 ngày)
REFRESH_TOKEN_EXPIRATION_DAYS=7
```

**Config Structure:**

```go
type JWTConfig struct {
    Secret            string        // Secret key để ký JWT
    Expiration        time.Duration // Access token expiration
    RefreshExpiration time.Duration // Refresh token expiration
}
```

### 7.2.8. Best Practices

**✅ Nên làm:**
- Access token ngắn hạn (15-30 phút cho production)
- Refresh token rotation mỗi lần refresh
- Hash refresh token trước khi lưu database
- HttpOnly + Secure + SameSite cho cookie
- Revoke tokens khi logout/đổi mật khẩu

**❌ Không nên:**
- Lưu refresh token trong localStorage (dễ bị XSS)
- Gửi refresh token trong Authorization header (chỉ dùng cookie)
- Trả về refresh token trong JSON response (chỉ set cookie)
- Dùng refresh token làm access token

---

## 7.3. Password Hashing Implementation

### 7.2.1. Bcrypt Hashing

```go
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    return string(bytes), err
}
```

**Chi tiết kỹ thuật:**
- **Algorithm**: bcrypt với `DefaultCost = 10` (2^10 = 1024 rounds)
- **Salt**: Tự động generate và embed trong hash string
- **Output Format**: `$2a$10$...` (version, cost, salt+hash)

**Hash Format:**
```
$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
│  │  │  │                              │
│  │  │  └─ Salt (22 chars)             └─ Hash (31 chars)
│  │  └─ Cost factor (10 = 2^10 rounds)
│  └─ Version (2a)
└─ Algorithm identifier
```

### 7.2.2. Password Verification

```go
func CheckPasswordHash(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**Process:**
1. Extract salt và cost từ hash string
2. Hash password với salt và cost đó
3. Compare với hash trong database
4. Return `true` nếu match, `false` nếu không

**Security:**
- **One-way**: Không thể reverse hash về password
- **Unique Salt**: Mỗi password có salt riêng (tự động generate)
- **Cost Factor**: Có thể tăng để chống brute force (trade-off với performance)

---

## 7.4. Rule Matching Algorithm Implementation

### 7.3.1. Cache Data Structures

```go
type BaseAuthorizationMiddleware struct {
    exactRulesMap              map[string][]models.Rule  // "METHOD|PATH" → Rules
    patternRulesByMethodAndSegs map[string]map[int][]models.Rule  // method → segmentCount → Rules
    cacheMutex                 sync.RWMutex
    lastRefresh                time.Time
    cacheTTL                   time.Duration
}
```

**Cấu trúc:**
- `exactRulesMap`: O(1) lookup cho exact matches
- `patternRulesByMethodAndSegs`: Nested map để filter nhanh pattern rules
  - Level 1: Filter theo HTTP method
  - Level 2: Filter theo số segments trong path
  - Level 3: Array of rules để iterate và match

### 7.3.2. Finding Matching Rules

```go
func (m *BaseAuthorizationMiddleware) findMatchingRules(method, path string) []models.Rule {
    m.cacheMutex.RLock()  // Read lock for concurrent access
    defer m.cacheMutex.RUnlock()
    
    // Step 1: O(1) exact match lookup
    key := fmt.Sprintf("%s|%s", method, path)
    exactMatches, hasExactMatch := m.exactRulesMap[key]
    if hasExactMatch && len(exactMatches) > 0 {
        return exactMatches  // Early exit
    }
    
    // Step 2: Pattern matching (only if no exact match)
    pathSegments := m.countSegments(path)
    methodPatterns, hasMethodPatterns := m.patternRulesByMethodAndSegs[method]
    if !hasMethodPatterns {
        return nil
    }
    
    rulesToCheck, hasMatchingSegments := methodPatterns[pathSegments]
    if !hasMatchingSegments {
        return nil
    }
    
    // Step 3: Iterate và match từng rule
    var patternMatches []models.Rule
    for _, rule := range rulesToCheck {
        if m.matchPath(rule.Path, path) {
            patternMatches = append(patternMatches, rule)
        }
    }
    
    return patternMatches
}
```

**Tối ưu hóa:**
1. **Early Exit**: Exact match → return ngay (không check patterns)
2. **Filter by Method**: Chỉ check patterns cùng method
3. **Filter by Segment Count**: Chỉ check patterns cùng số segments
4. **Segment-by-Segment Matching**: So sánh từng segment thay vì regex

### 7.3.3. Segment Counting Algorithm

```go
func (m *BaseAuthorizationMiddleware) countSegments(path string) int {
    if len(path) == 0 || path == "/" {
        return 0
    }
    start := 0
    if path[0] == '/' {
        start = 1  // Skip leading slash
    }
    if start >= len(path) {
        return 0
    }
    return strings.Count(path[start:], "/") + 1
}
```

**Ví dụ:**
- `/api/users` → 2 segments (`api`, `users`)
- `/api/users/123` → 3 segments (`api`, `users`, `123`)
- `/api/blogs/123/comments` → 4 segments

### 7.3.4. Path Pattern Matching Algorithm

```go
func (m *BaseAuthorizationMiddleware) matchPath(pattern, path string) bool {
    if pattern == path {
        return true  // Exact match
    }
    
    patternLen := len(pattern)
    pathLen := len(path)
    patternIdx := 0
    pathIdx := 0
    
    // Skip leading slashes
    if patternIdx < patternLen && pattern[patternIdx] == '/' {
        patternIdx++
    }
    if pathIdx < pathLen && path[pathIdx] == '/' {
        pathIdx++
    }
    
    // Match segment by segment
    for patternIdx < patternLen && pathIdx < pathLen {
        // Extract pattern segment
        patternStart := patternIdx
        for patternIdx < patternLen && pattern[patternIdx] != '/' {
            patternIdx++
        }
        patternSeg := pattern[patternStart:patternIdx]
        
        // Extract path segment
        pathStart := pathIdx
        for pathIdx < pathLen && path[pathIdx] != '/' {
            pathIdx++
        }
        pathSeg := path[pathStart:pathIdx]
        
        // Match: wildcard * matches any segment
        if patternSeg != "*" && patternSeg != pathSeg {
            return false
        }
        
        // Move to next segment
        if patternIdx < patternLen {
            patternIdx++
        }
        if pathIdx < pathLen {
            pathIdx++
        }
    }
    
    // Both must reach end
    return patternIdx >= patternLen && pathIdx >= pathLen
}
```

**Ví dụ matching:**
- Pattern: `GET|/api/users/*`, Path: `GET|/api/users/123` → ✅ Match
- Pattern: `GET|/api/blogs/*/comments`, Path: `GET|/api/blogs/123/comments` → ✅ Match
- Pattern: `GET|/api/users/*`, Path: `GET|/api/users/123/posts` → ❌ No match (khác số segments)

---

## 7.5. Cache Refresh Implementation

### 7.4.1. Cache Refresh Process

```go
func (m *BaseAuthorizationMiddleware) refreshCache() {
    // Load all rules from database
    rules, err := m.ruleRepo.GetAllRulesForCache()
    if err != nil {
        return  // Log error but don't fail
    }
    
    m.cacheMutex.Lock()  // Write lock - exclusive access
    defer m.cacheMutex.Unlock()
    
    // Rebuild cache structures
    exactRulesMap := make(map[string][]models.Rule)
    patternRulesByMethodAndSegs := make(map[string]map[int][]models.Rule)
    
    for _, rule := range rules {
        if strings.Contains(rule.Path, "*") {
            // Pattern rule: index by method and segment count
            segmentCount := m.countSegments(rule.Path)
            if patternRulesByMethodAndSegs[rule.Method] == nil {
                patternRulesByMethodAndSegs[rule.Method] = make(map[int][]models.Rule)
            }
            patternRulesByMethodAndSegs[rule.Method][segmentCount] = append(
                patternRulesByMethodAndSegs[rule.Method][segmentCount],
                rule,
            )
        } else {
            // Exact rule: index by "METHOD|PATH"
            key := fmt.Sprintf("%s|%s", rule.Method, rule.Path)
            exactRulesMap[key] = append(exactRulesMap[key], rule)
        }
    }
    
    // Atomic update
    m.exactRulesMap = exactRulesMap
    m.patternRulesByMethodAndSegs = patternRulesByMethodAndSegs
    m.lastRefresh = time.Now()
}
```

**Chi tiết:**
- **Thread Safety**: Write lock (`Lock()`) để đảm bảo exclusive access khi refresh
- **Atomic Update**: Update tất cả cache structures cùng lúc
- **Error Handling**: Nếu load rules fail, giữ nguyên cache cũ (không crash)

### 7.4.2. Cache Invalidation

```go
func (m *BaseAuthorizationMiddleware) InvalidateCache() {
    m.refreshCache()  // Force refresh immediately
}
```

**Khi nào gọi:**
- Sau khi `SyncRoutes()` - đồng bộ routes từ code
- Sau khi tạo/update/xóa rule qua API
- Manual refresh khi cần

**Thread Safety:**
- Read operations: `RLock()` - cho phép concurrent reads
- Write operations: `Lock()` - exclusive access
- Refresh: `Lock()` - exclusive access để rebuild cache

---

## 7.6. User ID Generation

### 7.5.1. ID Generation Algorithm

```go
const (
    IDLength = 12
    IDCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func GenerateID() (string, error) {
    bytes := make([]byte, IDLength)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    
    result := make([]byte, IDLength)
    charsetLen := len(IDCharset)
    
    for i := 0; i < IDLength; i++ {
        result[i] = IDCharset[int(bytes[i])%charsetLen]
    }
    
    return string(result), nil
}
```

**Chi tiết:**
- **Length**: 12 ký tự (đủ ngắn cho URL, đủ dài để tránh collision)
- **Character Set**: `a-zA-Z0-9` (62 ký tự)
- **Random Source**: `crypto/rand` (cryptographically secure)
- **Collision Probability**: ~1/62^12 ≈ 1/3.2×10^21 (rất thấp)

**Ví dụ IDs:**
- `aB3xY9mK2pQ1`
- `XyZ7wV4nR8tL`
- `mN5bC6dF9gH2`

---

## 7.7. Tóm tắt Implementation Details

### ✅ Key Implementation Points

1. **JWT Token**
   - Claims structure với RoleIDs được bảo vệ bởi signature
   - Algorithm confusion prevention với explicit method check
   - HMAC-SHA256 signing với secret key

2. **Refresh Token**
   - Dual-token strategy (access + refresh)
   - SHA256 hash trước khi lưu database
   - Cookie-based với HttpOnly + Secure + SameSite
   - Token rotation mỗi lần refresh

3. **Password Hashing**
   - bcrypt với DefaultCost (10 rounds)
   - Tự động salt generation và embedding
   - One-way hashing không thể reverse

4. **Rule Matching**
   - O(1) exact match lookup
   - Optimized pattern matching với nested maps
   - Segment-by-segment matching thay vì regex

5. **Cache Management**
   - Thread-safe với `sync.RWMutex`
   - Atomic cache refresh
   - Manual invalidation sau rule changes

6. **ID Generation**
   - Cryptographically secure random generation
   - 12-character alphanumeric IDs
   - Low collision probability

---

**Xem thêm:**
- [3. Middleware và Security](./03-middleware-security.md) - Luồng xử lý authentication và authorization
- [4. Hệ thống phân quyền](./04-he-thong-phan-quyen.md) - Rule-based authorization và evaluation
- [9. Tối ưu hóa và Best Practices](./09-toi-uu-hoa-best-practices.md) - Performance optimizations
- [Mục lục](./README.md)
