package handlers

import (
	"context"
	"errors"

	api "vpn/gen/node"
	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/service"
)

type UserHandler struct {
	api.UnimplementedUserServiceServer
	Service *service.SingBoxService
}

func (c *UserHandler) AddUser(ctx context.Context, in *api.AddUserRequest) (*api.AddUserResponse, error) {
	var user config.User

	switch value := in.GetCredentials().(type) {
	case *api.AddUserRequest_Hysteria2:
		user = config.Hysteria2User{
			Name:     value.Hysteria2.GetName(),
			Password: value.Hysteria2.GetPassword(),
		}
	case *api.AddUserRequest_Vless:
		user = config.VlessUser{
			Tag:  value.Vless.GetTag(),
			UUID: value.Vless.GetUuid(),
		}
	default:
		return nil, errors.New("user credentials required")
	}

	connection, err := c.Service.AddUser(in.GetInboundTag(), user)
	if err != nil {
		return nil, err
	}

	resp := &api.AddUserResponse{Inbound: inboundProto(connection)}

	switch value := user.(type) {
	case config.Hysteria2User:
		resp.Credentials = &api.AddUserResponse_Hysteria2{
			Hysteria2: &api.Hysteria2User{Name: value.Name, Password: value.Password},
		}
	case config.VlessUser:
		resp.Credentials = &api.AddUserResponse_Vless{
			Vless: &api.VlessUser{Tag: value.Tag, Uuid: value.UUID},
		}
	}

	return resp, nil
}

func (c *UserHandler) RemoveUser(ctx context.Context, in *api.RemoveUserRequest) (*api.RemoveUserResponse, error) {
	err := c.Service.RemoveUser(in.GetTag(), in.GetInboundTag())
	return &api.RemoveUserResponse{}, err
}

func (c *UserHandler) ModifyUser(ctx context.Context, in *api.ModifyUserRequest) (*api.ModifyUserResponse, error) {
	options := make([]config.UserOption, 0, len(in.GetOptions()))
	for _, option := range in.GetOptions() {
		switch value := option.GetValue().(type) {
		case *api.UserOption_Password:
			options = append(options, config.PasswordOption(value.Password))
		case *api.UserOption_Uuid:
			options = append(options, config.UUIDOption(value.Uuid))
		default:
			return nil, errors.New("unknown user option")
		}
	}

	err := c.Service.ModifyUser(in.GetInboundTag(), in.GetTag(), options)

	return &api.ModifyUserResponse{}, err
}
