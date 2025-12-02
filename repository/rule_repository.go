package repository

import (
	"github.com/techmaster-vietnam/authkit/models"
	"gorm.io/gorm"
)

// RuleFilter represents filter parameters for listing rules
type RuleFilter struct {
	Method  string // GET, POST, PUT, DELETE
	Path    string // Path chứa chuỗi (ví dụ: "blog")
	Type    string // PUBLIC, ALLOW, FORBID
	Fixed   *bool  // true hoặc false (pointer để phân biệt không có giá trị và false)
	Service string // Service name để tìm kiếm trên cột service_name (case-insensitive)
}

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

// Update updates a rule
func (r *RuleRepository) Update(rule *models.Rule) error {
	return r.db.Save(rule).Error
}

// Delete hard deletes a rule by ID
// Chỉ được sử dụng trong quá trình sync routes để cleanup rules cũ
// Không expose qua API để tránh xóa nhầm rules
func (r *RuleRepository) Delete(id string) error {
	return r.db.Unscoped().Where("id = ?", id).Delete(&models.Rule{}).Error
}

// List lists all rules (filtered by service_name if set and optional filters)
func (r *RuleRepository) List(filter RuleFilter) ([]models.Rule, error) {
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

	// Apply filters (AND logic)
	if filter.Method != "" {
		query = query.Where("method = ?", filter.Method)
	}
	if filter.Path != "" {
		query = query.Where("path LIKE ?", "%"+filter.Path+"%")
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Fixed != nil {
		query = query.Where("fixed = ?", *filter.Fixed)
	}
	if filter.Service != "" {
		// Case-insensitive search using LOWER() for PostgreSQL compatibility
		query = query.Where("LOWER(service_name) = LOWER(?)", filter.Service)
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

// GetRulesByRole gets all rules that a specific role can access
// Uses PostgreSQL function get_rules_by_role for better performance
// If roleName is "super_admin", returns all rules
// Otherwise, returns:
// - All rules with type = "PUBLIC"
// - All rules with type = "ALLOW" and roles = [] (empty array)
// - All rules with type = "ALLOW" and roles contains roleID
func (r *RuleRepository) GetRulesByRole(roleID uint, roleName string) ([]models.Rule, error) {
	var rules []models.Rule

	// Prepare service_name parameter (NULL for single-app mode)
	var serviceNameParam interface{}
	if r.serviceName != "" {
		serviceNameParam = r.serviceName
	} else {
		serviceNameParam = nil
	}

	// Call PostgreSQL function using GORM Raw with Model() to ensure proper struct mapping
	// Specify columns explicitly to match function return type
	// Column order must match the struct field order: ID, Method, Path, Type, Roles, Fixed, Description, ServiceName
	err := r.db.Model(&models.Rule{}).
		Raw("SELECT id, method, path, type, roles, fixed, description, service_name FROM get_rules_by_role(?, ?, ?)",
			roleID, roleName, serviceNameParam).
		Scan(&rules).Error

	return rules, err
}
