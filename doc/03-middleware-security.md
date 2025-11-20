# 3. Middleware và Security

Tài liệu này mô tả cách AuthKit bảo vệ ứng dụng của bạn thông qua hai lớp middleware: **Authentication** (xác thực người dùng) và **Authorization** (kiểm tra quyền truy cập).

> 🔐 **Authentication** = "Bạn là ai?" (Who are you?)  
> 🛡️ **Authorization** = "Bạn được phép làm gì?" (What can you do?)

---

## 3.1. Authentication Middleware - Người gác cổng

Authentication Middleware là lớp bảo vệ đầu tiên, đảm bảo chỉ những người dùng đã đăng nhập mới có thể truy cập vào các endpoint được bảo vệ.

### 3.1.1. Luồng xử lý Authentication

```mermaid
sequenceDiagram
    participant C as Client
    participant AM as AuthMiddleware
    participant JWT as JWT Validator
    participant DB as Database
    participant CTX as Context
    
    C->>AM: HTTP Request<br/>Authorization: Bearer <token>
    AM->>AM: Extract Token<br/>(Header hoặc Cookie)
    
    alt No Token
        AM-->>C: ❌ 401 Unauthorized<br/>"Token không được cung cấp"
    end
    
    AM->>JWT: ValidateToken(token, secret)
    JWT->>JWT: Verify Signature<br/>Verify Expiration<br/>Check Algorithm (HMAC-SHA256)
    
    alt Invalid Token
        JWT-->>AM: Error
        AM-->>C: ❌ 401 Unauthorized<br/>"Token không hợp lệ"
    end
    
    alt Valid Token
        JWT-->>AM: Claims {userID, email, roleIDs}
        AM->>DB: GetUserByID(userID)
        DB-->>AM: User object
        
        alt User Inactive
            AM-->>C: ❌ 403 Forbidden<br/>"Tài khoản đã bị vô hiệu hóa"
        end
        
        alt User Active
            AM->>CTX: Store in Context<br/>user, userID, roleIDs
            AM->>AM: Next() → Continue
        end
    end
```

### 3.1.2. Các bước chi tiết

#### **Bước 1: Extract Token** 🔍
Middleware tìm JWT token từ:
- **Ưu tiên**: `Authorization: Bearer <token>` header
- **Fallback**: Cookie `token`

```go
// Ví dụ header
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### **Bước 2: Validate Token** ✅
Token được validate với các kiểm tra:

1. **Parse Token**: Giải mã JWT token thành claims
2. **Verify Signature**: Kiểm tra chữ ký HMAC-SHA256
   - ⚠️ **Quan trọng**: Chỉ chấp nhận HMAC-SHA256 để prevent algorithm confusion attacks
3. **Check Expiration**: Kiểm tra token còn hạn không
4. **Extract Claims**: Lấy `userID`, `email`, `roleIDs` từ token

**Claims trong JWT Token:**
```json
{
  "user_id": "abc123xyz",
  "email": "user@example.com",
  "role_ids": [1, 2, 3],
  "exp": 1234567890,
  "iat": 1234567890
}
```

#### **Bước 3: Load User** 👤
- Query database để load user bằng `userID` từ claims
- Kiểm tra `user.IsActive()` - reject nếu tài khoản bị vô hiệu hóa

#### **Bước 4: Store in Context** 💾
Lưu thông tin vào Fiber context để các middleware/handler sau sử dụng:

```go
c.Locals("user", user)        // User object
c.Locals("userID", userID)    // User ID string
c.Locals("roleIDs", roleIDs)  // Role IDs từ JWT token
```

### 3.1.3. Bảo mật Role IDs trong Token

**Vấn đề**: Role IDs được lưu trong JWT token - liệu có an toàn không?

**Giải pháp**: ✅ **Rất an toàn** vì:

1. **Role IDs được bảo vệ bởi HMAC signature**
   - Token được ký bằng secret key
   - Nếu hacker modify `role_ids` → signature không match → token invalid

2. **Không cần query database**
   - Role IDs đã có sẵn trong token (sau khi validate)
   - Giảm tải database, tăng hiệu suất

3. **Algorithm confusion prevention**
   - Chỉ chấp nhận HMAC-SHA256
   - Reject các algorithm khác (RSA, ECDSA, etc.)

```mermaid
graph LR
    A[Client sends Token] -->|1. Extract| B[AuthMiddleware]
    B -->|2. Validate| C{JWT Validator}
    C -->|Signature OK?| D[Yes: Extract roleIDs]
    C -->|Signature FAIL| E[❌ Reject]
    D -->|3. Store| F[Context: roleIDs]
    F -->|4. Continue| G[Next Middleware]
    
    style D fill:#90EE90
    style E fill:#FFB6C1
```

---

## 3.2. Authorization Middleware - Người kiểm tra quyền

Sau khi đã xác thực người dùng, Authorization Middleware kiểm tra xem người dùng có quyền truy cập endpoint cụ thể không.

### 3.2.1. Luồng xử lý Authorization

```mermaid
flowchart TD
    Start([Request đến]) --> GetRules[Find Matching Rules<br/>from Cache]
    GetRules -->|No Rule| Deny1[❌ 403 Forbidden<br/>Default Deny]
    
    GetRules -->|Has Rules| CheckPublic{Has PUBLIC Rule?}
    CheckPublic -->|Yes| Allow1[✅ Allow<br/>Anonymous OK]
    
    CheckPublic -->|No| CheckAuth{User in Context?}
    CheckAuth -->|No| Deny2[❌ 401 Unauthorized<br/>Need Login]
    
    CheckAuth -->|Yes| GetRoles[Get Role IDs<br/>from Context]
    GetRoles --> CheckSuperAdmin{Is super_admin?}
    CheckSuperAdmin -->|Yes| Allow2[✅ Allow<br/>Bypass All Rules]
    
    CheckSuperAdmin -->|No| CheckForbid{Has FORBIDE Rule?}
    CheckForbid -->|Yes| CheckForbidRoles{User có<br/>Forbidden Role?}
    CheckForbidRoles -->|Yes| Deny3[❌ 403 Forbidden]
    CheckForbidRoles -->|No| CheckAllow{Has ALLOW Rule?}
    
    CheckForbid -->|No| CheckAllow
    CheckAllow -->|Yes| CheckAllowRoles{User có<br/>Allowed Role?}
    CheckAllowRoles -->|Yes| Allow3[✅ Allow]
    CheckAllowRoles -->|No| Deny4[❌ 403 Forbidden]
    
    CheckAllow -->|No| Deny5[❌ 403 Forbidden<br/>No Matching Rule]
    
    style Allow1 fill:#90EE90
    style Allow2 fill:#90EE90
    style Allow3 fill:#90EE90
    style Deny1 fill:#FFB6C1
    style Deny2 fill:#FFB6C1
    style Deny3 fill:#FFB6C1
    style Deny4 fill:#FFB6C1
    style Deny5 fill:#FFB6C1
```

### 3.2.2. Rule Matching Algorithm

Authorization Middleware tìm rules phù hợp với request:

#### **1. Exact Match (O(1) lookup)**
Tìm rule chính xác với `"METHOD|PATH"`:

```
Request: GET /api/users
Lookup: "GET|/api/users" → Rule found ✅
```

#### **2. Pattern Match (Wildcard)**
Nếu không có exact match, thử pattern matching với wildcard `*`:

```
Request: GET /api/users/123
Pattern: "GET|/api/users/*" → Match ✅
```

**Ví dụ pattern matching:**
- `GET|/api/users/*` matches `GET|/api/users/123`
- `GET|/api/blogs/*/comments` matches `GET|/api/blogs/1/comments`
- `*` matches bất kỳ segment nào

### 3.2.3. Rule Evaluation Order

Rules được đánh giá theo thứ tự ưu tiên:

1. **PUBLIC** (Early Exit) 🟢
   - Cho phép anonymous users
   - Không cần authentication
   - Ví dụ: Login, Register endpoints

2. **super_admin** (Early Exit) ⭐
   - Bypass tất cả rules
   - Chỉ cần có role super_admin
   - Emergency access, system administrators

3. **FORBIDE** (Ưu tiên cao) 🔴
   - Cấm các roles được chỉ định
   - Kiểm tra trước ALLOW rules
   - Ví dụ: `Forbid("guest")` → cấm role guest

4. **ALLOW** (Kiểm tra sau) 🟡
   - Cho phép các roles được chỉ định
   - Nếu roles rỗng → mọi user đã đăng nhập đều được
   - Ví dụ: `Allow("admin", "editor")` → chỉ admin và editor

### 3.2.4. X-Role-Context Header

Tính năng đặc biệt: Cho phép user giả lập một role cụ thể (nếu user có role đó):

```http
GET /api/users
Authorization: Bearer <token>
X-Role-Context: editor
```

**Cách hoạt động:**
- User có roles: `[admin, editor, viewer]`
- Gửi request với `X-Role-Context: editor`
- Authorization chỉ kiểm tra với role `editor` (tạm thời ignore các roles khác)
- Use case: Testing, role switching trong multi-tenant systems

---

## 3.3. Cơ chế Cache và Tối ưu Hiệu suất

Authorization Middleware sử dụng caching để tối ưu hiệu suất, tránh query database mỗi request.

### 3.3.1. Rules Cache Structure

```mermaid
graph TB
    subgraph "Rules Cache"
        ERM[exactRulesMap<br/>METHOD|PATH → Rules<br/>O(1) lookup]
        PRM[patternRulesByMethodAndSegs<br/>method → segmentCount → Rules<br/>Optimized pattern matching]
    end
    
    subgraph "Role Cache"
        SAID[superAdminID<br/>Cached super_admin ID<br/>O(1) check]
        RNID[roleNameToIDMap<br/>role name → ID<br/>For X-Role-Context]
    end
    
    DB[(Database)] -->|Load All Rules| ERM
    DB -->|Load All Rules| PRM
    DB -->|Load super_admin| SAID
    DB -->|Load All Roles| RNID
    
    Request -->|Lookup| ERM
    Request -->|Pattern Match| PRM
    Authz -->|Check| SAID
    Authz -->|Convert| RNID
```

### 3.3.2. Cache Refresh Strategy

**Khi nào cache được refresh?**

1. **Khởi động**: Load tất cả rules và roles khi middleware được khởi tạo
2. **Manual refresh**: Gọi `InvalidateCache()` sau khi:
   - Sync routes (`ak.SyncRoutes()`)
   - Update rules từ API
   - Thay đổi roles

**Cache TTL**: 5 minutes (có thể refresh manual bất cứ lúc nào)

### 3.3.3. Tối ưu hiệu suất

**1. Early Exit Patterns** ⚡
- PUBLIC rule → Allow ngay (không cần check roles)
- super_admin → Bypass ngay (không cần check rules)
- No rule → Deny ngay (không cần check gì)

**2. O(1) Lookups** ⚡
- Exact rules: Map lookup `O(1)`
- Pattern rules: Organized theo method và segment count
- Role IDs: Từ JWT token (không cần DB query)

**3. Batch Operations** ⚡
- Load tất cả rules một lần
- Batch convert role names → IDs khi sync routes
- Cache role name → ID mapping

**4. Thread Safety** 🔒
- `sync.RWMutex` để protect cache
- Read lock cho lookups (concurrent reads OK)
- Write lock cho refresh (exclusive)

---

## 3.4. Tóm tắt Security Best Practices

### ✅ Authentication Security

- **JWT Signature**: Role IDs được bảo vệ bởi HMAC signature
- **Algorithm Verification**: Chỉ chấp nhận HMAC-SHA256
- **Token Expiration**: Tokens có thời hạn để giảm risk nếu bị leak
- **User Active Check**: Reject inactive users ngay lập tức

### ✅ Authorization Security

- **Default Deny**: Không có rule → FORBIDE (security first)
- **Rule Priority**: FORBIDE rules có ưu tiên cao hơn ALLOW
- **super_admin Protection**: Không thể tạo/gán qua API
- **Role Context Validation**: X-Role-Context phải được validate

### ✅ Performance Security

- **Caching**: Rules được cache để tránh DB queries
- **Early Exits**: Tối ưu với early exit patterns
- **Thread Safety**: Concurrent-safe với mutex
- **No DB Queries**: Role IDs từ JWT token (đã validated)

---

**Xem thêm:**
- [2. Kiến trúc tổng thể](./02-kien-truc-tong-the.md) - Hiểu cách middleware tích hợp vào kiến trúc
- [4. Hệ thống phân quyền](./04-he-thong-phan-quyen.md) - Chi tiết về rules và roles
- [7. Cơ chế hoạt động chi tiết](./07-co-che-hoat-dong-chi-tiet.md) - JWT token generation và rule matching algorithm
- [9. Tối ưu hóa và Best Practices](./09-toi-uu-hoa-best-practices.md) - Security best practices chi tiết
- [Mục lục](./README.md)
