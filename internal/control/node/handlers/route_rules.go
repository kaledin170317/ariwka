package handlers

import (
	"context"

	api "vpn/gen/control"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/service"
)

type RouteRules struct {
	api.UnimplementedNodeRouteRuleServiceServer
	Service service.RouteRuleService
}

func (h RouteRules) List(ctx context.Context, req *api.ListNodeRouteRulesRequest) (*api.ListNodeRouteRulesResponse, error) {
	rules, err := h.Service.RouteRules(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}

	return &api.ListNodeRouteRulesResponse{Rules: rules}, nil
}

func (h RouteRules) Create(ctx context.Context, req *api.CreateNodeRouteRuleRequest) (*api.CreateNodeRouteRuleResponse, error) {
	err := h.Service.AddRouteRule(ctx, req.GetNodeId(), model.RouteRule{
		User:     req.User,
		RuleSet:  req.RuleSet,
		Action:   model.RouteAction(req.GetAction()),
		Outbound: req.GetOutbound(),
	})

	return &api.CreateNodeRouteRuleResponse{}, err
}

func (h RouteRules) Delete(ctx context.Context, req *api.DeleteNodeRouteRuleRequest) (*api.DeleteNodeRouteRuleResponse, error) {
	err := h.Service.DeleteRouteRule(ctx, req.GetNodeId(), req.GetIndex())
	return &api.DeleteNodeRouteRuleResponse{}, err
}
