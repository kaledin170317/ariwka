package handlers

import (
	"context"
	api "vpn/gen/node"
	common "vpn/gen/vpn/common"
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
		entry := &api.InboundConnection{Inbound: inboundProto(connection)}
		switch c := connection.(type) {
		case *inbound.Hysteria2:
			users := make([]*api.Hysteria2User, 0, len(c.Users))
			for _, user := range c.Users {
				users = append(users, &api.Hysteria2User{
					Name:     user.Name,
					Password: user.Password,
				})
			}

			entry.Hysteria2Users = users
		case *inbound.Vless:
			users := make([]string, 0, len(c.Users))
			for _, user := range c.Users {
				users = append(users, user.UUID)
			}

			entry.VlessUsers = users
		default:
			continue
		}

		resp.Inbounds = append(resp.Inbounds, entry)
	}

	return resp, nil
}

func inboundProto(connection inbound.Connection) *common.Inbound {
	base := &common.Inbound{
		Tag:  connection.GetTag(),
		Port: connection.GetPort(),
	}
	switch c := connection.(type) {
	case *inbound.Hysteria2:
		base.Address = c.Address
		base.Details = &common.Inbound_Hysteria2{
			Hysteria2: &common.Hysteria2Inbound{
				ServerName: c.TLS.ServerName,
				Insecure:   c.TLS.Insecure,
				PinSha256:  c.TLS.PinSHA256,
			},
		}
	case *inbound.Vless:
		base.Address = c.Address
		base.Details = &common.Inbound_Vless{
			Vless: &common.VlessInbound{
				ServerName: c.ServerName,
				PublicKey:  c.PublicKey,
				ShortId:    c.ShortID,
				Transport: &common.VlessTransport{
					Type:        common.VlessTransportType(c.Transport.Type),
					ServiceName: c.Transport.ServiceName,
					Host:        c.Transport.Host,
					Path:        c.Transport.Path,
				},
			},
		}
	}

	return base
}

func (c *VPNHandler) AddHysteria2Inbound(ctx context.Context, in *api.AddHysteria2InboundRequest) (*api.AddHysteria2InboundResponse, error) {
	connection, err := c.service.AddHysteria2Inbound(in.GetPort())
	if err != nil {
		return nil, err
	}

	return &api.AddHysteria2InboundResponse{Inbound: inboundProto(connection)}, nil
}

func (c *VPNHandler) AddVlessInbound(ctx context.Context, in *api.AddVlessInboundRequest) (*api.AddVlessInboundResponse, error) {
	t := in.GetTransport()

	connection, err := c.service.AddVlessInbound(in.GetPort(), in.GetServerName(), inbound.VlessTransportOptions{
		Type: inbound.VlessTransportType(t.GetType()),
		Host: t.GetHost(),
		Path: t.GetPath(),
	})
	if err != nil {
		return nil, err
	}

	return &api.AddVlessInboundResponse{Inbound: inboundProto(connection)}, nil
}

func (c *VPNHandler) RemoveInbound(ctx context.Context, in *api.RemoveInboundRequest) (*api.RemoveInboundResponse, error) {
	return &api.RemoveInboundResponse{}, c.service.RemoveInbound(in.GetTag())
}

func (c *VPNHandler) AddUser(ctx context.Context, in *api.AddUserRequest) (*api.AddUserResponse, error) {
	connection, tag, credential, err := c.service.AddUser(in.GetInboundTag())
	if err != nil {
		return nil, err
	}

	resp := &api.AddUserResponse{Inbound: inboundProto(connection)}
	switch connection.(type) {
	case *inbound.Hysteria2:
		resp.Credentials = &api.AddUserResponse_Hysteria2{
			Hysteria2: &api.Hysteria2User{Name: tag, Password: credential},
		}
	case *inbound.Vless:
		resp.Credentials = &api.AddUserResponse_Vless{
			Vless: &api.VlessUser{Tag: tag, Uuid: credential},
		}
	}

	return resp, nil
}

func (c *VPNHandler) RemoveUser(ctx context.Context, in *api.RemoveUserRequest) (*api.RemoveUserResponse, error) {
	err := c.service.RemoveUser(in.GetTag(), in.GetInboundTag())
	return &api.RemoveUserResponse{}, err
}
