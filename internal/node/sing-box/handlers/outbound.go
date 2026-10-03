package handlers

import (
	"context"
	"errors"

	api "vpn/gen/node"
	common "vpn/gen/vpn/common"
	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/service"
)

type OutboundHandler struct {
	api.UnimplementedOutboundServiceServer
	Service *service.SingBoxService
}

func (c *OutboundHandler) GetOutbounds(ctx context.Context, in *api.GetOutboundsRequest) (*api.GetOutboundsResponse, error) {
	items := c.Service.GetOutbounds()
	resp := &api.GetOutboundsResponse{Outbounds: make([]*api.Outbound, 0, len(items))}

	for _, item := range items {
		out := &api.Outbound{Tag: item.Tag}
		switch item.Type {
		case config.OutboundTypeDirect:
			out.Details = &api.Outbound_Direct{Direct: &api.DirectOutbound{}}
		case config.OutboundTypeBlock:
			out.Details = &api.Outbound_Block{Block: &api.BlockOutbound{}}
		case config.OutboundTypeVless:
			out.Details = &api.Outbound_Vless{Vless: &api.VlessOutbound{
				Server: item.Server,
				Port:   item.ServerPort,
				Uuid:   item.UUID,
				Inbound: &common.VlessInbound{
					ServerName: item.ServerName,
					PublicKey:  item.PublicKey,
					ShortId:    item.ShortID,
					Transport: &common.VlessTransport{
						Type:        transportType(item.Transport.Type),
						ServiceName: item.Transport.ServiceName,
						Host:        item.Transport.Host,
						Path:        item.Transport.Path,
					},
				},
			}}
		case config.OutboundTypeHysteria2:
			out.Details = &api.Outbound_Hysteria2{Hysteria2: &api.Hysteria2Outbound{
				Server:   item.Server,
				Port:     item.ServerPort,
				Password: item.Password,
				Inbound: &common.Hysteria2Inbound{
					ServerName: item.ServerName,
					Insecure:   item.Insecure,
				},
			}}
		}

		resp.Outbounds = append(resp.Outbounds, out)
	}

	return resp, nil
}

func (c *OutboundHandler) AddOutbound(ctx context.Context, in *api.AddOutboundRequest) (*api.AddOutboundResponse, error) {
	value := in.GetOutbound()
	if value == nil {
		return nil, errors.New("outbound required")
	}

	item := config.Outbound{Tag: value.GetTag()}
	switch details := value.GetDetails().(type) {
	case *api.Outbound_Direct:
		item.Type = config.OutboundTypeDirect
	case *api.Outbound_Block:
		item.Type = config.OutboundTypeBlock
	case *api.Outbound_Vless:
		value := details.Vless

		settings, err := vlessSettings(value.GetInbound())
		if err != nil {
			return nil, err
		}

		item.Type = config.OutboundTypeVless
		item.Server = value.GetServer()
		item.ServerPort = value.GetPort()
		item.UUID = value.GetUuid()
		item.ServerName = settings.ServerName
		item.PublicKey = settings.PublicKey
		item.ShortID = settings.ShortID
		item.Transport = settings.Transport
	case *api.Outbound_Hysteria2:
		value := details.Hysteria2

		settings, err := hysteria2Settings(value.GetInbound())
		if err != nil {
			return nil, err
		}

		item.Type = config.OutboundTypeHysteria2
		item.Server = value.GetServer()
		item.ServerPort = value.GetPort()
		item.Password = value.GetPassword()
		item.ServerName = settings.ServerName
		item.Insecure = settings.Insecure
	default:
		return nil, errors.New("outbound details required")
	}

	err := c.Service.AddOutbound(item)

	return &api.AddOutboundResponse{}, err
}

func (c *OutboundHandler) RemoveOutbound(ctx context.Context, in *api.RemoveOutboundRequest) (*api.RemoveOutboundResponse, error) {
	err := c.Service.RemoveOutbound(in.GetTag())
	return &api.RemoveOutboundResponse{}, err
}

func (c *OutboundHandler) ModifyOutbound(ctx context.Context, in *api.ModifyOutboundRequest) (*api.ModifyOutboundResponse, error) {
	options := make([]config.OutboundOption, 0, len(in.GetOptions()))
	for _, option := range in.GetOptions() {
		switch value := option.GetValue().(type) {
		case *api.OutboundOption_Server:
			options = append(options, config.OutboundServerOption(value.Server))
		case *api.OutboundOption_Port:
			options = append(options, config.OutboundPortOption(value.Port))
		case *api.OutboundOption_Uuid:
			options = append(options, config.OutboundUUIDOption(value.Uuid))
		case *api.OutboundOption_Password:
			options = append(options, config.OutboundPasswordOption(value.Password))
		case *api.OutboundOption_Vless:
			settings, err := vlessSettings(value.Vless)
			if err != nil {
				return nil, err
			}

			options = append(options, settings)
		case *api.OutboundOption_Hysteria2:
			settings, err := hysteria2Settings(value.Hysteria2)
			if err != nil {
				return nil, err
			}

			options = append(options, settings)
		default:
			return nil, errors.New("unknown outbound option")
		}
	}

	err := c.Service.ModifyOutbound(in.GetTag(), options)

	return &api.ModifyOutboundResponse{}, err
}

func vlessSettings(value *common.VlessInbound) (config.OutboundVlessOption, error) {
	if value == nil {
		return config.OutboundVlessOption{}, errors.New("VLESS inbound settings required")
	}

	return config.OutboundVlessOption{
		ServerName: value.GetServerName(),
		PublicKey:  value.GetPublicKey(),
		ShortID:    value.GetShortId(),
		Transport:  outboundTransport(value.GetTransport()),
	}, nil
}

func hysteria2Settings(value *common.Hysteria2Inbound) (config.OutboundHysteria2Option, error) {
	if value == nil || value.GetPinSha256() != "" {
		return config.OutboundHysteria2Option{}, errors.New("Hysteria2 inbound settings required without pin_sha256")
	}

	return config.OutboundHysteria2Option{
		ServerName: value.GetServerName(),
		Insecure:   value.GetInsecure(),
	}, nil
}

func outboundTransport(value *common.VlessTransport) config.OutboundTransport {
	return config.OutboundTransport{
		Type:        transportName(value.GetType()),
		ServiceName: value.GetServiceName(),
		Host:        value.GetHost(),
		Path:        value.GetPath(),
	}
}

func transportName(value common.VlessTransportType) string {
	switch value {
	case common.VlessTransportType_VLESS_TRANSPORT_TYPE_GRPC:
		return "grpc"
	case common.VlessTransportType_VLESS_TRANSPORT_TYPE_HTTP:
		return "http"
	default:
		return ""
	}
}

func transportType(value string) common.VlessTransportType {
	if value == "grpc" {
		return common.VlessTransportType_VLESS_TRANSPORT_TYPE_GRPC
	}

	if value == "http" {
		return common.VlessTransportType_VLESS_TRANSPORT_TYPE_HTTP
	}

	return common.VlessTransportType_VLESS_TRANSPORT_TYPE_UNSPECIFIED
}
