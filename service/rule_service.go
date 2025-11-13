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
}

// NewRuleService creates a new rule service
func NewRuleService(ruleRepo *repository.RuleRepository) *RuleService {
	return &RuleService{ruleRepo: ruleRepo}
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

	// Generate ID from Method and Path
	ruleID := fmt.Sprintf("%s|%s", req.Method, req.Path)
	rule := &models.Rule{
		ID:     ruleID,
		Method: req.Method,
		Path:   req.Path,
		Type:   ruleType,
		Roles:  req.Roles,
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
		rule.Roles = req.Roles
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
