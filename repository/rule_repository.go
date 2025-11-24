package repository

import (
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// RuleRepository handles rule database operations
type RuleRepository struct {
	db          *gorm.DB
	serviceName string // Service name for filtering rules (empty = no filter, backward compatible)
}

// GetServiceName returns the service name configured for this repository
// Returns empty string if in single-app mode
func (r *RuleRepository) GetServiceName() string {
	return r.serviceName
}

// NewRuleRepository creates a new rule repository
// If serviceName is empty, repository works in single-app mode (no filtering)
func NewRuleRepository(db *gorm.DB, serviceName string) *RuleRepository {
	// Truncate to max 20 characters if longer
	if len(serviceName) > 20 {
		serviceName = serviceName[:20]
	}
	return &RuleRepository{
		db:          db,
		serviceName: serviceName,
	}
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
// In microservice mode (serviceName set), also filters by service_name
// In single-app mode (serviceName empty), only matches rules without service_name
func (r *RuleRepository) GetByMethodAndPath(method, path string) (*models.Rule, error) {
	var rule models.Rule
	query := r.db.Where("method = ? AND path = ?", method, path)
	// Filter by service_name if set
	if r.serviceName != "" {
		query = query.Where("service_name = ?", r.serviceName)
	} else {
		// In single-app mode, only match rules without service_name (NULL)
		query = query.Where("service_name IS NULL")
	}
	err := query.First(&rule).Error
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

// List lists all rules (filtered by service_name if set)
func (r *RuleRepository) List() ([]models.Rule, error) {
	var rules []models.Rule
	query := r.db
	// Filter by service_name if set (backward compatible: if empty, no filter)
	if r.serviceName != "" {
		query = query.Where("service_name = ?", r.serviceName)
	} else {
		// In single-app mode, only load rules without service_name (NULL or empty string)
		// Check both NULL and empty string for backward compatibility
		query = query.Where("service_name IS NULL OR service_name = ''")
	}
	err := query.Find(&rules).Error
	return rules, err
}

// GetAllRulesForCache gets all rules for caching (used by middleware)
// Filtered by service_name if set (backward compatible)
func (r *RuleRepository) GetAllRulesForCache() ([]models.Rule, error) {
	var rules []models.Rule
	query := r.db
	// Filter by service_name if set (backward compatible: if empty, no filter)
	if r.serviceName != "" {
		query = query.Where("service_name = ?", r.serviceName)
	} else {
		// In single-app mode, only load rules without service_name (NULL or empty string)
		// Check both NULL and empty string for backward compatibility
		query = query.Where("service_name IS NULL OR service_name = ''")
	}
	err := query.Find(&rules).Error
	return rules, err
}
