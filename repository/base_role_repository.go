package repository

import (
	"fmt"
	"strings"
	"sync"

	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// BaseRoleRepository là generic repository cho Role models
// T phải implement RoleInterface
type BaseRoleRepository[T core.RoleInterface] struct {
	db *gorm.DB
	
	// Cache để tối ưu việc chuyển đổi giữa role_id và role_name
	cache struct {
		sync.RWMutex
		idToName map[uint]string // role_id -> role_name
		nameToID map[string]uint  // role_name -> role_id
	}
}

// NewBaseRoleRepository tạo mới BaseRoleRepository với generic type
func NewBaseRoleRepository[T core.RoleInterface](db *gorm.DB) *BaseRoleRepository[T] {
	repo := &BaseRoleRepository[T]{
		db: db,
	}
	// Khởi tạo cache maps
	repo.cache.idToName = make(map[uint]string)
	repo.cache.nameToID = make(map[string]uint)
	
	// Load cache khi khởi tạo
	if err := repo.LoadRoleCache(); err != nil {
		// Log error nhưng không fail initialization
		// Cache sẽ được load lại khi cần thiết
		_ = err
	}
	
	return repo
}

// Create tạo mới role
func (r *BaseRoleRepository[T]) Create(role T) error {
	// Sử dụng Select để chỉ định rõ các fields cần insert, bao gồm cả ID
	// Điều này đảm bảo ID được insert vào database ngay cả khi không phải auto-increment
	if err := r.db.Select("id", "name", "is_system").Create(&role).Error; err != nil {
		return err
	}
	
	// Cập nhật cache sau khi tạo role thành công
	r.updateCache(role.GetID(), role.GetName())
	
	return nil
}

// GetByID lấy role theo ID
func (r *BaseRoleRepository[T]) GetByID(id uint) (T, error) {
	var role T
	err := r.db.Where("id = ?", id).First(&role).Error
	return role, err
}

// GetByName lấy role theo name
func (r *BaseRoleRepository[T]) GetByName(name string) (T, error) {
	var role T
	err := r.db.Where("name = ?", name).First(&role).Error
	return role, err
}

// GetByIDs lấy roles theo IDs (batch query)
func (r *BaseRoleRepository[T]) GetByIDs(ids []uint) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}
	var roles []T
	err := r.db.Where("id IN ?", ids).Find(&roles).Error
	return roles, err
}

// Update cập nhật role
func (r *BaseRoleRepository[T]) Update(role T) error {
	// Lấy role cũ để xóa khỏi cache nếu name thay đổi
	oldRole, err := r.GetByID(role.GetID())
	if err == nil {
		oldName := oldRole.GetName()
		newName := role.GetName()
		
		// Nếu name thay đổi, cần xóa entry cũ trong cache
		if oldName != newName {
			r.cache.Lock()
			delete(r.cache.nameToID, oldName)
			r.cache.Unlock()
		}
	}
	
	if err := r.db.Save(&role).Error; err != nil {
		return err
	}
	
	// Cập nhật cache sau khi update thành công
	r.updateCache(role.GetID(), role.GetName())
	
	return nil
}

// Delete hard delete role (chỉ nếu không phải system role)
// Sử dụng stored procedure để đảm bảo tính nhất quán dữ liệu:
// 1. Xóa khỏi bảng user_roles
// 2. Xóa role_id khỏi mảng rules.roles
// 3. Xóa khỏi bảng roles
func (r *BaseRoleRepository[T]) Delete(id uint) error {
	var role T
	if err := r.db.First(&role, id).Error; err != nil {
		return err
	}
	if role.IsSystem() {
		return gorm.ErrRecordNotFound // Cannot delete system roles
	}

	roleName := role.GetName()

	// Gọi stored procedure để xóa role và dọn dẹp dữ liệu liên quan
	if err := r.db.Exec("SELECT delete_role(?)", id).Error; err != nil {
		return err
	}
	
	// Xóa khỏi cache sau khi xóa thành công
	r.removeFromCache(id, roleName)
	
	return nil
}

// List lấy danh sách tất cả roles
func (r *BaseRoleRepository[T]) List() ([]T, error) {
	var roles []T
	err := r.db.Find(&roles).Error
	return roles, err
}

// AddRoleToUser thêm role cho user
// Note: Method này vẫn sử dụng models.BaseUser vì cần làm việc với many2many relationship
// Sử dụng PostgreSQL UPSERT (ON CONFLICT DO NOTHING) để tối ưu hiệu suất
func (r *BaseRoleRepository[T]) AddRoleToUser(userID string, roleID uint) error {
	var user models.BaseUser
	var role T

	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}

	// Sử dụng PostgreSQL UPSERT: nếu (user_id, role_id) đã tồn tại thì không làm gì
	// PRIMARY KEY constraint trên (user_id, role_id) sẽ tự động xử lý conflict
	return r.db.Exec(
		"INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2) ON CONFLICT (user_id, role_id) DO NOTHING",
		userID, roleID,
	).Error
}

// RemoveRoleFromUser xóa role khỏi user
func (r *BaseRoleRepository[T]) RemoveRoleFromUser(userID string, roleID uint) error {
	var user models.BaseUser
	var role T

	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}

	// Kiểm tra user có role đó hay không trước khi xóa
	var count int64
	err := r.db.Table("user_roles").
		Where("user_id = ? AND role_id = ?", userID, roleID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}

	return r.db.Model(&user).Association("Roles").Delete(&role)
}

// CheckUserHasRole kiểm tra user có role cụ thể không
func (r *BaseRoleRepository[T]) CheckUserHasRole(userID string, roleName string) (bool, error) {
	var count int64
	err := r.db.Table("user_roles").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, roleName).
		Count(&count).Error
	return count > 0, err
}

// ListRolesOfUser lấy danh sách roles của user
// Query trực tiếp roles từ database thông qua JOIN với user_roles
// để đảm bảo GORM tự động scan vào đúng type T
func (r *BaseRoleRepository[T]) ListRolesOfUser(userID string) ([]T, error) {
	// Kiểm tra user có tồn tại không
	var user models.BaseUser
	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return nil, err
	}

	// Query trực tiếp roles từ bảng roles thông qua JOIN với user_roles
	// Sử dụng type []T để GORM tự động scan vào đúng type
	var roles []T
	err := r.db.Table("roles").
		Joins("JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error
	if err != nil {
		return nil, err
	}

	// Đảm bảo luôn trả về empty slice thay vì nil
	if roles == nil {
		return []T{}, nil
	}

	return roles, nil
}

// ListUsersHasRole lấy danh sách users có role cụ thể
// Trả về []interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) ListUsersHasRole(roleName string) ([]interface{}, error) {
	var users []models.BaseUser
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("roles.name = ?", roleName).
		Preload("Roles").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	// Convert []models.BaseUser sang []interface{}
	result := make([]interface{}, len(users))
	for i := range users {
		result[i] = users[i]
	}
	return result, nil
}

// ListUsersHasRoleId lấy danh sách users có role theo ID
// Trả về []interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) ListUsersHasRoleId(roleID uint) ([]interface{}, error) {
	var users []models.BaseUser
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Where("user_roles.role_id = ?", roleID).
		Preload("Roles").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	// Convert []models.BaseUser sang []interface{}
	result := make([]interface{}, len(users))
	for i := range users {
		result[i] = users[i]
	}
	return result, nil
}

// ListUsersHasRoleName lấy danh sách users có role theo tên
// Trả về []interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) ListUsersHasRoleName(roleName string) ([]interface{}, error) {
	var users []models.BaseUser
	err := r.db.Table("users").
		Joins("JOIN user_roles ON users.id = user_roles.user_id").
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("roles.name = ?", roleName).
		Preload("Roles").
		Find(&users).Error
	if err != nil {
		return nil, err
	}
	// Convert []models.BaseUser sang []interface{}
	result := make([]interface{}, len(users))
	for i := range users {
		result[i] = users[i]
	}
	return result, nil
}

// GetIDsByNames lấy role IDs theo role names (batch query)
// Tối ưu: sử dụng cache trước, chỉ query DB nếu không có trong cache
func (r *BaseRoleRepository[T]) GetIDsByNames(names []string) (map[string]uint, error) {
	if len(names) == 0 {
		return make(map[string]uint), nil
	}
	
	result := make(map[string]uint)
	missingNames := make([]string, 0)
	
	// Kiểm tra cache trước
	r.cache.RLock()
	for _, name := range names {
		if id, exists := r.cache.nameToID[name]; exists {
			result[name] = id
		} else {
			missingNames = append(missingNames, name)
		}
	}
	r.cache.RUnlock()
	
	// Nếu tất cả đều có trong cache, trả về ngay
	if len(missingNames) == 0 {
		return result, nil
	}
	
	// Query DB cho các names không có trong cache
	var roles []T
	err := r.db.Where("name IN ?", missingNames).Find(&roles).Error
	if err != nil {
		return nil, err
	}
	
	// Cập nhật cache và result với kết quả từ DB
	r.cache.Lock()
	for _, role := range roles {
		id := role.GetID()
		name := role.GetName()
		r.cache.idToName[id] = name
		r.cache.nameToID[name] = id
		result[name] = id
	}
	r.cache.Unlock()
	
	return result, nil
}

// GetNamesByIDs lấy role names theo role IDs từ cache (không query DB)
// Trả về map role_id -> role_name
func (r *BaseRoleRepository[T]) GetNamesByIDs(ids []uint) map[uint]string {
	if len(ids) == 0 {
		return make(map[uint]string)
	}
	
	result := make(map[uint]string)
	r.cache.RLock()
	for _, id := range ids {
		if name, exists := r.cache.idToName[id]; exists {
			result[id] = name
		}
	}
	r.cache.RUnlock()
	
	return result
}

// UpdateUserRoles cập nhật toàn bộ roles của user
// Xóa tất cả roles hiện tại và thêm các roles mới
// Xử lý đúng cả hai trường hợp:
// 1. User có [A, B, C], cập nhật thành [D, E, F] (không trùng) -> Xóa A, B, C và thêm D, E, F
// 2. User có [A, B, C], cập nhật thành [B, C, F, X] (trùng một phần) -> Xóa A, B, C và thêm B, C, F, X
func (r *BaseRoleRepository[T]) UpdateUserRoles(userID string, roleIDs []uint) error {
	var user models.BaseUser
	if err := r.db.Where("id = ?", userID).First(&user).Error; err != nil {
		return err
	}

	// Kiểm tra tất cả roles có tồn tại không
	if len(roleIDs) > 0 {
		var roles []T
		if err := r.db.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
		if len(roles) != len(roleIDs) {
			return gorm.ErrRecordNotFound
		}
	}

	// Sử dụng transaction để đảm bảo tính nhất quán
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Xóa tất cả roles hiện tại của user
		// Điều này đảm bảo kết quả cuối cùng chính xác là danh sách roles mới
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", userID).Error; err != nil {
			return err
		}

		// Thêm các roles mới (batch insert để tối ưu hiệu suất)
		if len(roleIDs) > 0 {
			// Sử dụng batch insert với VALUES clause để tối ưu
			// Tạo câu lệnh INSERT với nhiều VALUES
			query := "INSERT INTO user_roles (user_id, role_id) VALUES "
			args := make([]interface{}, 0, len(roleIDs)*2)
			placeholders := make([]string, 0, len(roleIDs))
			
			for i, roleID := range roleIDs {
				placeholders = append(placeholders, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
				args = append(args, userID, roleID)
			}
			
			query += strings.Join(placeholders, ", ")
			query += " ON CONFLICT (user_id, role_id) DO NOTHING"
			
			if err := tx.Exec(query, args...).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// LoadRoleCache load tất cả roles từ database vào cache
// Được gọi khi khởi động app và khi cần refresh cache
func (r *BaseRoleRepository[T]) LoadRoleCache() error {
	roles, err := r.List()
	if err != nil {
		return err
	}
	
	// Tạo maps mới
	idToName := make(map[uint]string, len(roles))
	nameToID := make(map[string]uint, len(roles))
	
	// Populate maps
	for _, role := range roles {
		id := role.GetID()
		name := role.GetName()
		idToName[id] = name
		nameToID[name] = id
	}
	
	// Cập nhật cache với lock
	r.cache.Lock()
	r.cache.idToName = idToName
	r.cache.nameToID = nameToID
	r.cache.Unlock()
	
	return nil
}

// RefreshRoleCache refresh cache bằng cách load lại từ database
// Có thể gọi thủ công khi cần refresh cache
func (r *BaseRoleRepository[T]) RefreshRoleCache() error {
	return r.LoadRoleCache()
}

// GetRoleNameByID lấy role name từ role ID từ cache (không truy vấn database)
// Trả về (name, true) nếu tìm thấy, (empty string, false) nếu không tìm thấy
func (r *BaseRoleRepository[T]) GetRoleNameByID(id uint) (string, bool) {
	r.cache.RLock()
	defer r.cache.RUnlock()
	name, exists := r.cache.idToName[id]
	return name, exists
}

// GetRoleIDByName lấy role ID từ role name từ cache (không truy vấn database)
// Trả về (id, true) nếu tìm thấy, (0, false) nếu không tìm thấy
func (r *BaseRoleRepository[T]) GetRoleIDByName(name string) (uint, bool) {
	r.cache.RLock()
	defer r.cache.RUnlock()
	id, exists := r.cache.nameToID[name]
	return id, exists
}

// updateCache cập nhật cache với role mới hoặc đã cập nhật
func (r *BaseRoleRepository[T]) updateCache(id uint, name string) {
	r.cache.Lock()
	defer r.cache.Unlock()
	r.cache.idToName[id] = name
	r.cache.nameToID[name] = id
}

// removeFromCache xóa role khỏi cache
func (r *BaseRoleRepository[T]) removeFromCache(id uint, name string) {
	r.cache.Lock()
	defer r.cache.Unlock()
	delete(r.cache.idToName, id)
	delete(r.cache.nameToID, name)
}

// GetRoleNamesByUserIDs lấy role names cho nhiều users cùng lúc (tối ưu với một query duy nhất)
// Trả về map[userID][]roleName - map user ID sang mảng role names
// Sử dụng LEFT JOIN và GROUP BY để tối ưu hiệu suất
func (r *BaseRoleRepository[T]) GetRoleNamesByUserIDs(userIDs []string) (map[string][]string, error) {
	if len(userIDs) == 0 {
		return make(map[string][]string), nil
	}

	// Sử dụng raw SQL với PostgreSQL STRING_AGG để group role names theo user_id
	// Query tối ưu: một query duy nhất với LEFT JOIN và GROUP BY
	type Result struct {
		UserID   string
		RoleNames string // Comma-separated role names
	}

	var results []Result
	// Sử dụng GORM's Where để xử lý IN clause với slice một cách an toàn
	query := r.db.Table("user_roles").
		Select("user_roles.user_id, COALESCE(STRING_AGG(roles.name, ', ' ORDER BY roles.name), '') as role_names").
		Joins("LEFT JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id IN ?", userIDs).
		Group("user_roles.user_id")
	
	err := query.Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// Convert kết quả sang map[userID][]roleName
	resultMap := make(map[string][]string, len(results))
	for _, res := range results {
		if res.RoleNames == "" {
			// User không có role nào
			resultMap[res.UserID] = []string{}
		} else {
			// Split comma-separated string thành mảng
			roleNames := strings.Split(res.RoleNames, ", ")
			resultMap[res.UserID] = roleNames
		}
	}

	// Đảm bảo tất cả userIDs đều có trong map (kể cả những user không có role)
	for _, userID := range userIDs {
		if _, exists := resultMap[userID]; !exists {
			resultMap[userID] = []string{}
		}
	}

	return resultMap, nil
}

// DB trả về interface{} để match với RoleRepositoryInterface
func (r *BaseRoleRepository[T]) DB() interface{} {
	return r.db
}
