package handlers

import (
	"errors"
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/middleware"
	"github.com/techmaster-vietnam/authkit/service"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// BaseRuleHandler là generic rule handler
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseRuleHandler[TUser core.UserInterface, TRole core.RoleInterface] struct {
	ruleService             *service.RuleService
	authorizationMiddleware *middleware.BaseAuthorizationMiddleware[TUser, TRole]
}

// NewBaseRuleHandler tạo mới BaseRuleHandler với generic types
func NewBaseRuleHandler[TUser core.UserInterface, TRole core.RoleInterface](
	ruleService *service.RuleService,
	authzMiddleware *middleware.BaseAuthorizationMiddleware[TUser, TRole],
) *BaseRuleHandler[TUser, TRole] {
	return &BaseRuleHandler[TUser, TRole]{
		ruleService:             ruleService,
		authorizationMiddleware: authzMiddleware,
	}
}

// UpdateRule handles update rule request
// PUT /api/rules
func (h *BaseRuleHandler[TUser, TRole]) UpdateRule(c *fiber.Ctx) error {
	var req service.UpdateRuleRequest
	if err := c.BodyParser(&req); err != nil {
		return goerrorkit.NewValidationError("Dữ liệu không hợp lệ", map[string]interface{}{
			"error": err.Error(),
		})
	}

	if req.ID == "" {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"field": "id",
		})
	}

	rule, err := h.ruleService.UpdateRule(req.ID, req)
	if err != nil {
		return err
	}

	// Note: RuleService đã tự động invalidate cache thông qua cacheInvalidator
	// Việc gọi lại ở đây chỉ để đảm bảo backward compatibility
	// Nếu RuleService đã được inject cacheInvalidator, việc gọi này là redundant nhưng không gây hại
	h.authorizationMiddleware.InvalidateCache()

	return c.JSON(fiber.Map{
		"data": rule,
	})
}

// ListRules handles list rules request
// GET /api/rules?method=GET&path=blog&type=ALLOW&fixed=true
func (h *BaseRuleHandler[TUser, TRole]) ListRules(c *fiber.Ctx) error {
	// Lấy query parameters
	method := c.Query("method")
	path := c.Query("path")
	typeParam := c.Query("type")
	fixedParam := c.Query("fixed")
	serviceParam := c.Query("service")

	// Validate method
	if method != "" {
		validMethods := map[string]bool{
			"GET":    true,
			"POST":   true,
			"PUT":    true,
			"DELETE": true,
		}
		if !validMethods[method] {
			return goerrorkit.NewValidationError("Method không hợp lệ", map[string]interface{}{
				"field":    "method",
				"received": method,
				"allowed":  []string{"GET", "POST", "PUT", "DELETE"},
			})
		}
	}

	// Validate type
	if typeParam != "" {
		validTypes := map[string]bool{
			"PUBLIC": true,
			"ALLOW":  true,
			"FORBID": true,
		}
		if !validTypes[typeParam] {
			return goerrorkit.NewValidationError("Type không hợp lệ", map[string]interface{}{
				"field":    "type",
				"received": typeParam,
				"allowed":  []string{"PUBLIC", "ALLOW", "FORBID"},
			})
		}
	}

	// Validate fixed (mặc định false nếu không có)
	var fixed *bool
	if fixedParam != "" {
		switch fixedParam {
		case "true":
			val := true
			fixed = &val
		case "false":
			val := false
			fixed = &val
		default:
			return goerrorkit.NewValidationError("Fixed phải là true hoặc false", map[string]interface{}{
				"field":    "fixed",
				"received": fixedParam,
				"allowed":  []string{"true", "false"},
			})
		}
	}

	// Tạo filter struct
	filter := service.RuleFilter{
		Method:  method,
		Path:    path,
		Type:    typeParam,
		Fixed:   fixed,
		Service: serviceParam,
	}

	rules, err := h.ruleService.ListRules(filter)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"data": rules,
	})
}

// GetByID handles get rule by ID request
// GET /api/rules/:id
// ID có dạng: "GET|/api/blogs/*", "GET|/api/roles/*/users", "PUT|/api/users/*/roles"
// ID sẽ được URL encode khi truyền qua REST API (ví dụ: "GET%7C%2Fapi%2Fblogs%2F%2A")
// Fiber tự động decode URL parameters, nhưng cần decode thêm để xử lý các ký tự đặc biệt
func (h *BaseRuleHandler[TUser, TRole]) GetByID(c *fiber.Ctx) error {
	// Lấy id từ URL parameter (Fiber đã tự động decode một lần)
	id := c.Params("id")
	if id == "" {
		return goerrorkit.NewValidationError("ID không hợp lệ", map[string]interface{}{
			"field": "id",
		})
	}

	// Decode thêm một lần nữa để xử lý các ký tự đặc biệt như "|", "/", "*"
	// Sử dụng QueryUnescape vì nó xử lý cả %7C (|), %2F (/), %2A (*)
	decodedID, err := url.QueryUnescape(id)
	if err != nil {
		// Nếu decode lỗi, thử sử dụng id gốc (có thể đã được decode rồi)
		decodedID = id
	}

	// Lấy rule từ service
	rule, err := h.ruleService.GetByID(decodedID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy rule").WithData(map[string]interface{}{
				"rule_id": decodedID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy rule")
	}

	return c.JSON(fiber.Map{
		"data": rule,
	})
}
