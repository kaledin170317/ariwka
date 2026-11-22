package handlers

import (
	"context"
	"errors"

	"google.golang.org/protobuf/proto"

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

	policy := policyFromProto(req.GetPolicy())
	if req.GetPolicy().ApplyToNodes == nil || req.GetPolicy().ApplyToClients == nil {
		policies, err := h.Service.List(ctx)
		if err != nil {
			return nil, err
		}
		for _, current := range policies {
			if current.Tag != policy.Tag {
				continue
			}
			if req.GetPolicy().ApplyToNodes == nil {
				policy.ApplyToNodes = current.ApplyToNodes
			}
			if req.GetPolicy().ApplyToClients == nil {
				policy.ApplyToClients = current.ApplyToClients
			}
			break
		}
	}
	err := h.Service.Update(ctx, policy)

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
			Tag:            p.Tag,
			RuleFile:       p.RuleFile,
			RuleCode:       p.RuleCode,
			ApplyToNodes:   proto.Bool(p.ApplyToNodes),
			ApplyToClients: proto.Bool(p.ApplyToClients),
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
	applyToNodes := true
	if p.ApplyToNodes != nil {
		applyToNodes = p.GetApplyToNodes()
	}
	return model.Policy{
		Tag:            p.GetTag(),
		RuleFile:       p.GetRuleFile(),
		RuleCode:       p.GetRuleCode(),
		ApplyToNodes:   applyToNodes,
		ApplyToClients: p.GetApplyToClients(),
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
