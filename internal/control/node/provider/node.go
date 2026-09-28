package provider

import (
	"context"
	"errors"

	api "vpn/gen/node"
	common "vpn/gen/vpn/common/v1"
	"vpn/internal/control/node/model"
)

func (n *Node) AddUser(ctx context.Context, configID, inbound string) error {
	_, err := n.client.AddUser(ctx, &api.AddUserRequest{ConfigId: configID, Inbound: inbound})
	return err
}

func (n *Node) RemoveUser(ctx context.Context, configID, inbound string) error {
	_, err := n.client.RemoveUser(ctx, &api.RemoveUserRequest{ConfigId: configID, Inbound: inbound})
	return err
}

func (n *Node) AddInbound(ctx context.Context, tag string, kind common.InboundType, port int32) error {
	_, err := n.client.AddInbound(ctx, &api.AddInboundRequest{Tag: tag, Type: kind, Port: port})
	return err
}

func (n *Node) AddRuleSet(ctx context.Context, name string) error {
	_, err := n.client.AddRuleSet(ctx, &api.AddRuleSetRequest{Name: name})
	return err
}

func (n *Node) RuleSets(ctx context.Context) ([]string, error) {
	resp, err := n.client.GetRuleSets(ctx, &api.GetRuleSetsRequest{})
	if err != nil {
		return nil, err
	}
	return resp.GetRuleSets(), nil
}

func (n *Node) AddRouteRule(ctx context.Context, rule model.RouteRule) error {
	if !rule.Action.IsValid() {
		return errors.New("unsupported route action")
	}
	_, err := n.client.AddRouteRule(ctx, &api.AddRouteRuleRequest{
		User: rule.User, RuleSet: rule.RuleSet, Action: api.RouteAction(rule.Action),
	})
	return err
}

func (n *Node) Inbounds(ctx context.Context) ([]model.Inbound, error) {
	resp, err := n.client.GetInbounds(ctx, &api.GetInboundsRequest{})
	if err != nil {
		return nil, err
	}
	inbounds := make([]model.Inbound, 0, len(resp.GetInbounds()))
	for _, entry := range resp.GetInbounds() {
		h := entry.GetHysteria2()
		if h == nil {
			continue
		}
		users := make([]model.InboundUser, 0, len(h.GetUsers()))
		for _, user := range h.GetUsers() {
			users = append(users, model.InboundUser{Name: user.GetName(), Password: user.GetPassword()})
		}
		inbounds = append(inbounds, model.Inbound{
			Tag: h.GetTag(),
			Hysteria2: &model.Hysteria2{
				Address: h.GetAddress(), Port: h.GetPort(), Users: users,
				TLS: model.TLS{
					ServerName: h.GetTls().GetServerName(), Insecure: h.GetTls().GetInsecure(),
					PinSHA256: h.GetTls().GetPinSha256(),
				},
			},
		})
	}
	return inbounds, nil
}
