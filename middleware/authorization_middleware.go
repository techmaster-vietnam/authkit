package middleware

import (
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
)

// AuthorizationMiddleware handles authorization based on rules
type AuthorizationMiddleware struct {
	ruleRepo    *repository.RuleRepository
	roleRepo    *repository.RoleRepository
	userRepo    *repository.UserRepository
	rulesCache  []models.Rule
	cacheMutex  sync.RWMutex
	lastRefresh time.Time
	cacheTTL    time.Duration
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(
	ruleRepo *repository.RuleRepository,
	roleRepo *repository.RoleRepository,
	userRepo *repository.UserRepository,
) *AuthorizationMiddleware {
	mw := &AuthorizationMiddleware{
		ruleRepo:   ruleRepo,
		roleRepo:   roleRepo,
		userRepo:   userRepo,
		cacheTTL:   5 * time.Minute, // Cache rules for 5 minutes
		rulesCache: []models.Rule{},
	}

	// Load initial rules
	mw.refreshCache()

	return mw
}

// Authorize middleware checks authorization based on rules
func (m *AuthorizationMiddleware) Authorize() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		path := c.Path()

		// Refresh cache if needed
		m.refreshCacheIfNeeded()

		// Get all matching rules (sorted by priority)
		matchingRules := m.findMatchingRules(method, path)

		// If no rule found, default to FORBIDE
		if len(matchingRules) == 0 {
			return goerrorkit.NewAuthError(403, "Không có quyền truy cập endpoint này").WithData(map[string]interface{}{
				"method": method,
				"path":   path,
			})
		}

		// Check for PUBLIC rule first (allows anonymous)
		for _, rule := range matchingRules {
			if rule.Type == models.RuleTypePublic {
				return c.Next()
			}
		}

		// All other rule types require authentication
		// Reject anonymous users immediately
		user, ok := GetUserFromContext(c)
		if !ok {
			return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
		}

		// Get user roles (with optional role context from header)
		userRoles, err := m.roleRepo.ListRolesOfUser(user.ID)
		if err != nil {
			return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy roles của user").WithData(map[string]interface{}{
				"user_id": user.ID,
			})
		}

		userRoleNames := make(map[string]bool)
		for _, role := range userRoles {
			userRoleNames[role.Name] = true
		}

		// Check for optional X-Role-Context header
		// If provided, validate that user has this role
		roleContext := c.Get("X-Role-Context")
		if roleContext != "" {
			if !userRoleNames[roleContext] {
				return goerrorkit.NewAuthError(403, "Không có quyền sử dụng role context này").WithData(map[string]interface{}{
					"requested_role": roleContext,
					"user_roles":     userRoleNames,
				})
			}
			// Filter to only use the specified role context
			userRoleNames = map[string]bool{roleContext: true}
		}

		// Check for super admin role (bypass all rules)
		if userRoleNames["super_admin"] {
			return c.Next()
		}

		// Process rules in priority order
		// FORBIDE rules have higher priority than ALLOW rules
		hasForbidMatch := false
		hasAllowMatch := false
		hasAuthMatch := false

		for _, rule := range matchingRules {
			switch rule.Type {
			case models.RuleTypeForbid:
				// FORBIDE rules: check if user has forbidden roles
				if len(rule.Roles) == 0 {
					// Empty roles array means forbid everyone
					hasForbidMatch = true
				} else {
					// Check if user has any of the forbidden roles
					for _, roleName := range rule.Roles {
						if userRoleNames[roleName] {
							hasForbidMatch = true
							break
						}
					}
				}

			case models.RuleTypeAllow:
				// ALLOW rules: check if user has allowed roles
				if len(rule.Roles) == 0 {
					// Empty roles array means any authenticated user can access
					hasAllowMatch = true
				} else {
					// Check if user has any of the allowed roles
					for _, roleName := range rule.Roles {
						if userRoleNames[roleName] {
							hasAllowMatch = true
							break
						}
					}
				}

			case models.RuleTypeAuth:
				// AUTHENTICATED: any authenticated user can access
				hasAuthMatch = true
			}
		}

		// Apply priority: FORBIDE > ALLOW > AUTHENTICATED
		if hasForbidMatch {
			return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
				"method":    method,
				"path":      path,
				"user_roles": userRoleNames,
			})
		}

		if hasAllowMatch || hasAuthMatch {
			return c.Next()
		}

		// Default deny if no rule matches user's roles
		return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
			"method":    method,
			"path":      path,
			"user_roles": userRoleNames,
		})
	}
}

// findMatchingRules finds all matching rules for method and path, sorted by priority
func (m *AuthorizationMiddleware) findMatchingRules(method, path string) []models.Rule {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	var exactMatches []models.Rule
	var patternMatches []models.Rule

	// Find exact matches first
	for i := range m.rulesCache {
		rule := m.rulesCache[i]
		if rule.Method == method && rule.Path == path {
			exactMatches = append(exactMatches, rule)
		}
	}

	// Try pattern matching (simple wildcard support)
	for i := range m.rulesCache {
		rule := m.rulesCache[i]
		// Skip if already found as exact match
		if rule.Method == method && rule.Path != path && m.matchPath(rule.Path, path) {
			patternMatches = append(patternMatches, rule)
		}
	}

	// Combine: exact matches first, then pattern matches
	// Rules are already sorted by priority from database query
	allMatches := append(exactMatches, patternMatches...)

	return allMatches
}

// matchPath matches path pattern (supports * wildcard)
func (m *AuthorizationMiddleware) matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// Simple wildcard matching: * matches any segment
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	if len(patternParts) != len(pathParts) {
		return false
	}

	for i := range patternParts {
		if patternParts[i] != "*" && patternParts[i] != pathParts[i] {
			return false
		}
	}

	return true
}

// refreshCache refreshes the rules cache
func (m *AuthorizationMiddleware) refreshCache() {
	rules, err := m.ruleRepo.GetAllRulesForCache()
	if err != nil {
		// Log error but don't fail
		return
	}

	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	m.rulesCache = rules
	m.lastRefresh = time.Now()
}

// refreshCacheIfNeeded refreshes cache if TTL has expired
func (m *AuthorizationMiddleware) refreshCacheIfNeeded() {
	m.cacheMutex.RLock()
	needsRefresh := time.Since(m.lastRefresh) > m.cacheTTL
	m.cacheMutex.RUnlock()

	if needsRefresh {
		m.refreshCache()
	}
}

// InvalidateCache invalidates the cache (call this after rule changes)
func (m *AuthorizationMiddleware) InvalidateCache() {
	m.refreshCache()
}

