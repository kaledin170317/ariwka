package handlers

import (
	"context"

	api "vpn/gen/control"
	"vpn/internal/control/node/outbound/service"
)

type Outbounds struct {
	api.UnimplementedNodeOutboundServiceServer
	Service service.OutboundService
}

func (h Outbounds) List(ctx context.Context, req *api.ListNodeOutboundsRequest) (*api.ListNodeOutboundsResponse, error) {
	outbounds, err := h.Service.Outbounds(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}

	return &api.ListNodeOutboundsResponse{Outbounds: outbounds}, nil
}

func (h Outbounds) Create(ctx context.Context, req *api.CreateNodeOutboundRequest) (*api.CreateNodeOutboundResponse, error) {
	err := h.Service.AddOutbound(ctx, req.GetNodeId(), req.GetOutbound())
	return &api.CreateNodeOutboundResponse{}, err
}

func (h Outbounds) Delete(ctx context.Context, req *api.DeleteNodeOutboundRequest) (*api.DeleteNodeOutboundResponse, error) {
	err := h.Service.DeleteOutbound(ctx, req.GetNodeId(), req.GetTag())
	return &api.DeleteNodeOutboundResponse{}, err
}

func (h Outbounds) Update(ctx context.Context, req *api.UpdateNodeOutboundRequest) (*api.UpdateNodeOutboundResponse, error) {
	err := h.Service.ModifyOutbound(ctx, req.GetNodeId(), req.GetTag(), req.GetOptions())
	return &api.UpdateNodeOutboundResponse{}, err
}
