package utils

import (
	"crypto/rand"
)

const (
	// IDLength là độ dài của ID (12 ký tự)
	IDLength = 12
	// IDCharset là bộ ký tự được sử dụng để tạo ID (a-zA-Z0-9)
	IDCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// GenerateID tạo một ID ngẫu nhiên 12 ký tự từ a-zA-Z0-9
// Sử dụng crypto/rand để đảm bảo tính ngẫu nhiên và an toàn
func GenerateID() (string, error) {
	bytes := make([]byte, IDLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	result := make([]byte, IDLength)
	charsetLen := len(IDCharset)
	
	for i := 0; i < IDLength; i++ {
		result[i] = IDCharset[int(bytes[i])%charsetLen]
	}
	
	return string(result), nil
}

