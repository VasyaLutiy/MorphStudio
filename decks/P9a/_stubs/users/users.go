package users

import (
	"errors"
	"sync"
	"time"
)

var (
	ErrNotFound  = errors.New("user not found")
	ErrExists    = errors.New("user exists")
	ErrBadName   = errors.New("bad user name")
	ErrChatTaken = errors.New("chat already has a user")
)

type User struct {
	Name      string    `json:"name"`
	LinuxUser string    `json:"linux_user"`
	ChatID    int64     `json:"chat_id"`
	TokenHash string    `json:"token_hash"`
	CreatedAt time.Time `json:"created_at"`
}

type Users struct {
	mu        sync.Mutex
	base      string
	adminHash string
}

func Open(base string, adminToken string) (*Users, error) {
	panic("stub Open")
}

func (u *Users) Base() string {
	panic("stub Base")
}

func (u *Users) Add(v User) error {
	panic("stub Add")
}

func (u *Users) Get(name string) (User, error) {
	panic("stub Get")
}

func (u *Users) List() []User {
	panic("stub List")
}

func (u *Users) Delete(name string) error {
	panic("stub Delete")
}
