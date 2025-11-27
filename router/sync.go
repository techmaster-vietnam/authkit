package router

import (
	"fmt"

	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// SyncRoutesToDatabase đồng bộ routes từ code vào database
//   - Nếu Override=true: luôn ghi đè cấu hình từ code lên database (luôn update)
//   - Nếu Override=false và Fixed=true: chỉ tạo mới nếu chưa tồn tại, không update
//   - Nếu Override=false và Fixed=false: chỉ tạo mới nếu chưa tồn tại, không update nếu đã tồn tại
//     (giữ nguyên Type và Roles từ database vì đó là mong muốn của người dùng)
//   - Convert role names (string) từ routes → role IDs (uint) khi lưu vào DB
//   - Tự động xóa rules không còn trong registry (cleanup rules cũ)
//   - serviceName: nếu empty, rule sẽ có service_name = NULL (single-app mode)
//                  nếu set, rule sẽ có service_name = serviceName (microservice mode)
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

		if route.Override {
			// Override=true: luôn ghi đè cấu hình từ code lên database
			_, err := ruleRepo.GetByID(ruleID)
			if err == gorm.ErrRecordNotFound {
				// Rule chưa tồn tại, tạo mới
				if createErr := ruleRepo.Create(rule); createErr != nil {
					return goerrorkit.WrapWithMessage(createErr, fmt.Sprintf("Failed to create override rule %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			} else if err != nil {
				// Lỗi khác khi query
				return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to check override rule %s", ruleID)).
					WithData(map[string]interface{}{
						"rule_id": ruleID,
						"method":  route.Method,
						"path":    route.Path,
					})
			} else {
				// Rule đã tồn tại, update để ghi đè từ code
				if updateErr := ruleRepo.Update(rule); updateErr != nil {
					return goerrorkit.WrapWithMessage(updateErr, fmt.Sprintf("Failed to update override rule %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			}
		} else if route.Fixed {
			// Override=false và Fixed=true: chỉ tạo mới nếu chưa tồn tại, không update
			_, err := ruleRepo.GetByID(ruleID)
			if err == gorm.ErrRecordNotFound {
				// Rule chưa tồn tại, tạo mới
				if createErr := ruleRepo.Create(rule); createErr != nil {
					return goerrorkit.WrapWithMessage(createErr, fmt.Sprintf("Failed to create fixed rule %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			} else if err != nil {
				// Lỗi khác khi query
				return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to check fixed rule %s", ruleID)).
					WithData(map[string]interface{}{
						"rule_id": ruleID,
						"method":  route.Method,
						"path":    route.Path,
					})
			}
			// Rule đã tồn tại, bỏ qua (không update)
		} else {
			// Override=false và Fixed=false: chỉ tạo mới nếu chưa tồn tại, không update nếu đã tồn tại
			// Giữ nguyên Type và Roles từ database vì đó là mong muốn của người dùng
			_, err := ruleRepo.GetByID(ruleID)
			if err == gorm.ErrRecordNotFound {
				// Rule chưa tồn tại, tạo mới
				if createErr := ruleRepo.Create(rule); createErr != nil {
					return goerrorkit.WrapWithMessage(createErr, fmt.Sprintf("Failed to create rule %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			} else if err != nil {
				// Lỗi khác khi query
				return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to check rule %s", ruleID)).
					WithData(map[string]interface{}{
						"rule_id": ruleID,
						"method":  route.Method,
						"path":    route.Path,
					})
			}
			// Rule đã tồn tại, bỏ qua (không update) để giữ nguyên Type và Roles từ database
		}
	}

	return nil
}
