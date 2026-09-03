package authz

// Principal represents an authenticated user or service principal
// extracted from JWT claims
type Principal struct {
	// Subject is the "sub" claim from JWT (unique identifier)
	Subject string `json:"subject"`

	// Roles is the "roles" claim from JWT (array of role strings)
	Roles []string `json:"roles,omitempty"`

	// Attributes are custom claims extracted from JWT
	// excluding standard claims (sub, roles, exp, iat, etc.)
	Attributes map[string]string `json:"attributes,omitempty"`
}
