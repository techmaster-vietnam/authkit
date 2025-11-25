package middleware

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/core"
	"github.com/techmaster-vietnam/authkit/models"
	"github.com/techmaster-vietnam/authkit/repository"
	"github.com/techmaster-vietnam/goerrorkit"
)

// BaseAuthorizationMiddleware là generic authorization middleware
// TUser phải implement UserInterface, TRole phải implement RoleInterface
type BaseAuthorizationMiddleware[TUser core.UserInterface, TRole core.RoleInterface] struct {
	ruleRepo                    *repository.RuleRepository
	roleRepo                    *repository.BaseRoleRepository[TRole]
	userRepo                    *repository.BaseUserRepository[TUser]
	exactRulesMap               map[string][]models.Rule         // key = "METHOD|PATH" for O(1) lookup
	patternRulesByMethodAndSegs map[string]map[int][]models.Rule // Optimized: key1=method, key2=segmentCount
	cacheMutex                  sync.RWMutex
	lastRefresh                 time.Time
	cacheTTL                    time.Duration
	superAdminID                *uint           // Cached super_admin role ID (nil if not loaded yet)
	roleNameToIDMap             map[string]uint // Cached role name -> ID mapping for X-Role-Context
	roleNameCacheMutex          sync.RWMutex    // Mutex for role name cache
}

// NewBaseAuthorizationMiddleware tạo mới BaseAuthorizationMiddleware với generic types
func NewBaseAuthorizationMiddleware[TUser core.UserInterface, TRole core.RoleInterface](
	ruleRepo *repository.RuleRepository,
	roleRepo *repository.BaseRoleRepository[TRole],
	userRepo *repository.BaseUserRepository[TUser],
) *BaseAuthorizationMiddleware[TUser, TRole] {
	mw := &BaseAuthorizationMiddleware[TUser, TRole]{
		ruleRepo:                    ruleRepo,
		roleRepo:                    roleRepo,
		userRepo:                    userRepo,
		cacheTTL:                    100 * time.Minute, // Cache rules for 100 minutes
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		superAdminID:                nil,
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	return mw
}

// Authorize middleware checks authorization based on rules
func (m *BaseAuthorizationMiddleware[TUser, TRole]) Authorize() fiber.Handler {
	return func(c *fiber.Ctx) error {
		method := c.Method()
		path := c.Path()
		// Get all matching rules
		matchingRules := m.findMatchingRules(method, path)

		// If no rule found, default to FORBID
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
		user, ok := GetUserFromContextGeneric[TUser](c)
		if !ok {
			return goerrorkit.NewAuthError(401, "Yêu cầu đăng nhập")
		}

		// Get role IDs from context (from validated JWT token - no DB query needed)
		roleIDs, ok := GetRoleIDsFromContext(c)
		if !ok {
			// Fallback: if roles not in context, query from DB
			userRoles, err := m.roleRepo.ListRolesOfUser(user.GetID())
			if err != nil {
				return goerrorkit.WrapWithMessage(err, "Lỗi khi lấy roles của user").WithData(map[string]interface{}{
					"user_id": user.GetID(),
				})
			}
			roleIDs = make([]uint, 0, len(userRoles))
			for _, role := range userRoles {
				roleIDs = append(roleIDs, role.GetID())
			}
		}

		// Convert role IDs to map for O(1) lookup
		userRoleIDs := make(map[uint]bool, len(roleIDs))
		for _, roleID := range roleIDs {
			userRoleIDs[roleID] = true
		}

		// Check for super admin role (bypass all rules) - using cached ID
		if m.getSuperAdminID() != nil && userRoleIDs[*m.getSuperAdminID()] {
			return c.Next()
		}

		// Check for optional X-Role-Context header
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

		// Process rules: FORBID rules have higher priority than ALLOW rules
		hasForbidRule := false
		for _, rule := range matchingRules {
			if rule.Type == models.AccessForbid {
				hasForbidRule = true
				// FORBID rules: check if user has forbidden roles
				if len(rule.Roles) == 0 {
					return goerrorkit.NewAuthError(403, "Không có quyền truy cập").WithData(map[string]interface{}{
						"method":        method,
						"path":          path,
						"user_role_ids": roleIDs,
					})
				}
				// Check if user has any of the forbidden roles
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

		// If only FORBID rule exists and user is not forbidden, allow access immediately
		if hasForbidRule {
			// Quick check: does any ALLOW rule exist?
			hasAllowRule := false
			for _, rule := range matchingRules {
				if rule.Type == models.AccessAllow {
					hasAllowRule = true
					break
				}
			}
			if !hasAllowRule {
				return c.Next()
			}
		}

		// Check ALLOW rules only if no FORBID match or FORBID + ALLOW both exist
		for _, rule := range matchingRules {
			if rule.Type == models.AccessAllow {
				// ALLOW rules: check if user has allowed roles
				// Empty roles array means any authenticated user can access
				if len(rule.Roles) == 0 {
					return c.Next()
				}
				// Check if user has any of the allowed roles
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
}

// findMatchingRules finds all matching rules for method and path
func (m *BaseAuthorizationMiddleware[TUser, TRole]) findMatchingRules(method, path string) []models.Rule {
	m.cacheMutex.RLock()
	defer m.cacheMutex.RUnlock()

	// O(1) lookup for exact match
	key := fmt.Sprintf("%s|%s", method, path)
	exactMatches, hasExactMatch := m.exactRulesMap[key]

	if hasExactMatch && len(exactMatches) > 0 {
		return exactMatches
	}

	// Check pattern matching
	pathSegments := m.countSegments(path)
	methodPatterns, hasMethodPatterns := m.patternRulesByMethodAndSegs[method]
	if !hasMethodPatterns {
		return nil
	}

	rulesToCheck, hasMatchingSegments := methodPatterns[pathSegments]
	if !hasMatchingSegments {
		return nil
	}

	var patternMatches []models.Rule
	for _, rule := range rulesToCheck {
		if m.matchPath(rule.Path, path) {
			patternMatches = append(patternMatches, rule)
		}
	}

	return patternMatches
}

// countSegments counts the number of segments in a path
func (m *BaseAuthorizationMiddleware[TUser, TRole]) countSegments(path string) int {
	if len(path) == 0 || path == "/" {
		return 0
	}
	start := 0
	if path[0] == '/' {
		start = 1
	}
	if start >= len(path) {
		return 0
	}
	return strings.Count(path[start:], "/") + 1
}

// matchPath matches path pattern (supports * wildcard)
func (m *BaseAuthorizationMiddleware[TUser, TRole]) matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	patternLen := len(pattern)
	pathLen := len(path)
	patternIdx := 0
	pathIdx := 0

	if patternIdx < patternLen && pattern[patternIdx] == '/' {
		patternIdx++
	}
	if pathIdx < pathLen && path[pathIdx] == '/' {
		pathIdx++
	}

	for patternIdx < patternLen && pathIdx < pathLen {
		patternStart := patternIdx
		for patternIdx < patternLen && pattern[patternIdx] != '/' {
			patternIdx++
		}
		patternSeg := pattern[patternStart:patternIdx]

		pathStart := pathIdx
		for pathIdx < pathLen && path[pathIdx] != '/' {
			pathIdx++
		}
		pathSeg := path[pathStart:pathIdx]

		if patternSeg != "*" && patternSeg != pathSeg {
			return false
		}

		if patternIdx < patternLen {
			patternIdx++
		}
		if pathIdx < pathLen {
			pathIdx++
		}
	}

	return patternIdx >= patternLen && pathIdx >= pathLen
}

// refreshCache refreshes the rules cache
func (m *BaseAuthorizationMiddleware[TUser, TRole]) refreshCache() {
	rules, err := m.ruleRepo.GetAllRulesForCache()
	if err != nil {
		// Log error để debug - lỗi này có thể khiến cache rỗng
		goerrorkit.LogError(goerrorkit.WrapWithMessage(err, "Lỗi khi load rules từ database để refresh cache").WithData(map[string]interface{}{
			"service_name": m.ruleRepo.GetServiceName(),
		}), "BaseAuthorizationMiddleware.refreshCache")
		return
	}

	// Log warning nếu không có rules nào được load
	if len(rules) == 0 {
		goerrorkit.LogError(goerrorkit.NewBusinessError(404, "Không có rules nào được load vào cache").WithData(map[string]interface{}{
			"service_name": m.ruleRepo.GetServiceName(),
			"hint":         "Kiểm tra xem đã sync routes chưa (ak.SyncRoutes()) và service_name có khớp không",
		}), "BaseAuthorizationMiddleware.refreshCache")
	}

	m.cacheMutex.Lock()
	defer m.cacheMutex.Unlock()

	exactRulesMap := make(map[string][]models.Rule)
	patternRulesByMethodAndSegs := make(map[string]map[int][]models.Rule)

	for _, rule := range rules {
		if strings.Contains(rule.Path, "*") {
			segmentCount := m.countSegments(rule.Path)
			if patternRulesByMethodAndSegs[rule.Method] == nil {
				patternRulesByMethodAndSegs[rule.Method] = make(map[int][]models.Rule)
			}
			patternRulesByMethodAndSegs[rule.Method][segmentCount] = append(
				patternRulesByMethodAndSegs[rule.Method][segmentCount],
				rule,
			)
		} else {
			key := fmt.Sprintf("%s|%s", rule.Method, rule.Path)
			exactRulesMap[key] = append(exactRulesMap[key], rule)
		}
	}

	m.exactRulesMap = exactRulesMap
	m.patternRulesByMethodAndSegs = patternRulesByMethodAndSegs
	m.lastRefresh = time.Now()
}

// InvalidateCache invalidates the cache
func (m *BaseAuthorizationMiddleware[TUser, TRole]) InvalidateCache() {
	m.refreshCache()
}

// getSuperAdminID returns cached super_admin role ID (lazy loads if not cached)
func (m *BaseAuthorizationMiddleware[TUser, TRole]) getSuperAdminID() *uint {
	// Check cache first
	m.roleNameCacheMutex.RLock()
	if m.superAdminID != nil {
		defer m.roleNameCacheMutex.RUnlock()
		return m.superAdminID
	}
	m.roleNameCacheMutex.RUnlock()

	// Not cached, lazy load from DB
	superAdmin, err := m.roleRepo.GetByName("super_admin")
	if err != nil {
		// Role không tồn tại hoặc lỗi DB - trả về nil để skip super_admin check
		return nil
	}

	// Update cache
	m.roleNameCacheMutex.Lock()
	superAdminID := superAdmin.GetID()
	m.superAdminID = &superAdminID
	m.roleNameToIDMap["super_admin"] = superAdminID
	m.roleNameCacheMutex.Unlock()

	return m.superAdminID
}

// getRoleIDByName gets role ID by name with caching
func (m *BaseAuthorizationMiddleware[TUser, TRole]) getRoleIDByName(roleName string) (uint, error) {
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
	roleID := role.GetID()
	m.roleNameCacheMutex.Lock()
	m.roleNameToIDMap[roleName] = roleID
	m.roleNameCacheMutex.Unlock()

	return roleID, nil
}
