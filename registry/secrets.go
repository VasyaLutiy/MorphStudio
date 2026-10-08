package registry

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ErrNoSecret is returned when a project has no stored secret of a kind.
var ErrNoSecret = errors.New("no secret")

// ErrEmptySecret is returned when a secret value is empty.
var ErrEmptySecret = errors.New("empty secret")

// SecretKinds lists the secret kinds the registry stores.
var SecretKinds = []string{"github-token", "git-credentials", "session-token"}

func isSecretKind(kind string) bool {
	for _, k := range SecretKinds {
		if k == kind {
			return true
		}
	}
	return false
}

// SecretPath returns the file a project's secret of kind is stored in.
func (r *Registry) SecretPath(name, kind string) string {
	return filepath.Join(r.base, name, kind)
}

// PutSecret stores value as the project's secret of kind.
func (r *Registry) PutSecret(name, kind, value string) error {
	if _, err := r.Get(name); err != nil {
		return err
	}
	if !isSecretKind(kind) {
		return fmt.Errorf("registry: unknown secret kind %q", kind)
	}
	if value == "" {
		return ErrEmptySecret
	}
	dir := filepath.Join(r.base, name)
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
	if err := os.Chmod(tmp, 0o600); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, r.SecretPath(name, kind)); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// HasSecret reports whether the project has a secret of kind.
func (r *Registry) HasSecret(name, kind string) bool {
	_, err := os.Stat(r.SecretPath(name, kind))
	return err == nil
}

// ReadSecret returns the project's secret of kind verbatim.
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

// DeleteSecret removes the project's secret of kind.
func (r *Registry) DeleteSecret(name, kind string) error {
	err := os.Remove(r.SecretPath(name, kind))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
