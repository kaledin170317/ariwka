package handlers

import (
	"context"

	api "vpn/gen/control"
	"vpn/internal/control/config/service"
	usercontext "vpn/internal/control/user/context"
)

type Configs struct {
	api.UnimplementedConfigServiceServer
	Service service.Service
}

func (h Configs) CreateMyProfile(ctx context.Context, req *api.CreateMyProfileRequest) (*api.CreateProfileResponse, error) {
	u, _ := usercontext.User(ctx)
	id, connection, err := h.Service.Create(ctx, u.ID, req.GetNodeId(), req.GetInbound())

	return &api.CreateProfileResponse{
		Id:         id,
		Connection: connection,
	}, err
}

func (h Configs) CreateUserProfile(ctx context.Context, req *api.CreateUserProfileRequest) (*api.CreateProfileResponse, error) {
	id, connection, err := h.Service.Create(ctx, req.GetUserId(), req.GetNodeId(), req.GetInbound())

	return &api.CreateProfileResponse{
		Id:         id,
		Connection: connection,
	}, err
}

func (h Configs) GetConfig(ctx context.Context, _ *api.GetConfigRequest) (*api.GetConfigResponse, error) {
	u, _ := usercontext.User(ctx)

	connections, err := h.Service.Get(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	return &api.GetConfigResponse{Connections: connections}, nil
}

func (h Configs) GetProfile(ctx context.Context, req *api.GetProfileRequest) (*api.GetProfileResponse, error) {
	u, _ := usercontext.User(ctx)
	link, err := h.Service.GetProfile(ctx, u.ID, req.GetId())

	return &api.GetProfileResponse{Connection: link}, err
}
