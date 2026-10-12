package users

import (
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("users: user not found")
var ErrExists = errors.New("users: user exists")
var ErrBadName = errors.New("users: bad user name")
var ErrChatTaken = errors.New("users: chat bound to another user")
var ErrEmptyToken = errors.New("users: empty token")
var ErrTokenTaken = errors.New("users: token taken")

var errStub = errors.New("stub users")

type User struct {
	Name      string    `json:"name"`
	LinuxUser string    `json:"linux_user"`
	ChatID    int64     `json:"chat_id"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type Registry struct {
	mu    sync.Mutex
	base  string
	users []User
}

func HashToken(token string) string                              { return "" }
func Open(base string) (*Registry, error)                        { return nil, errStub }
func (r *Registry) Base() string                                 { return "" }
func (r *Registry) Add(name string, now time.Time) (User, error) { return User{}, errStub }
func (r *Registry) Get(name string) (User, error)                { return User{}, errStub }
func (r *Registry) List() []User                                 { return nil }
func (r *Registry) ByChat(chatID int64) (User, error)            { return User{}, errStub }
func (r *Registry) SetChat(name string, chatID int64) error      { return errStub }
func (r *Registry) SetToken(name, token string) error            { return errStub }
