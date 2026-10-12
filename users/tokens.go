package users

import (
	"crypto/subtle"
	"encoding/hex"
)

// LookupToken resolves a bearer token to exactly one user. The empty token
// never matches; otherwise the SHA-256 hex of token is compared, over the
// decoded bytes and in constant time, against every stored TokenHash and the
// first name-sorted match is returned.
func (u *Users) LookupToken(token string) (User, bool) {
	if token == "" {
		return User{}, false
	}
	want, err := hex.DecodeString(sha256Hex(token))
	if err != nil {
		return User{}, false
	}
	for _, v := range u.List() {
		got, err := hex.DecodeString(v.TokenHash)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare(got, want) == 1 {
			return v, true
		}
	}
	return User{}, false
}

// IsAdmin reports whether token is the operator's admin token. The empty
// token never matches, and a registry opened without an admin token has no
// admin at all.
func (u *Users) IsAdmin(token string) bool {
	if token == "" || u.adminHash == "" {
		return false
	}
	want, err := hex.DecodeString(sha256Hex(token))
	if err != nil {
		return false
	}
	got, err := hex.DecodeString(u.adminHash)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(got, want) == 1
}
