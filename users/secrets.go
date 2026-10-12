package users

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrNoSecret    = errors.New("no secret")
	ErrEmptySecret = errors.New("empty secret")
)

var SecretKinds = []string{"anthropic-key", "github-token"}

func isSecretKind(kind string) bool {
	for _, k := range SecretKinds {
		if k == kind {
			return true
		}
	}
	return false
}

func (u *Users) SecretPath(name, kind string) string {
	return filepath.Join(u.base, name, kind)
}

func (u *Users) PutSecret(name, kind, value string) error {
	if _, ok := u.byName[name]; !ok {
		return ErrNotFound
	}
	if !isSecretKind(kind) {
		return fmt.Errorf("users: unknown secret kind %q", kind)
	}
	if value == "" {
		return ErrEmptySecret
	}
	dir := filepath.Join(u.base, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".secret-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if _, err := tmp.WriteString(value); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(dir, kind)); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

func (u *Users) HasSecret(name, kind string) bool {
	_, err := os.Stat(u.SecretPath(name, kind))
	return err == nil
}

func (u *Users) ReadSecret(name, kind string) (string, error) {
	if _, ok := u.byName[name]; !ok {
		return "", ErrNotFound
	}
	data, err := os.ReadFile(u.SecretPath(name, kind))
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoSecret
		}
		return "", err
	}
	return string(data), nil
}

func (u *Users) DeleteSecret(name, kind string) error {
	if _, ok := u.byName[name]; !ok {
		return ErrNotFound
	}
	if err := os.Remove(u.SecretPath(name, kind)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
