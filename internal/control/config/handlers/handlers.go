package handlers

import (
	"context"

	api "vpn/gen/control"
	"vpn/internal/control/config/service"
	"vpn/internal/control/happ"
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

func (h Configs) DeleteMyProfile(ctx context.Context, req *api.DeleteMyProfileRequest) (*api.DeleteMyProfileResponse, error) {
	u, _ := usercontext.User(ctx)
	err := h.Service.Delete(ctx, u.ID, req.GetId())

	return &api.DeleteMyProfileResponse{}, err
}

func (h Configs) DeleteUserProfile(ctx context.Context, req *api.DeleteUserProfileRequest) (*api.DeleteUserProfileResponse, error) {
	err := h.Service.Delete(ctx, req.GetUserId(), req.GetId())
	return &api.DeleteUserProfileResponse{}, err
}

func (h Configs) GetConfig(ctx context.Context, _ *api.GetConfigRequest) (*api.GetConfigResponse, error) {
	u, _ := usercontext.User(ctx)

	profiles, err := h.Service.GetProfiles(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	resp := &api.GetConfigResponse{
		Connections: make([]string, 0, len(profiles)),
		Profiles:    make([]*api.ProfileConnection, 0, len(profiles)),
	}
	for _, profile := range profiles {
		resp.Connections = append(resp.Connections, profile.Link)
		resp.Profiles = append(resp.Profiles, &api.ProfileConnection{
			Id:         profile.ID,
			Connection: profile.Link,
		})
	}

	return resp, nil
}

func (h Configs) GetProfile(ctx context.Context, req *api.GetProfileRequest) (*api.GetProfileResponse, error) {
	u, _ := usercontext.User(ctx)
	link, err := h.Service.GetProfile(ctx, u.ID, req.GetId())

	return &api.GetProfileResponse{Connection: link}, err
}

func (h Configs) GetSubscriptionLink(ctx context.Context, _ *api.GetSubscriptionLinkRequest) (*api.GetSubscriptionLinkResponse, error) {
	u, _ := usercontext.User(ctx)

	url, err := happ.URL(u.ID)
	if err != nil {
		return nil, err
	}

	return &api.GetSubscriptionLinkResponse{Url: url}, nil
}

func (h Configs) GetDefaultSubscriptionLink(ctx context.Context, _ *api.GetSubscriptionLinkRequest) (*api.GetSubscriptionLinkResponse, error) {
	u, _ := usercontext.User(ctx)

	url, err := happ.DefaultURL(u.ID)
	if err != nil {
		return nil, err
	}

	return &api.GetSubscriptionLinkResponse{Url: url}, nil
}
