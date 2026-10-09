// Package authctx는 인증된 요청의 사용자·세션을 컨텍스트로 전달한다.
package authctx

import "context"

type key int

const (
	userKey key = iota
	sessionKey
)

func With(ctx context.Context, userID, sessionID string) context.Context {
	ctx = context.WithValue(ctx, userKey, userID)
	return context.WithValue(ctx, sessionKey, sessionID)
}

func UserID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userKey).(string)
	return v, ok && v != ""
}

func SessionID(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(sessionKey).(string)
	return v, ok && v != ""
}
