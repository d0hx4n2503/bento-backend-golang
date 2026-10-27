package models

// Principal is the authenticated identity carried by a session token.
type Principal struct {
	Subject string `json:"sub"`
	Email   string `json:"email,omitempty"`
	Role    string `json:"role,omitempty"`
}
