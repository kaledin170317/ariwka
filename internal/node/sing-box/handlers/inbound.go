package handlers

import (
	"context"
	"errors"

	api "vpn/gen/node"
	common "vpn/gen/vpn/common"
	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/model/inbound"
)

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
			resp.Inbounds = append(resp.Inbounds, entry)
		case *inbound.Vless:
			users := make([]string, 0, len(c.Users))

			details := make([]*api.VlessUser, 0, len(c.Users))
			for _, user := range c.Users {
				users = append(users, user.UUID)
				details = append(details, &api.VlessUser{Tag: user.Tag, Uuid: user.UUID})
			}

			entry.VlessUsers = users
			entry.VlessUserDetails = details
			resp.Inbounds = append(resp.Inbounds, entry)
		}
	}

	return resp, nil
}

func (c *VPNHandler) AddInbound(ctx context.Context, in *api.AddInboundRequest) (*api.AddInboundResponse, error) {
	var (
		connection inbound.Connection
		err        error
	)

	switch details := in.GetDetails().(type) {
	case *api.AddInboundRequest_Hysteria2:
		connection, err = c.service.AddHysteria2Inbound(in.GetTag(), in.GetPort())
	case *api.AddInboundRequest_Vless:
		v := details.Vless
		t := v.GetTransport()
		connection, err = c.service.AddVlessInbound(
			in.GetTag(),
			in.GetPort(),
			v.GetServerName(),
			inbound.VlessTransportOptions{
				Type: inbound.VlessTransportType(t.GetType()),
				Host: t.GetHost(),
				Path: t.GetPath(),
			},
			v.GetPrivateKey(),
			v.GetShortId(),
			v.GetServiceName(),
		)
	default:
		return nil, errors.New("inbound details required")
	}

	if err != nil {
		return nil, err
	}

	return &api.AddInboundResponse{Inbound: inboundProto(connection)}, nil
}

func (c *VPNHandler) RemoveInbound(ctx context.Context, in *api.RemoveInboundRequest) (*api.RemoveInboundResponse, error) {
	return &api.RemoveInboundResponse{}, c.service.RemoveInbound(in.GetTag())
}

func (c *VPNHandler) ModifyInbound(ctx context.Context, in *api.ModifyInboundRequest) (*api.ModifyInboundResponse, error) {
	options := make([]config.InboundOption, 0, len(in.GetOptions()))
	for _, option := range in.GetOptions() {
		switch value := option.GetValue().(type) {
		case *api.InboundOption_Port:
			options = append(options, config.PortOption(value.Port))
		case *api.InboundOption_ServerName:
			options = append(options, config.ServerNameOption(value.ServerName))
		case *api.InboundOption_Transport:
			t := value.Transport
			options = append(options, config.TransportOption{
				Type:        inbound.VlessTransportType(t.GetType()).String(),
				ServiceName: t.GetServiceName(),
				Host:        t.GetHost(),
				Path:        t.GetPath(),
			})
		case *api.InboundOption_PrivateKey:
			options = append(options, config.PrivateKeyOption(value.PrivateKey))
		case *api.InboundOption_ShortId:
			options = append(options, config.ShortIDOption(value.ShortId))
		default:
			return nil, errors.New("unknown inbound option")
		}
	}

	err := c.service.ModifyInbound(in.GetTag(), options)

	return &api.ModifyInboundResponse{}, err
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
