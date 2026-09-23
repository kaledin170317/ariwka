package handlers

import (
	"context"

	api "vpn/gen/control"
	"vpn/internal/control/user/service"
)

type Users struct {
	api.UnimplementedUserServiceServer
	Service service.Service
}

func (h Users) GetUsers(ctx context.Context, _ *api.GetUsersRequest) (*api.GetUsersResponse, error) {
	users, err := h.Service.List(ctx)
	if err != nil {
		return nil, err
	}
	resp := &api.GetUsersResponse{Users: make([]*api.User, 0, len(users))}
	for _, u := range users {
		resp.Users = append(resp.Users, &api.User{Id: u.ID, Username: u.Username})
	}
	return resp, nil
}
