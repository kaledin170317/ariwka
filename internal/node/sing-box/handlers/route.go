package handlers

import (
	"context"

	api "vpn/gen/node"
	"vpn/internal/node/sing-box/config"
)

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
