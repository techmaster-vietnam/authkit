package middleware

import (
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/models"
)

// Test countSegments - pure function, không cần DB
func TestBaseAuthorizationMiddleware_CountSegments(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{}

	tests := []struct {
		name     string
		path     string
		expected int
	}{
		{"empty path", "", 0},
		{"root path", "/", 0},
		{"single segment", "/api", 1},
		{"two segments", "/api/users", 2},
		{"three segments", "/api/users/123", 3},
		{"path without leading slash", "api/users", 2},
		{"path with trailing slash", "/api/users/", 3},
		{"nested path", "/api/v1/users/123/posts", 5},
		{"deeply nested", "/api/v1/users/123/posts/456/comments", 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mw.countSegments(tt.path)
			if result != tt.expected {
				t.Errorf("countSegments(%q) = %d, expected %d", tt.path, result, tt.expected)
			}
		})
	}
}

// Test matchPath - pure function, không cần DB
func TestBaseAuthorizationMiddleware_MatchPath(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{}

	tests := []struct {
		name     string
		pattern  string
		path     string
		expected bool
	}{
		{"exact match", "/api/users", "/api/users", true},
		{"exact match root", "/", "/", true},
		{"wildcard match single segment", "/api/*", "/api/users", true},
		{"wildcard match multiple segments", "/api/*/posts", "/api/users/posts", true},
		{"wildcard match with ID", "/api/users/*", "/api/users/123", true},
		{"no match different path", "/api/users", "/api/posts", false},
		{"no match different segments", "/api/users", "/api/users/123", false},
		{"wildcard in middle", "/api/*/posts", "/api/users/posts", true},
		{"wildcard at end", "/api/users/*", "/api/users/123", true},
		{"multiple wildcards", "/api/*/posts/*", "/api/users/posts/123", true},
		{"no match - too many segments", "/api/users", "/api/users/123/posts", false},
		{"no match - too few segments", "/api/users/123", "/api/users", false},
		{"path without leading slash", "api/users", "api/users", true},
		{"pattern without leading slash", "api/*", "api/users", true},
		{"complex wildcard pattern", "/api/*/users/*/posts", "/api/v1/users/123/posts", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mw.matchPath(tt.pattern, tt.path)
			if result != tt.expected {
				t.Errorf("matchPath(%q, %q) = %v, expected %v", tt.pattern, tt.path, result, tt.expected)
			}
		})
	}
}

// Test findMatchingRules với exact matches
func TestBaseAuthorizationMiddleware_FindMatchingRules_ExactMatch(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
	}

	// Setup cache với exact rules
	mw.cacheMutex.Lock()
	mw.exactRulesMap["GET|/api/users"] = []models.Rule{
		{
			ID:     "GET|/api/users",
			Method: "GET",
			Path:   "/api/users",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1, 2},
		},
	}
	mw.cacheMutex.Unlock()

	rules := mw.findMatchingRules("GET", "/api/users")
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "GET|/api/users" {
		t.Errorf("Expected rule ID 'GET|/api/users', got %q", rules[0].ID)
	}
}

// Test findMatchingRules với pattern matches
func TestBaseAuthorizationMiddleware_FindMatchingRules_PatternMatch(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
	}

	// Setup cache với pattern rules
	mw.cacheMutex.Lock()
	if mw.patternRulesByMethodAndSegs["GET"] == nil {
		mw.patternRulesByMethodAndSegs["GET"] = make(map[int][]models.Rule)
	}
	mw.patternRulesByMethodAndSegs["GET"][3] = []models.Rule{
		{
			ID:     "GET|/api/users/*",
			Method: "GET",
			Path:   "/api/users/*",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1},
		},
	}
	mw.cacheMutex.Unlock()

	rules := mw.findMatchingRules("GET", "/api/users/123")
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "GET|/api/users/*" {
		t.Errorf("Expected rule ID 'GET|/api/users/*', got %q", rules[0].ID)
	}
}

// Test findMatchingRules - no match
func TestBaseAuthorizationMiddleware_FindMatchingRules_NoMatch(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
	}

	rules := mw.findMatchingRules("GET", "/api/unknown")
	if len(rules) > 0 {
		t.Errorf("Expected no rules, got %d rules", len(rules))
	}
}

// Test findMatchingRules - exact match takes priority over pattern
func TestBaseAuthorizationMiddleware_FindMatchingRules_ExactTakesPriority(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
	}

	mw.cacheMutex.Lock()
	// Setup exact rule
	mw.exactRulesMap["GET|/api/users/123"] = []models.Rule{
		{
			ID:     "GET|/api/users/123",
			Method: "GET",
			Path:   "/api/users/123",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1},
		},
	}
	// Setup pattern rule that could also match
	if mw.patternRulesByMethodAndSegs["GET"] == nil {
		mw.patternRulesByMethodAndSegs["GET"] = make(map[int][]models.Rule)
	}
	mw.patternRulesByMethodAndSegs["GET"][3] = []models.Rule{
		{
			ID:     "GET|/api/users/*",
			Method: "GET",
			Path:   "/api/users/*",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{2},
		},
	}
	mw.cacheMutex.Unlock()

	rules := mw.findMatchingRules("GET", "/api/users/123")
	// Should return exact match only
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "GET|/api/users/123" {
		t.Errorf("Expected exact match 'GET|/api/users/123', got %q", rules[0].ID)
	}
	if len(rules[0].Roles) != 1 || rules[0].Roles[0] != 1 {
		t.Errorf("Expected role ID 1, got %v", rules[0].Roles)
	}
}

// Helper middleware để setup user context trong tests với generic types
func setupUserContextMiddlewareGeneric[TUser interface{ GetID() string }](userID string, roleIDs []uint, user TUser) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("user", user)
		c.Locals("userID", userID)
		c.Locals("roleIDs", roleIDs)
		return c.Next()
	}
}

// Test Authorize - no rule found (default FORBID)
func TestBaseAuthorizationMiddleware_Authorize_NoRuleFound(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	app := fiber.New()
	app.Get("/api/unknown", mw.Authorize(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/unknown", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode == 200 {
		t.Errorf("Expected error status (403 or 500), got 200 OK")
	}
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	if !strings.Contains(bodyStr, "quyền") && !strings.Contains(bodyStr, "403") {
		t.Errorf("Expected error message about permission, got: %s", bodyStr)
	}
}

// Test Authorize - anonymous user với non-PUBLIC rule
func TestBaseAuthorizationMiddleware_Authorize_AnonymousUser(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	// Setup cache với ALLOW rule (requires auth)
	mw.cacheMutex.Lock()
	mw.exactRulesMap["GET|/api/protected"] = []models.Rule{
		{
			ID:     "GET|/api/protected",
			Method: "GET",
			Path:   "/api/protected",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1},
		},
	}
	mw.cacheMutex.Unlock()

	app := fiber.New()
	app.Get("/api/protected", mw.Authorize(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode == 200 {
		t.Errorf("Expected error status (401 or 500), got 200 OK")
	}
	body := make([]byte, 1024)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	if !strings.Contains(bodyStr, "đăng nhập") && !strings.Contains(bodyStr, "401") {
		t.Errorf("Expected error message about login required, got: %s", bodyStr)
	}
}

// Test Authorize với PUBLIC rule
func TestBaseAuthorizationMiddleware_Authorize_PublicRule(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	// Setup cache với PUBLIC rule
	mw.cacheMutex.Lock()
	mw.exactRulesMap["GET|/api/public"] = []models.Rule{
		{
			ID:     "GET|/api/public",
			Method: "GET",
			Path:   "/api/public",
			Type:   models.AccessPublic,
			Roles:  models.IntArray{},
		},
	}
	mw.cacheMutex.Unlock()

	app := fiber.New()
	app.Get("/api/public", mw.Authorize(), func(c *fiber.Ctx) error {
		return c.SendString("OK")
	})

	req := httptest.NewRequest("GET", "/api/public", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

// Test Authorize với user có role được ALLOW
func TestBaseAuthorizationMiddleware_Authorize_UserWithAllowedRole(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	// Define role IDs
	adminRoleID := uint(1)
	editorRoleID := uint(2)
	userRoleID := uint(3)

	// Setup cache với ALLOW rule chỉ cho admin và editor
	mw.cacheMutex.Lock()
	mw.exactRulesMap["GET|/api/posts"] = []models.Rule{
		{
			ID:     "GET|/api/posts",
			Method: "GET",
			Path:   "/api/posts",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{adminRoleID, editorRoleID},
		},
	}
	mw.cacheMutex.Unlock()

	// Test 1: User có admin role - được phép truy cập
	t.Run("admin_role_allowed", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "user1", Email: "test@example.com", Active: true}
		app.Get("/api/posts", setupUserContextMiddlewareGeneric("user1", []uint{adminRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/api/posts", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("Admin user should be allowed (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})

	// Test 2: User có editor role - được phép truy cập
	t.Run("editor_role_allowed", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "user2", Email: "test@example.com", Active: true}
		app.Get("/api/posts", setupUserContextMiddlewareGeneric("user2", []uint{editorRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/api/posts", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("Editor user should be allowed (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})

	// Test 3: User có user role (không có admin/editor) - bị từ chối
	t.Run("user_role_denied", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "user3", Email: "test@example.com", Active: true}
		app.Get("/api/posts", setupUserContextMiddlewareGeneric("user3", []uint{userRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/api/posts", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode == 200 {
			t.Error("Regular user should be denied access (not 200)")
		}
	})
}

// Test Authorize với user có role bị FORBID
func TestBaseAuthorizationMiddleware_Authorize_UserWithForbiddenRole(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	// Define role IDs
	adminRoleID := uint(1)
	bannedRoleID := uint(99)

	// Setup cache với FORBID rule cho banned role
	mw.cacheMutex.Lock()
	if mw.patternRulesByMethodAndSegs["DELETE"] == nil {
		mw.patternRulesByMethodAndSegs["DELETE"] = make(map[int][]models.Rule)
	}
	mw.patternRulesByMethodAndSegs["DELETE"][3] = []models.Rule{
		{
			ID:     "DELETE|/api/posts/*",
			Method: "DELETE",
			Path:   "/api/posts/*",
			Type:   models.AccessForbid,
			Roles:  models.IntArray{bannedRoleID},
		},
		{
			ID:     "DELETE|/api/posts/*",
			Method: "DELETE",
			Path:   "/api/posts/*",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{adminRoleID},
		},
	}
	mw.cacheMutex.Unlock()

	// Test 1: User có banned role - bị từ chối (FORBID có priority cao hơn)
	t.Run("banned_role_forbidden", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "banned_user", Email: "test@example.com", Active: true}
		app.Delete("/api/posts/*", setupUserContextMiddlewareGeneric("banned_user", []uint{bannedRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("DELETE", "/api/posts/123", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode == 200 {
			t.Error("Banned user should be forbidden (not 200)")
		}
	})

	// Test 2: User có admin role - được phép (FORBID không áp dụng cho admin)
	t.Run("admin_role_allowed", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "admin_user", Email: "test@example.com", Active: true}
		app.Delete("/api/posts/*", setupUserContextMiddlewareGeneric("admin_user", []uint{adminRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("DELETE", "/api/posts/123", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("Admin user should be allowed (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})
}

// Test Authorize với ALLOW rule empty roles (any authenticated user)
func TestBaseAuthorizationMiddleware_Authorize_AllowAnyAuthenticatedUser(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	// Setup cache với ALLOW rule empty roles (any authenticated user)
	mw.cacheMutex.Lock()
	mw.exactRulesMap["GET|/api/profile"] = []models.Rule{
		{
			ID:     "GET|/api/profile",
			Method: "GET",
			Path:   "/api/profile",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{},
		},
	}
	mw.cacheMutex.Unlock()

	// Test: Bất kỳ authenticated user nào cũng được phép
	t.Run("any_authenticated_user_allowed", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "any_user", Email: "test@example.com", Active: true}
		app.Get("/api/profile", setupUserContextMiddlewareGeneric("any_user", []uint{999}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("GET", "/api/profile", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("Any authenticated user should be allowed (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})
}

// Test Authorize với FORBID rule empty roles (forbid everyone)
func TestBaseAuthorizationMiddleware_Authorize_ForbidEveryone(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	// Setup cache với FORBID rule empty roles (forbid everyone)
	mw.cacheMutex.Lock()
	mw.exactRulesMap["POST|/api/admin/delete-all"] = []models.Rule{
		{
			ID:     "POST|/api/admin/delete-all",
			Method: "POST",
			Path:   "/api/admin/delete-all",
			Type:   models.AccessForbid,
			Roles:  models.IntArray{},
		},
	}
	mw.cacheMutex.Unlock()

	// Test: Ngay cả admin cũng bị từ chối
	t.Run("everyone_forbidden", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "admin_user", Email: "test@example.com", Active: true}
		app.Post("/api/admin/delete-all", setupUserContextMiddlewareGeneric("admin_user", []uint{1}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("POST", "/api/admin/delete-all", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode == 200 {
			t.Error("Everyone should be forbidden, even admin (not 200)")
		}
	})
}

// Test Authorize với super admin bypass
func TestBaseAuthorizationMiddleware_Authorize_SuperAdminBypass(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	superAdminRoleID := uint(1)
	regularRoleID := uint(2)

	// Setup super admin ID trong cache
	mw.roleNameCacheMutex.Lock()
	mw.superAdminID = &superAdminRoleID
	mw.roleNameCacheMutex.Unlock()

	// Setup cache với FORBID rule cho regular role
	mw.cacheMutex.Lock()
	mw.exactRulesMap["DELETE|/api/system"] = []models.Rule{
		{
			ID:     "DELETE|/api/system",
			Method: "DELETE",
			Path:   "/api/system",
			Type:   models.AccessForbid,
			Roles:  models.IntArray{regularRoleID},
		},
		{
			ID:     "DELETE|/api/system",
			Method: "DELETE",
			Path:   "/api/system",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{},
		},
	}
	mw.cacheMutex.Unlock()

	// Test 1: Super admin bypass tất cả rules
	t.Run("super_admin_bypass", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "super_admin", Email: "test@example.com", Active: true}
		app.Delete("/api/system", setupUserContextMiddlewareGeneric("super_admin", []uint{superAdminRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("DELETE", "/api/system", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("Super admin should bypass all rules (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})

	// Test 2: Regular user bị từ chối
	t.Run("regular_user_denied", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "regular_user", Email: "test@example.com", Active: true}
		app.Delete("/api/system", setupUserContextMiddlewareGeneric("regular_user", []uint{regularRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("DELETE", "/api/system", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode == 200 {
			t.Error("Regular user should be denied (not 200)")
		}
	})
}

// Test Authorize với X-Role-Context header
func TestBaseAuthorizationMiddleware_Authorize_RoleContextHeader(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	adminRoleID := uint(1)
	editorRoleID := uint(2)
	viewerRoleID := uint(3)

	// Setup role name cache
	mw.roleNameCacheMutex.Lock()
	mw.roleNameToIDMap["admin"] = adminRoleID
	mw.roleNameToIDMap["editor"] = editorRoleID
	mw.roleNameToIDMap["viewer"] = viewerRoleID
	mw.roleNameCacheMutex.Unlock()

	// Setup cache với rule chỉ cho editor (pattern match)
	mw.cacheMutex.Lock()
	if mw.patternRulesByMethodAndSegs["PUT"] == nil {
		mw.patternRulesByMethodAndSegs["PUT"] = make(map[int][]models.Rule)
	}
	mw.patternRulesByMethodAndSegs["PUT"][3] = []models.Rule{
		{
			ID:     "PUT|/api/posts/*",
			Method: "PUT",
			Path:   "/api/posts/*",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{editorRoleID},
		},
	}
	mw.cacheMutex.Unlock()

	// Test 1: User có cả admin và editor role, dùng X-Role-Context=editor - được phép
	t.Run("role_context_valid", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "user_with_multiple_roles", Email: "test@example.com", Active: true}
		app.Put("/api/posts/*", setupUserContextMiddlewareGeneric("user_with_multiple_roles", []uint{adminRoleID, editorRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("PUT", "/api/posts/123", nil)
		req.Header.Set("X-Role-Context", "editor")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("User with editor role context should be allowed (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})

	// Test 2: User có cả admin và editor role, dùng X-Role-Context=admin - bị từ chối (admin không được phép)
	t.Run("role_context_wrong_role", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "user_with_multiple_roles", Email: "test@example.com", Active: true}
		app.Put("/api/posts/*", setupUserContextMiddlewareGeneric("user_with_multiple_roles", []uint{adminRoleID, editorRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("PUT", "/api/posts/123", nil)
		req.Header.Set("X-Role-Context", "admin")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode == 200 {
			t.Error("User with admin role context should be denied (only editor allowed)")
		}
	})

	// Test 3: User không có role được yêu cầu trong X-Role-Context - bị từ chối
	t.Run("role_context_user_doesnt_have_role", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "user_without_editor", Email: "test@example.com", Active: true}
		app.Put("/api/posts/*", setupUserContextMiddlewareGeneric("user_without_editor", []uint{viewerRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("PUT", "/api/posts/123", nil)
		req.Header.Set("X-Role-Context", "editor")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode == 200 {
			t.Error("User without editor role should be denied")
		}
	})
}

// Test Authorize với multiple rules cùng endpoint (FORBID và ALLOW)
func TestBaseAuthorizationMiddleware_Authorize_MultipleRulesSameEndpoint(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		roleNameToIDMap:             make(map[string]uint),
		roleNameCacheMutex:          sync.RWMutex{},
	}

	adminRoleID := uint(1)
	bannedRoleID := uint(99)
	regularRoleID := uint(2)

	// Setup cache với cả FORBID và ALLOW rules
	mw.cacheMutex.Lock()
	mw.exactRulesMap["POST|/api/comments"] = []models.Rule{
		{
			ID:     "POST|/api/comments",
			Method: "POST",
			Path:   "/api/comments",
			Type:   models.AccessForbid,
			Roles:  models.IntArray{bannedRoleID},
		},
		{
			ID:     "POST|/api/comments",
			Method: "POST",
			Path:   "/api/comments",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{adminRoleID, regularRoleID},
		},
	}
	mw.cacheMutex.Unlock()

	// Test 1: Banned user - bị từ chối (FORBID có priority)
	t.Run("banned_user_forbidden", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "banned_user", Email: "test@example.com", Active: true}
		app.Post("/api/comments", setupUserContextMiddlewareGeneric("banned_user", []uint{bannedRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("POST", "/api/comments", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode == 200 {
			t.Error("Banned user should be forbidden (FORBID has priority)")
		}
	})

	// Test 2: Admin user - được phép
	t.Run("admin_user_allowed", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "admin_user", Email: "test@example.com", Active: true}
		app.Post("/api/comments", setupUserContextMiddlewareGeneric("admin_user", []uint{adminRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("POST", "/api/comments", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("Admin user should be allowed (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})

	// Test 3: Regular user - được phép
	t.Run("regular_user_allowed", func(t *testing.T) {
		app := fiber.New()
		user := &models.BaseUser{ID: "regular_user", Email: "test@example.com", Active: true}
		app.Post("/api/comments", setupUserContextMiddlewareGeneric("regular_user", []uint{regularRoleID}, user), mw.Authorize(), func(c *fiber.Ctx) error {
			return c.SendString("OK")
		})

		req := httptest.NewRequest("POST", "/api/comments", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			body := make([]byte, 1024)
			n, _ := resp.Body.Read(body)
			t.Errorf("Regular user should be allowed (200), got %d. Response: %s", resp.StatusCode, string(body[:n]))
		}
	})
}

// Test findMatchingRules với multiple pattern rules cùng segment count
func TestBaseAuthorizationMiddleware_FindMatchingRules_MultiplePatterns(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
	}

	mw.cacheMutex.Lock()
	if mw.patternRulesByMethodAndSegs["GET"] == nil {
		mw.patternRulesByMethodAndSegs["GET"] = make(map[int][]models.Rule)
	}
	mw.patternRulesByMethodAndSegs["GET"][3] = []models.Rule{
		{
			ID:     "GET|/api/users/*",
			Method: "GET",
			Path:   "/api/users/*",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1},
		},
		{
			ID:     "GET|/api/posts/*",
			Method: "GET",
			Path:   "/api/posts/*",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{2},
		},
	}
	mw.cacheMutex.Unlock()

	// Test match với /api/users/123
	rules := mw.findMatchingRules("GET", "/api/users/123")
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule for /api/users/123, got %d", len(rules))
	}
	if rules[0].ID != "GET|/api/users/*" {
		t.Errorf("Expected 'GET|/api/users/*', got %q", rules[0].ID)
	}

	// Test match với /api/posts/456
	rules = mw.findMatchingRules("GET", "/api/posts/456")
	if len(rules) != 1 {
		t.Fatalf("Expected 1 rule for /api/posts/456, got %d", len(rules))
	}
	if rules[0].ID != "GET|/api/posts/*" {
		t.Errorf("Expected 'GET|/api/posts/*', got %q", rules[0].ID)
	}
}

// Test findMatchingRules với different methods
func TestBaseAuthorizationMiddleware_FindMatchingRules_DifferentMethods(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
	}

	mw.cacheMutex.Lock()
	mw.exactRulesMap["GET|/api/users"] = []models.Rule{
		{ID: "GET|/api/users", Method: "GET", Path: "/api/users", Type: models.AccessAllow},
	}
	mw.exactRulesMap["POST|/api/users"] = []models.Rule{
		{ID: "POST|/api/users", Method: "POST", Path: "/api/users", Type: models.AccessAllow},
	}
	mw.cacheMutex.Unlock()

	// Test GET
	rules := mw.findMatchingRules("GET", "/api/users")
	if len(rules) != 1 || rules[0].Method != "GET" {
		t.Errorf("GET request should match GET rule")
	}

	// Test POST
	rules = mw.findMatchingRules("POST", "/api/users")
	if len(rules) != 1 || rules[0].Method != "POST" {
		t.Errorf("POST request should match POST rule")
	}

	// Test GET không match POST rule
	rules = mw.findMatchingRules("GET", "/api/users")
	if len(rules) > 0 && rules[0].Method == "POST" {
		t.Errorf("GET request should not match POST rule")
	}
}

// Test refreshCache logic (simulated, không cần DB)
func TestBaseAuthorizationMiddleware_RefreshCache_Logic(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		exactRulesMap:               make(map[string][]models.Rule),
		patternRulesByMethodAndSegs: make(map[string]map[int][]models.Rule),
		cacheMutex:                  sync.RWMutex{},
		cacheTTL:                    5 * time.Minute,
	}

	// Mock rules data
	mockRules := []models.Rule{
		{
			ID:     "GET|/api/users",
			Method: "GET",
			Path:   "/api/users",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1, 2},
		},
		{
			ID:     "POST|/api/users",
			Method: "POST",
			Path:   "/api/users",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1},
		},
		{
			ID:     "GET|/api/users/*",
			Method: "GET",
			Path:   "/api/users/*",
			Type:   models.AccessAllow,
			Roles:  models.IntArray{1},
		},
		{
			ID:     "DELETE|/api/posts/*",
			Method: "DELETE",
			Path:   "/api/posts/*",
			Type:   models.AccessForbid,
			Roles:  models.IntArray{3},
		},
	}

	mw.cacheMutex.Lock()
	// Simulate refreshCache logic
	exactRulesMap := make(map[string][]models.Rule)
	patternRulesByMethodAndSegs := make(map[string]map[int][]models.Rule)

	for _, rule := range mockRules {
		if strings.Contains(rule.Path, "*") {
			segmentCount := mw.countSegments(rule.Path)
			if patternRulesByMethodAndSegs[rule.Method] == nil {
				patternRulesByMethodAndSegs[rule.Method] = make(map[int][]models.Rule)
			}
			patternRulesByMethodAndSegs[rule.Method][segmentCount] = append(
				patternRulesByMethodAndSegs[rule.Method][segmentCount],
				rule,
			)
		} else {
			key := rule.Method + "|" + rule.Path
			exactRulesMap[key] = append(exactRulesMap[key], rule)
		}
	}

	mw.exactRulesMap = exactRulesMap
	mw.patternRulesByMethodAndSegs = patternRulesByMethodAndSegs
	mw.lastRefresh = time.Now()
	mw.cacheMutex.Unlock()

	// Verify cache được build đúng
	mw.cacheMutex.RLock()
	if len(mw.exactRulesMap) != 2 {
		t.Errorf("Expected 2 exact rules, got %d", len(mw.exactRulesMap))
	}
	if mw.patternRulesByMethodAndSegs["GET"] == nil {
		t.Error("Expected GET pattern rules")
	}
	if len(mw.patternRulesByMethodAndSegs["GET"][3]) != 1 {
		t.Errorf("Expected 1 GET pattern rule with 3 segments, got %d", len(mw.patternRulesByMethodAndSegs["GET"][3]))
	}
	if len(mw.patternRulesByMethodAndSegs["DELETE"][3]) != 1 {
		t.Errorf("Expected 1 DELETE pattern rule with 3 segments, got %d", len(mw.patternRulesByMethodAndSegs["DELETE"][3]))
	}
	mw.cacheMutex.RUnlock()
}

// Test getRoleIDByName với cache
func TestBaseAuthorizationMiddleware_GetRoleIDByName_WithCache(t *testing.T) {
	mw := &BaseAuthorizationMiddleware[*models.BaseUser, *models.BaseRole]{
		roleNameToIDMap:    make(map[string]uint),
		roleNameCacheMutex: sync.RWMutex{},
	}

	// Setup cache
	mw.roleNameCacheMutex.Lock()
	mw.roleNameToIDMap["admin"] = 1
	mw.roleNameToIDMap["editor"] = 2
	mw.roleNameCacheMutex.Unlock()

	// Test cache hit
	roleID, err := mw.getRoleIDByName("admin")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if roleID != 1 {
		t.Errorf("Expected role ID 1, got %d", roleID)
	}

	// Test cache hit again
	roleID, err = mw.getRoleIDByName("editor")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if roleID != 2 {
		t.Errorf("Expected role ID 2, got %d", roleID)
	}
}
