package handlers

import (
	"context"

	api "vpn/gen/node"
	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/model/inbound"
	"vpn/internal/node/sing-box/service"
)

type VPNHandler struct {
	api.UnimplementedVPNServiceServer

	service service.SingBoxService
}

func NewVPNHandler(service service.SingBoxService) *VPNHandler {
	return &VPNHandler{service: service}
}

func (c *VPNHandler) AddRuleSet(ctx context.Context, in *api.AddRuleSetRequest) (*api.AddRuleSetResponse, error) {
	return &api.AddRuleSetResponse{}, c.service.AddRemoteRule(in.Name)
}

func (c *VPNHandler) GetRuleSets(ctx context.Context, in *api.GetRuleSetsRequest) (*api.GetRuleSetsResponse, error) {
	ruleSets := c.service.GetRuleSets()

	return &api.GetRuleSetsResponse{RuleSets: ruleSets}, nil
}

func (c *VPNHandler) AddRouteRule(ctx context.Context, in *api.AddRouteRuleRequest) (*api.AddRouteRuleResponse, error) {
	return &api.AddRouteRuleResponse{}, c.service.AddRoute(in.User, in.RuleSet, config.RouteAction(in.Action))
}

func (c *VPNHandler) GetInbounds(ctx context.Context, in *api.GetInboundsRequest) (*api.GetInboundsResponse, error) {
	inbounds, err := c.service.GetInbounds()
	if err != nil {
		return nil, err
	}

	resp := &api.GetInboundsResponse{
		Inbounds: make([]*api.InboundConnection, 0, len(inbounds)),
	}

	for _, connection := range inbounds {
		switch connection.Type {
		case inbound.InboundTypeHysteria2:
			users := make([]*api.Hysteria2User, 0, len(connection.Users))

			for _, user := range connection.Users {
				users = append(users, &api.Hysteria2User{
					Name:     user.Name,
					Password: user.Password,
				})
			}

			resp.Inbounds = append(resp.Inbounds, &api.InboundConnection{
				Connection: &api.InboundConnection_Hysteria2{
					Hysteria2: &api.Hysteria2Connection{
						Tag:     connection.Tag,
						Address: connection.GetHost(),
						Port:    connection.Port,
						Tls: &api.ClientTLS{
							Enabled:    connection.TLS.Enabled,
							ServerName: connection.TLS.ServerName,
							Insecure:   connection.TLS.Insecure,
							PinSha256:  connection.TLS.PinSHA256,
						},
						Users: users,
					},
				},
			})
		}
	}

	return resp, nil
}

func (c *VPNHandler) AddInbound(ctx context.Context, in *api.AddInboundRequest) (*api.AddInboundResponse, error) {
	return &api.AddInboundResponse{}, c.service.AddInbound(in.GetTag(), inbound.InboundType(in.GetType()))
}

func (c *VPNHandler) AddUser(ctx context.Context, in *api.AddUserRequest) (*api.AddUserResponse, error) {
	err := c.service.AddUser(in.GetConfigId(), in.GetInbound())
	return &api.AddUserResponse{}, err
}

func (c *VPNHandler) RemoveUser(ctx context.Context, in *api.RemoveUserRequest) (*api.RemoveUserResponse, error) {
	err := c.service.RemoveUser(in.GetConfigId(), in.GetInbound())
	return &api.RemoveUserResponse{}, err
}
