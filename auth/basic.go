package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

// Users maps a user name to a hash string built by HashPassword.
type Users map[string]string

// HashPassword returns "sha256$" + salt + "$" + the lower-case hex of
// sha256.Sum256([]byte(salt + password)).
func HashPassword(password, salt string) string {
	sum := sha256.Sum256([]byte(salt + password))
	return "sha256$" + salt + "$" + hex.EncodeToString(sum[:])
}

// Verify reports whether password matches the stored hash of user.
func Verify(users Users, user, password string) bool {
	hash, ok := users[user]
	if !ok {
		return false
	}
	parts := strings.SplitN(hash, "$", 3)
	if len(parts) != 3 || parts[0] != "sha256" {
		return false
	}
	want := HashPassword(password, parts[1])
	return subtle.ConstantTimeCompare([]byte(want), []byte(hash)) == 1
}

// Require guards next with HTTP Basic authentication against users.
func Require(users Users, realm string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, password, ok := r.BasicAuth()
		if !ok || !Verify(users, user, password) {
			w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("{\"error\":{\"code\":\"unauthorized\"}}\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
