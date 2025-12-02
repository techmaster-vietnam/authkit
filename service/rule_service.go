package service

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// RuleFilter represents filter parameters for listing rules
// Alias của repository.RuleFilter để giữ interface service layer
type RuleFilter = repository.RuleFilter

// RuleService handles rule business logic
type RuleService struct {
	ruleRepo         *repository.RuleRepository
	roleRepo         *repository.RoleRepository
	cacheInvalidator core.CacheInvalidator // Optional: để invalidate rules cache khi rule thay đổi
}

// NewRuleService creates a new rule service
func NewRuleService(ruleRepo *repository.RuleRepository, roleRepo *repository.RoleRepository) *RuleService {
	return &RuleService{
		ruleRepo: ruleRepo,
		roleRepo: roleRepo,
	}
}

// SetCacheInvalidator sets cache invalidator để service có thể invalidate rules cache
// Nên được gọi sau khi khởi tạo service nếu cần invalidate cache khi rule thay đổi
func (s *RuleService) SetCacheInvalidator(invalidator core.CacheInvalidator) {
	s.cacheInvalidator = invalidator
}

// UpdateRuleRequest represents update rule request
type UpdateRuleRequest struct {
	ID          string      `json:"id"`          // ID của rule (ví dụ: "POST|/api/auth/login")
	Type        string      `json:"type"`        // PUBLIC, ALLOW, hoặc FORBID
	Roles       interface{} `json:"roles"`       // []uint (IDs) hoặc []string (names) hoặc mảng rỗng
	Description string      `json:"description"` // Mô tả rule
}

// UpdateRule updates a rule
func (s *RuleService) UpdateRule(ruleID string, req UpdateRuleRequest) (*models.Rule, error) {
	rule, err := s.ruleRepo.GetByID(ruleID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, goerrorkit.NewBusinessError(404, "Không tìm thấy rule").WithData(map[string]interface{}{
				"rule_id": ruleID,
			})
		}
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy rule")
	}

	// Kiểm tra rule.fixed = true thì không cho phép cập nhật
	if rule.Fixed {
		return nil, goerrorkit.NewBusinessError(403, "Không thể cập nhật rule này vì rule.fixed = true. Rule này được quyết định từ code Golang và không thể thay đổi từ database").WithData(map[string]interface{}{
			"rule_id": ruleID,
			"fixed":   rule.Fixed,
		})
	}

	// Validate và cập nhật Type
	if req.Type != "" {
		ruleType := models.AccessType(req.Type)
		if ruleType != models.AccessPublic && ruleType != models.AccessAllow &&
			ruleType != models.AccessForbid {
			return nil, goerrorkit.NewValidationError("Type phải là PUBLIC, ALLOW hoặc FORBID", map[string]interface{}{
				"field":    "type",
				"received": req.Type,
				"allowed":  []string{"PUBLIC", "ALLOW", "FORBID"},
			})
		}
		rule.Type = ruleType
	}

	// Parse và validate roles (hỗ trợ cả []uint và []string)
	if req.Roles != nil {
		var roleIDs []uint
		rolesValue := reflect.ValueOf(req.Roles)
		if !rolesValue.IsValid() || rolesValue.Kind() != reflect.Slice {
			return nil, goerrorkit.NewValidationError("Roles phải là một mảng", map[string]interface{}{
				"field": "roles",
			})
		}

		// Xử lý mảng rỗng
		if rolesValue.Len() == 0 {
			roleIDs = []uint{}
		} else {
			// Kiểm tra phần tử đầu tiên để xác định loại (uint hay string)
			firstElem := rolesValue.Index(0)
			// Xử lý trường hợp interface{} (khi JSON được parse)
			elemKind := firstElem.Kind()
			if elemKind == reflect.Interface {
				// Unwrap interface để lấy kiểu thực tế
				if firstElem.Elem().IsValid() {
					elemKind = firstElem.Elem().Kind()
				}
			}

			needValidateIDs := false // Flag để biết có cần validate IDs không
			switch elemKind {
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				// Mảng số (IDs) - cần validate tồn tại
				roleIDs = make([]uint, rolesValue.Len())
				for i := 0; i < rolesValue.Len(); i++ {
					elem := rolesValue.Index(i)
					if elem.Kind() == reflect.Interface && elem.Elem().IsValid() {
						elem = elem.Elem()
					}
					roleIDs[i] = uint(elem.Uint())
				}
				needValidateIDs = true
			case reflect.String:
				// Mảng chuỗi (names) - cần convert sang IDs và validate tồn tại
				roleNames := make([]string, rolesValue.Len())
				for i := 0; i < rolesValue.Len(); i++ {
					elem := rolesValue.Index(i)
					if elem.Kind() == reflect.Interface && elem.Elem().IsValid() {
						elem = elem.Elem()
					}
					roleNames[i] = elem.String()
				}
				// Lấy role IDs từ names
				roleNameToIDMap, err := s.roleRepo.GetIDsByNames(roleNames)
				if err != nil {
					return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy role IDs từ names").
						WithData(map[string]interface{}{
							"role_names": roleNames,
						})
				}
				// Kiểm tra tất cả names có tồn tại không
				missingRoles := make([]string, 0)
				for _, name := range roleNames {
					if _, exists := roleNameToIDMap[name]; !exists {
						missingRoles = append(missingRoles, name)
					}
				}
				if len(missingRoles) > 0 {
					return nil, goerrorkit.NewBusinessError(404, "Một số roles không tồn tại trong bảng roles").WithData(map[string]interface{}{
						"missing_roles": missingRoles,
					})
				}
				// Convert sang mảng IDs
				roleIDs = make([]uint, 0, len(roleNames))
				for _, name := range roleNames {
					roleIDs = append(roleIDs, roleNameToIDMap[name])
				}
				// Không cần validate lại vì đã kiểm tra names tồn tại
			case reflect.Float64:
				// Xử lý trường hợp JSON parse số thành float64 - cần validate tồn tại
				roleIDs = make([]uint, rolesValue.Len())
				for i := 0; i < rolesValue.Len(); i++ {
					elem := rolesValue.Index(i)
					if elem.Kind() == reflect.Interface && elem.Elem().IsValid() {
						elem = elem.Elem()
					}
					roleIDs[i] = uint(elem.Float())
				}
				needValidateIDs = true
			default:
				return nil, goerrorkit.NewValidationError("Roles phải là mảng số (IDs) hoặc mảng chuỗi (names)", map[string]interface{}{
					"field": "roles",
				})
			}

			// Kiểm tra tất cả role IDs có tồn tại trong database không (chỉ khi cần)
			if needValidateIDs && len(roleIDs) > 0 {
				roles, err := s.roleRepo.GetByIDs(roleIDs)
				if err != nil {
					return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra roles tồn tại")
				}
				// Tạo map để kiểm tra nhanh
				existingRoleIDs := make(map[uint]bool)
				for _, role := range roles {
					existingRoleIDs[role.ID] = true
				}
				// Tìm các role IDs không tồn tại
				missingRoleIDs := make([]uint, 0)
				for _, roleID := range roleIDs {
					if !existingRoleIDs[roleID] {
						missingRoleIDs = append(missingRoleIDs, roleID)
					}
				}
				if len(missingRoleIDs) > 0 {
					return nil, goerrorkit.NewBusinessError(404, "Một số roles không tồn tại trong bảng roles").WithData(map[string]interface{}{
						"missing_role_ids": missingRoleIDs,
					})
				}
			}
		}

		rule.Roles = models.FromUintSlice(roleIDs)
	}

	// Cập nhật description nếu có
	if req.Description != "" {
		rule.Description = req.Description
	}

	if err := s.ruleRepo.Update(rule); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật rule")
	}

	// Invalidate rules cache sau khi cập nhật rule thành công
	if s.cacheInvalidator != nil {
		s.cacheInvalidator.InvalidateRulesCache()
	}

	return rule, nil
}

// ListRules lists all rules with optional filters
func (s *RuleService) ListRules(filter RuleFilter) ([]models.Rule, error) {
	rules, err := s.ruleRepo.List(filter)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách rule")
	}
	return rules, nil
}

// GetByID gets a rule by ID
func (s *RuleService) GetByID(ruleID string) (*models.Rule, error) {
	rule, err := s.ruleRepo.GetByID(ruleID)
	if err != nil {
		return nil, err
	}
	return rule, nil
}

// GetRulesByRole gets all rules that a specific role can access
// roleIDName can be either role ID (number) or role name (string)
// Returns:
// - All rules if role is "super_admin"
// - Otherwise: PUBLIC rules + ALLOW rules with empty roles + ALLOW rules containing the role ID
func (s *RuleService) GetRulesByRole(roleIDName string) ([]models.Rule, error) {
	var roleID uint
	var roleName string
	var err error

	// Try to parse as number (role ID)
	parsedID, parseErr := strconv.ParseUint(roleIDName, 10, 32)
	if parseErr == nil {
		// It's a number, use as role ID
		roleID = uint(parsedID)
		// Get role name from ID
		role, err := s.roleRepo.GetByID(roleID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, goerrorkit.NewBusinessError(404, "Không tìm thấy role").WithData(map[string]interface{}{
					"role_id": roleIDName,
				})
			}
			return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin role")
		}
		roleName = role.Name
	} else {
		// It's a string, treat as role name
		roleName = roleIDName
		// Get role ID from name
		role, err := s.roleRepo.GetByName(roleName)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, goerrorkit.NewBusinessError(404, "Không tìm thấy role").WithData(map[string]interface{}{
					"role_name": roleIDName,
				})
			}
			return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy thông tin role")
		}
		roleID = role.ID
	}

	// Get rules from repository
	rules, err := s.ruleRepo.GetRulesByRole(roleID, roleName)
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách rules theo role")
	}

	return rules, nil
}
