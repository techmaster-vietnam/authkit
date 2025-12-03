package router

import (
	"fmt"

	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// SyncRoutesToDatabase đồng bộ routes từ code vào database
// Logic đồng bộ:
//   - Fixed: LUÔN được đồng bộ từ code sang DB
//   - Type và Roles:
//     * Nếu Fixed=true: LUÔN ghi đè từ code lên database (code là source of truth)
//     * Nếu Fixed=false và Override=true: ghi đè từ code lên database (code là source of truth, nhưng cho phép sửa tạm thời qua API)
//     * Nếu Fixed=false và Override=false: giữ nguyên từ database (cho phép người dùng chỉnh sửa qua API)
//   - Description, Method, Path: luôn được đồng bộ từ code
//   - Convert role names (string) từ routes → role IDs (uint) khi lưu vào DB
//   - Tự động xóa rules không còn trong registry (cleanup rules cũ)
//   - serviceName: nếu empty, rule sẽ có service_name = NULL (single-app mode)
//                  nếu set, rule sẽ có service_name = serviceName (microservice mode)
//
// Các trường hợp chuyển đổi được xử lý:
//   1. Fixed() → không Fixed(): update Fixed=false trong DB, giữ nguyên Type và Roles
//   2. không Fixed() → Fixed(): update Fixed=true và ghi đè Type, Roles từ code
//   3. không Fixed() → Override(): update Fixed=false, ghi đè Type và Roles từ code
//   4. Override() → Fixed(): update Fixed=true và ghi đè Type, Roles từ code
//   5. không Override() → Override(): ghi đè Type và Roles từ code, update Fixed=false
//
// Về Override:
//   - Override hữu ích khi bạn muốn code là source of truth nhưng vẫn cho phép admin test/sửa tạm thời qua API
//   - Khác với Fixed: Override cho phép sửa từ DB (nhưng sẽ bị ghi đè khi sync), Fixed không cho phép sửa từ DB
//   - Use case: Development/testing environments, khi cần test quyền tạm thời nhưng vẫn muốn code là source of truth
func SyncRoutesToDatabase(registry *RouteRegistry, ruleRepo *repository.RuleRepository, roleRepo *repository.RoleRepository, serviceName string) error {
	routes := registry.GetAllRoutes()

	// Truncate serviceName to max 20 characters if longer
	ruleServiceName := serviceName
	if len(ruleServiceName) > 20 {
		ruleServiceName = ruleServiceName[:20]
	}

	// Lấy tất cả rules hiện tại trong DB (filtered by service_name)
	existingRules, err := ruleRepo.List(repository.RuleFilter{})
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to list existing rules")
	}

	// Tạo map các rule IDs từ registry để check nhanh
	registryRuleIDs := make(map[string]bool)
	for _, route := range routes {
		ruleID := fmt.Sprintf("%s|%s", route.Method, route.FullPath)
		registryRuleIDs[ruleID] = true
	}

	// Xóa các rules không còn trong registry
	for _, existingRule := range existingRules {
		if !registryRuleIDs[existingRule.ID] {
			// Rule không còn trong registry, xóa khỏi DB
			if deleteErr := ruleRepo.Delete(existingRule.ID); deleteErr != nil {
				return goerrorkit.WrapWithMessage(deleteErr, fmt.Sprintf("Failed to delete obsolete rule %s", existingRule.ID)).
					WithData(map[string]interface{}{
						"rule_id": existingRule.ID,
						"method":  existingRule.Method,
						"path":    existingRule.Path,
					})
			}
		}
	}

	// Collect all unique role names from all routes for batch conversion
	roleNameSet := make(map[string]bool)
	for _, route := range routes {
		for _, roleName := range route.Roles {
			roleNameSet[roleName] = true
		}
	}

	// Convert all role names to IDs in one batch query (optimized)
	roleNames := make([]string, 0, len(roleNameSet))
	for roleName := range roleNameSet {
		roleNames = append(roleNames, roleName)
	}
	roleNameToIDMap, err := roleRepo.GetIDsByNames(roleNames)
	if err != nil {
		return goerrorkit.WrapWithMessage(err, "Failed to convert role names to IDs").
			WithData(map[string]interface{}{
				"role_names": roleNames,
			})
	}

	// Convert role names to IDs for each route
	for _, route := range routes {
		ruleID := fmt.Sprintf("%s|%s", route.Method, route.FullPath)

		// Convert role names to role IDs
		roleIDs := make([]uint, 0, len(route.Roles))
		for _, roleName := range route.Roles {
			if roleID, exists := roleNameToIDMap[roleName]; exists {
				roleIDs = append(roleIDs, roleID)
			} else {
				// Role name not found - log warning but continue
				// This allows routes to be registered even if role doesn't exist yet
				// The role might be created later
			}
		}

		// Kiểm tra xem rule đã tồn tại trong DB chưa
		existingRule, err := ruleRepo.GetByID(ruleID)
		if err != nil && err != gorm.ErrRecordNotFound {
			// Lỗi khác khi query
			return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to check rule %s", ruleID)).
				WithData(map[string]interface{}{
					"rule_id": ruleID,
					"method":  route.Method,
					"path":    route.Path,
				})
		}

		if err == gorm.ErrRecordNotFound {
			// Rule chưa tồn tại, tạo mới với tất cả thông tin từ code
			rule := &models.Rule{
				ID:          ruleID,
				Method:      route.Method,
				Path:        route.FullPath, // Sử dụng FullPath để sync vào DB
				Type:        route.AccessType,
				Roles:       models.FromUintSlice(roleIDs), // Store role IDs instead of names
				Fixed:       route.Fixed,
				Description: route.Description,
				ServiceName: ruleServiceName, // Empty string will be stored as NULL in DB
			}
			if createErr := ruleRepo.Create(rule); createErr != nil {
				return goerrorkit.WrapWithMessage(createErr, fmt.Sprintf("Failed to create rule %s", ruleID)).
					WithData(map[string]interface{}{
						"rule_id": ruleID,
						"method":  route.Method,
						"path":    route.Path,
					})
			}
		} else {
			// Rule đã tồn tại trong DB
			// Luôn đồng bộ Fixed, Description, Method, Path từ code
			// Với Type và Roles:
			//   - Nếu Fixed=true: LUÔN ghi đè từ code (code là source of truth)
			//   - Nếu Fixed=false và Override=true: ghi đè từ code (code là source of truth, nhưng cho phép sửa tạm thời qua API)
			//   - Nếu Fixed=false và Override=false: giữ nguyên từ DB (cho phép người dùng chỉnh sửa qua API)

			// Tạo rule mới với thông tin từ code
			ruleFromCode := &models.Rule{
				ID:          ruleID,
				Method:      route.Method,
				Path:        route.FullPath,
				Type:        route.AccessType,
				Roles:       models.FromUintSlice(roleIDs),
				Fixed:       route.Fixed,
				Description: route.Description,
				ServiceName: ruleServiceName,
			}

			if route.Fixed {
				// Fixed=true: LUÔN ghi đè Type và Roles từ code (code là source of truth)
				// Rule này không thể sửa từ API (bị reject trong RuleService.UpdateRule)
				if updateErr := ruleRepo.Update(ruleFromCode); updateErr != nil {
					return goerrorkit.WrapWithMessage(updateErr, fmt.Sprintf("Failed to update fixed rule %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			} else if route.Override {
				// Fixed=false và Override=true: ghi đè Type và Roles từ code
				// Cho phép sửa từ API tạm thời, nhưng khi sync sẽ bị ghi đè lại
				if updateErr := ruleRepo.Update(ruleFromCode); updateErr != nil {
					return goerrorkit.WrapWithMessage(updateErr, fmt.Sprintf("Failed to update override rule %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			} else {
				// Fixed=false và Override=false: chỉ update metadata (Fixed, Description, Method, Path)
				// Giữ nguyên Type và Roles từ DB (cho phép người dùng chỉnh sửa qua API)
				ruleToUpdate := &models.Rule{
					ID:          ruleID,
					Method:      route.Method,      // Update từ code
					Path:        route.FullPath,    // Update từ code
					Type:        existingRule.Type, // Giữ nguyên từ DB
					Roles:       existingRule.Roles, // Giữ nguyên từ DB
					Fixed:       route.Fixed,        // Update từ code
					Description: route.Description,  // Update từ code
					ServiceName: ruleServiceName,    // Update từ code
				}
				if updateErr := ruleRepo.Update(ruleToUpdate); updateErr != nil {
					return goerrorkit.WrapWithMessage(updateErr, fmt.Sprintf("Failed to update rule metadata %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			}
		}
	}

	return nil
}
