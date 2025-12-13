package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/vsevolod-ryzhov/gofermart/internal/service"
)

type contextKey string

const UserIDKey contextKey = "userID"

func respondWithJSONError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
}

func Auth(authService *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var tokenString string

			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					tokenString = parts[1]
				}
			}

			if tokenString == "" {
				cookie, err := r.Cookie("auth_token")
				if err == nil && cookie.Value != "" {
					tokenString = cookie.Value
				}
			}

			if tokenString == "" {
				respondWithJSONError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			userID, err := authService.ValidateToken(tokenString)
			if err != nil {
				respondWithJSONError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) (int, bool) {
	userID := r.Context().Value(UserIDKey)
	if userID == nil {
		return 0, false
	}
	return userID.(int), true
}
