package usercontext

import (
	"context"

	"vpn/internal/control/user/model"
)

type userKey struct{}

func WithUser(ctx context.Context, u model.User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

func User(ctx context.Context) (model.User, bool) {
	u, ok := ctx.Value(userKey{}).(model.User)
	return u, ok
}
