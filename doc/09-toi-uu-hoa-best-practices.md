# 9. Tối ưu hóa Authorization Performance

Tài liệu này tập trung vào **tối ưu tốc độ authorization** - hoạt động chạy liên tục trên mỗi request. Authorization middleware là hot path quan trọng nhất trong AuthKit.

> 📖 **Lưu ý**: Để hiểu về implementation details, xem [7. Cơ chế hoạt động chi tiết](./07-co-che-hoat-dong-chi-tiet.md). Để hiểu về caching strategy, xem [3. Middleware và Security](./03-middleware-security.md).

---

## 9.1. Authorization Flow - Hot Path Analysis

Authorization middleware chạy trên **mỗi request** đến protected endpoints. Đây là luồng xử lý:

```go
func (m *BaseAuthorizationMiddleware) Authorize() fiber.Handler {
    return func(c *fiber.Ctx) error {
        // 1. Find matching rules (O(1) hoặc optimized pattern matching)
        matchingRules := m.findMatchingRules(method, path)
        
        // 2. Early exit: No rule → Deny
        if len(matchingRules) == 0 { return deny }
        
        // 3. Early exit: PUBLIC rule → Allow
        if hasPublicRule(matchingRules) { return allow }
        
        // 4. Get role IDs from JWT token (no DB query!)
        roleIDs := GetRoleIDsFromContext(c)
        
        // 5. Early exit: super_admin → Allow
        if isSuperAdmin(roleIDs) { return allow }
        
        // 6. Check FORBID rules (priority)
        // 7. Check ALLOW rules
    }
}
```

**Performance Critical Points:**
- ✅ Rule lookup: Phải O(1) hoặc gần O(1)
- ✅ Role IDs: Không được query DB
- ✅ Early exits: Phải check sớm nhất có thể
- ✅ Cache: Phải thread-safe và fast

---

## 9.2. Tối ưu Rule Matching

### 9.2.1. Exact Match - O(1) Lookup

**Implementation:**
```go
// O(1) lookup với map
key := fmt.Sprintf("%s|%s", method, path)
exactMatches, hasExactMatch := m.exactRulesMap[key]
if hasExactMatch && len(exactMatches) > 0 {
    return exactMatches  // Early exit, không check patterns
}
```

**Performance:**
- **Time Complexity**: O(1) - constant time lookup
- **Space Complexity**: O(n) - n là số exact rules
- **Best Case**: Hầu hết routes là exact matches → O(1) cho mọi request

**Best Practice:**
- ✅ Ưu tiên exact routes thay vì pattern routes khi có thể
- ✅ Ví dụ: `/api/users` tốt hơn `/api/users/*` nếu không cần dynamic ID

### 9.2.2. Pattern Matching - Optimized với Nested Maps

**Implementation:**
```go
// Filter 1: By method
methodPatterns := m.patternRulesByMethodAndSegs[method]

// Filter 2: By segment count
pathSegments := m.countSegments(path)
rulesToCheck := methodPatterns[pathSegments]

// Filter 3: Match từng rule
for _, rule := range rulesToCheck {
    if m.matchPath(rule.Path, path) {
        matches = append(matches, rule)
    }
}
```

**Performance:**
- **Time Complexity**: O(k) với k << n (k là số rules cùng method và segment count)
- **Optimization**: Filter theo method trước → giảm 90%+ rules cần check
- **Optimization**: Filter theo segment count → giảm thêm 80%+ rules

**Best Practice:**
- ✅ Sử dụng pattern routes chỉ khi cần thiết (dynamic IDs)
- ✅ Tránh quá nhiều pattern routes → tăng số rules cần iterate

### 9.2.3. Segment Counting - Zero Allocation

**Implementation:**
```go
func countSegments(path string) int {
    if len(path) == 0 || path == "/" {
        return 0
    }
    start := 0
    if path[0] == '/' {
        start = 1  // Skip leading slash
    }
    return strings.Count(path[start:], "/") + 1
}
```

**Performance:**
- **Zero Allocation**: Không tạo slice hay string mới
- **Fast**: Chỉ đếm ký tự `/` trong path
- **O(n)**: n là độ dài path (thường < 100 chars)

---

## 9.3. Tối ưu Role Checking

### 9.3.1. Role IDs từ JWT Token - Zero DB Query

**Critical Optimization:**
```go
// ✅ Tốt: Role IDs từ JWT token (đã validated)
roleIDs, ok := GetRoleIDsFromContext(c)
if !ok {
    // Fallback: Query DB (chỉ khi cần)
    userRoles, _ := m.roleRepo.ListRolesOfUser(userID)
    // ...
}
```

**Performance Impact:**
- **Without JWT**: Mỗi request → 1 DB query để lấy roles
- **With JWT**: Zero DB queries → **100x faster** (DB query ~1-5ms vs memory lookup ~0.001ms)

**Best Practice:**
- ✅ Luôn đảm bảo role IDs được lưu trong JWT token
- ✅ Validate token signature để đảm bảo role IDs không bị tamper
- ✅ Không query DB để lấy roles nếu đã có trong token

### 9.3.2. Role ID Map - O(1) Lookup

**Implementation:**
```go
// Convert role IDs array → map for O(1) lookup
userRoleIDs := make(map[uint]bool, len(roleIDs))
for _, roleID := range roleIDs {
    userRoleIDs[roleID] = true
}

// Check role: O(1) lookup
if userRoleIDs[roleID] {
    // User has this role
}
```

**Performance:**
- **Array lookup**: O(n) - phải iterate qua tất cả roles
- **Map lookup**: O(1) - constant time
- **Impact**: Với 10 roles → 10x faster

**Best Practice:**
- ✅ Luôn convert role IDs array → map trước khi check
- ✅ Pre-allocate map với capacity: `make(map[uint]bool, len(roleIDs))`

### 9.3.3. super_admin Cache - O(1) Check

**Implementation:**
```go
// Cache super_admin ID khi khởi động
superAdminID := m.getSuperAdminID()  // O(1) lookup từ cache

// Check super_admin: O(1)
if superAdminID != nil && userRoleIDs[*superAdminID] {
    return c.Next()  // Early exit, bypass all rules
}
```

**Performance:**
- **Without cache**: Mỗi request → query DB để check super_admin
- **With cache**: O(1) memory lookup → **1000x faster**

---

## 9.4. Early Exit Patterns

Early exits là kỹ thuật quan trọng nhất để tối ưu authorization:

### 9.4.1. Early Exit Order (từ nhanh nhất đến chậm nhất)

```go
// 1. No rule → Deny (O(1) check)
if len(matchingRules) == 0 { return deny }

// 2. PUBLIC rule → Allow (O(k) với k = số rules, thường k=1)
if hasPublicRule(matchingRules) { return allow }

// 3. No user → Deny (O(1) check)
if user == nil { return deny }

// 4. super_admin → Allow (O(1) check với cache)
if isSuperAdmin(roleIDs) { return allow }

// 5. FORBID rules (O(k) với k = số FORBID rules)
// 6. ALLOW rules (O(k) với k = số ALLOW rules)
```

**Performance Impact:**
- **PUBLIC routes**: Chỉ cần 2 checks (no rule, PUBLIC) → ~0.01ms
- **super_admin routes**: Chỉ cần 4 checks → ~0.02ms
- **Normal routes**: Cần check tất cả rules → ~0.1-1ms

### 9.4.2. Rule Evaluation Order

**Priority Order:**
1. **PUBLIC** (highest priority) - Check đầu tiên
2. **super_admin** - Check sau PUBLIC
3. **FORBID** - Check trước ALLOW
4. **ALLOW** - Check cuối cùng

**Lý do:**
- PUBLIC và super_admin có thể early exit → check sớm nhất
- FORBID có priority cao hơn ALLOW → check trước
- Nếu user bị FORBID → không cần check ALLOW

---

## 9.5. Cache Management

### 9.5.1. Cache Structure Optimization

**Rules Cache:**
```go
// Exact rules: O(1) lookup
exactRulesMap map[string][]models.Rule  // "METHOD|PATH" → Rules

// Pattern rules: Optimized nested map
patternRulesByMethodAndSegs map[string]map[int][]models.Rule
// method → segmentCount → Rules
```

**Role Cache:**
```go
superAdminID *uint                    // Cached super_admin ID
roleNameToIDMap map[string]uint      // Role name → ID mapping
```

**Memory Usage:**
- Rules cache: ~1-10MB (tùy số rules)
- Role cache: ~1KB (chỉ vài roles)
- **Trade-off**: Memory nhỏ để đổi lấy tốc độ lookup cực nhanh

### 9.5.2. Thread Safety - RWMutex

**Implementation:**
```go
// Read lock: Cho phép concurrent reads
m.cacheMutex.RLock()
defer m.cacheMutex.RUnlock()
rules := m.exactRulesMap[key]

// Write lock: Exclusive access khi refresh
m.cacheMutex.Lock()
defer m.cacheMutex.Unlock()
m.exactRulesMap = newRulesMap
```

**Performance:**
- **Read lock**: Cho phép nhiều goroutines đọc cùng lúc
- **Write lock**: Chặn tất cả reads khi refresh
- **Impact**: Concurrent requests không block nhau khi đọc cache

**Best Practice:**
- ✅ Sử dụng RWMutex thay vì Mutex để cho phép concurrent reads
- ✅ Refresh cache ngoài giờ cao điểm nếu có thể
- ✅ Atomic update: Update tất cả cache structures cùng lúc

### 9.5.3. Cache Invalidation Strategy

**Khi nào refresh cache:**
- Sau khi `SyncRoutes()` - đồng bộ routes từ code
- Sau khi tạo/update/xóa rule qua API
- Manual refresh: `InvalidateCache()`

**Best Practice:**
```go
// ✅ Đúng: Refresh cache sau khi sync routes
ak.SyncRoutes()
ak.InvalidateCache()

// ✅ Đúng: Refresh cache sau khi update rule
ruleHandler.UpdateRule(c)
authorizationMiddleware.InvalidateCache()
```

---

## 9.6. Performance Benchmarks

### 9.6.1. Typical Performance (per request)

| Operation | Time | Notes |
|-----------|------|-------|
| Exact rule lookup | ~0.001ms | O(1) map lookup |
| Pattern rule lookup | ~0.01-0.1ms | O(k) với k << n |
| Role ID check (from JWT) | ~0.001ms | O(1) map lookup |
| Role ID check (from DB) | ~1-5ms | DB query + network |
| super_admin check | ~0.001ms | O(1) cached lookup |
| **Total (optimized)** | **~0.01-0.1ms** | Với JWT token |
| **Total (unoptimized)** | **~5-10ms** | Query DB mỗi request |

### 9.6.2. Throughput Impact

**Với optimization:**
- ~10,000-100,000 requests/second (tùy hardware)
- CPU-bound, không phụ thuộc DB

**Không optimization:**
- ~100-1,000 requests/second (bị giới hạn bởi DB)
- DB-bound, bottleneck ở database queries

---

## 9.7. Best Practices Summary

### ✅ Do's

1. **Luôn sử dụng JWT token với role IDs**
   - Zero DB queries cho role checking
   - 100x faster than DB queries

2. **Ưu tiên exact routes thay vì pattern routes**
   - O(1) lookup vs O(k) pattern matching
   - Faster và đơn giản hơn

3. **Convert role IDs array → map trước khi check**
   - O(1) lookup vs O(n) array iteration
   - 10x faster với nhiều roles

4. **Refresh cache sau khi thay đổi rules**
   - Đảm bảo cache luôn up-to-date
   - Tránh stale data

5. **Sử dụng RWMutex cho cache**
   - Cho phép concurrent reads
   - Tăng throughput

### ❌ Don'ts

1. **Không query DB để lấy roles nếu đã có trong JWT**
   - Chậm hơn 100x
   - Tạo bottleneck ở database

2. **Không check roles bằng array iteration**
   - Chậm hơn 10x với nhiều roles
   - Luôn convert sang map

3. **Không refresh cache quá thường xuyên**
   - Write lock block tất cả reads
   - Chỉ refresh khi cần thiết

4. **Không tạo quá nhiều pattern routes**
   - Tăng số rules cần iterate
   - Giảm performance của pattern matching

---

**Xem thêm:**
- [3. Middleware và Security](./03-middleware-security.md) - Caching strategy và early exits
- [7. Cơ chế hoạt động chi tiết](./07-co-che-hoat-dong-chi-tiet.md) - Implementation details của rule matching
- [8. Tích hợp và Sử dụng](./08-tich-hop-su-dung.md) - Best practices khi tích hợp
- [Mục lục](./README.md)
