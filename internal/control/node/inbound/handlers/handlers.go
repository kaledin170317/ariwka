package handlers

import (
	"context"
	"errors"

	api "vpn/gen/control"
	common "vpn/gen/vpn/common"
	"vpn/internal/control/node/inbound/service"
	"vpn/internal/control/node/model"
)

type Inbounds struct {
	api.UnimplementedNodeInboundServiceServer
	Service service.InboundService
}

func (h Inbounds) List(ctx context.Context, _ *api.ListNodeInboundsRequest) (*api.ListNodeInboundsResponse, error) {
	inbounds, err := h.Service.ListInbounds(ctx)
	if err != nil {
		return nil, err
	}

	resp := &api.ListNodeInboundsResponse{Inbounds: make([]*api.NodeInbound, 0, len(inbounds))}
	for _, item := range inbounds {
		if result := inboundProto(item.Inbound); result != nil {
			result.Network = common.InboundNetwork(item.Inbound.Network)
			resp.Inbounds = append(resp.Inbounds, &api.NodeInbound{NodeId: item.NodeID, Inbound: result})
		}
	}

	return resp, nil
}

func (h Inbounds) Create(ctx context.Context, req *api.CreateNodeInboundRequest) (*api.CreateNodeInboundResponse, error) {
	var created model.Inbound

	var err error

	switch details := req.GetDetails().(type) {
	case *api.CreateNodeInboundRequest_Hysteria2:
		created, err = h.Service.AddHysteria2Inbound(ctx, req.GetNodeId(), req.GetPort())
	case *api.CreateNodeInboundRequest_Vless:
		t := details.Vless.GetTransport()
		created, err = h.Service.AddVlessInbound(ctx, req.GetNodeId(), req.GetPort(), details.Vless.GetServerName(), model.VlessTransportOptions{
			Type: model.VlessTransportType(t.GetType()),
			Host: t.GetHost(),
			Path: t.GetPath(),
		})
	default:
		return nil, errors.New("inbound type required")
	}

	if err != nil {
		return nil, err
	}

	return &api.CreateNodeInboundResponse{Inbound: inboundProto(created)}, nil
}

func (h Inbounds) Delete(ctx context.Context, req *api.DeleteNodeInboundRequest) (*api.DeleteNodeInboundResponse, error) {
	err := h.Service.DeleteInbound(ctx, req.GetNodeId(), req.GetTag())
	return &api.DeleteNodeInboundResponse{}, err
}

func inboundProto(inbound model.Inbound) *common.Inbound {
	if h := inbound.Hysteria2; h != nil {
		return &common.Inbound{
			Tag:     inbound.Tag,
			Port:    h.Port,
			Address: h.Address,
			Details: &common.Inbound_Hysteria2{
				Hysteria2: &common.Hysteria2Inbound{
					ServerName: h.TLS.ServerName,
					Insecure:   h.TLS.Insecure,
					PinSha256:  h.TLS.PinSHA256,
				},
			},
		}
	}

	if v := inbound.Vless; v != nil {
		return &common.Inbound{
			Tag:     inbound.Tag,
			Port:    v.Port,
			Address: v.Address,
			Details: &common.Inbound_Vless{
				Vless: &common.VlessInbound{
					Transport: &common.VlessTransport{
						Type:        common.VlessTransportType(v.Transport.Type),
						ServiceName: v.Transport.ServiceName,
						Host:        v.Transport.Host,
						Path:        v.Transport.Path,
					},
					ServerName: v.ServerName,
					PublicKey:  v.PublicKey,
					ShortId:    v.ShortID,
				},
			},
		}
	}

	return nil
}
