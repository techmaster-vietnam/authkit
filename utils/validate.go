package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/techmaster-vietnam/authkit/config"
	"github.com/techmaster-vietnam/goerrorkit"
)

// ValidateEmail kiểm tra format email hợp lệ
// Email hợp lệ phải:
// - Không rỗng
// - Chứa ký tự @
// - Có format hợp lệ (local@domain)
// - Tuân theo RFC 5321 (local part tối đa 64 ký tự, domain tối đa 255 ký tự)
func ValidateEmail(email string) error {
	// Kiểm tra email rỗng
	if strings.TrimSpace(email) == "" {
		return goerrorkit.NewValidationError("Email là bắt buộc", map[string]interface{}{
			"field": "email",
		})
	}

	// Kiểm tra email có chứa @ không
	if !strings.Contains(email, "@") {
		return goerrorkit.NewValidationError("Email không hợp lệ: thiếu ký tự @", map[string]interface{}{
			"field": "email",
			"value": email,
		})
	}

	// Regex pattern để validate email format
	// Pattern này kiểm tra:
	// - Local part: chứa chữ cái, số, dấu chấm, gạch dưới, dấu gạch ngang
	// - @ symbol
	// - Domain part: chứa chữ cái, số, dấu chấm, dấu gạch ngang
	// - TLD: ít nhất 2 ký tự
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return goerrorkit.NewValidationError("Email không hợp lệ: format không đúng", map[string]interface{}{
			"field": "email",
			"value": email,
		})
	}

	// Kiểm tra độ dài email (RFC 5321: local part tối đa 64 ký tự, domain tối đa 255 ký tự)
	if len(email) > 320 { // Tổng độ dài email không quá 320 ký tự
		return goerrorkit.NewValidationError("Email quá dài (tối đa 320 ký tự)", map[string]interface{}{
			"field":      "email",
			"value":      email,
			"max_length": 320,
		})
	}

	// Kiểm tra local part và domain part
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return goerrorkit.NewValidationError("Email không hợp lệ: format không đúng", map[string]interface{}{
			"field": "email",
			"value": email,
		})
	}

	localPart := parts[0]
	domainPart := parts[1]

	// Kiểm tra local part không rỗng và không quá dài
	if len(localPart) == 0 || len(localPart) > 64 {
		return goerrorkit.NewValidationError("Email không hợp lệ: phần trước @ không hợp lệ", map[string]interface{}{
			"field": "email",
			"value": email,
		})
	}

	// Kiểm tra domain part không rỗng và không quá dài
	if len(domainPart) == 0 || len(domainPart) > 255 {
		return goerrorkit.NewValidationError("Email không hợp lệ: phần sau @ không hợp lệ", map[string]interface{}{
			"field": "email",
			"value": email,
		})
	}

	// Kiểm tra domain có chứa ít nhất một dấu chấm (có TLD)
	if !strings.Contains(domainPart, ".") {
		return goerrorkit.NewValidationError("Email không hợp lệ: domain phải có TLD", map[string]interface{}{
			"field": "email",
			"value": email,
		})
	}

	return nil
}

// ValidatePassword kiểm tra password theo các quy tắc được cấu hình
// Password hợp lệ phải:
// - Đạt độ dài tối thiểu (theo config)
// - Chứa chữ hoa (nếu RequireUppercase = true)
// - Chứa chữ thường (nếu RequireLowercase = true)
// - Chứa chữ số (nếu RequireDigit = true)
// - Chứa ký tự đặc biệt (nếu RequireSpecialChar = true)
func ValidatePassword(password string, cfg config.PasswordConfig) error {
	// Kiểm tra password rỗng
	if strings.TrimSpace(password) == "" {
		return goerrorkit.NewValidationError("Mật khẩu là bắt buộc", map[string]interface{}{
			"field": "password",
		})
	}

	// Kiểm tra độ dài tối thiểu
	if len(password) < cfg.MinLength {
		return goerrorkit.NewValidationError(fmt.Sprintf("Mật khẩu phải có ít nhất %d ký tự", cfg.MinLength), map[string]interface{}{
			"field":      "password",
			"min_length": cfg.MinLength,
		})
	}

	var hasUppercase, hasLowercase, hasDigit bool
	specialCharCount := 0

	// Định nghĩa ký tự đặc biệt
	specialChars := "!@#$%^&*()_+-=[]{}|;:,.<>?/~`"

	// Kiểm tra từng ký tự trong password
	for _, char := range password {
		if unicode.IsUpper(char) {
			hasUppercase = true
		}
		if unicode.IsLower(char) {
			hasLowercase = true
		}
		if unicode.IsDigit(char) {
			hasDigit = true
		}
		if strings.ContainsRune(specialChars, char) {
			specialCharCount++
		}
	}

	// Kiểm tra các yêu cầu
	var errors []string
	var errorData = make(map[string]interface{})

	if cfg.RequireUppercase && !hasUppercase {
		errors = append(errors, "chữ hoa")
		errorData["require_uppercase"] = true
	}

	if cfg.RequireLowercase && !hasLowercase {
		errors = append(errors, "chữ thường")
		errorData["require_lowercase"] = true
	}

	if cfg.RequireDigit && !hasDigit {
		errors = append(errors, "chữ số")
		errorData["require_digit"] = true
	}

	if cfg.RequireSpecialChar {
		if specialCharCount < cfg.MinSpecialChars {
			errors = append(errors, "ký tự đặc biệt")
			errorData["require_special_char"] = true
			errorData["min_special_chars"] = cfg.MinSpecialChars
			errorData["found_special_chars"] = specialCharCount
		}
	}

	// Nếu có lỗi, trả về thông báo lỗi
	if len(errors) > 0 {
		errorMsg := "Mật khẩu phải chứa ít nhất: " + strings.Join(errors, ", ")
		errorData["field"] = "password"
		errorData["min_length"] = cfg.MinLength
		return goerrorkit.NewValidationError(errorMsg, errorData)
	}

	return nil
}
