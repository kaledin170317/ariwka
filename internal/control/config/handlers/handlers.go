package handlers

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	api "vpn/gen/control"
	"vpn/internal/control/config/service"
	"vpn/internal/control/happ"
	usercontext "vpn/internal/control/user/context"
)

type Configs struct {
	api.UnimplementedConfigServiceServer
	Service service.Service
}

func (h Configs) CreateMyProfile(context.Context, *api.CreateMyProfileRequest) (*api.CreateProfileResponse, error) {
	return nil, status.Error(codes.FailedPrecondition, "create a device config with a network mode instead")
}

func (h Configs) CreateUserProfile(context.Context, *api.CreateUserProfileRequest) (*api.CreateProfileResponse, error) {
	return nil, status.Error(codes.FailedPrecondition, "create a device config with a network mode instead")
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

func (h Configs) CreateMyDevice(ctx context.Context, req *api.CreateDeviceRequest) (*api.Device, error) {
	u, _ := usercontext.User(ctx)

	device, err := h.Service.CreateDevice(ctx, u.ID, req.GetName(), req.GetDefaultNetwork())
	if err != nil {
		return nil, err
	}

	url, err := happ.DeviceURL(device.ID)
	if err != nil {
		return nil, err
	}

	return &api.Device{Id: device.ID, Name: device.Name, Url: url, DefaultNetwork: device.DefaultNetwork}, nil
}

func (h Configs) ListMyDevices(ctx context.Context, _ *api.ListDevicesRequest) (*api.ListDevicesResponse, error) {
	u, _ := usercontext.User(ctx)

	devices, err := h.Service.ListDevices(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	resp := &api.ListDevicesResponse{Devices: make([]*api.Device, 0, len(devices))}
	for _, device := range devices {
		url, urlErr := happ.DeviceURL(device.ID)
		if urlErr != nil {
			return nil, urlErr
		}

		resp.Devices = append(resp.Devices, &api.Device{Id: device.ID, Name: device.Name, Url: url, Hwid: device.HWID, DefaultNetwork: device.DefaultNetwork})
	}

	return resp, nil
}

func (h Configs) DeleteMyDevice(ctx context.Context, req *api.DeleteDeviceRequest) (*api.DeleteDeviceResponse, error) {
	u, _ := usercontext.User(ctx)
	err := h.Service.DeleteDevice(ctx, u.ID, req.GetId())

	return &api.DeleteDeviceResponse{}, err
}
