package users

import (
	"errors"
	"time"
)

// ErrNotFound is returned when a user is not stored.
var ErrNotFound = errors.New("user not found")

// ErrExists is returned when a user name is already stored.
var ErrExists = errors.New("user exists")

// ErrBadName is returned when a user name is not acceptable.
var ErrBadName = errors.New("bad user name")

// ErrEmptyToken is returned when a token is empty.
var ErrEmptyToken = errors.New("empty token")

var errStubUserRegistry = errors.New("stub user registry")

// User is one registered user.
type User struct {
	Name      string    `json:"name"`
	LinuxUser string    `json:"linux_user"`
	ChatID    int64     `json:"chat_id"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
}

// Store is the user store rooted at a base directory.
type Store struct {
	base      string
	users     []User
	adminHash string
}

func hashToken(token string) string { return "" }

// Open creates base when missing and reads its users.json.
func Open(base string, adminToken string) (*Store, error) {
	return nil, errStubUserRegistry
}

// Base returns the directory the store lives in.
func (s *Store) Base() string { return "" }

// Add validates and stores a user with its token hash, rewriting the whole list.
func (s *Store) Add(u User, token string) error { return errStubUserRegistry }

// Get returns the stored user with the given name.
func (s *Store) Get(name string) (User, error) { return User{}, errStubUserRegistry }

// List returns a new sorted copy of the stored users.
func (s *Store) List() []User { return nil }
