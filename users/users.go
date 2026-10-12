package users

import (
	"bytes"
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
	base      string
	adminHash string
	byName    map[string]User
}

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

const usersFileName = "users.json"

func Open(base string, adminToken string) (*Users, error) {
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	u := &Users{
		base:   base,
		byName: map[string]User{},
	}
	if adminToken != "" {
		u.adminHash = sha256Hex(adminToken)
	}
	path := filepath.Join(base, usersFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return u, nil
		}
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("users: read %s: not a JSON array", path)
	}
	var list []User
	if err := json.Unmarshal(trimmed, &list); err != nil {
		return nil, fmt.Errorf("users: read %s: %w", path, err)
	}
	for _, v := range list {
		u.byName[v.Name] = v
	}
	return u, nil
}

func (u *Users) Base() string { return u.base }

func (u *Users) Add(v User) error {
	if !nameRE.MatchString(v.Name) {
		return ErrBadName
	}
	if _, ok := u.byName[v.Name]; ok {
		return ErrExists
	}
	for _, ex := range u.byName {
		if ex.ChatID == v.ChatID {
			return ErrChatTaken
		}
	}
	u.byName[v.Name] = v
	return u.persist()
}

func (u *Users) Get(name string) (User, error) {
	v, ok := u.byName[name]
	if !ok {
		return User{}, ErrNotFound
	}
	return v, nil
}

func (u *Users) List() []User {
	out := make([]User, 0, len(u.byName))
	for _, v := range u.byName {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (u *Users) Delete(name string) error {
	if _, ok := u.byName[name]; !ok {
		return ErrNotFound
	}
	delete(u.byName, name)
	if err := os.RemoveAll(filepath.Join(u.base, name)); err != nil {
		return err
	}
	return u.persist()
}

func (u *Users) persist() error {
	data, err := json.MarshalIndent(u.List(), "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(u.base, "users-*.json")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	return os.Rename(name, filepath.Join(u.base, usersFileName))
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
