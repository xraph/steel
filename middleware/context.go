package middleware

import (
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// =============================================================================
// Context Helpers
// =============================================================================

// GetRequestID retrieves request ID from context
func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value("request_id").(string); ok {
		return id
	}
	return ""
}

// GetJWTToken retrieves JWT token from context
func GetJWTToken(r *http.Request) *jwt.Token {
	if token, ok := r.Context().Value("jwt_token").(*jwt.Token); ok {
		return token
	}
	return nil
}

// GetJWTClaims retrieves JWT claims from context
func GetJWTClaims(r *http.Request) jwt.Claims {
	if claims, ok := r.Context().Value("jwt_claims").(jwt.Claims); ok {
		return claims
	}
	return nil
}

// GetUserID helper to extract user ID from JWT claims
func GetUserID(r *http.Request) string {
	claims := GetJWTClaims(r)
	if claims == nil {
		return ""
	}

	if mapClaims, ok := claims.(jwt.MapClaims); ok {
		if userID, exists := mapClaims["user_id"]; exists {
			if id, ok := userID.(string); ok {
				return id
			}
		}
		if sub, exists := mapClaims["sub"]; exists {
			if id, ok := sub.(string); ok {
				return id
			}
		}
	}

	return ""
}
