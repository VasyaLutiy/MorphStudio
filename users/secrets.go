package users

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// The sentinel errors of the per-user secret store.
var (
	ErrNoSecret    = errors.New("no secret")
	ErrEmptySecret = errors.New("empty secret")
)

// SecretKinds are the kinds of per-user secret the store accepts.
var SecretKinds = []string{"anthropic-key", "github-token"}

// validKind reports whether kind is one of SecretKinds.
func validKind(kind string) bool {
	for _, k := range SecretKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// SecretPath returns the path of the file holding the secret of the given
// kind for the named user. The name is not validated: the caller does.
func (s *Store) SecretPath(name, kind string) string {
	return filepath.Join(s.base, name, kind)
}

// PutSecret stores value as the secret of the given kind for the named user.
// An unknown user is ErrNotFound, an unknown kind is an error beginning
// "users: unknown secret kind", an empty value is ErrEmptySecret. The user
// directory is created mode 0o700 when it is missing and the value is written
// through a temporary file of mode 0o600 renamed onto SecretPath, so the
// resulting file has mode 0o600 even when it existed with another mode. The
// value never appears in an error text.
func (s *Store) PutSecret(name, kind, value string) error {
	if _, ok := s.users[name]; !ok {
		return ErrNotFound
	}
	if !validKind(kind) {
		return fmt.Errorf("users: unknown secret kind %q", kind)
	}
	if value == "" {
		return ErrEmptySecret
	}
	dir := filepath.Join(s.base, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".secret-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.WriteString(value); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, filepath.Join(dir, kind)); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// HasSecret reports whether the secret of the given kind exists for the named
// user.
func (s *Store) HasSecret(name, kind string) bool {
	_, err := os.Stat(s.SecretPath(name, kind))
	return err == nil
}

// ReadSecret returns the content of the secret of the given kind verbatim. A
// missing secret is ErrNoSecret; an unknown user is ErrNotFound. The value is
// never part of an error text.
func (s *Store) ReadSecret(name, kind string) (string, error) {
	if _, ok := s.users[name]; !ok {
		return "", ErrNotFound
	}
	data, err := os.ReadFile(s.SecretPath(name, kind))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrNoSecret
		}
		return "", err
	}
	return string(data), nil
}

// DeleteSecret removes the secret of the given kind for the named user. A
// missing secret is not an error.
func (s *Store) DeleteSecret(name, kind string) error {
	if err := os.Remove(s.SecretPath(name, kind)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
