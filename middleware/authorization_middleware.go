package middleware

import (
	"fmt"
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
	ruleRepo           *repository.RuleRepository
	roleRepo           *repository.RoleRepository
	userRepo           *repository.UserRepository
	exactRulesMap      map[string][]models.Rule // key = "METHOD|PATH" for O(1) lookup
	patternRules       []models.Rule            // Rules with wildcard patterns
	cacheMutex         sync.RWMutex
	lastRefresh        time.Time
	cacheTTL           time.Duration
	superAdminID       *uint           // Cached super_admin role ID (nil if not loaded yet)
	roleNameToIDMap    map[string]uint // Cached role name -> ID mapping for X-Role-Context
	roleNameCacheMutex sync.RWMutex    // Mutex for role name cache
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(
	ruleRepo *repository.RuleRepository,
	roleRepo *repository.RoleRepository,
	userRepo *repository.UserRepository,
) *AuthorizationMiddleware {
	mw := &AuthorizationMiddleware{
		ruleRepo:           ruleRepo,
		roleRepo:           roleRepo,
		userRepo:           userRepo,
		cacheTTL:           5 * time.Minute, // Cache rules for 5 minutes
		exactRulesMap:      make(map[string][]models.Rule),
		patternRules:       []models.Rule{},
		superAdminID:       nil,
		roleNameToIDMap:    make(map[string]uint),
		roleNameCacheMutex: sync.RWMutex{},
	}

	// Load initial rules
	mw.refreshCache()
	// Load super_admin ID and common role names cache
	mw.loadRoleNameCache()

	return mw
}

// Authorize middleware checks authorization based on rules
func (m *AuthorizationMiddleware) Authorize() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		path := c.Path()

		// Refresh cache if needed
		// m.refreshCacheIfNeeded()

		// Get all matching rules
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
			if rule.Type == models.AccessPublic {
				return c.Next()
			}
		}

		// All other rule types require authentication
		// Reject anonymous users immediately
		user, ok := GetUserFromContext(c)
		if !ok {
			return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
		}

		// Get role IDs from context (from validated JWT token - no DB query needed)
		// Role IDs are safe because they come from a token that has been validated
		// If hacker modified role_ids, ValidateToken would have failed
		roleIDs, ok := GetRoleIDsFromContext(c)
		if !ok {
			// Fallback: if roles not in context, query from DB (backward compatibility)
			userRoles, err := m.roleRepo.ListRolesOfUser(user.ID)
			if err != nil {
				return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy roles của user").WithData(map[string]interface{}{
					"user_id": user.ID,
				})
			}
			roleIDs = make([]uint, 0, len(userRoles))
			for _, role := range userRoles {
				roleIDs = append(roleIDs, role.ID)
			}
		}

		// Convert role IDs to map for O(1) lookup (no DB query needed!)
		userRoleIDs := make(map[uint]bool, len(roleIDs))
		for _, roleID := range roleIDs {
			userRoleIDs[roleID] = true
		}

		// Check for super admin role (bypass all rules) - using cached ID
		if m.getSuperAdminID() != nil && userRoleIDs[*m.getSuperAdminID()] {
			return c.Next()
		}

		// Check for optional X-Role-Context header
		// If provided, validate that user has this role and filter to only that role
		roleContext := c.Get("X-Role-Context")
		if roleContext != "" {
			roleContextID, err := m.getRoleIDByName(roleContext)
			if err != nil {
				return goerrorkit.NewAuthError(403, "Không có quyền sử dụng role context này").WithData(map[string]interface{}{
					"requested_role": roleContext,
					"error":          "Role không tồn tại",
				})
			}
			if !userRoleIDs[roleContextID] {
				return goerrorkit.NewAuthError(403, "Không có quyền sử dụng role context này").WithData(map[string]interface{}{
					"requested_role": roleContext,
					"user_role_ids":  roleIDs,
				})
			}
			// Filter to only use the specified role context
			userRoleIDs = map[uint]bool{roleContextID: true}
		}

		// Process rules: FORBIDE rules have higher priority than ALLOW rules
		hasForbidMatch := false
		hasAllowMatch := false

		for _, rule := range matchingRules {
			switch rule.Type {
			case models.AccessForbid:
				// FORBIDE rules: check if user has forbidden roles
				if len(rule.Roles) == 0 {
					// Empty roles array means forbid everyone
					hasForbidMatch = true
				} else {
					// Check if user has any of the forbidden roles (compare IDs directly)
					for _, roleID := range rule.Roles {
						if userRoleIDs[roleID] {
							hasForbidMatch = true
							break
						}
					}
				}

			case models.AccessAllow:
				// ALLOW rules: check if user has allowed roles
				// Empty roles array means any authenticated user can access
				if len(rule.Roles) == 0 {
					hasAllowMatch = true
				} else {
					// Check if user has any of the allowed roles (compare IDs directly)
					for _, roleID := range rule.Roles {
						if userRoleIDs[roleID] {
							hasAllowMatch = true
							break
						}
					}
				}
			}
		}

		// Apply priority: FORBIDE > ALLOW
		if hasForbidMatch {
			return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
				"method":        method,
				"path":          path,
				"user_role_ids": roleIDs,
			})
		}

		if hasAllowMatch {
			return c.Next()
		}

		// Default deny if no rule matches user's roles
		return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
			"method":        method,
			"path":          path,
			"user_role_ids": roleIDs,
		})
	}
}

// findMatchingRules finds all matching rules for method and path
// Returns exact matches first, then pattern matches
// Optimized with O(1) lookup for exact matches using map
func (m *AuthorizationMiddleware) findMatchingRules(method, path string) []models.Rule {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// O(1) lookup for exact match
	key := fmt.Sprintf("%s|%s", method, path)
	exactMatches, hasExactMatch := m.exactRulesMap[key]

	// If exact match found, return it immediately (no need to check patterns)
	if hasExactMatch && len(exactMatches) > 0 {
		return exactMatches
	}

	// Only check pattern matching if no exact match found
	var patternMatches []models.Rule
	for _, rule := range m.patternRules {
		if rule.Method == method && m.matchPath(rule.Path, path) {
			patternMatches = append(patternMatches, rule)
		}
	}

	return patternMatches
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

// refreshCache refreshes the rules cache and rebuilds the map structure
func (m *AuthorizationMiddleware) refreshCache() {
	rules, err := m.ruleRepo.GetAllRulesForCache()
	if err != nil {
		// Log error but don't fail
		return
	}

	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	// Rebuild exact rules map and pattern rules list
	exactRulesMap := make(map[string][]models.Rule)
	patternRules := make([]models.Rule, 0)

	for _, rule := range rules {
		// Check if rule contains wildcard pattern
		// Rules với path parameters đã được convert thành * khi sync vào DB từ auth_router.go
		if strings.Contains(rule.Path, "*") {
			patternRules = append(patternRules, rule)
		} else {
			// Build key for exact match: "METHOD|PATH"
			key := fmt.Sprintf("%s|%s", rule.Method, rule.Path)
			exactRulesMap[key] = append(exactRulesMap[key], rule)
		}
	}

	// Update cache
	m.exactRulesMap = exactRulesMap
	m.patternRules = patternRules
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

// loadRoleNameCache loads super_admin ID and common role names into cache
// This is called once at initialization to avoid repeated DB queries
func (m *AuthorizationMiddleware) loadRoleNameCache() {
	// Load super_admin ID
	superAdmin, err := m.roleRepo.GetByName("super_admin")
	if err == nil {
		m.roleNameCacheMutex.Lock()
		m.superAdminID = &superAdmin.ID
		m.roleNameToIDMap["super_admin"] = superAdmin.ID
		m.roleNameCacheMutex.Unlock()
	}

	// Pre-load common role names (optional optimization)
	commonRoles := []string{"admin", "editor", "author", "reader"}
	roleMap, err := m.roleRepo.GetIDsByNames(commonRoles)
	if err == nil {
		m.roleNameCacheMutex.Lock()
		for name, id := range roleMap {
			m.roleNameToIDMap[name] = id
		}
		m.roleNameCacheMutex.Unlock()
	}
}

// getSuperAdminID returns cached super_admin role ID
func (m *AuthorizationMiddleware) getSuperAdminID() *uint {
	m.roleNameCacheMutex.RLock()
	defer m.roleNameCacheMutex.RUnlock()
	return m.superAdminID
}

// getRoleIDByName gets role ID by name with caching
// First checks cache, if not found, queries DB and updates cache
func (m *AuthorizationMiddleware) getRoleIDByName(roleName string) (uint, error) {
	// Check cache first
	m.roleNameCacheMutex.RLock()
	if roleID, exists := m.roleNameToIDMap[roleName]; exists {
		m.roleNameCacheMutex.RUnlock()
		return roleID, nil
	}
	m.roleNameCacheMutex.RUnlock()

	// Not in cache, query DB
	role, err := m.roleRepo.GetByName(roleName)
	if err != nil {
		return 0, err
	}

	// Update cache
	m.roleNameCacheMutex.Lock()
	m.roleNameToIDMap[roleName] = role.ID
	m.roleNameCacheMutex.Unlock()

	return role.ID, nil
}
