package users

import "errors"

var ErrNoSecret = errors.New("users: no secret")
var ErrEmptySecret = errors.New("users: empty secret")

var SecretKinds = []string{"anthropic-key", "github-token"}

func (r *Registry) SecretPath(name, kind string) string      { return "" }
func (r *Registry) PutSecret(name, kind, value string) error { return errors.New("stub PutSecret") }
func (r *Registry) HasSecret(name, kind string) bool         { return false }
func (r *Registry) ReadSecret(name, kind string) (string, error) {
	return "", errors.New("stub ReadSecret")
}
func (r *Registry) DeleteSecret(name, kind string) error { return errors.New("stub DeleteSecret") }
