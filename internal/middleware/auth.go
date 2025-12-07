package middleware

import (
	"net/http"
	"strings"
)

// JWTValidator ensures the user is authenticated.
// In a real scenario, this would verify the signature using a public key.
func JWTValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization Header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid Authorization Header Format", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		// Mock validation for this exercise.
		// In production: jwt.Parse(token, ...)
		if token == "" {
			http.Error(w, "Invalid Token", http.StatusUnauthorized)
			return
		}

		// Proceed
		next.ServeHTTP(w, r)
	})
}
