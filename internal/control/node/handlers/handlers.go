package handlers

import (
	"context"

	api "vpn/gen/control"
	common "vpn/gen/vpn/common"
	"vpn/internal/control/node/model"
	"vpn/internal/control/node/service"
)

type Nodes struct {
	api.UnimplementedNodeServiceServer
	Service service.Service
}

func (h Nodes) AddNode(ctx context.Context, req *api.AddNodeRequest) (*api.Node, error) {
	n, err := h.Service.Create(ctx, req.GetName(), req.GetAddress(), req.GetDescription())
	if err != nil {
		return nil, err
	}

	return nodeProto(n), nil
}

func (h Nodes) ModifyNode(ctx context.Context, req *api.ModifyNodeRequest) (*api.Node, error) {
	n, err := h.Service.Modify(ctx, req.GetId(), req.GetOptions())
	if err != nil {
		return nil, err
	}

	return nodeProto(n), nil
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
		resp.Nodes = append(resp.Nodes, nodeProto(n))
	}

	return resp, nil
}

func (h Nodes) GetNodeInbounds(ctx context.Context, req *api.GetNodeInboundsRequest) (*api.GetNodeInboundsResponse, error) {
	inbounds, err := h.Service.Inbounds(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}

	resp := &api.GetNodeInboundsResponse{Inbounds: make([]*common.Inbound, 0, len(inbounds))}
	for _, inbound := range inbounds {
		if result := inboundProto(inbound); result != nil {
			resp.Inbounds = append(resp.Inbounds, result)
		}
	}

	return resp, nil
}

func (h Nodes) AddNodeHysteria2Inbound(ctx context.Context, req *api.AddNodeHysteria2InboundRequest) (*api.AddNodeHysteria2InboundResponse, error) {
	created, err := h.Service.AddHysteria2Inbound(ctx, req.GetNodeId(), req.GetPort())
	if err != nil {
		return nil, err
	}

	return &api.AddNodeHysteria2InboundResponse{Inbound: inboundProto(created)}, nil
}

func (h Nodes) AddNodeVlessInbound(ctx context.Context, req *api.AddNodeVlessInboundRequest) (*api.AddNodeVlessInboundResponse, error) {
	t := req.GetTransport()

	created, err := h.Service.AddVlessInbound(ctx, req.GetNodeId(), req.GetPort(), req.GetServerName(), model.VlessTransportOptions{
		Type: model.VlessTransportType(t.GetType()),
		Host: t.GetHost(),
		Path: t.GetPath(),
	})
	if err != nil {
		return nil, err
	}

	return &api.AddNodeVlessInboundResponse{Inbound: inboundProto(created)}, nil
}

func (h Nodes) DeleteNodeInbound(ctx context.Context, req *api.DeleteNodeInboundRequest) (*api.DeleteNodeInboundResponse, error) {
	err := h.Service.DeleteInbound(ctx, req.GetNodeId(), req.GetTag())
	return &api.DeleteNodeInboundResponse{}, err
}

func (h Nodes) GetNodeRuleSets(ctx context.Context, req *api.GetNodeRuleSetsRequest) (*api.GetNodeRuleSetsResponse, error) {
	rules, err := h.Service.RuleSets(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}

	return &api.GetNodeRuleSetsResponse{RuleSets: rules}, nil
}

func (h Nodes) AddNodeRuleSet(ctx context.Context, req *api.AddNodeRuleSetRequest) (*api.AddNodeRuleSetResponse, error) {
	err := h.Service.AddRuleSet(ctx, req.GetNodeId(), req.GetName(), req.GetUrl())
	return &api.AddNodeRuleSetResponse{}, err
}

func (h Nodes) RemoveNodeRuleSet(ctx context.Context, req *api.RemoveNodeRuleSetRequest) (*api.RemoveNodeRuleSetResponse, error) {
	err := h.Service.RemoveRuleSet(ctx, req.GetNodeId(), req.GetName())
	return &api.RemoveNodeRuleSetResponse{}, err
}

func (h Nodes) ModifyNodeRuleSet(ctx context.Context, req *api.ModifyNodeRuleSetRequest) (*api.ModifyNodeRuleSetResponse, error) {
	err := h.Service.ModifyRuleSet(ctx, req.GetNodeId(), req.GetName(), req.GetUrl())
	return &api.ModifyNodeRuleSetResponse{}, err
}

func (h Nodes) AddNodeRouteRule(ctx context.Context, req *api.AddNodeRouteRuleRequest) (*api.AddNodeRouteRuleResponse, error) {
	err := h.Service.AddRouteRule(ctx, req.GetNodeId(), model.RouteRule{
		User:     req.User,
		RuleSet:  req.RuleSet,
		Action:   model.RouteAction(req.GetAction()),
		Outbound: req.GetOutbound(),
	})

	return &api.AddNodeRouteRuleResponse{}, err
}

func (h Nodes) GetNodeRouteRules(ctx context.Context, req *api.GetNodeRouteRulesRequest) (*api.GetNodeRouteRulesResponse, error) {
	rules, err := h.Service.RouteRules(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}

	return &api.GetNodeRouteRulesResponse{Rules: rules}, nil
}

func (h Nodes) RemoveNodeRouteRule(ctx context.Context, req *api.RemoveNodeRouteRuleRequest) (*api.RemoveNodeRouteRuleResponse, error) {
	err := h.Service.RemoveRouteRule(ctx, req.GetNodeId(), req.GetIndex())
	return &api.RemoveNodeRouteRuleResponse{}, err
}

func (h Nodes) GetNodeOutbounds(ctx context.Context, req *api.GetNodeOutboundsRequest) (*api.GetNodeOutboundsResponse, error) {
	outbounds, err := h.Service.Outbounds(ctx, req.GetNodeId())
	if err != nil {
		return nil, err
	}

	return &api.GetNodeOutboundsResponse{Outbounds: outbounds}, nil
}

func (h Nodes) AddNodeOutbound(ctx context.Context, req *api.AddNodeOutboundRequest) (*api.AddNodeOutboundResponse, error) {
	err := h.Service.AddOutbound(ctx, req.GetNodeId(), req.GetOutbound())
	return &api.AddNodeOutboundResponse{}, err
}

func (h Nodes) RemoveNodeOutbound(ctx context.Context, req *api.RemoveNodeOutboundRequest) (*api.RemoveNodeOutboundResponse, error) {
	err := h.Service.RemoveOutbound(ctx, req.GetNodeId(), req.GetTag())
	return &api.RemoveNodeOutboundResponse{}, err
}

func (h Nodes) ModifyNodeOutbound(ctx context.Context, req *api.ModifyNodeOutboundRequest) (*api.ModifyNodeOutboundResponse, error) {
	err := h.Service.ModifyOutbound(ctx, req.GetNodeId(), req.GetTag(), req.GetOptions())
	return &api.ModifyNodeOutboundResponse{}, err
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

func inboundProto(inbound model.Inbound) *common.Inbound {
	if h := inbound.Hysteria2; h != nil {
		return &common.Inbound{
			Tag:     inbound.Tag,
			Port:    h.Port,
			Address: h.Address,
			Details: &common.Inbound_Hysteria2{
				Hysteria2: &common.Hysteria2Inbound{
					ServerName: h.TLS.ServerName,
					Insecure:   h.TLS.Insecure,
					PinSha256:  h.TLS.PinSHA256,
				},
			},
		}
	}

	if v := inbound.Vless; v != nil {
		return &common.Inbound{
			Tag:     inbound.Tag,
			Port:    v.Port,
			Address: v.Address,
			Details: &common.Inbound_Vless{
				Vless: &common.VlessInbound{
					Transport: &common.VlessTransport{
						Type:        common.VlessTransportType(v.Transport.Type),
						ServiceName: v.Transport.ServiceName,
						Host:        v.Transport.Host,
						Path:        v.Transport.Path,
					},
					ServerName: v.ServerName,
					PublicKey:  v.PublicKey,
					ShortId:    v.ShortID,
				},
			},
		}
	}

	return nil
}
