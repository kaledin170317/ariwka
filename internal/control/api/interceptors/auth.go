package interceptors

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	api "vpn/gen/control"
	"vpn/internal/control/user/model"
	"vpn/pkg/auth"
)

type userKey struct{}

func User(ctx context.Context) (model.User, bool) {
	u, ok := ctx.Value(userKey{}).(model.User)
	return u, ok
}

type role int

const (
	public role = iota
	user
	admin
)

var methods = map[string]role{
	api.AuthService_Login_FullMethodName:    public,
	api.AuthService_Register_FullMethodName: public,
	api.UserService_GetUsers_FullMethodName: admin,
}

type Users interface {
	Get(context.Context, string) (model.User, error)
}

type Auth struct {
	JWT   auth.JWT
	Users Users
}

func (a Auth) Intercept(ctx context.Context, req any, info *grpc.UnaryServerInfo, next grpc.UnaryHandler) (any, error) {
	access, ok := methods[info.FullMethod]
	if !ok {
		access = user
	}
	if access == public {
		return next(ctx, req)
	}

	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get("authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return nil, errors.New("valid bearer token required")
	}

	id, err := a.JWT.Verify(strings.TrimPrefix(values[0], "Bearer "))
	if err != nil {
		return nil, err
	}

	u, err := a.Users.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	if access == admin && u.Role != "admin" {
		return nil, errors.New("admin access required")
	}

	return next(context.WithValue(ctx, userKey{}, u), req)
}
