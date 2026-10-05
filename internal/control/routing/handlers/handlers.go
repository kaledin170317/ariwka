package handlers

import (
	"context"
	"errors"

	api "vpn/gen/control"
	"vpn/internal/control/routing/model"
	"vpn/internal/control/routing/service"
)

type Handler struct {
	api.UnimplementedRoutingServiceServer
	Service service.Service
}

func (h Handler) Create(ctx context.Context, req *api.CreateRoutePolicyRequest) (*api.CreateRoutePolicyResponse, error) {
	if req.GetPolicy() == nil {
		return nil, errors.New("route policy required")
	}

	err := h.Service.Create(ctx, policyFromProto(req.GetPolicy()))

	return &api.CreateRoutePolicyResponse{}, err
}

func (h Handler) Update(ctx context.Context, req *api.UpdateRoutePolicyRequest) (*api.UpdateRoutePolicyResponse, error) {
	if req.GetPolicy() == nil {
		return nil, errors.New("route policy required")
	}

	err := h.Service.Update(ctx, policyFromProto(req.GetPolicy()))

	return &api.UpdateRoutePolicyResponse{}, err
}

func (h Handler) Delete(ctx context.Context, req *api.DeleteRoutePolicyRequest) (*api.DeleteRoutePolicyResponse, error) {
	err := h.Service.Delete(ctx, req.GetTag())
	return &api.DeleteRoutePolicyResponse{}, err
}

func (h Handler) List(ctx context.Context, _ *api.ListRoutePoliciesRequest) (*api.ListRoutePoliciesResponse, error) {
	policies, err := h.Service.List(ctx)
	if err != nil {
		return nil, err
	}

	resp := &api.ListRoutePoliciesResponse{Policies: make([]*api.RoutePolicy, 0, len(policies))}
	for _, p := range policies {
		resp.Policies = append(resp.Policies, &api.RoutePolicy{
			Tag:     p.Tag,
			RuleSet: p.RuleSet,
			Source: &api.NodeFilter{
				Country:    p.Source.Country,
				NotCountry: p.Source.NotCountry,
			},
			Target: &api.NodeFilter{
				Country:    p.Target.Country,
				NotCountry: p.Target.NotCountry,
			},
		})
	}

	return resp, nil
}

func policyFromProto(p *api.RoutePolicy) model.Policy {
	return model.Policy{
		Tag:     p.GetTag(),
		RuleSet: p.GetRuleSet(),
		Source: model.Filter{
			Country:    p.GetSource().GetCountry(),
			NotCountry: p.GetSource().GetNotCountry(),
		},
		Target: model.Filter{
			Country:    p.GetTarget().GetCountry(),
			NotCountry: p.GetTarget().GetNotCountry(),
		},
	}
}
