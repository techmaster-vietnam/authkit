package core

// CacheInvalidator định nghĩa interface để invalidate cache
// Cho phép service layer invalidate cache mà không cần biết chi tiết implementation
type CacheInvalidator interface {
	// InvalidateRulesCache invalidates rules cache trong authorization middleware
	InvalidateRulesCache()
}

