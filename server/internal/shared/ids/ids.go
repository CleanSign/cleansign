// Package ids는 새 식별자를 만든다.
package ids

import "github.com/google/uuid"

func New() string { return uuid.NewString() }
