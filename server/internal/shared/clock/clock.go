// Package clock은 시간을 주입할 수 있게 해서 시간에 따라 달라지는 규칙을 테스트하기 쉽게 한다.
package clock

import "time"

type Clock interface {
	Now() time.Time
}

type Real struct{}

func (Real) Now() time.Time { return time.Now().UTC() }

// Fixed는 테스트에서 시간을 고정하거나 직접 움직일 때 쓴다.
type Fixed struct{ T time.Time }

func (f *Fixed) Now() time.Time { return f.T }

func (f *Fixed) Advance(d time.Duration) { f.T = f.T.Add(d) }
