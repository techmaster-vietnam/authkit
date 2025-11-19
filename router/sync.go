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
func SyncRoutesToDatabase(registry *RouteRegistry, ruleRepo *repository.RuleRepository) error {
	routes := registry.GetAllRoutes()

	for _, route := range routes {
		ruleID := fmt.Sprintf("%s|%s", route.Method, route.FullPath)
		
		rule := &models.Rule{
			ID:          ruleID,
			Method:      route.Method,
			Path:        route.FullPath, // Sử dụng FullPath để sync vào DB
			Type:        route.AccessType,
			Roles:       route.Roles,
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

