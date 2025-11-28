package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration
type Config struct {
	JWT         JWTConfig
	Server      ServerConfig
	Password    PasswordConfig
	ServiceName string // Service name for microservice architecture (max 20 chars, empty = single app mode)
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret            string        // Secret key để ký JWT token
	Expiration        time.Duration // Access token expiration (mặc định: 24h)
	RefreshExpiration time.Duration // Refresh token expiration (mặc định: 7 ngày)
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	CookieSecure bool // Cookie Secure flag (true = HTTPS only, false = allow HTTP for dev)
}

// PasswordConfig holds password validation configuration
type PasswordConfig struct {
	MinLength          int  // Độ dài tối thiểu (mặc định: 6)
	RequireUppercase   bool // Yêu cầu chữ hoa (mặc định: true)
	RequireLowercase   bool // Yêu cầu chữ thường (mặc định: true)
	RequireDigit       bool // Yêu cầu chữ số (mặc định: true)
	RequireSpecialChar bool // Yêu cầu ký tự đặc biệt (mặc định: false)
	MinSpecialChars    int  // Số lượng ký tự đặc biệt tối thiểu (mặc định: 0)
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	// Parse JWT expiration (supports decimal values like 0.00833 for 30 seconds)
	jwtExpirationHoursStr := getEnv("JWT_EXPIRATION_HOURS", "24")
	jwtExpirationHours, _ := strconv.ParseFloat(jwtExpirationHoursStr, 64)
	
	// Parse refresh token expiration (supports decimal values like 0.000694 for 60 seconds)
	refreshExpirationDaysStr := getEnv("REFRESH_TOKEN_EXPIRATION_DAYS", "7")
	refreshExpirationDays, _ := strconv.ParseFloat(refreshExpirationDaysStr, 64)
	
	readTimeout, _ := strconv.Atoi(getEnv("READ_TIMEOUT_SECONDS", "10"))
	writeTimeout, _ := strconv.Atoi(getEnv("WRITE_TIMEOUT_SECONDS", "10"))

	// Password configuration
	passwordMinLength, _ := strconv.Atoi(getEnv("PASSWORD_MIN_LENGTH", "6"))
	passwordRequireUppercase := getEnv("PASSWORD_REQUIRE_UPPERCASE", "true") == "true"
	passwordRequireLowercase := getEnv("PASSWORD_REQUIRE_LOWERCASE", "true") == "true"
	passwordRequireDigit := getEnv("PASSWORD_REQUIRE_DIGIT", "true") == "true"
	passwordRequireSpecialChar := getEnv("PASSWORD_REQUIRE_SPECIAL_CHAR", "false") == "true"
	passwordMinSpecialChars, _ := strconv.Atoi(getEnv("PASSWORD_MIN_SPECIAL_CHARS", "0"))

	serviceName := getEnv("SERVICE_NAME", "")
	// Truncate to max 20 characters if longer
	if len(serviceName) > 20 {
		serviceName = serviceName[:20]
	}

	// Cookie Secure configuration (default: true for production security)
	cookieSecure := getEnv("COOKIE_SECURE", "true") == "true"

	return &Config{
		JWT: JWTConfig{
			Secret:            getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
			Expiration:        time.Duration(jwtExpirationHours * float64(time.Hour)),
			RefreshExpiration: time.Duration(refreshExpirationDays * 24 * float64(time.Hour)),
		},
		Server: ServerConfig{
			Port:         getEnv("PORT", "3000"),
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
			CookieSecure: cookieSecure,
		},
		Password: PasswordConfig{
			MinLength:          passwordMinLength,
			RequireUppercase:   passwordRequireUppercase,
			RequireLowercase:   passwordRequireLowercase,
			RequireDigit:       passwordRequireDigit,
			RequireSpecialChar: passwordRequireSpecialChar,
			MinSpecialChars:    passwordMinSpecialChars,
		},
		ServiceName: serviceName,
	}
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
