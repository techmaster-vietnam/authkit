package router

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/middleware"
)

// AuthRouter wrapper cho fiber.Router với fluent API để cấu hình routes và phân quyền
type AuthRouter struct {
	router      fiber.Router
	registry    *RouteRegistry
	authMw      *middleware.AuthMiddleware
	authzMw     *middleware.AuthorizationMiddleware
	prefix      string // Prefix path của group (để build full path)
}

// NewAuthRouter tạo mới AuthRouter
func NewAuthRouter(
	router fiber.Router,
	registry *RouteRegistry,
	authMw *middleware.AuthMiddleware,
	authzMw *middleware.AuthorizationMiddleware,
) *AuthRouter {
	return &AuthRouter{
		router:   router,
		registry: registry,
		authMw:   authMw,
		authzMw:  authzMw,
		prefix:   "", // Root router không có prefix
	}
}

// Get tạo GET route với fluent API
func (ar *AuthRouter) Get(path string, handler fiber.Handler) *RouteBuilder {
	return ar.createRouteBuilder("GET", path, handler)
}

// Post tạo POST route với fluent API
func (ar *AuthRouter) Post(path string, handler fiber.Handler) *RouteBuilder {
	return ar.createRouteBuilder("POST", path, handler)
}

// Put tạo PUT route với fluent API
func (ar *AuthRouter) Put(path string, handler fiber.Handler) *RouteBuilder {
	return ar.createRouteBuilder("PUT", path, handler)
}

// Delete tạo DELETE route với fluent API
func (ar *AuthRouter) Delete(path string, handler fiber.Handler) *RouteBuilder {
	return ar.createRouteBuilder("DELETE", path, handler)
}

// Patch tạo PATCH route với fluent API
func (ar *AuthRouter) Patch(path string, handler fiber.Handler) *RouteBuilder {
	return ar.createRouteBuilder("PATCH", path, handler)
}

// Group tạo router group với middleware tùy chọn
func (ar *AuthRouter) Group(prefix string, handlers ...fiber.Handler) *AuthRouter {
	group := ar.router.Group(prefix, handlers...)
	newRouter := NewAuthRouter(group, ar.registry, ar.authMw, ar.authzMw)
	// Build full prefix path
	newRouter.prefix = strings.TrimSuffix(ar.prefix, "/") + "/" + strings.TrimPrefix(prefix, "/")
	newRouter.prefix = strings.TrimPrefix(newRouter.prefix, "/")
	if newRouter.prefix != "" {
		newRouter.prefix = "/" + newRouter.prefix
	}
	return newRouter
}

// convertPathToPattern converts path parameters to wildcard pattern
// Ví dụ: /blogs/:id -> /blogs/*, /users/:user_id/roles/:role_id -> /users/*/roles/*
func convertPathToPattern(path string) string {
	// Split path by "/"
	parts := strings.Split(path, "/")
	
	// Convert each part: if starts with ":", replace with "*"
	for i, part := range parts {
		if strings.HasPrefix(part, ":") {
			parts[i] = "*"
		}
	}
	
	// Join back
	return strings.Join(parts, "/")
}

// createRouteBuilder tạo RouteBuilder cho route
func (ar *AuthRouter) createRouteBuilder(method, path string, handler fiber.Handler) *RouteBuilder {
	// Build full path từ prefix và path
	fullPath := strings.TrimSuffix(ar.prefix, "/") + "/" + strings.TrimPrefix(path, "/")
	fullPath = strings.TrimPrefix(fullPath, "/")
	if fullPath != "" {
		fullPath = "/" + fullPath
		// Normalize: remove trailing slash (except for root)
		fullPath = strings.TrimSuffix(fullPath, "/")
		if fullPath == "" {
			fullPath = "/"
		}
	} else {
		fullPath = "/"
	}
	
	// Convert path parameters to pattern for rule matching
	// Ví dụ: /api/blogs/:id -> /api/blogs/*
	fullPathPattern := convertPathToPattern(fullPath)
	
	return &RouteBuilder{
		metadata: &RouteMetadata{
			Method:      method,
			Path:        path,            // Relative path (để register vào router)
			FullPath:    fullPathPattern, // Full path pattern (để sync vào DB và match)
			Handler:     handler,
			AccessType:  "", // Sẽ được set bởi Public/Allow/Forbid
			Roles:       []string{},
			Fixed:       false,
			Description: "",
		},
		router:   ar.router,
		registry: ar.registry,
		authMw:   ar.authMw,
		authzMw:  ar.authzMw,
	}
}

