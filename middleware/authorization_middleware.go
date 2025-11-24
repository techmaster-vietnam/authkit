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
	ruleRepo                    *repository.RuleRepository
	roleRepo                    *repository.RoleRepository
	userRepo                    *repository.UserRepository
	exactRulesMap               map[string][]models.Rule         // key = "METHOD|PATH" for O(1) lookup
	patternRulesByMethodAndSegs map[string]map[int][]models.Rule // Optimized: key1=method, key2=segmentCount
	cacheMutex                  sync.RWMutex
	lastRefresh                 time.Time
	cacheTTL                    time.Duration
	superAdminID                *uint           // Cached super_admin role ID (nil if not loaded yet)
	roleNameToIDMap             map[string]uint // Cached role name -> ID mapping for X-Role-Context
	roleNameCacheMutex          sync.RWMutex    // Mutex for role name cache
}

// NewAuthorizationMiddleware creates a new authorization middleware
func NewAuthorizationMiddleware(
	ruleRepo *repository.RuleRepository,
	roleRepo *repository.RoleRepository,
	userRepo *repository.UserRepository,
) *AuthorizationMiddleware {
	mw := &AuthorizationMiddleware{
		ruleRepo:                    ruleRepo,
		roleRepo:                    roleRepo,
		userRepo:                    userRepo,
		cacheTTL:                    5 * time.Minute, // Cache rules for 5 minutes
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		superAdminID:                nil,
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	// Load initial rules
	mw.refreshCache()
	// Load super_admin ID and common role names cache
	mw.loadRoleNameCache()

	return mw
}

// authorizeHandler is the actual authorization handler function
// Separated from Authorize() to allow easier debugging (named function instead of closure)
func (m *AuthorizationMiddleware) authorizeHandler(c *fiber.Ctx) error {
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

	// Check for PUBLIC rule first (allows anonymous) - early exit
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
	// Early exit optimization: check FORBIDE first, if found, reject immediately
	for _, rule := range matchingRules {
		if rule.Type == models.AccessForbid {
			// FORBIDE rules: check if user has forbidden roles
			if len(rule.Roles) == 0 {
				// Empty roles array means forbid everyone - reject immediately
				return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
					"method":        method,
					"path":          path,
					"user_role_ids": roleIDs,
				})
			}
			// Check if user has any of the forbidden roles (compare IDs directly)
			for _, roleID := range rule.Roles {
				if userRoleIDs[roleID] {
					return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
						"method":        method,
						"path":          path,
						"user_role_ids": roleIDs,
					})
				}
			}
		}
	}

	// Check ALLOW rules only if no FORBIDE match
	for _, rule := range matchingRules {
		if rule.Type == models.AccessAllow {
			// ALLOW rules: check if user has allowed roles
			// Empty roles array means any authenticated user can access
			if len(rule.Roles) == 0 {
				return c.Next()
			}
			// Check if user has any of the allowed roles (compare IDs directly)
			for _, roleID := range rule.Roles {
				if userRoleIDs[roleID] {
					return c.Next()
				}
			}
		}
	}

	// Default deny if no rule matches user's roles
	return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
		"method":        method,
		"path":          path,
		"user_role_ids": roleIDs,
	})
}

// Authorize middleware checks authorization based on rules
func (m *AuthorizationMiddleware) Authorize() fiber.Handler {
	return m.authorizeHandler
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
	// Optimized: only check patterns with same method and segment count
	pathSegments := m.countSegments(path)
	methodPatterns, hasMethodPatterns := m.patternRulesByMethodAndSegs[method]
	if !hasMethodPatterns {
		return nil
	}

	// Only check patterns with matching segment count
	rulesToCheck, hasMatchingSegments := methodPatterns[pathSegments]
	if !hasMatchingSegments {
		return nil
	}

	// Now only check the filtered rules (much smaller set)
	var patternMatches []models.Rule
	for _, rule := range rulesToCheck {
		if m.matchPath(rule.Path, path) {
			patternMatches = append(patternMatches, rule)
		}
	}

	return patternMatches
}

// countSegments counts the number of segments in a path (optimized, no allocation)
// Example: "/api/users/123" -> 3 segments
func (m *AuthorizationMiddleware) countSegments(path string) int {
	if len(path) == 0 || path == "/" {
		return 0
	}
	// Count '/' characters (faster than Split)
	// Skip leading '/' if present
	start := 0
	if path[0] == '/' {
		start = 1
	}
	if start >= len(path) {
		return 0
	}
	// Count segments by counting '/' + 1
	return strings.Count(path[start:], "/") + 1
}

// matchPath matches path pattern (supports * wildcard)
// Optimized: assumes pattern and path have same segment count (pre-filtered)
// Uses manual segment comparison to avoid allocations from strings.Split
func (m *AuthorizationMiddleware) matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// Manual segment-by-segment comparison (no allocation)
	patternLen := len(pattern)
	pathLen := len(path)
	patternIdx := 0
	pathIdx := 0

	// Skip leading slash if present
	if patternIdx < patternLen && pattern[patternIdx] == '/' {
		patternIdx++
	}
	if pathIdx < pathLen && path[pathIdx] == '/' {
		pathIdx++
	}

	// Compare segments one by one
	for patternIdx < patternLen && pathIdx < pathLen {
		// Find end of current segment in pattern
		patternStart := patternIdx
		for patternIdx < patternLen && pattern[patternIdx] != '/' {
			patternIdx++
		}
		patternSeg := pattern[patternStart:patternIdx]

		// Find end of current segment in path
		pathStart := pathIdx
		for pathIdx < pathLen && path[pathIdx] != '/' {
			pathIdx++
		}
		pathSeg := path[pathStart:pathIdx]

		// Compare segments: wildcard matches anything
		if patternSeg != "*" && patternSeg != pathSeg {
			return false
		}

		// Skip trailing slash before next segment
		if patternIdx < patternLen {
			patternIdx++
		}
		if pathIdx < pathLen {
			pathIdx++
		}
	}

	// Both should have reached the end
	return patternIdx >= patternLen && pathIdx >= pathLen
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

	// Rebuild exact rules map and optimized pattern rules index
	exactRulesMap := make(map[string][]models.Rule)
	patternRulesByMethodAndSegs := make(map[string]map[int][]models.Rule)

	for _, rule := range rules {
		// Check if rule contains wildcard pattern
		// Rules với path parameters đã được convert thành * khi sync vào DB từ auth_router.go
		if strings.Contains(rule.Path, "*") {
			// Index by method and segment count for O(1) filtering
			segmentCount := m.countSegments(rule.Path)
			if patternRulesByMethodAndSegs[rule.Method] == nil {
				patternRulesByMethodAndSegs[rule.Method] = make(map[int][]models.Rule)
			}
			patternRulesByMethodAndSegs[rule.Method][segmentCount] = append(
				patternRulesByMethodAndSegs[rule.Method][segmentCount],
				rule,
			)
		} else {
			// Build key for exact match: "METHOD|PATH"
			key := fmt.Sprintf("%s|%s", rule.Method, rule.Path)
			exactRulesMap[key] = append(exactRulesMap[key], rule)
		}
	}

	// Update cache atomically
	m.exactRulesMap = exactRulesMap
	m.patternRulesByMethodAndSegs = patternRulesByMethodAndSegs
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
