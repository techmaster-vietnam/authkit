package main

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit"
	"github.com/techmaster-vietnam/authkit/utils"
	"gorm.io/gorm"
)

// DemoHandler demonstrates new features: username in JWT, custom fields, role conversion
type DemoHandler struct {
	ak *authkit.AuthKit[*CustomUser, *authkit.BaseRole]
}

// NewDemoHandler creates a new demo handler
func NewDemoHandler(ak *authkit.AuthKit[*CustomUser, *authkit.BaseRole]) *DemoHandler {
	return &DemoHandler{
		ak: ak,
	}
}

// LoginWithUsername demonstrates login with username in JWT token
// POST /api/demo/login-with-username
func (h *DemoHandler) LoginWithUsername(c *fiber.Ctx) error {
	// Debug: Log method and path để kiểm tra tại sao route này được gọi
	fmt.Printf("[DEBUG] LoginWithUsername được gọi với method=%s, path=%s, route=%s\n",
		c.Method(), c.Path(), c.Route().Path)

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
			"debug": fiber.Map{
				"method": c.Method(),
				"path":   c.Path(),
				"route":  c.Route().Path,
			},
		})
	}

	// Get user
	user, err := h.ak.UserRepo.GetByEmail(req.Email)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.Status(401).JSON(fiber.Map{
				"error": "Email hoặc mật khẩu không đúng",
			})
		}
		return c.Status(500).JSON(fiber.Map{
			"error": "Lỗi khi đăng nhập",
		})
	}

	// Check password
	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return c.Status(401).JSON(fiber.Map{
			"error": "Email hoặc mật khẩu không đúng",
		})
	}

	// Get user roles
	userRoles := user.GetRoles()
	roleIDs := utils.ExtractRoleIDsFromRoleInterfaces(userRoles)
	roleNames := utils.ExtractRoleNamesFromRoleInterfaces(userRoles)

	// Get custom user fields if available
	var mobile, address string
	// user is already *CustomUser from GetByEmail
	mobile = user.Mobile
	address = user.Address

	// Generate token with username and role names using flexible API
	config := utils.ClaimsConfig{
		Username:   user.GetFullName(), // Use FullName as username
		RoleFormat: "both",             // Include both IDs and names
		RoleIDs:    roleIDs,
		RoleNames:  roleNames,
		CustomFields: map[string]interface{}{
			"mobile":  mobile,
			"address": address,
		},
	}

	token, err := utils.GenerateTokenFlexible(
		user.GetID(),
		user.GetEmail(),
		config,
		h.ak.Config.JWT.Secret,
		h.ak.Config.JWT.Expiration,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Lỗi khi tạo token",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"token":   token,
		"user":    user,
		"message": "Token được tạo với username và custom fields (mobile, address)",
	})
}

// GetTokenInfo demonstrates extracting information from flexible token
// GET /api/demo/token-info
func (h *DemoHandler) GetTokenInfo(c *fiber.Ctx) error {
	// Get token from Authorization header
	token := c.Get("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	// Validate flexible token
	claims, err := utils.ValidateTokenFlexible(token, h.ak.Config.JWT.Secret)
	if err != nil {
		return c.Status(401).JSON(fiber.Map{
			"error": "Invalid token",
		})
	}

	// Extract information from claims
	result := fiber.Map{
		"user_id": claims["user_id"],
		"email":   claims["email"],
	}

	// Extract username if present
	if username, ok := claims["username"].(string); ok {
		result["username"] = username
	}

	// Extract role IDs if present
	if roleIDs, ok := claims["role_ids"].([]interface{}); ok {
		ids := make([]uint, len(roleIDs))
		for i, id := range roleIDs {
			ids[i] = uint(id.(float64))
		}
		result["role_ids"] = ids

		// Convert role IDs to names using repository
		roles, err := h.ak.RoleRepo.GetByIDs(ids)
		if err == nil {
			// Convert []*authkit.BaseRole to []authkit.Role
			roleModels := make([]authkit.Role, len(roles))
			for i, r := range roles {
				roleModels[i] = *r
			}
			roleNames := utils.ExtractRoleNamesFromRoles(roleModels)
			result["role_names_from_ids"] = roleNames
		}
	}

	// Extract role names if present
	if roleNames, ok := claims["roles"].([]interface{}); ok {
		names := make([]string, len(roleNames))
		for i, name := range roleNames {
			names[i] = name.(string)
		}
		result["role_names"] = names

		// Convert role names to IDs using repository
		nameToIDMap, err := h.ak.RoleRepo.GetIDsByNames(names)
		if err == nil {
			ids := utils.ConvertRoleNameMapToIDs(nameToIDMap, names)
			result["role_ids_from_names"] = ids
		}
	}

	// Extract custom fields
	customFields := make(fiber.Map)
	for k, v := range claims {
		if k != "user_id" && k != "email" && k != "username" && k != "role_ids" && k != "roles" &&
			k != "exp" && k != "iat" && k != "nbf" && k != "iss" {
			customFields[k] = v
		}
	}
	if len(customFields) > 0 {
		result["custom_fields"] = customFields
	}

	return c.JSON(fiber.Map{
		"success": true,
		"claims":  result,
		"message": "Thông tin được extract từ flexible token",
	})
}

// GetRouteRegistry returns all registered routes from RouteRegistry
// GET /api/routeregistry
func (h *DemoHandler) GetRouteRegistry(c *fiber.Ctx) error {
	routes := h.ak.RouteRegistry.GetAllRoutes()

	// Convert RouteMetadata sang response DTO (bỏ Handler field)
	routeDTOs := make([]fiber.Map, len(routes))
	for i, route := range routes {
		// Convert AccessType sang string
		accessType := string(route.AccessType)
		// AccessType có thể là "PUBLIC", "ALLOW", "FORBID" hoặc empty
		// Nếu empty, mặc định là "Public" để tương thích với frontend
		if accessType == "" {
			accessType = "Public"
		} else {
			// Convert "PUBLIC" -> "Public", "ALLOW" -> "Allow", "FORBID" -> "Forbid"
			switch accessType {
			case "PUBLIC":
				accessType = "Public"
			case "ALLOW":
				accessType = "Allow"
			case "FORBID":
				accessType = "Forbid"
			}
		}

		routeDTOs[i] = fiber.Map{
			"method":      route.Method,
			"path":        route.FullPath, // Sử dụng FullPath thay vì Path
			"type":        accessType,
			"roles":       route.Roles, // Đã là []string (role names)
			"fixed":       route.Fixed,
			"description": route.Description,
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    routeDTOs,
		"count":   len(routeDTOs),
	})
}

// Foo returns a simple "ok" JSON response
// GET /api/foo
func (h *DemoHandler) Foo(c *fiber.Ctx) error {
	return c.JSON("ok")
}
