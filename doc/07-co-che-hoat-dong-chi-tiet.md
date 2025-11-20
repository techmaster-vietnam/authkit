# 7. Cơ chế hoạt động chi tiết

Tài liệu này mô tả chi tiết **implementation** của các cơ chế bên trong AuthKit ở mức code, bao gồm cấu trúc dữ liệu, thuật toán và các chi tiết kỹ thuật.

> 📖 **Lưu ý**: Tài liệu này tập trung vào **implementation details** và **code-level explanations**. Để hiểu về **luồng xử lý** và **cách sử dụng**, xem các tài liệu khác:
> - [3. Middleware và Security](./03-middleware-security.md) - Luồng authentication và authorization
> - [4. Hệ thống phân quyền](./04-he-thong-phan-quyen.md) - Rule matching và evaluation

---

## 7.1. JWT Token Implementation

### 7.1.1. Claims Structure

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

---

## 7.2. Password Hashing Implementation

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
│  │  │  └─ Salt (22 chars)            └─ Hash (31 chars)
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

## 7.3. Rule Matching Algorithm Implementation

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

## 7.4. Cache Refresh Implementation

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

## 7.5. User ID Generation

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

## 7.6. Tóm tắt Implementation Details

### ✅ Key Implementation Points

1. **JWT Token**
   - Claims structure với RoleIDs được bảo vệ bởi signature
   - Algorithm confusion prevention với explicit method check
   - HMAC-SHA256 signing với secret key

2. **Password Hashing**
   - bcrypt với DefaultCost (10 rounds)
   - Tự động salt generation và embedding
   - One-way hashing không thể reverse

3. **Rule Matching**
   - O(1) exact match lookup
   - Optimized pattern matching với nested maps
   - Segment-by-segment matching thay vì regex

4. **Cache Management**
   - Thread-safe với `sync.RWMutex`
   - Atomic cache refresh
   - Manual invalidation sau rule changes

5. **ID Generation**
   - Cryptographically secure random generation
   - 12-character alphanumeric IDs
   - Low collision probability

---

**Xem thêm:**
- [3. Middleware và Security](./03-middleware-security.md) - Luồng xử lý authentication và authorization
- [4. Hệ thống phân quyền](./04-he-thong-phan-quyen.md) - Rule-based authorization và evaluation
- [9. Tối ưu hóa và Best Practices](./09-toi-uu-hoa-best-practices.md) - Performance optimizations
- [Mục lục](./README.md)
