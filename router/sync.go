package router

import (
	"fmt"

	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// SyncRoutesToDatabase đồng bộ routes từ code vào database
// - Nếu Fixed=true: chỉ tạo mới nếu chưa tồn tại, không update
// - Nếu Fixed=false: upsert (tạo mới hoặc update)
// - Convert role names (string) từ routes → role IDs (uint) khi lưu vào DB
func SyncRoutesToDatabase(registry *RouteRegistry, ruleRepo *repository.RuleRepository, roleRepo *repository.RoleRepository) error {
	routes := registry.GetAllRoutes()

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
		}

		if route.Fixed {
			// Fixed=true: chỉ tạo mới nếu chưa tồn tại, không update
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
			// Fixed=false: upsert (tạo mới hoặc update)
			// Thử update trước, nếu không tồn tại thì create
			_, err := ruleRepo.GetByID(ruleID)
			if err == gorm.ErrRecordNotFound {
				// Chưa tồn tại, tạo mới
				if createErr := ruleRepo.Create(rule); createErr != nil {
					return goerrorkit.WrapWithMessage(createErr, fmt.Sprintf("Failed to create rule %s", ruleID)).
						WithData(map[string]interface{}{
							"rule_id": ruleID,
							"method":  route.Method,
							"path":    route.Path,
						})
				}
			} else if err != nil {
				return goerrorkit.WrapWithMessage(err, fmt.Sprintf("Failed to check rule %s", ruleID)).
					WithData(map[string]interface{}{
						"rule_id": ruleID,
						"method":  route.Method,
						"path":    route.Path,
					})
			} else {
				// Đã tồn tại, update
				if updateErr := ruleRepo.Update(rule); updateErr != nil {
					return goerrorkit.WrapWithMessage(updateErr, fmt.Sprintf("Failed to update rule %s", ruleID)).
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

