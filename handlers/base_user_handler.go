package handlers

import (
	"net/url"
	"reflect"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/authkit/utils"
	"github.com/techmaster-vietnam/goerrorkit"
)

// BaseUserHandler là generic user management handler
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseUserHandler[TUser core.UserInterface, TRole core.RoleInterface] struct {
	authService *service.BaseAuthService[TUser, TRole]
	roleRepo    *repository.BaseRoleRepository[TRole]
}

// NewBaseUserHandler tạo mới BaseUserHandler với generic types
func NewBaseUserHandler[TUser core.UserInterface, TRole core.RoleInterface](
	authService *service.BaseAuthService[TUser, TRole],
	roleRepo *repository.BaseRoleRepository[TRole],
) *BaseUserHandler[TUser, TRole] {
	return &BaseUserHandler[TUser, TRole]{
		authService: authService,
		roleRepo:    roleRepo,
	}
}

// GetProfile handles get profile request
// GET /api/user/profile
// Trả về profile của chính user đang đăng nhập
func (h *BaseUserHandler[TUser, TRole]) GetProfile(c *fiber.Ctx) error {
	user, ok := middleware.GetUserFromContextGeneric[TUser](c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	return c.JSON(fiber.Map{
		"data": user,
	})
}

// UpdateProfile handles update profile request
// PUT /api/user/profile
// User chỉ có thể cập nhật profile của chính mình
func (h *BaseUserHandler[TUser, TRole]) UpdateProfile(c *fiber.Ctx) error {
	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	// Parse request body vào map để lấy cả các trường custom
	var requestMap map[string]interface{}
	if err := c.BodyParser(&requestMap); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Extract full_name
	fullName := ""
	if fn, ok := requestMap["full_name"].(string); ok {
		fullName = fn
	}

	// Các trường còn lại (không phải full_name) là custom fields
	// Loại trừ các trường đặc biệt không nên được set trực tiếp
	excludedFields := map[string]bool{
		"full_name":  true,
		"email":      true, // Không cho phép đổi email
		"password":   true, // Password được đổi qua ChangePassword
		"id":         true, // Không cho phép set ID
		"created_at": true,
		"updated_at": true,
		"deleted_at": true,
		"roles":      true, // Roles được quản lý riêng
	}

	customFields := make(map[string]interface{})
	for key, value := range requestMap {
		if !excludedFields[key] {
			customFields[key] = value
		}
	}

	user, err := h.authService.UpdateProfile(userID, fullName, customFields)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"data": user,
	})
}

// DeleteProfile handles delete profile request
// DELETE /api/user/profile
func (h *BaseUserHandler[TUser, TRole]) DeleteProfile(c *fiber.Ctx) error {
	userID, ok := middleware.GetUserIDFromContext(c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	if err := h.authService.DeleteProfile(userID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Xóa tài khoản thành công",
	})
}

// GetProfileByID handles get profile by identifier request
// GET /api/user/:id
// Chỉ dành cho admin và super_admin để xem profile của bất kỳ user nào
// Route đã được giới hạn bằng .Allow("admin", "super_admin") nên middleware sẽ tự động kiểm tra
// identifier có thể là: id, email, hoặc mobile
// Trả về user kèm theo danh sách roles
func (h *BaseUserHandler[TUser, TRole]) GetProfileByID(c *fiber.Ctx) error {
	// Lấy identifier từ URL parameter (có thể là id, email, hoặc mobile)
	identifier := c.Params("id")
	if identifier == "" {
		return goerrorkit.NewValidationError("Identifier là bắt buộc (id, email, hoặc mobile)", map[string]interface{}{
			"field": "identifier",
		})
	}

	// Decode URL-encoded identifier (ví dụ: bob%40gmail.com -> bob@gmail.com)
	decodedIdentifier, err := url.QueryUnescape(identifier)
	if err != nil {
		// Nếu decode thất bại, sử dụng giá trị gốc
		decodedIdentifier = identifier
	}

	// Lấy thông tin user từ database (tự động phát hiện loại identifier)
	targetUser, err := h.authService.GetUserByIdentifier(decodedIdentifier)
	if err != nil {
		return err
	}

	// Lấy roles của user
	userID := targetUser.GetID()
	roles, err := h.roleRepo.ListRolesOfUser(userID)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách roles")
	}

	// Đảm bảo roles không nil
	if roles == nil {
		roles = []TRole{}
	}

	// Format response với roles dạng [{role_id, role_name}]
	type RoleInfo struct {
		ID   uint   `json:"role_id"`
		Name string `json:"role_name"`
	}

	rolesInfo := make([]RoleInfo, 0, len(roles))
	for _, role := range roles {
		rolesInfo = append(rolesInfo, RoleInfo{
			ID:   role.GetID(),
			Name: role.GetName(),
		})
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"user":  targetUser,
			"roles": rolesInfo,
		},
	})
}

// UpdateProfileByID handles update profile by user ID request
// PUT /api/user/:id
// Chỉ dành cho admin và super_admin để cập nhật profile của người khác
// - super_admin: có thể cập nhật mọi profile
// - admin: chỉ có thể cập nhật profile của chính mình hoặc profile có role khác "admin"
// id phải là user ID (không nhận email hay mobile)
func (h *BaseUserHandler[TUser, TRole]) UpdateProfileByID(c *fiber.Ctx) error {
	// Lấy user hiện tại từ context
	currentUser, ok := middleware.GetUserFromContextGeneric[TUser](c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	// Lấy role IDs từ context
	roleIDs, ok := middleware.GetRoleIDsFromContext(c)
	if !ok {
		// Fallback: query từ DB nếu không có trong context
		roles, err := h.roleRepo.ListRolesOfUser(currentUser.GetID())
		if err != nil {
			return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy roles của user")
		}
		roleIDs = make([]uint, 0, len(roles))
		for _, role := range roles {
			roleIDs = append(roleIDs, role.GetID())
		}
	}

	// Kiểm tra user hiện tại có role super_admin hoặc admin không
	// Lấy role names từ role IDs
	currentUserRoles, err := h.roleRepo.GetByIDs(roleIDs)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin roles")
	}

	currentUserRoleNames := make(map[string]bool)
	for _, role := range currentUserRoles {
		currentUserRoleNames[role.GetName()] = true
	}

	isSuperAdmin := currentUserRoleNames["super_admin"]
	isAdmin := currentUserRoleNames["admin"]

	if !isSuperAdmin && !isAdmin {
		// Middleware đã kiểm tra .Allow("admin", "super_admin") nên trường hợp này không nên xảy ra
		// Nhưng để an toàn, vẫn kiểm tra lại
		return goerrorkit.NewAuthError(403, "Chỉ admin và super_admin mới có quyền cập nhật profile của người khác")
	}

	// Lấy user ID từ URL parameter
	userID := c.Params("id")
	if userID == "" {
		return goerrorkit.NewValidationError("User ID là bắt buộc", map[string]interface{}{
			"field": "id",
		})
	}

	// Lấy thông tin target user từ database bằng ID
	targetUser, err := h.authService.GetUserByID(userID)
	if err != nil {
		return err
	}

	// Kiểm tra phân quyền:
	// - super_admin: có thể cập nhật mọi profile
	// - admin: chỉ có thể cập nhật profile của chính mình hoặc profile có role khác "admin"
	if !isSuperAdmin && isAdmin {
		// Nếu là admin (không phải super_admin), kiểm tra:
		// 1. Có phải đang cập nhật profile của chính mình không?
		if targetUser.GetID() == currentUser.GetID() {
			// Được phép cập nhật profile của chính mình
		} else {
			// 2. Kiểm tra target user có role "admin" không?
			hasAdminRole, err := h.roleRepo.CheckUserHasRole(targetUser.GetID(), "admin")
			if err != nil {
				return goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra role của target user")
			}
			if hasAdminRole {
				return goerrorkit.NewAuthError(403, "Admin không được phép cập nhật profile của user có role 'admin'. Chỉ super_admin mới có quyền này").WithData(map[string]interface{}{
					"target_user_id": targetUser.GetID(),
				})
			}
			// Target user không có role admin, được phép cập nhật
		}
	}

	// Parse request body vào map để lấy cả các trường custom
	var requestMap map[string]interface{}
	if err := c.BodyParser(&requestMap); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Extract full_name
	fullName := ""
	if fn, ok := requestMap["full_name"].(string); ok {
		fullName = fn
	}

	// Các trường còn lại (không phải full_name) là custom fields
	// Loại trừ các trường đặc biệt không nên được set trực tiếp
	excludedFields := map[string]bool{
		"full_name":  true,
		"email":      true, // Không cho phép đổi email
		"password":   true, // Password được đổi qua ChangePassword
		"id":         true, // Không cho phép set ID
		"created_at": true,
		"updated_at": true,
		"deleted_at": true,
		"roles":      true, // Roles được quản lý riêng
	}

	customFields := make(map[string]interface{})
	for key, value := range requestMap {
		if !excludedFields[key] {
			customFields[key] = value
		}
	}

	// Cập nhật profile của target user
	updatedUser, err := h.authService.UpdateProfile(targetUser.GetID(), fullName, customFields)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"data": updatedUser,
	})
}

// DeleteUserByID handles delete user by ID request
// DELETE /api/user/:id
// Chỉ dành cho admin và super_admin để xóa user
// - super_admin: xóa bất kỳ user nào không chứa role "super_admin". Xóa là hard delete
// - admin: chỉ xóa các user không có role "admin" và "super_admin". Xóa là soft delete
func (h *BaseUserHandler[TUser, TRole]) DeleteUserByID(c *fiber.Ctx) error {
	// Lấy user hiện tại từ context
	currentUser, ok := middleware.GetUserFromContextGeneric[TUser](c)
	if !ok {
		return goerrorkit.NewAuthError(401, "Không tìm thấy thông tin người dùng")
	}

	// Lấy user ID từ URL parameter
	targetUserID := c.Params("id")
	if targetUserID == "" {
		return goerrorkit.NewValidationError("User ID là bắt buộc", map[string]interface{}{
			"field": "id",
		})
	}

	// Gọi service để xóa user với logic phân quyền
	if err := h.authService.DeleteUserByID(currentUser.GetID(), targetUserID); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"message": "Xóa user thành công",
	})
}

// ListUsers handles list users request với pagination và filter
// GET /api/user?page=1&page_size=10&email=test&full_name=John&mobile=123&address=Hanoi
// Chỉ dành cho admin và super_admin
// Trả về danh sách users với các trường: id, email, full_name, mobile, address
// Filter params:
//   - email: filter email chứa text
//   - full_name: filter full_name chứa text
//   - mobile: filter mobile chứa text (custom field)
//   - address: filter address chứa text (custom field)
//   - Các custom fields khác có thể được thêm vào query params và sẽ được filter động
//
// Pagination params:
//   - page: số trang (bắt đầu từ 1, mặc định 1)
//   - page_size: số lượng items mỗi trang (mặc định 10)
//   - enable_pagination: bật/tắt pagination thủ công (true/false, mặc định auto)
//   - pagination_threshold: ngưỡng để tự động bật pagination (mặc định 100)
//   - max_page_size: giới hạn tối đa page_size khi có pagination (mặc định 100)
//
// Logic pagination:
//   - Nếu enable_pagination = true: luôn dùng pagination
//   - Nếu enable_pagination = false: luôn không dùng pagination (trả về tất cả)
//   - Nếu enable_pagination không được chỉ định (auto): tự động quyết định
//   - Nếu total users > pagination_threshold: dùng pagination
//   - Nếu total users <= pagination_threshold: không dùng pagination (trả về tất cả)
func (h *BaseUserHandler[TUser, TRole]) ListUsers(c *fiber.Ctx) error {
	// Parse pagination params từ query string
	page := c.QueryInt("page", 1)
	pageSize := c.QueryInt("page_size", 40)

	// Parse enable_pagination (true/false/auto)
	var enablePagination *bool
	enablePaginationStr := c.Query("enable_pagination")
	switch enablePaginationStr {
	case "true":
		val := true
		enablePagination = &val
	case "false":
		val := false
		enablePagination = &val
	default:
		// Nếu không phải true/false hoặc rỗng thì giữ nil (auto mode)
	}

	// Parse pagination_threshold và max_page_size
	paginationThreshold := c.QueryInt("pagination_threshold", 100)
	maxPageSize := c.QueryInt("max_page_size", 100)

	// Parse sort params từ query string
	sortBy := c.Query("sort_by")
	order := c.Query("order", "asc") // Mặc định là "asc" nếu không có
	// Validate order chỉ nhận "asc" hoặc "desc"
	if order != "asc" && order != "desc" {
		order = "asc" // Fallback về "asc" nếu không hợp lệ
	}

	// Parse filter params từ query string
	filter := &repository.UserFilter{
		Email:    c.Query("email"),
		FullName: c.Query("full_name"),
		SortBy:   sortBy,
		Order:    order,
		Custom:   make(map[string]string),
	}

	// Parse tất cả custom fields từ query params một cách động
	// Không hardcode bất kỳ custom field nào - hoàn toàn tự động
	queryArgs := c.Context().QueryArgs()
	queryArgs.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		valueStr := string(value)

		// Loại trừ các params đã được parse (standard fields, pagination và sort)
		excludedParams := map[string]bool{
			"page":                 true,
			"page_size":            true,
			"pageSize":             true, // Alternative format
			"enable_pagination":    true,
			"pagination_threshold": true,
			"max_page_size":        true,
			"email":                true,
			"full_name":            true,
			"sort_by":              true,
			"order":                true,
		}

		if !excludedParams[keyStr] && valueStr != "" {
			// Tất cả các params còn lại đều là custom field filters
			filter.Custom[keyStr] = valueStr
		}
	})

	// Tạo options cho service
	options := service.ListUsersOptions{
		Page:                page,
		PageSize:            pageSize,
		EnablePagination:    enablePagination,
		PaginationThreshold: paginationThreshold,
		MaxPageSize:         maxPageSize,
		SortBy:              sortBy,
		Order:               order,
	}

	// Gọi service để lấy danh sách users với filter và options
	result, err := h.authService.ListUsersWithOptions(options, filter)
	if err != nil {
		return err
	}

	// Tạo response DTO động - sử dụng map để hỗ trợ custom fields
	userList := make([]map[string]interface{}, 0, len(result.Users))
	for _, user := range result.Users {
		item := map[string]interface{}{
			"id":        user.GetID(),
			"email":     user.GetEmail(),
			"full_name": user.GetFullName(),
		}

		// Lấy tất cả các trường custom bằng reflection
		// Tự động detect và thêm vào response
		userValue := reflect.ValueOf(user)
		if userValue.Kind() == reflect.Ptr {
			userValue = userValue.Elem()
		}

		if userValue.Kind() == reflect.Struct {
			userType := userValue.Type()

			// Duyệt qua tất cả các fields của struct
			for i := 0; i < userType.NumField(); i++ {
				field := userType.Field(i)
				fieldValue := userValue.Field(i)

				// Bỏ qua các trường đã có trong BaseUser hoặc các trường đặc biệt
				excludedFields := map[string]bool{
					"BaseUser":  true,
					"ID":        true,
					"Email":     true,
					"Password":  true,
					"FullName":  true,
					"Active":    true,
					"CreatedAt": true,
					"UpdatedAt": true,
					"DeletedAt": true,
					"Roles":     true,
				}

				if excludedFields[field.Name] {
					continue
				}

				// Chỉ lấy các trường có thể convert sang JSON (string, int, bool, etc.)
				if !fieldValue.IsValid() || !fieldValue.CanInterface() {
					continue
				}

				// Convert field name sang snake_case cho JSON key
				jsonKey := utils.PascalToSnakeCase(field.Name)

				// Lấy giá trị field
				fieldInterface := fieldValue.Interface()

				// Chỉ thêm các giá trị không rỗng (optional)
				// Hoặc có thể thêm tất cả để đảm bảo đầy đủ thông tin
				switch v := fieldInterface.(type) {
				case string:
					if v != "" {
						item[jsonKey] = v
					}
				case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
					item[jsonKey] = v
				case float32, float64:
					item[jsonKey] = v
				case bool:
					item[jsonKey] = v
				default:
					// Thêm các type khác nếu cần
					item[jsonKey] = v
				}
			}
		}

		userList = append(userList, item)
	}

	// Tạo response data với pagination info (chỉ khi có pagination)
	responseData := fiber.Map{
		"users":              userList,
		"total":              result.Total,
		"pagination_enabled": result.PaginationEnabled,
	}

	// Chỉ thêm pagination info khi có pagination
	if result.PaginationEnabled {
		if result.Page != nil {
			responseData["page"] = *result.Page
		}
		if result.PageSize != nil {
			responseData["page_size"] = *result.PageSize
		}
		if result.TotalPages != nil {
			responseData["total_pages"] = *result.TotalPages
		}
	}

	return c.JSON(fiber.Map{
		"data": responseData,
	})
}
