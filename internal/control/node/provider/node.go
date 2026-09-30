package provider

import (
	"context"
	"errors"

	api "vpn/gen/node"
	common "vpn/gen/vpn/common"
	"vpn/internal/control/node/model"
)

func (n *Node) AddUser(ctx context.Context, inboundTag string) (model.Inbound, string, error) {
	resp, err := n.client.AddUser(ctx, &api.AddUserRequest{InboundTag: inboundTag})
	if err != nil {
		return model.Inbound{}, "", err
	}

	created := inboundModel(resp.GetInbound())

	var tag string

	if h := created.Hysteria2; h != nil && resp.GetHysteria2() != nil {
		tag = resp.GetHysteria2().GetName()
		h.Users = []model.InboundUser{
			{Name: tag, Password: resp.GetHysteria2().GetPassword()},
		}
	} else if v := created.Vless; v != nil && resp.GetVless() != nil {
		tag = resp.GetVless().GetTag()
		v.Users = []string{resp.GetVless().GetUuid()}
	}

	if tag == "" {
		return model.Inbound{}, "", errors.New("node returned no user tag")
	}

	return created, tag, nil
}

func (n *Node) RemoveUser(ctx context.Context, userTag, inboundTag string) error {
	_, err := n.client.RemoveUser(ctx, &api.RemoveUserRequest{
		Tag:        userTag,
		InboundTag: inboundTag,
	})

	return err
}

func (n *Node) AddHysteria2Inbound(ctx context.Context, port int32) (model.Inbound, error) {
	resp, err := n.client.AddHysteria2Inbound(ctx, &api.AddHysteria2InboundRequest{Port: port})
	if err != nil {
		return model.Inbound{}, err
	}

	return inboundModel(resp.GetInbound()), nil
}

func (n *Node) AddVlessInbound(ctx context.Context, port int32, serverName string, transport model.VlessTransportOptions) (model.Inbound, error) {
	resp, err := n.client.AddVlessInbound(ctx, &api.AddVlessInboundRequest{
		Port:       port,
		ServerName: serverName,
		Transport:  transportProto(transport),
	})
	if err != nil {
		return model.Inbound{}, err
	}

	return inboundModel(resp.GetInbound()), nil
}

func (n *Node) RemoveInbound(ctx context.Context, tag string) error {
	_, err := n.client.RemoveInbound(ctx, &api.RemoveInboundRequest{Tag: tag})
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
		User:    rule.User,
		RuleSet: rule.RuleSet,
		Action:  api.RouteAction(rule.Action),
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
		connection := inboundModel(entry.GetInbound())
		if connection.Vless != nil {
			connection.Vless.Users = entry.GetVlessUsers()
		} else if connection.Hysteria2 != nil {
			users := make([]model.InboundUser, 0, len(entry.GetHysteria2Users()))
			for _, user := range entry.GetHysteria2Users() {
				users = append(users, model.InboundUser{
					Name:     user.GetName(),
					Password: user.GetPassword(),
				})
			}

			connection.Hysteria2.Users = users
		} else {
			continue
		}

		inbounds = append(inbounds, connection)
	}

	return inbounds, nil
}

func inboundModel(base *common.Inbound) model.Inbound {
	if base == nil {
		return model.Inbound{}
	}

	connection := model.Inbound{Tag: base.GetTag()}
	if h := base.GetHysteria2(); h != nil {
		connection.Hysteria2 = &model.Hysteria2{
			Address: base.GetAddress(),
			Port:    base.GetPort(),
			TLS: model.TLS{
				ServerName: h.GetServerName(),
				Insecure:   h.GetInsecure(),
				PinSHA256:  h.GetPinSha256(),
			},
		}
	}

	if v := base.GetVless(); v != nil {
		connection.Vless = &model.Vless{
			Address:    base.GetAddress(),
			Port:       base.GetPort(),
			ServerName: v.GetServerName(),
			PublicKey:  v.GetPublicKey(),
			ShortID:    v.GetShortId(),
			Transport: model.VlessTransport{
				Type:        model.VlessTransportType(v.GetTransport().GetType()),
				ServiceName: v.GetTransport().GetServiceName(),
				Host:        v.GetTransport().GetHost(),
				Path:        v.GetTransport().GetPath(),
			},
		}
	}

	return connection
}

func transportProto(t model.VlessTransportOptions) *common.VlessTransportOptions {
	return &common.VlessTransportOptions{
		Type: common.VlessTransportType(t.Type),
		Host: t.Host,
		Path: t.Path,
	}
}
