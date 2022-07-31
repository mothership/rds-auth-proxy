package discovery

import "errors"

// Possible errors returned by disovery clients
var (
	// ErrTargetNotFound shoould be returned when a target lookup by host
	// or name fails
	ErrTargetNotFound = errors.New("target not found")
)
