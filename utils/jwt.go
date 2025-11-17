package utils

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents JWT claims
type JWTClaims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	RoleIDs []uint `json:"role_ids"` // Role IDs stored in token - protected by signature
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
