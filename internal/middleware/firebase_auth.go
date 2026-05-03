// Package middleware provides HTTP middleware for the DISHA AI backend.
package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// authContextKey is an unexported type for context keys in this package.
type authContextKey string

const (
	// UIDKey is the context key for the authenticated Firebase UID.
	UIDKey authContextKey = "firebaseUID"
)

// FirebaseAuth is a middleware that verifies Firebase ID tokens.
type FirebaseAuth struct {
	client *auth.Client
}

// NewFirebaseAuth initialises a FirebaseAuth middleware using the provided
// service-account credentials JSON string and project ID.
func NewFirebaseAuth(ctx context.Context, projectID, credsJSON string) (*FirebaseAuth, error) {
	var opts []option.ClientOption
	if credsJSON != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credsJSON)))
	}

	cfg := &firebase.Config{ProjectID: projectID}
	app, err := firebase.NewApp(ctx, cfg, opts...)
	if err != nil {
		return nil, err
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, err
	}

	return &FirebaseAuth{client: client}, nil
}

func writeAuthError(w http.ResponseWriter, code, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// Middleware returns an http.Handler middleware that extracts and verifies the
// Bearer token from the Authorization header. On success, it injects the UID
// into the request context. On failure, it returns 401 Unauthorized.
func (fa *FirebaseAuth) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			writeAuthError(w, "UNAUTHORIZED", "Missing or invalid Authorization header", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		decoded, err := fa.client.VerifyIDToken(r.Context(), token)
		if err != nil {
			writeAuthError(w, "UNAUTHORIZED", "Invalid or expired Firebase token", http.StatusUnauthorized)
			return
		}

		// Inject the UID into the request context
		ctx := context.WithValue(r.Context(), UIDKey, decoded.UID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UIDFromContext retrieves the Firebase UID injected by the FirebaseAuth middleware.
// Returns empty string if not present.
func UIDFromContext(ctx context.Context) string {
	uid, _ := ctx.Value(UIDKey).(string)
	return uid
}
