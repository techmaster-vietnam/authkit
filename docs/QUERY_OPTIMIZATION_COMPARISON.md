# So sánh các cách tiếp cận: Lấy User và Roles

## Vấn đề hiện tại

Trong hàm `Login` của `AuthService`, có 2 lệnh query:

```go
// Bước 1: Lấy user theo email
user, err := s.userRepo.GetByEmail(req.Email)

// Bước 2: Lấy roles của user
userRoles, err := s.roleRepo.ListRolesOfUser(user.ID)
```

**⚠️ PHÁT HIỆN QUAN TRỌNG:** 
- `GetByEmail()` đã sử dụng `Preload("Roles")` nên `user.Roles` đã được load sẵn
- Lệnh `ListRolesOfUser()` là **THỪA** và không cần thiết!

---

## Cách 1: Hiện tại (2 queries riêng biệt - ĐANG THỪA)

### Code hiện tại:
```go
user, err := s.userRepo.GetByEmail(req.Email)  // Đã Preload Roles
userRoles, err := s.roleRepo.ListRolesOfUser(user.ID)  // THỪA!
```

### Cách GORM thực thi:
1. **Query 1:** `SELECT * FROM users WHERE email = ?` + `SELECT * FROM roles INNER JOIN user_roles ON ... WHERE user_id = ?`
2. **Query 2:** `SELECT * FROM users WHERE id = ?` + `SELECT * FROM roles INNER JOIN user_roles ON ... WHERE user_id = ?` (THỪA!)

### Ưu điểm:
- ❌ Không có ưu điểm nào - đang query thừa

### Nhược điểm:
- ❌ **2 round-trips** đến database (mặc dù query 2 là thừa)
- ❌ **Lãng phí tài nguyên** - query lại data đã có sẵn
- ❌ **Latency cao hơn** - phải chờ 2 queries
- ❌ **Code không tối ưu**

### Performance:
- **Round-trips:** 2 (1 thừa)
- **Network latency:** 2x
- **Database load:** 2x queries

---

## Cách 2: Sử dụng Preload (1 query - TỐI ƯU NHẤT)

### Code đề xuất:
```go
user, err := s.userRepo.GetByEmail(req.Email)  // Đã Preload Roles
// user.Roles đã có sẵn, không cần query thêm!
userRoles := user.Roles
```

### Cách GORM thực thi:
1. **Query 1:** `SELECT * FROM users WHERE email = ?`
2. **Query 2:** `SELECT * FROM roles INNER JOIN user_roles ON roles.id = user_roles.role_id WHERE user_roles.user_id = ?`

**Lưu ý:** GORM Preload vẫn thực hiện 2 queries riêng biệt (không phải JOIN), nhưng trong cùng 1 transaction context.

### Ưu điểm:
- ✅ **Đơn giản** - chỉ cần 1 lệnh gọi
- ✅ **Code sạch** - sử dụng relationship có sẵn
- ✅ **Tận dụng GORM** - không cần viết SQL thủ công
- ✅ **Dễ maintain** - theo chuẩn GORM
- ✅ **Đã có sẵn** trong code hiện tại

### Nhược điểm:
- ⚠️ GORM Preload vẫn thực hiện 2 queries (không phải JOIN)
- ⚠️ Nếu cần tối ưu tuyệt đối, có thể dùng JOIN thủ công

### Performance:
- **Round-trips:** 2 queries (nhưng trong cùng context)
- **Network latency:** ~1x (GORM batch queries)
- **Database load:** 2 queries nhưng tối ưu hơn

---

## Cách 3: Single Query với JOIN (Raw SQL hoặc GORM Joins)

### Code đề xuất:
```go
// Trong UserRepository
func (r *UserRepository) GetByEmailWithRoles(email string) (*models.User, []models.Role, error) {
    var user models.User
    var roles []models.Role
    
    // Option 1: Raw SQL với JOIN
    err := r.db.Raw(`
        SELECT u.*, r.id as role_id, r.name as role_name, r.description as role_description,
               r.is_system as role_is_system, r.created_at as role_created_at, r.updated_at as role_updated_at
        FROM users u
        LEFT JOIN user_roles ur ON u.id = ur.user_id
        LEFT JOIN roles r ON ur.role_id = r.id
        WHERE u.email = ? AND u.deleted_at IS NULL
    `, email).Scan(&user).Error
    
    // Hoặc Option 2: GORM Joins
    err := r.db.Table("users").
        Select("users.*, roles.*").
        Joins("LEFT JOIN user_roles ON users.id = user_roles.user_id").
        Joins("LEFT JOIN roles ON user_roles.role_id = roles.id").
        Where("users.email = ?", email).
        Scan(&user).Error
    
    return &user, roles, err
}
```

### Cách thực thi:
1. **Query duy nhất:** `SELECT ... FROM users LEFT JOIN user_roles ... LEFT JOIN roles ... WHERE email = ?`

### Ưu điểm:
- ✅ **1 round-trip** duy nhất đến database
- ✅ **Latency thấp nhất** - chỉ 1 network call
- ✅ **Tối ưu database** - database có thể optimize JOIN tốt hơn
- ✅ **Giảm network overhead**

### Nhược điểm:
- ❌ **Code phức tạp hơn** - phải map kết quả thủ công
- ❌ **Khó maintain** - phải viết SQL thủ công
- ❌ **Mất tính type-safe** của GORM
- ❌ **Khó debug** - phải xử lý NULL values từ LEFT JOIN
- ❌ **Không tận dụng được GORM relationships**

### Performance:
- **Round-trips:** 1
- **Network latency:** 1x (tốt nhất)
- **Database load:** 1 query với JOIN

---

## Cách 4: Stored Procedure

### Code đề xuất:
```sql
-- Tạo stored procedure
DELIMITER //
CREATE PROCEDURE GetUserWithRoles(IN p_email VARCHAR(255))
BEGIN
    SELECT u.*, 
           JSON_ARRAYAGG(
               JSON_OBJECT(
                   'id', r.id,
                   'name', r.name,
                   'description', r.description,
                   'is_system', r.is_system,
                   'created_at', r.created_at,
                   'updated_at', r.updated_at
               )
           ) as roles
    FROM users u
    LEFT JOIN user_roles ur ON u.id = ur.user_id
    LEFT JOIN roles r ON ur.role_id = r.id
    WHERE u.email = p_email AND u.deleted_at IS NULL
    GROUP BY u.id;
END //
DELIMITER ;
```

```go
// Trong UserRepository
func (r *UserRepository) GetByEmailWithRolesSP(email string) (*models.User, []models.Role, error) {
    var result struct {
        models.User
        RolesJSON string `gorm:"column:roles"`
    }
    
    err := r.db.Raw("CALL GetUserWithRoles(?)", email).Scan(&result).Error
    if err != nil {
        return nil, nil, err
    }
    
    // Parse JSON roles
    var roles []models.Role
    json.Unmarshal([]byte(result.RolesJSON), &roles)
    
    return &result.User, roles, nil
}
```

### Ưu điểm:
- ✅ **1 round-trip** đến database
- ✅ **Logic tập trung** ở database layer
- ✅ **Có thể tối ưu** ở database level (indexes, query plan)
- ✅ **Giảm network traffic** - chỉ 1 call

### Nhược điểm:
- ❌ **Khó maintain** - logic nằm ở database, khó version control
- ❌ **Không portable** - phụ thuộc vào database cụ thể (MySQL/PostgreSQL khác nhau)
- ❌ **Khó test** - phải setup database để test
- ❌ **Khó debug** - phải vào database để debug
- ❌ **Mất tính linh hoạt** - khó thay đổi logic
- ❌ **Phức tạp hơn** - phải parse JSON, handle NULL
- ❌ **Không tận dụng được GORM** - phải viết raw SQL
- ❌ **Migration phức tạp** - phải quản lý stored procedures

### Performance:
- **Round-trips:** 1
- **Network latency:** 1x
- **Database load:** 1 stored procedure call

---

## Bảng so sánh tổng hợp

| Tiêu chí | Cách 1 (Hiện tại - THỪA) | Cách 2 (Preload) | Cách 3 (JOIN) | Cách 4 (Stored Procedure) |
|----------|-------------------------|-----------------|---------------|---------------------------|
| **Số queries** | 2 (1 thừa) | 2 (GORM batch) | 1 | 1 |
| **Round-trips** | 2 | ~1-2 | 1 | 1 |
| **Code đơn giản** | ⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Dễ maintain** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Performance** | ⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Type-safe** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Portable** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ | ⭐⭐ |
| **Tận dụng GORM** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐ | ⭐ |
| **Debug dễ** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |

---

## Kết luận và Khuyến nghị

### 🎯 Khuyến nghị: **Cách 2 - Sử dụng Preload (đã có sẵn)**

**Lý do:**
1. ✅ Code đã có sẵn `Preload("Roles")` trong `GetByEmail()`
2. ✅ Đơn giản nhất - chỉ cần xóa dòng code thừa
3. ✅ Dễ maintain và debug
4. ✅ Performance đã tốt với GORM batch queries
5. ✅ Type-safe và portable
6. ✅ Theo best practices của GORM

### Code đề xuất sửa:

```go
// TRƯỚC (SAI - đang query thừa):
user, err := s.userRepo.GetByEmail(req.Email)
if err != nil {
    // handle error
}

userRoles, err := s.roleRepo.ListRolesOfUser(user.ID)  // ❌ THỪA!
if err != nil {
    // handle error
}

// SAU (ĐÚNG - sử dụng data đã có):
user, err := s.userRepo.GetByEmail(req.Email)
if err != nil {
    // handle error
}

// user.Roles đã được load sẵn từ Preload!
userRoles := user.Roles  // ✅ Sử dụng data có sẵn

// Extract role IDs
roleIDs := make([]uint, 0, len(userRoles))
for _, role := range userRoles {
    roleIDs = append(roleIDs, role.ID)
}
```

### Khi nào nên dùng Cách 3 (JOIN) hoặc Cách 4 (Stored Procedure)?

Chỉ nên xem xét khi:
- ⚠️ **Performance là ưu tiên số 1** và đã đo được bottleneck thực sự
- ⚠️ **Scale lớn** - hàng triệu requests/giây
- ⚠️ **Network latency rất cao** (cross-region database)
- ⚠️ **Đã profile và xác định** đây là bottleneck thực sự

**Lưu ý:** Với hầu hết ứng dụng, sự khác biệt performance giữa Cách 2 và Cách 3/4 là **không đáng kể** (< 10ms), nhưng chi phí maintain lại cao hơn nhiều.

---

## Action Items

1. ✅ **Ngay lập tức:** Xóa dòng `ListRolesOfUser()` thừa trong `auth_service.go`
2. ✅ Sử dụng `user.Roles` trực tiếp (đã được Preload)
3. ⚠️ **Tùy chọn:** Nếu cần tối ưu hơn nữa, cân nhắc Cách 3 (JOIN) sau khi đã đo được bottleneck thực sự

