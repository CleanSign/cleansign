// Package authctx passes the authenticated request's user and session through the context.
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
