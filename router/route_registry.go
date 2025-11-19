package router

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/techmaster-vietnam/authkit/models"
)

// RouteMetadata lưu thông tin route được khai báo trong code
type RouteMetadata struct {
	Method      string
	Path        string // Relative path (để register vào router)
	FullPath    string // Full path bao gồm prefix (để sync vào DB)
	Handler     fiber.Handler
	AccessType  models.AccessType
	Roles       []string
	Fixed       bool
	Description string
}

// RouteRegistry quản lý tất cả routes được đăng ký từ code
type RouteRegistry struct {
	routes      []*RouteMetadata
	exactMap    map[string]*RouteMetadata // O(1) lookup: "METHOD|PATH" -> RouteMetadata
	patternList []*RouteMetadata          // Routes có wildcard patterns
	mutex       sync.RWMutex
}

// NewRouteRegistry tạo mới RouteRegistry
func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{
		routes:      make([]*RouteMetadata, 0),
		exactMap:    make(map[string]*RouteMetadata),
		patternList: make([]*RouteMetadata, 0),
	}
}

// Register đăng ký một route vào registry
func (rr *RouteRegistry) Register(route *RouteMetadata) {
	rr.mutex.Lock()
	defer rr.mutex.Unlock()

	// Thêm vào danh sách routes
	rr.routes = append(rr.routes, route)

	// Kiểm tra xem có wildcard pattern không (sử dụng FullPath)
	if strings.Contains(route.FullPath, "*") {
		rr.patternList = append(rr.patternList, route)
	} else {
		// Thêm vào exact map cho O(1) lookup (sử dụng FullPath)
		key := fmt.Sprintf("%s|%s", route.Method, route.FullPath)
		rr.exactMap[key] = route
	}
}

// GetAllRoutes trả về tất cả routes đã đăng ký
func (rr *RouteRegistry) GetAllRoutes() []*RouteMetadata {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	routes := make([]*RouteMetadata, len(rr.routes))
	copy(routes, rr.routes)
	return routes
}

// FindRoute tìm route theo method và path
func (rr *RouteRegistry) FindRoute(method, path string) *RouteMetadata {
	rr.mutex.RLock()
	defer rr.mutex.RUnlock()

	// Thử exact match trước (O(1)) - sử dụng FullPath
	key := fmt.Sprintf("%s|%s", method, path)
	if route, found := rr.exactMap[key]; found {
		return route
	}

	// Nếu không tìm thấy, thử pattern matching - sử dụng FullPath
	for _, route := range rr.patternList {
		if route.Method == method && matchPath(route.FullPath, path) {
			return route
		}
	}

	return nil
}

// matchPath kiểm tra path có match với pattern không (hỗ trợ wildcard *)
func matchPath(pattern, path string) bool {
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

