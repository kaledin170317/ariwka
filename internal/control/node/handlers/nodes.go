package handlers

import (
	"context"

	api "vpn/gen/control"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/service"
)

type Nodes struct {
	api.UnimplementedNodeServiceServer
	Service service.Service
}

func (h Nodes) List(ctx context.Context, _ *api.ListNodesRequest) (*api.ListNodesResponse, error) {
	nodes, err := h.Service.List(ctx)
	if err != nil {
		return nil, err
	}

	resp := &api.ListNodesResponse{Nodes: make([]*api.Node, 0, len(nodes))}
	for _, n := range nodes {
		resp.Nodes = append(resp.Nodes, nodeProto(n))
	}

	return resp, nil
}

func (h Nodes) Create(ctx context.Context, req *api.CreateNodeRequest) (*api.Node, error) {
	n, err := h.Service.Create(ctx, req.GetName(), req.GetAddress(), req.GetDescription())
	if err != nil {
		return nil, err
	}

	return nodeProto(n), nil
}

func (h Nodes) Delete(ctx context.Context, req *api.DeleteNodeRequest) (*api.DeleteNodeResponse, error) {
	err := h.Service.Delete(ctx, req.GetId())
	return &api.DeleteNodeResponse{}, err
}

func (h Nodes) Update(ctx context.Context, req *api.UpdateNodeRequest) (*api.Node, error) {
	n, err := h.Service.Modify(ctx, req.GetId(), req.GetOptions())
	if err != nil {
		return nil, err
	}

	return nodeProto(n), nil
}

func nodeProto(n model.Node) *api.Node {
	return &api.Node{
		Id:          n.ID,
		Name:        n.Name,
		Address:     n.Address,
		Description: n.Description,
		Country:     n.Country,
	}
}
