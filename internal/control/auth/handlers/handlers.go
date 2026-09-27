package handlers

import (
	"context"

	api "vpn/gen/control"
	"vpn/internal/control/auth/service"
)

type Auth struct {
	api.UnimplementedAuthServiceServer
	Service service.Service
}

func (h Auth) Login(ctx context.Context, req *api.LoginRequest) (*api.LoginResponse, error) {
	token, err := h.Service.Login(ctx, req.GetUsername(), req.GetPassword())
	return &api.LoginResponse{Token: token}, err
}

func (h Auth) Register(ctx context.Context, req *api.RegisterRequest) (*api.User, error) {
	u, err := h.Service.Register(ctx, req.GetUsername(), req.GetPassword())
	if err != nil {
		return nil, err
	}
	return &api.User{Id: u.ID, Username: u.Username}, nil
}

func (h Auth) LoginTelegramUser(ctx context.Context, req *api.LoginTelegramUserRequest) (*api.LoginResponse, error) {
	token, err := h.Service.LoginTelegramUser(ctx, req.GetTelegramId())
	return &api.LoginResponse{Token: token}, err
}
