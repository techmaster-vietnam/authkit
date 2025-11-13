package repository

import (
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// RuleRepository handles rule database operations
type RuleRepository struct {
	db *gorm.DB
}

// NewRuleRepository creates a new rule repository
func NewRuleRepository(db *gorm.DB) *RuleRepository {
	return &RuleRepository{db: db}
}

// Create creates a new rule
func (r *RuleRepository) Create(rule *models.Rule) error {
	return r.db.Create(rule).Error
}

// GetByID gets a rule by ID
func (r *RuleRepository) GetByID(id string) (*models.Rule, error) {
	var rule models.Rule
	err := r.db.Where("id = ?", id).First(&rule).Error
	return &rule, err
}

// GetByMethodAndPath gets a rule by method and path
func (r *RuleRepository) GetByMethodAndPath(method, path string) (*models.Rule, error) {
	var rule models.Rule
	err := r.db.Where("method = ? AND path = ?", method, path).First(&rule).Error
	return &rule, err
}

// Update updates a rule
func (r *RuleRepository) Update(rule *models.Rule) error {
	return r.db.Save(rule).Error
}

// Delete hard deletes a rule
func (r *RuleRepository) Delete(id string) error {
	return r.db.Unscoped().Delete(&models.Rule{}, id).Error
}

// List lists all rules
func (r *RuleRepository) List() ([]models.Rule, error) {
	var rules []models.Rule
	err := r.db.Find(&rules).Error
	return rules, err
}

// GetAllRulesForCache gets all rules for caching (used by middleware)
func (r *RuleRepository) GetAllRulesForCache() ([]models.Rule, error) {
	var rules []models.Rule
	err := r.db.Find(&rules).Error
	return rules, err
}
