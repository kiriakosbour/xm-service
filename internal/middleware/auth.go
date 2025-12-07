package middleware

import (
	"context"
	"net/http"
	"strings"
)

// ContextKey is used for context values
type ContextKey string

const (
	// UserIDKey is the context key for the authenticated user ID
	UserIDKey ContextKey = "userID"
)

// JWTAuth is a middleware that validates JWT tokens
// This is a mock implementation for the exercise
// In production, you would:
// 1. Parse the JWT token
// 2. Verify the signature using a secret or public key
// 3. Check token expiration
// 4. Extract claims and add to context
func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error": "missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
			return
		}

		token := parts[1]
		if token == "" {
			http.Error(w, `{"error": "empty token"}`, http.StatusUnauthorized)
			return
		}

		// Mock validation: In production, verify the JWT signature and claims
		// For this exercise, we accept any non-empty token
		// Example of what production code would look like:
		/*
			claims := &jwt.RegisteredClaims{}
			parsedToken, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
				return []byte(jwtSecret), nil
			})
			if err != nil || !parsedToken.Valid {
				http.Error(w, `{"error": "invalid token"}`, http.StatusUnauthorized)
				return
			}
		*/

		// Add mock user ID to context
		ctx := context.WithValue(r.Context(), UserIDKey, "mock-user-id")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserID extracts the user ID from the context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}
