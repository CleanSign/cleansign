// Package clock makes time injectable so that time-dependent rules are easy to test.
package clock

import "time"

type Clock interface {
	Now() time.Time
}

type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }

// Fixed is used in tests to freeze time or move it manually.
type Fixed struct{ T time.Time }

func (f *Fixed) Now() time.Time { return f.T }

func (f *Fixed) Advance(d time.Duration) { f.T = f.T.Add(d) }
