// Package users: per-user secrets stored as files under base/<name>/<kind>,
// mode 0600, never returned in error texts and never logged.
package users

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

var (
	ErrNoSecret    = errors.New("users: no secret")
	ErrEmptySecret = errors.New("users: empty secret")
)

// SecretKinds are the accepted secret kinds.
var SecretKinds = []string{"anthropic-key", "github-token"}

func knownSecretKind(kind string) bool {
	for _, k := range SecretKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// SecretPath returns the file path of the secret of kind kind for name.
func (r *Registry) SecretPath(name, kind string) string {
	return filepath.Join(r.base, name, kind)
}

// PutSecret writes value as the secret of kind kind for name. The file is
// written atomically through a temporary file of mode 0600 in the user's
// directory and renamed onto the destination, so an existing file with
// another mode ends up 0600. The value never appears in any returned error.
func (r *Registry) PutSecret(name, kind, value string) error {
	if _, err := r.Get(name); err != nil {
		return err
	}
	if !knownSecretKind(kind) {
		return fmt.Errorf("users: unknown secret kind %q", kind)
	}
	if value == "" {
		return ErrEmptySecret
	}
	dir := filepath.Join(r.base, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "secret-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(value); err != nil {
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
	if err := os.Rename(tmpName, filepath.Join(dir, kind)); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// HasSecret reports whether the secret of kind kind exists for name.
func (r *Registry) HasSecret(name, kind string) bool {
	if _, err := os.Stat(r.SecretPath(name, kind)); err != nil {
		return false
	}
	return true
}

// ReadSecret returns the verbatim content of the secret of kind kind for
// name. A missing file yields ("", ErrNoSecret); an unknown user yields
// ("", ErrNotFound).
func (r *Registry) ReadSecret(name, kind string) (string, error) {
	if _, err := r.Get(name); err != nil {
		return "", err
	}
	data, err := os.ReadFile(r.SecretPath(name, kind))
	if err != nil {
		if os.IsNotExist(err) {
			return "", ErrNoSecret
		}
		return "", err
	}
	return string(data), nil
}

// DeleteSecret removes the secret of kind kind for name. A missing file is
// not an error; an unknown user yields ErrNotFound.
func (r *Registry) DeleteSecret(name, kind string) error {
	if _, err := r.Get(name); err != nil {
		return err
	}
	err := os.Remove(r.SecretPath(name, kind))
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}
