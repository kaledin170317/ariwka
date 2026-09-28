package handlers

import (
	"context"

	api "vpn/gen/control"
	common "vpn/gen/vpn/common/v1"
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
	return &api.Node{Id: n.ID, Name: n.Name, Address: n.Address}, nil
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
		resp.Nodes = append(resp.Nodes, &api.Node{Id: n.ID, Name: n.Name, Address: n.Address})
	}
	return resp, nil
}

func (h Nodes) GetNodeInbounds(ctx context.Context, req *api.GetNodeInboundsRequest) (*api.GetNodeInboundsResponse, error) {
	inbounds, err := h.Service.Inbounds(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}
	resp := &api.GetNodeInboundsResponse{Inbounds: make([]*api.Inbound, 0, len(inbounds))}
	for _, inbound := range inbounds {
		if inbound.Hysteria2 != nil {
			resp.Inbounds = append(resp.Inbounds, &api.Inbound{
				Tag: inbound.Tag, Type: common.InboundType_INBOUND_TYPE_HYSTERIA2, Port: inbound.Hysteria2.Port,
			})
		}
	}
	return resp, nil
}

func (h Nodes) AddNodeInbound(ctx context.Context, req *api.AddNodeInboundRequest) (*api.AddNodeInboundResponse, error) {
	tag, err := h.Service.AddInbound(ctx, req.GetNodeId(), req.GetType(), req.GetPort())
	if err != nil {
		return nil, err
	}
	return &api.AddNodeInboundResponse{Tag: tag}, nil
}
