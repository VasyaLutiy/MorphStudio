package users

type Caller struct {
	User  string
	Admin bool
}

func (c Caller) Sees(owner string) bool                             { return false }
func (r *Registry) Lookup(adminToken, bearer string) (Caller, bool) { return Caller{}, false }
