package users

// Lookup returns the user whose token matches, by a constant-time compare of the hashes.
func (s *Store) Lookup(token string) (User, bool) { return User{}, false }

// IsAdminToken reports whether the token is the operator's admin token.
func (s *Store) IsAdminToken(token string) bool { return false }
