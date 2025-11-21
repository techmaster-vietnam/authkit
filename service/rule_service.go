package service

import (
	"errors"
	"fmt"

	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
	"gorm.io/gorm"
)

// RuleService handles rule business logic
type RuleService struct {
	ruleRepo *repository.RuleRepository
	roleRepo *repository.RoleRepository
}

// NewRuleService creates a new rule service
func NewRuleService(ruleRepo *repository.RuleRepository, roleRepo *repository.RoleRepository) *RuleService {
	return &RuleService{
		ruleRepo: ruleRepo,
		roleRepo: roleRepo,
	}
}

// AddRuleRequest represents add rule request
type AddRuleRequest struct {
	Method string   `json:"method"`
	Path   string   `json:"path"`
	Type   string   `json:"type"` // PUBLIC, ALLOW, FORBIDE
	Roles  []string `json:"roles"`
}

// UpdateRuleRequest represents update rule request
type UpdateRuleRequest struct {
	Type  string   `json:"type"`
	Roles []string `json:"roles"`
}

// AddRule creates a new rule
func (s *RuleService) AddRule(req AddRuleRequest) (*models.Rule, error) {
	// Validate input
	if req.Method == "" {
		return nil, goerrorkit.NewValidationError("Method là bắt buộc", map[string]interface{}{
			"field": "method",
		})
	}
	if req.Path == "" {
		return nil, goerrorkit.NewValidationError("Path là bắt buộc", map[string]interface{}{
			"field": "path",
		})
	}

	ruleType := models.AccessType(req.Type)
	if ruleType != models.AccessPublic && ruleType != models.AccessAllow &&
		ruleType != models.AccessForbid {
		return nil, goerrorkit.NewValidationError("Type phải là PUBLIC, ALLOW hoặc FORBIDE", map[string]interface{}{
			"field":    "type",
			"received": req.Type,
			"allowed":  []string{"PUBLIC", "ALLOW", "FORBIDE"},
		})
	}

	// Check if rule already exists
	_, err := s.ruleRepo.GetByMethodAndPath(req.Method, req.Path)
	if err == nil {
		return nil, goerrorkit.NewBusinessError(409, "Rule đã tồn tại cho method và path này").WithData(map[string]interface{}{
			"method": req.Method,
			"path":   req.Path,
		})
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi kiểm tra rule")
	}

	// Convert role names to role IDs
	roleIDs := make([]uint, 0)
	if len(req.Roles) > 0 {
		roleNameToIDMap, err := s.roleRepo.GetIDsByNames(req.Roles)
		if err != nil {
			return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi convert role names sang IDs").
				WithData(map[string]interface{}{
					"role_names": req.Roles,
				})
		}
		// Convert map to slice, skip roles that don't exist
		for _, roleName := range req.Roles {
			if roleID, exists := roleNameToIDMap[roleName]; exists {
				roleIDs = append(roleIDs, roleID)
			}
		}
	}

	// Generate ID from Method and Path
	ruleID := fmt.Sprintf("%s|%s", req.Method, req.Path)
	
	// Get service name from repository (empty in single-app mode)
	serviceName := s.ruleRepo.GetServiceName()
	// Truncate to max 20 characters if longer
	if len(serviceName) > 20 {
		serviceName = serviceName[:20]
	}
	
	rule := &models.Rule{
		ID:          ruleID,
		Method:      req.Method,
		Path:        req.Path,
		Type:        ruleType,
		Roles:       models.FromUintSlice(roleIDs), // Store role IDs instead of names
		ServiceName: serviceName,                   // Auto-set from repository (empty = NULL in DB)
	}

	if err := s.ruleRepo.Create(rule); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi tạo rule")
	}

	return rule, nil
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

	if req.Type != "" {
		ruleType := models.AccessType(req.Type)
		if ruleType != models.AccessPublic && ruleType != models.AccessAllow &&
			ruleType != models.AccessForbid {
			return nil, goerrorkit.NewValidationError("Type phải là PUBLIC, ALLOW hoặc FORBIDE", map[string]interface{}{
				"field":    "type",
				"received": req.Type,
				"allowed":  []string{"PUBLIC", "ALLOW", "FORBIDE"},
			})
		}
		rule.Type = ruleType
	}

	if req.Roles != nil {
		// Convert role names to role IDs
		roleIDs := make([]uint, 0)
		if len(req.Roles) > 0 {
			roleNameToIDMap, err := s.roleRepo.GetIDsByNames(req.Roles)
			if err != nil {
				return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi convert role names sang IDs").
					WithData(map[string]interface{}{
						"role_names": req.Roles,
					})
			}
			// Convert map to slice, skip roles that don't exist
			for _, roleName := range req.Roles {
				if roleID, exists := roleNameToIDMap[roleName]; exists {
					roleIDs = append(roleIDs, roleID)
				}
			}
		}
		rule.Roles = models.FromUintSlice(roleIDs) // Store role IDs instead of names
	}

	if err := s.ruleRepo.Update(rule); err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi cập nhật rule")
	}

	return rule, nil
}

// RemoveRule removes a rule
func (s *RuleService) RemoveRule(ruleID string) error {
	if err := s.ruleRepo.Delete(ruleID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return goerrorkit.NewBusinessError(404, "Không tìm thấy rule").WithData(map[string]interface{}{
				"rule_id": ruleID,
			})
		}
		return goerrorkit.WrapWithMessage(err, "Lỗi khi xóa rule")
	}
	return nil
}

// ListRules lists all rules
func (s *RuleService) ListRules() ([]models.Rule, error) {
	rules, err := s.ruleRepo.List()
	if err != nil {
		return nil, goerrorkit.WrapWithMessage(err, "Lỗi khi lấy danh sách rule")
	}
	return rules, nil
}
