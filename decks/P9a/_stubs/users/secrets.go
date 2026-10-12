package users

import "errors"

var (
	ErrNoSecret    = errors.New("no secret")
	ErrEmptySecret = errors.New("empty secret")
)

var SecretKinds = []string{"anthropic-key", "github-token"}

func (u *Users) SecretPath(name, kind string) string {
	panic("stub SecretPath")
}

func (u *Users) PutSecret(name, kind, value string) error {
	panic("stub PutSecret")
}

func (u *Users) HasSecret(name, kind string) bool {
	panic("stub HasSecret")
}

func (u *Users) ReadSecret(name, kind string) (string, error) {
	panic("stub ReadSecret")
}

func (u *Users) DeleteSecret(name, kind string) error {
	panic("stub DeleteSecret")
}
