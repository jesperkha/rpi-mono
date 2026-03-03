package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
	"time"
)

const (
	sessionCookieName = "session"
	sessionDuration   = 7 * 24 * time.Hour
)

type AuthMiddleware struct {
	passwordHash string
	sessions     map[string]time.Time
}

func NewAuthMiddleware(passwordHash string) *AuthMiddleware {
	return &AuthMiddleware{
		passwordHash: passwordHash,
		sessions:     make(map[string]time.Time),
	}
}

// Middleware checks if the request has a valid session cookie
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow access to login page and static assets
		if r.URL.Path == "/login" ||
			strings.HasPrefix(r.URL.Path, "/assets/") ||
			r.URL.Path == "/manifest.json" ||
			r.URL.Path == "/ping" {
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || !a.isValidSession(cookie.Value) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SECURITY: Plain SHA-256 is a fast hash with no salt — vulnerable to
// brute-force and rainbow table attacks. Use bcrypt or argon2id instead.

// ValidatePassword checks if the provided password matches the stored hash
func (a *AuthMiddleware) ValidatePassword(password string) bool {
	hash := sha256.Sum256([]byte(password))
	hashStr := hex.EncodeToString(hash[:])
	return subtle.ConstantTimeCompare([]byte(hashStr), []byte(a.passwordHash)) == 1
}

// CreateSession creates a new session and returns the session token
func (a *AuthMiddleware) CreateSession() string {
	token := generateToken()
	a.sessions[token] = time.Now().Add(sessionDuration)
	return token
}

// InvalidateSession removes a session
func (a *AuthMiddleware) InvalidateSession(token string) {
	delete(a.sessions, token)
}

func (a *AuthMiddleware) isValidSession(token string) bool {
	expiry, exists := a.sessions[token]
	if !exists {
		return false
	}
	if time.Now().After(expiry) {
		delete(a.sessions, token)
		return false
	}
	return true
}

func generateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate secure random token: " + err.Error())
	}
	return hex.EncodeToString(b)
}
