package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username,omitempty"` // Optional username field
	RoleIDs  []uint `json:"role_ids"`          // Role IDs stored in token - protected by signature
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token with role IDs
// Role IDs are protected by HMAC-SHA256 signature - cannot be tampered without detection
func GenerateToken(userID string, email string, roleIDs []uint, secret string, expiration time.Duration) (string, error) {
	claims := JWTClaims{
		UserID:  userID,
		Email:   email,
		RoleIDs: roleIDs,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "authkit",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateToken validates a JWT token and verifies signature
// Returns claims only if token signature is valid - prevents role tampering
func ValidateToken(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	// Only return claims if token is valid (signature verified)
	// If hacker modifies role_ids, signature won't match and token.Valid will be false
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// ClaimsConfig configures custom claims for flexible token generation
type ClaimsConfig struct {
	// Username to include in token (optional)
	Username string
	
	// Custom fields to add to token claims
	CustomFields map[string]interface{}
	
	// Role format: "ids" ([]uint), "names" ([]string), or "both"
	RoleFormat string // "ids" | "names" | "both"
	
	// Role IDs (when RoleFormat is "ids" or "both")
	RoleIDs []uint
	
	// Role Names (when RoleFormat is "names" or "both")
	RoleNames []string
}

// GenerateTokenFlexible generates a JWT token with flexible claims configuration
// This function supports:
// - Username field
// - Custom fields
// - Role IDs, Role Names, or both
// Role IDs are protected by HMAC-SHA256 signature - cannot be tampered without detection
func GenerateTokenFlexible(
	userID string,
	email string,
	config ClaimsConfig,
	secret string,
	expiration time.Duration,
) (string, error) {
	// Use MapClaims for flexibility
	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"exp":     time.Now().Add(expiration).Unix(),
		"iat":     time.Now().Unix(),
		"nbf":     time.Now().Unix(),
		"iss":     "authkit",
	}
	
	// Add username if provided
	if config.Username != "" {
		claims["username"] = config.Username
	}
	
	// Add roles based on format
	if config.RoleFormat == "names" || config.RoleFormat == "both" {
		if len(config.RoleNames) > 0 {
			claims["roles"] = config.RoleNames
		}
	}
	if config.RoleFormat == "ids" || config.RoleFormat == "both" {
		if len(config.RoleIDs) > 0 {
			claims["role_ids"] = config.RoleIDs
		}
	}
	
	// Add custom fields (skip username if already set)
	for k, v := range config.CustomFields {
		if k != "username" || config.Username == "" {
			claims[k] = v
		}
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateTokenFlexible validates a JWT token and returns MapClaims for flexible extraction
// Returns claims only if token signature is valid - prevents role tampering
func ValidateTokenFlexible(tokenString, secret string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Only return claims if token is valid (signature verified)
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	
	return nil, jwt.ErrSignatureInvalid
}

// ValidateTokenAndExtractRoleIDs validates a JWT token and extracts role IDs
// Supports both JWTClaims (from GenerateToken) and MapClaims (from GenerateTokenFlexible)
// Returns userID, email, and roleIDs if token is valid
func ValidateTokenAndExtractRoleIDs(tokenString, secret string) (userID string, email string, roleIDs []uint, err error) {
	// First try to validate as JWTClaims (standard token)
	claims, err := ValidateToken(tokenString, secret)
	if err == nil && claims != nil {
		// Standard token format
		roleIDs = claims.RoleIDs
		if roleIDs == nil {
			roleIDs = []uint{}
		}
		return claims.UserID, claims.Email, roleIDs, nil
	}

	// If standard validation failed, try flexible token format
	mapClaims, err2 := ValidateTokenFlexible(tokenString, secret)
	if err2 == nil && mapClaims != nil {
		// Flexible token format (MapClaims)
		// Extract userID
		if uid, ok := mapClaims["user_id"].(string); ok {
			userID = uid
		} else {
			return "", "", nil, fmt.Errorf("user_id not found in token")
		}

		// Extract email
		if em, ok := mapClaims["email"].(string); ok {
			email = em
		} else {
			return "", "", nil, fmt.Errorf("email not found in token")
		}

		// Extract role_ids from MapClaims
		roleIDs = []uint{}
		if roleIDsRaw, ok := mapClaims["role_ids"]; ok {
			switch v := roleIDsRaw.(type) {
			case []interface{}:
				// Convert []interface{} to []uint
				for _, id := range v {
					switch idVal := id.(type) {
					case float64:
						roleIDs = append(roleIDs, uint(idVal))
					case uint:
						roleIDs = append(roleIDs, idVal)
					case int:
						roleIDs = append(roleIDs, uint(idVal))
					}
				}
			case []uint:
				roleIDs = v
			case []float64:
				// JSON numbers are parsed as float64
				for _, id := range v {
					roleIDs = append(roleIDs, uint(id))
				}
			}
		}

		return userID, email, roleIDs, nil
	}

	// Both validations failed
	if err != nil {
		return "", "", nil, err
	}
	return "", "", nil, err2
}
