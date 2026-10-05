package repo

import (
	"context"

	api "vpn/gen/node"
	"vpn/internal/control/node/provider"
)

type Nodes struct{ Provider *provider.Provider }

func (r Nodes) List(ctx context.Context, nodeID string) ([]*api.Outbound, error) {
	node, err := r.Provider.Get(nodeID)
	if err != nil {
		return nil, err
	}

	return node.Outbounds(ctx)
}

func (r Nodes) Create(ctx context.Context, nodeID string, outbound *api.Outbound) error {
	node, err := r.Provider.Get(nodeID)
	if err != nil {
		return err
	}

	return node.AddOutbound(ctx, outbound)
}

func (r Nodes) Delete(ctx context.Context, nodeID, tag string) error {
	node, err := r.Provider.Get(nodeID)
	if err != nil {
		return err
	}

	return node.RemoveOutbound(ctx, tag)
}

func (r Nodes) Update(ctx context.Context, nodeID, tag string, options []*api.OutboundOption) error {
	node, err := r.Provider.Get(nodeID)
	if err != nil {
		return err
	}

	return node.ModifyOutbound(ctx, tag, options)
}
