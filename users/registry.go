// Package users implements the user registry that backs the per-user HTTP
// and MCP bearer tokens.
package users

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

// The sentinel errors of the user registry. None of them is a file system
// error: every error of the file system is returned as is.
var (
	ErrNotFound   = errors.New("user not found")
	ErrExists     = errors.New("user exists")
	ErrBadName    = errors.New("bad user name")
	ErrEmptyToken = errors.New("empty token")
)

// nameRE is the pattern a user name must match.
var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

// User is one stored user. TokenHash is the lowercase hex SHA-256 of the
// user's bearer token; the token itself is never stored.
type User struct {
	Name      string    `json:"name"`
	LinuxUser string    `json:"linux_user"`
	ChatID    int64     `json:"chat_id"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is a user registry backed by base/users.json, rewritten whole and
// atomically on every change.
type Store struct {
	base           string
	users          map[string]User
	adminTokenHash string
}

// hashToken returns the lowercase hex SHA-256 of token.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Open creates base (mode 0o700) when it is missing and loads base/users.json
// from it. A missing file yields an empty store; a file that is not a JSON
// array of User is refused with an error beginning "users: read ". The admin
// token is hashed with hashToken and kept; "" means that there is no admin
// token.
func Open(base string, adminToken string) (*Store, error) {
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	s := &Store{
		base:  base,
		users: make(map[string]User),
	}
	if adminToken != "" {
		s.adminTokenHash = hashToken(adminToken)
	}
	data, err := os.ReadFile(filepath.Join(base, "users.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return s, nil
		}
		return nil, err
	}
	var list []User
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("users: read %w", err)
	}
	if list == nil {
		return nil, fmt.Errorf("users: read %w", errors.New("not a JSON array of users"))
	}
	for _, u := range list {
		s.users[u.Name] = u
	}
	return s, nil
}

// Base returns the directory the store reads and writes.
func (s *Store) Base() string {
	return s.base
}

// Add stores u under u.Name with the given token. The user's TokenHash is
// overwritten with hashToken(token) and the whole list is rewritten
// atomically. A bad name is ErrBadName, a name already stored is ErrExists,
// an empty token is ErrEmptyToken.
func (s *Store) Add(u User, token string) error {
	if !nameRE.MatchString(u.Name) {
		return ErrBadName
	}
	if _, ok := s.users[u.Name]; ok {
		return ErrExists
	}
	if token == "" {
		return ErrEmptyToken
	}
	u.TokenHash = hashToken(token)
	s.users[u.Name] = u
	if err := s.save(); err != nil {
		delete(s.users, u.Name)
		return err
	}
	return nil
}

// Get returns the user named name or (User{}, ErrNotFound).
func (s *Store) Get(name string) (User, error) {
	u, ok := s.users[name]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

// List returns a new slice of the users sorted by Name. The slice is never
// nil; an empty store yields an empty, non-nil slice.
func (s *Store) List() []User {
	return s.sorted()
}

func (s *Store) sorted() []User {
	out := make([]User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// save writes the whole user list as a JSON array sorted by Name to a
// temporary file in base and renames it onto users.json.
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.sorted(), "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.base, "users-*.json")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(s.base, "users.json")); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
