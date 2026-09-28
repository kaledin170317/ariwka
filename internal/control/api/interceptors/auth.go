package interceptors

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	api "vpn/gen/control"
	usercontext "vpn/internal/control/user/context"
	"vpn/internal/control/user/model"
	"vpn/pkg/auth"
)

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(public, user, admin)
type AccessLevel int32

var methods = map[string]AccessLevel{
	api.AuthService_Login_FullMethodName:               AccessLevelPublic,
	api.AuthService_Register_FullMethodName:            AccessLevelPublic,
	api.AuthService_LoginTelegramUser_FullMethodName:   AccessLevelAdmin,
	api.UserService_GetUsers_FullMethodName:            AccessLevelAdmin,
	api.NodeService_GetNodes_FullMethodName:            AccessLevelUser,
	api.NodeService_AddNode_FullMethodName:             AccessLevelAdmin,
	api.NodeService_DeleteNode_FullMethodName:          AccessLevelAdmin,
	api.NodeService_GetNodeInbounds_FullMethodName:     AccessLevelUser,
	api.NodeService_AddNodeInbound_FullMethodName:      AccessLevelAdmin,
	api.ConfigService_CreateMyProfile_FullMethodName:   AccessLevelUser,
	api.ConfigService_CreateUserProfile_FullMethodName: AccessLevelAdmin,
	api.ConfigService_GetConfig_FullMethodName:         AccessLevelUser,
	api.ConfigService_GetProfile_FullMethodName:        AccessLevelUser,
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
		return nil, errors.New("access level not configured")
	}
	if access == AccessLevelPublic {
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

	if access == AccessLevelAdmin && u.Role != "admin" {
		return nil, errors.New("admin access required")
	}
	return next(usercontext.WithUser(ctx, u), req)
}
