package auth

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Validator checks API keys.
type Validator struct {
	keys map[string]struct{}
}

// NewValidator creates a validator from configured API keys.
func NewValidator(keys []string) *Validator {
	set := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k != "" {
			set[k] = struct{}{}
		}
	}
	return &Validator{keys: set}
}

// Enabled returns true when at least one API key is configured.
func (v *Validator) Enabled() bool {
	return len(v.keys) > 0
}

// Valid reports whether the key is authorized.
func (v *Validator) Valid(key string) bool {
	if !v.Enabled() {
		return true
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	_, ok := v.keys[key]
	return ok
}

// Middleware protects routes when API keys are configured.
func (v *Validator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !v.Enabled() {
			next.ServeHTTP(w, r)
			return
		}
		key := extractAPIKey(r)
		if !v.Valid(key) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractAPIKey(r *http.Request) string {
	if key := r.Header.Get("X-API-Key"); key != "" {
		return key
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

// ConstantTimeValid compares a key using constant-time comparison.
func (v *Validator) ConstantTimeValid(key string) bool {
	if !v.Enabled() {
		return true
	}
	for candidate := range v.keys {
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(key)) == 1 {
			return true
		}
	}
	return false
}
