package handlers

import (
	"context"

	api "vpn/gen/node"
	"vpn/internal/node/sing-box/config"
	"vpn/internal/node/sing-box/service"
)

type RouteHandler struct {
	api.UnimplementedRouteServiceServer
	Service *service.SingBoxService
}

func (c *RouteHandler) SyncDNSRoutes(ctx context.Context, in *api.SyncDNSRoutesRequest) (*api.SyncDNSRoutesResponse, error) {
	routes := make([]config.DNSRoute, 0, len(in.GetRoutes()))
	for _, route := range in.GetRoutes() {
		routes = append(routes, config.DNSRoute{
			RuleSet:  route.GetRuleSet(),
			Outbound: route.GetOutbound(),
		})
	}

	err := c.Service.SyncDNSRoutes(routes, in.GetManagedRuleSets())

	return &api.SyncDNSRoutesResponse{}, err
}

func (c *RouteHandler) AddRuleSet(ctx context.Context, in *api.AddRuleSetRequest) (*api.AddRuleSetResponse, error) {
	err := c.Service.AddRuleSet(in.GetName(), in.GetUrl(), in.GetFormat(), in.GetUpdateIntervalSeconds())
	return &api.AddRuleSetResponse{}, err
}

func (c *RouteHandler) GetRuleSets(ctx context.Context, in *api.GetRuleSetsRequest) (*api.GetRuleSetsResponse, error) {
	ruleSets := c.Service.GetRuleSets()

	resp := &api.GetRuleSetsResponse{RuleSets: make([]*api.RuleSet, 0, len(ruleSets))}
	for _, rule := range ruleSets {
		resp.RuleSets = append(resp.RuleSets, &api.RuleSet{
			Name:                  rule.Name,
			Url:                   rule.URL,
			Format:                rule.Format,
			UpdateIntervalSeconds: rule.UpdateIntervalSeconds,
		})
	}

	return resp, nil
}

func (c *RouteHandler) RemoveRuleSet(ctx context.Context, in *api.RemoveRuleSetRequest) (*api.RemoveRuleSetResponse, error) {
	err := c.Service.RemoveRuleSet(in.GetName())
	return &api.RemoveRuleSetResponse{}, err
}

func (c *RouteHandler) ModifyRuleSet(ctx context.Context, in *api.ModifyRuleSetRequest) (*api.ModifyRuleSetResponse, error) {
	err := c.Service.ModifyRuleSet(in.GetName(), in.GetUrl(), in.GetFormat(), in.GetUpdateIntervalSeconds())
	return &api.ModifyRuleSetResponse{}, err
}

func (c *RouteHandler) AddRouteRule(ctx context.Context, in *api.AddRouteRuleRequest) (*api.AddRouteRuleResponse, error) {
	err := c.Service.AddRoute(in.User, in.RuleSet, config.RouteAction(in.Action), in.GetOutbound())
	return &api.AddRouteRuleResponse{}, err
}

func (c *RouteHandler) GetRouteRules(ctx context.Context, in *api.GetRouteRulesRequest) (*api.GetRouteRulesResponse, error) {
	rules := c.Service.GetRouteRules()

	resp := &api.GetRouteRulesResponse{Rules: make([]*api.RouteRule, 0, len(rules))}
	for _, rule := range rules {
		resp.Rules = append(resp.Rules, &api.RouteRule{
			Users:    rule.Users,
			RuleSets: rule.RuleSets,
			Action:   api.RouteAction(rule.Action),
			Outbound: rule.Outbound,
		})
	}

	return resp, nil
}

func (c *RouteHandler) RemoveRouteRule(ctx context.Context, in *api.RemoveRouteRuleRequest) (*api.RemoveRouteRuleResponse, error) {
	err := c.Service.RemoveRouteRule(in.GetIndex())
	return &api.RemoveRouteRuleResponse{}, err
}
