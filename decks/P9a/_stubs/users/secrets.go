package users

import "errors"

// ErrNoSecret is returned when a user has no stored secret of a kind.
var ErrNoSecret = errors.New("no secret")

// ErrEmptySecret is returned when a secret value is empty.
var ErrEmptySecret = errors.New("empty secret")

// SecretKinds are the secret kinds a user may store.
var SecretKinds = []string{"anthropic-key", "github-token"}

var errStubUserSecrets = errors.New("stub user secrets")

// SecretPath is the file a user's secret of a kind lives in.
func (s *Store) SecretPath(name, kind string) string { return "" }

// PutSecret writes a user's secret of a kind with mode 0600.
func (s *Store) PutSecret(name, kind, value string) error { return errStubUserSecrets }

// HasSecret reports whether a user has a secret of a kind.
func (s *Store) HasSecret(name, kind string) bool { return false }

// ReadSecret returns a user's secret of a kind verbatim.
func (s *Store) ReadSecret(name, kind string) (string, error) {
	return "", errStubUserSecrets
}

// DeleteSecret removes a user's secret of a kind.
func (s *Store) DeleteSecret(name, kind string) error { return errStubUserSecrets }
