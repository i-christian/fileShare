package validator

import "time"

type ApiKey struct {
	KeyName string    `json:"key_name"`
	Expires time.Time `json:"expires_at,omitzero"`
	Scope   []string  `json:"scope"`
}

func ValidateAPIKeyLogin(v *Validator, key *ApiKey) {
	v.Check(len(key.Scope) > 0, "scope", "must be provided")
	v.Check(len(key.KeyName) > 2 && len(key.KeyName) <= 50, "key_name", "must be between 3 and 50 characters long")
	if !key.Expires.IsZero() {
		v.Check(IsValidTimeFormat(time.RFC3339, key.Expires.Format(time.RFC3339)) && key.Expires.After(time.Now()),
			"expires_at", "must be a valid value and not in the past")
	}
}
