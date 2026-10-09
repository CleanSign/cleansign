// Package ids creates new identifiers.
package ids

import "github.com/google/uuid"

func New() string { return uuid.NewString() }
