// Package errs는 계층을 가로지르는 공용 오류를 담는다. HTTP 상태로의 변환은 server 계층이 한다.
package errs

import "errors"

var (
	// ErrUnauthorized는 로그인하지 않았거나 토큰이 유효하지 않을 때 쓴다.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrNotFound는 대상이 없거나 본인 소유가 아닐 때 쓴다. 남의 것은 존재 여부도 알려 주지 않는다.
	ErrNotFound = errors.New("not found")
)
