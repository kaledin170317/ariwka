package handlers

import (
	"context"

	api "vpn/gen/control"
	common "vpn/gen/vpn/common"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/service"
)

type Nodes struct {
	api.UnimplementedNodeServiceServer
	Service service.Service
}

func (h Nodes) AddNode(ctx context.Context, req *api.AddNodeRequest) (*api.Node, error) {
	n, err := h.Service.Create(ctx, req.GetName(), req.GetAddress())
	if err != nil {
		return nil, err
	}

	return &api.Node{
		Id:      n.ID,
		Name:    n.Name,
		Address: n.Address,
	}, nil
}

func (h Nodes) DeleteNode(ctx context.Context, req *api.DeleteNodeRequest) (*api.DeleteNodeResponse, error) {
	err := h.Service.Delete(ctx, req.GetId())
	return &api.DeleteNodeResponse{}, err
}

func (h Nodes) GetNodes(ctx context.Context, _ *api.GetNodesRequest) (*api.GetNodesResponse, error) {
	nodes, err := h.Service.List(ctx)
	if err != nil {
		return nil, err
	}

	resp := &api.GetNodesResponse{Nodes: make([]*api.Node, 0, len(nodes))}
	for _, n := range nodes {
		resp.Nodes = append(resp.Nodes, &api.Node{
			Id:      n.ID,
			Name:    n.Name,
			Address: n.Address,
		})
	}

	return resp, nil
}

func (h Nodes) GetNodeInbounds(ctx context.Context, req *api.GetNodeInboundsRequest) (*api.GetNodeInboundsResponse, error) {
	inbounds, err := h.Service.Inbounds(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}

	resp := &api.GetNodeInboundsResponse{Inbounds: make([]*common.Inbound, 0, len(inbounds))}
	for _, inbound := range inbounds {
		if result := inboundProto(inbound); result != nil {
			resp.Inbounds = append(resp.Inbounds, result)
		}
	}

	return resp, nil
}

func (h Nodes) AddNodeHysteria2Inbound(ctx context.Context, req *api.AddNodeHysteria2InboundRequest) (*api.AddNodeHysteria2InboundResponse, error) {
	created, err := h.Service.AddHysteria2Inbound(ctx, req.GetNodeId(), req.GetPort())
	if err != nil {
		return nil, err
	}

	return &api.AddNodeHysteria2InboundResponse{Inbound: inboundProto(created)}, nil
}

func (h Nodes) AddNodeVlessInbound(ctx context.Context, req *api.AddNodeVlessInboundRequest) (*api.AddNodeVlessInboundResponse, error) {
	t := req.GetTransport()

	created, err := h.Service.AddVlessInbound(ctx, req.GetNodeId(), req.GetPort(), req.GetServerName(), model.VlessTransportOptions{
		Type: model.VlessTransportType(t.GetType()),
		Host: t.GetHost(),
		Path: t.GetPath(),
	})
	if err != nil {
		return nil, err
	}

	return &api.AddNodeVlessInboundResponse{Inbound: inboundProto(created)}, nil
}

func (h Nodes) DeleteNodeInbound(ctx context.Context, req *api.DeleteNodeInboundRequest) (*api.DeleteNodeInboundResponse, error) {
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
