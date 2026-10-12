// Package users implements the morphd user registry: the users of morphd in
// one JSON file, rewritten whole and atomically; a token is stored only as
// the SHA-256 hash of its bytes.
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
	"sync"
	"time"
)

var (
	ErrNotFound   = errors.New("users: user not found")
	ErrExists     = errors.New("users: user exists")
	ErrBadName    = errors.New("users: bad user name")
	ErrChatTaken  = errors.New("users: chat bound to another user")
	ErrEmptyToken = errors.New("users: empty token")
	ErrTokenTaken = errors.New("users: token taken")
)

// User is one registered user.
type User struct {
	Name      string    `json:"name"`
	LinuxUser string    `json:"linux_user"`
	ChatID    int64     `json:"chat_id"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// Registry is the user registry backed by base/users.json.
type Registry struct {
	mu    sync.Mutex
	base  string
	users []User
}

var nameRE = regexp.MustCompile(`^[a-z][a-z0-9]{0,15}$`)

// HashToken returns the lower-case hex of the SHA-256 of the token's bytes.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func usersPath(base string) string {
	return filepath.Join(base, "users.json")
}

// Open creates base with mode 0o700 when missing and reads base/users.json.
// A missing file yields an empty registry; a file that is not a JSON array
// of User yields an error beginning "users: read ".
func Open(base string) (*Registry, error) {
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	r := &Registry{base: base}
	data, err := os.ReadFile(usersPath(base))
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, err
	}
	var list []User
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("users: read %s: %w", usersPath(base), err)
	}
	if list == nil {
		return nil, fmt.Errorf("users: read %s: not a JSON array of users", usersPath(base))
	}
	r.users = list
	return r, nil
}

// Base returns the registry's base directory.
func (r *Registry) Base() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.base
}

// writeLocked rewrites the whole users.json file atomically, sorted by Name.
// It must be called with r.mu held.
func (r *Registry) writeLocked(users []User) error {
	sorted := make([]User, len(users))
	copy(sorted, users)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	data, err := json.MarshalIndent(sorted, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(r.base, "users-*.json")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, usersPath(r.base)); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// Add registers a new user named name, created at now.
func (r *Registry) Add(name string, now time.Time) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !nameRE.MatchString(name) {
		return User{}, ErrBadName
	}
	for _, u := range r.users {
		if u.Name == name {
			return User{}, ErrExists
		}
	}
	u := User{Name: name, LinuxUser: "mu-" + name, CreatedAt: now}
	next := make([]User, len(r.users), len(r.users)+1)
	copy(next, r.users)
	next = append(next, u)
	if err := r.writeLocked(next); err != nil {
		return User{}, err
	}
	r.users = next
	return u, nil
}

// Get returns the user named name, or ErrNotFound.
func (r *Registry) Get(name string) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if u.Name == name {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

// List returns a new slice of all users, sorted by Name.
func (r *Registry) List() []User {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]User, len(r.users))
	copy(out, r.users)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ByChat returns the user bound to chatID, or ErrNotFound.
func (r *Registry) ByChat(chatID int64) (User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if chatID == 0 {
		return User{}, ErrNotFound
	}
	for _, u := range r.users {
		if u.ChatID == chatID {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

// SetChat binds chatID to the user named name; chatID 0 unbinds.
func (r *Registry) SetChat(name string, chatID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	idx := -1
	for i, u := range r.users {
		if u.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrNotFound
	}
	if chatID != 0 {
		for i, u := range r.users {
			if i != idx && u.ChatID == chatID {
				return ErrChatTaken
			}
		}
	}
	next := make([]User, len(r.users))
	copy(next, r.users)
	next[idx].ChatID = chatID
	if err := r.writeLocked(next); err != nil {
		return err
	}
	r.users = next
	return nil
}

// SetToken sets the token of the user named name, storing only its hash.
func (r *Registry) SetToken(name, token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	idx := -1
	for i, u := range r.users {
		if u.Name == name {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrNotFound
	}
	if token == "" {
		return ErrEmptyToken
	}
	h := HashToken(token)
	for i, u := range r.users {
		if i != idx && u.TokenHash == h {
			return ErrTokenTaken
		}
	}
	next := make([]User, len(r.users))
	copy(next, r.users)
	next[idx].TokenHash = h
	if err := r.writeLocked(next); err != nil {
		return err
	}
	r.users = next
	return nil
}
