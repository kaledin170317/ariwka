package repo

import (
	"context"

	api "vpn/gen/node"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/provider"
)

type Nodes struct{ Provider *provider.Provider }

func (r Nodes) List(ctx context.Context, nodeID string) ([]*api.RouteRule, error) {
	node, err := r.Provider.Get(nodeID)
	if err != nil {
		return nil, err
	}

	return node.RouteRules(ctx)
}

func (r Nodes) Create(ctx context.Context, nodeID string, rule model.RouteRule) error {
	node, err := r.Provider.Get(nodeID)
	if err != nil {
		return err
	}

	return node.AddRouteRule(ctx, rule)
}

func (r Nodes) Delete(ctx context.Context, nodeID string, index uint32) error {
	node, err := r.Provider.Get(nodeID)
	if err != nil {
		return err
	}

	return node.RemoveRouteRule(ctx, index)
}
