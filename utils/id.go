package utils

import (
	"github.com/techmaster-vietnam/authkit/core"
)

// GenerateID tạo một ID ngẫu nhiên 12 ký tự từ a-zA-Z0-9
// Re-export từ core để backward compatibility
// Sử dụng crypto/rand để đảm bảo tính ngẫu nhiên và an toàn
func GenerateID() (string, error) {
	return core.GenerateID()
}

// IDLength và IDCharset được re-export từ core để backward compatibility
const (
	IDLength  = core.IDLength
	IDCharset = core.IDCharset
)

