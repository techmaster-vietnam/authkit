package repository

import (
	"reflect"
	"strings"

	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/utils"
	"gorm.io/gorm"
)

// UserFilter represents filter parameters for listing users
// Hỗ trợ filter các trường standard (email, full_name) và custom fields động
type UserFilter struct {
	Email    string            // Filter email chứa text
	FullName string            // Filter full_name chứa text
	Custom   map[string]string // Filter custom fields: key là tên field (snake_case hoặc PascalCase), value là text cần tìm
	SortBy   string            // Trường để sort (email, full_name, hoặc custom field)
	Order    string            // Thứ tự sort: "asc" hoặc "desc" (mặc định "asc")
}

// BaseUserRepository là generic repository cho User models
// T phải implement UserInterface
type BaseUserRepository[T core.UserInterface] struct {
	db *gorm.DB
}

// NewBaseUserRepository tạo mới BaseUserRepository với generic type
func NewBaseUserRepository[T core.UserInterface](db *gorm.DB) *BaseUserRepository[T] {
	return &BaseUserRepository[T]{db: db}
}

// Create tạo mới user
func (r *BaseUserRepository[T]) Create(user T) error {
	return r.db.Create(&user).Error
}

// GetByID lấy user theo ID
func (r *BaseUserRepository[T]) GetByID(id string) (T, error) {
	var user T
	err := r.db.Preload("Roles").Where("id = ?", id).First(&user).Error
	return user, err
}

// GetByEmail lấy user theo email
func (r *BaseUserRepository[T]) GetByEmail(email string) (T, error) {
	var user T
	err := r.db.Preload("Roles").Where("email = ?", email).First(&user).Error
	return user, err
}

// GetByMobile lấy user theo mobile
func (r *BaseUserRepository[T]) GetByMobile(mobile string) (T, error) {
	var user T
	err := r.db.Preload("Roles").Where("mobile = ?", mobile).First(&user).Error
	return user, err
}

// Update cập nhật user
func (r *BaseUserRepository[T]) Update(user T) error {
	return r.db.Save(&user).Error
}

// Delete soft delete user
func (r *BaseUserRepository[T]) Delete(id string) error {
	var user T
	return r.db.Where("id = ?", id).Delete(&user).Error
}

// HardDelete hard delete user (permanently remove from database)
func (r *BaseUserRepository[T]) HardDelete(id string) error {
	var user T
	return r.db.Unscoped().Where("id = ?", id).Delete(&user).Error
}

// List lấy danh sách users với pagination và filter
func (r *BaseUserRepository[T]) List(offset, limit int, filter interface{}) ([]T, int64, error) {
	var users []T
	var total int64
	var zero T

	// Build query với filters
	query := r.db.Model(&zero)

	// Apply filters nếu có
	var userFilter *UserFilter
	if filter != nil {
		// Convert filter sang *UserFilter nếu có thể
		if f, ok := filter.(*UserFilter); ok {
			userFilter = f
		} else {
			// Nếu filter không phải *UserFilter, bỏ qua filter
			userFilter = nil
		}
	}

	if userFilter != nil {
		// Filter email (standard field)
		if userFilter.Email != "" {
			query = query.Where("email LIKE ?", "%"+userFilter.Email+"%")
		}

		// Filter full_name (standard field)
		if userFilter.FullName != "" {
			query = query.Where("full_name LIKE ?", "%"+userFilter.FullName+"%")
		}

		// Filter custom fields động
		if len(userFilter.Custom) > 0 {
			query = r.applyCustomFilters(query, &zero, userFilter.Custom)
		}
	}

	// Count total với filters (không áp dụng sort cho count)
	countQuery := query
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Áp dụng sort nếu có
	if userFilter != nil && userFilter.SortBy != "" {
		columnName := r.getSortColumnName(&zero, userFilter.SortBy)
		if columnName != "" {
			order := userFilter.Order
			if order != "asc" && order != "desc" {
				order = "asc" // Fallback về "asc" nếu không hợp lệ
			}
			query = query.Order(columnName + " " + order)
		}
	}

	// Get users với pagination và sort
	err := query.Preload("Roles").Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

// Count đếm tổng số users với filter (không lấy dữ liệu, chỉ đếm)
func (r *BaseUserRepository[T]) Count(filter interface{}) (int64, error) {
	var total int64
	var zero T

	// Build query với filters
	query := r.db.Model(&zero)

	// Apply filters nếu có
	var userFilter *UserFilter
	if filter != nil {
		if f, ok := filter.(*UserFilter); ok {
			userFilter = f
		} else {
			userFilter = nil
		}
	}

	if userFilter != nil {
		// Filter email (standard field)
		if userFilter.Email != "" {
			query = query.Where("email LIKE ?", "%"+userFilter.Email+"%")
		}

		// Filter full_name (standard field)
		if userFilter.FullName != "" {
			query = query.Where("full_name LIKE ?", "%"+userFilter.FullName+"%")
		}

		// Filter custom fields động
		if len(userFilter.Custom) > 0 {
			query = r.applyCustomFilters(query, &zero, userFilter.Custom)
		}
	}

	// Count total với filters
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}

	return total, nil
}

// applyCustomFilters áp dụng filter cho các custom fields bằng reflection
// Tự động detect tên column trong database từ struct field
func (r *BaseUserRepository[T]) applyCustomFilters(query *gorm.DB, zero *T, customFilters map[string]string) *gorm.DB {
	// Lấy type của struct (unwrap pointer nếu cần)
	userType := reflect.TypeOf(*zero)
	if userType.Kind() == reflect.Ptr {
		userType = userType.Elem()
	}

	if userType.Kind() != reflect.Struct {
		return query // Không phải struct thì không filter
	}

	// Duyệt qua các custom filters
	for fieldName, filterValue := range customFilters {
		if filterValue == "" {
			continue // Bỏ qua filter rỗng
		}

		// Tìm field trong struct
		var field reflect.StructField
		var found bool

		// Thử tìm với tên chính xác (case-sensitive)
		if f, ok := userType.FieldByName(fieldName); ok {
			field = f
			found = true
		} else {
			// Thử tìm với PascalCase nếu fieldName là snake_case
			pascalCase := r.snakeToPascalCase(fieldName)
			if f, ok := userType.FieldByName(pascalCase); ok {
				field = f
				found = true
			} else {
				// Tìm không phân biệt hoa thường
				for i := 0; i < userType.NumField(); i++ {
					f := userType.Field(i)
					if strings.EqualFold(f.Name, fieldName) || strings.EqualFold(f.Name, pascalCase) {
						field = f
						found = true
						break
					}
				}
			}
		}

		if !found {
			continue // Field không tồn tại, bỏ qua
		}

		// Lấy tên column từ GORM tag hoặc convert từ field name
		columnName := r.getColumnName(field)
		if columnName == "" {
			continue // Không có column name hợp lệ
		}

		// Áp dụng filter với LIKE
		query = query.Where(columnName+" LIKE ?", "%"+filterValue+"%")
	}

	return query
}

// getColumnName lấy tên column từ struct field
// Ưu tiên GORM tag "column", nếu không có thì convert PascalCase sang snake_case
func (r *BaseUserRepository[T]) getColumnName(field reflect.StructField) string {
	// Kiểm tra GORM tag
	gormTag := field.Tag.Get("gorm")
	if gormTag != "" {
		// Parse gorm tag để tìm column name
		// Format: gorm:"column:column_name" hoặc gorm:"column:column_name;type:varchar(200)"
		parts := strings.Split(gormTag, ";")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "column:") {
				columnName := strings.TrimPrefix(part, "column:")
				// Remove any additional attributes after column name
				if idx := strings.Index(columnName, " "); idx != -1 {
					columnName = columnName[:idx]
				}
				return columnName
			}
		}
	}

	// Nếu không có GORM tag, convert PascalCase sang snake_case
	return utils.PascalToSnakeCase(field.Name)
}

// getSortColumnName lấy tên column từ sort_by field name
// Hỗ trợ cả standard fields (email, full_name) và custom fields
func (r *BaseUserRepository[T]) getSortColumnName(zero *T, sortBy string) string {
	if sortBy == "" {
		return ""
	}

	// Standard fields mapping
	standardFields := map[string]string{
		"email":     "email",
		"full_name": "full_name",
		"fullName":  "full_name",
		"id":        "id",
		"created_at": "created_at",
		"updated_at": "updated_at",
	}

	// Kiểm tra xem có phải standard field không
	if columnName, ok := standardFields[sortBy]; ok {
		return columnName
	}

	// Nếu không phải standard field, tìm trong struct bằng reflection
	userType := reflect.TypeOf(*zero)
	// Unwrap pointer nếu cần
	for userType.Kind() == reflect.Ptr {
		userType = userType.Elem()
	}

	if userType.Kind() != reflect.Struct {
		return "" // Không phải struct thì không sort
	}

	// Tìm field trong struct
	var field reflect.StructField
	var found bool

	// Thử tìm với tên chính xác (case-sensitive)
	if f, ok := userType.FieldByName(sortBy); ok {
		field = f
		found = true
	} else {
		// Thử tìm với PascalCase nếu sortBy là snake_case
		pascalCase := r.snakeToPascalCase(sortBy)
		if f, ok := userType.FieldByName(pascalCase); ok {
			field = f
			found = true
		} else {
			// Tìm không phân biệt hoa thường
			for i := 0; i < userType.NumField(); i++ {
				f := userType.Field(i)
				if strings.EqualFold(f.Name, sortBy) || strings.EqualFold(f.Name, pascalCase) {
					field = f
					found = true
					break
				}
			}
		}
	}

	if !found {
		return "" // Field không tồn tại
	}

	// Lấy tên column từ field
	return r.getColumnName(field)
}

// snakeToPascalCase converts snake_case to PascalCase
// Ví dụ: "mobile" -> "Mobile", "full_name" -> "FullName"
func (r *BaseUserRepository[T]) snakeToPascalCase(snake string) string {
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

// DB trả về *gorm.DB nhưng dùng interface{} để match với UserRepositoryInterface
func (r *BaseUserRepository[T]) DB() interface{} {
	return r.db
}
