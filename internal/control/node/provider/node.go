package provider

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/curve25519"

	api "vpn/gen/node"
	common "vpn/gen/vpn/common"
	"vpn/internal/control/node/model"
)

func (n *Node) AddHysteria2User(ctx context.Context, inboundTag string, user model.InboundUser) (model.Inbound, error) {
	return n.addUser(ctx, &api.AddUserRequest{
		InboundTag: inboundTag,
		Credentials: &api.AddUserRequest_Hysteria2{
			Hysteria2: &api.Hysteria2User{Name: user.Name, Password: user.Password},
		},
	})
}

func (n *Node) AddVlessUser(ctx context.Context, inboundTag string, user model.VlessUser) (model.Inbound, error) {
	return n.addUser(ctx, &api.AddUserRequest{
		InboundTag: inboundTag,
		Credentials: &api.AddUserRequest_Vless{
			Vless: &api.VlessUser{Tag: user.Tag, Uuid: user.UUID},
		},
	})
}

func (n *Node) RemoveUser(ctx context.Context, userTag, inboundTag string) error {
	_, err := n.users.RemoveUser(ctx, &api.RemoveUserRequest{
		Tag:        userTag,
		InboundTag: inboundTag,
	})

	return err
}

func (n *Node) AddHysteria2Inbound(ctx context.Context, tag string, port int32) (model.Inbound, error) {
	resp, err := n.inbounds.AddInbound(ctx, &api.AddInboundRequest{
		Tag:  tag,
		Port: port,
		Details: &api.AddInboundRequest_Hysteria2{
			Hysteria2: &api.Hysteria2InboundOptions{},
		},
	})
	if err != nil {
		return model.Inbound{}, err
	}

	return inboundModel(resp.GetInbound()), nil
}

func (n *Node) AddVlessInbound(ctx context.Context, config model.InboundConfig) (model.Inbound, error) {
	resp, err := n.inbounds.AddInbound(ctx, &api.AddInboundRequest{
		Tag:  config.Tag,
		Port: config.Port,
		Details: &api.AddInboundRequest_Vless{
			Vless: &api.VlessInboundOptions{
				ServerName:  config.ServerName,
				Transport:   transportProto(config.Transport),
				PrivateKey:  config.PrivateKey,
				ShortId:     config.ShortID,
				ServiceName: config.ServiceName,
			},
		},
	})
	if err != nil {
		return model.Inbound{}, err
	}

	return inboundModel(resp.GetInbound()), nil
}

func (n *Node) RemoveInbound(ctx context.Context, tag string) error {
	_, err := n.inbounds.RemoveInbound(ctx, &api.RemoveInboundRequest{Tag: tag})
	return err
}

func (n *Node) ModifyInbound(ctx context.Context, config model.InboundConfig, current model.Inbound) error {
	options := make([]*api.InboundOption, 0)

	port := int32(0)
	if current.Hysteria2 != nil {
		port = current.Hysteria2.Port
	} else if current.Vless != nil {
		port = current.Vless.Port
	}

	if port != config.Port {
		options = append(options, &api.InboundOption{Value: &api.InboundOption_Port{Port: config.Port}})
	}

	if config.Type == model.InboundTypeVless {
		v := current.Vless
		if v == nil {
			return fmt.Errorf("inbound %s cannot be modified", config.Tag)
		}

		if config.PrivateKey == "" {
			return nil // An inbound created before the migration has no stored REALITY key.
		}

		if v.ServerName != config.ServerName {
			options = append(options, &api.InboundOption{Value: &api.InboundOption_ServerName{ServerName: config.ServerName}})
		}

		if v.Transport.Type != config.Transport.Type || v.Transport.Host != config.Transport.Host || v.Transport.Path != config.Transport.Path || v.Transport.ServiceName != config.ServiceName {
			options = append(options, &api.InboundOption{Value: &api.InboundOption_Transport{Transport: &common.VlessTransport{
				Type:        common.VlessTransportType(config.Transport.Type),
				ServiceName: config.ServiceName,
				Host:        config.Transport.Host,
				Path:        config.Transport.Path,
			}}})
		}

		key, err := base64.RawURLEncoding.DecodeString(config.PrivateKey)
		if err != nil {
			return err
		}

		public, err := curve25519.X25519(key, curve25519.Basepoint)
		if err != nil {
			return err
		}

		if v.PublicKey != base64.RawURLEncoding.EncodeToString(public) {
			options = append(options, &api.InboundOption{Value: &api.InboundOption_PrivateKey{PrivateKey: config.PrivateKey}})
		}

		if v.ShortID != config.ShortID {
			options = append(options, &api.InboundOption{Value: &api.InboundOption_ShortId{ShortId: config.ShortID}})
		}
	} else if current.Hysteria2 == nil {
		return fmt.Errorf("inbound %s has wrong type", config.Tag)
	}

	if len(options) == 0 {
		return nil
	}

	_, err := n.inbounds.ModifyInbound(ctx, &api.ModifyInboundRequest{Tag: config.Tag, Options: options})

	return err
}

func (n *Node) ModifyUser(ctx context.Context, inboundTag, tag string, option *api.UserOption) error {
	_, err := n.users.ModifyUser(ctx, &api.ModifyUserRequest{
		InboundTag: inboundTag,
		Tag:        tag,
		Options:    []*api.UserOption{option},
	})

	return err
}

func (n *Node) ModifyHysteria2User(ctx context.Context, inboundTag, tag, password string) error {
	return n.ModifyUser(ctx, inboundTag, tag, &api.UserOption{Value: &api.UserOption_Password{Password: password}})
}

func (n *Node) ModifyVlessUser(ctx context.Context, inboundTag, tag, uuid string) error {
	return n.ModifyUser(ctx, inboundTag, tag, &api.UserOption{Value: &api.UserOption_Uuid{Uuid: uuid}})
}

func (n *Node) AddRuleSetConfig(ctx context.Context, name, url, format string, interval uint32) error {
	_, err := n.routes.AddRuleSet(ctx, &api.AddRuleSetRequest{
		Name: name, Url: url, Format: format, UpdateIntervalSeconds: interval,
	})

	return err
}

func (n *Node) RuleSets(ctx context.Context) ([]*api.RuleSet, error) {
	resp, err := n.routes.GetRuleSets(ctx, &api.GetRuleSetsRequest{})
	if err != nil {
		return nil, err
	}

	return resp.GetRuleSets(), nil
}

func (n *Node) RemoveRuleSet(ctx context.Context, name string) error {
	_, err := n.routes.RemoveRuleSet(ctx, &api.RemoveRuleSetRequest{Name: name})
	return err
}

func (n *Node) ModifyRuleSetConfig(ctx context.Context, name, url, format string, interval uint32) error {
	_, err := n.routes.ModifyRuleSet(ctx, &api.ModifyRuleSetRequest{
		Name: name, Url: url, Format: format, UpdateIntervalSeconds: interval,
	})

	return err
}

func (n *Node) RouteRules(ctx context.Context) ([]*api.RouteRule, error) {
	resp, err := n.routes.GetRouteRules(ctx, &api.GetRouteRulesRequest{})
	if err != nil {
		return nil, err
	}

	return resp.GetRules(), nil
}

func (n *Node) SyncDNSRoutes(ctx context.Context, routes []*api.DNSRoute, managedRuleSets []string) error {
	_, err := n.routes.SyncDNSRoutes(ctx, &api.SyncDNSRoutesRequest{
		Routes:          routes,
		ManagedRuleSets: managedRuleSets,
	})

	return err
}

func (n *Node) RemoveRouteRule(ctx context.Context, index uint32) error {
	_, err := n.routes.RemoveRouteRule(ctx, &api.RemoveRouteRuleRequest{Index: index})
	return err
}

func (n *Node) Outbounds(ctx context.Context) ([]*api.Outbound, error) {
	resp, err := n.outbounds.GetOutbounds(ctx, &api.GetOutboundsRequest{})
	if err != nil {
		return nil, err
	}

	return resp.GetOutbounds(), nil
}

func (n *Node) AddOutbound(ctx context.Context, outbound *api.Outbound) error {
	_, err := n.outbounds.AddOutbound(ctx, &api.AddOutboundRequest{Outbound: outbound})
	return err
}

func (n *Node) RemoveOutbound(ctx context.Context, tag string) error {
	_, err := n.outbounds.RemoveOutbound(ctx, &api.RemoveOutboundRequest{Tag: tag})
	return err
}

func (n *Node) ModifyOutbound(ctx context.Context, tag string, options []*api.OutboundOption) error {
	_, err := n.outbounds.ModifyOutbound(ctx, &api.ModifyOutboundRequest{Tag: tag, Options: options})
	return err
}

func (n *Node) AddRouteRule(ctx context.Context, rule model.RouteRule) error {
	if !rule.Action.IsValid() {
		return errors.New("unsupported route action")
	}

	_, err := n.routes.AddRouteRule(ctx, &api.AddRouteRuleRequest{
		User:     rule.User,
		RuleSet:  rule.RuleSet,
		Action:   api.RouteAction(rule.Action),
		Outbound: rule.Outbound,
	})

	return err
}

func (n *Node) Inbounds(ctx context.Context) ([]model.Inbound, error) {
	resp, err := n.inbounds.GetInbounds(ctx, &api.GetInboundsRequest{})
	if err != nil {
		return nil, err
	}

	inbounds := make([]model.Inbound, 0, len(resp.GetInbounds()))
	for _, entry := range resp.GetInbounds() {
		connection := inboundModel(entry.GetInbound())
		if connection.Vless != nil {
			connection.Vless.Users = entry.GetVlessUsers()
			for _, user := range entry.GetVlessUserDetails() {
				connection.Vless.UserDetails = append(connection.Vless.UserDetails, model.VlessUser{
					Tag:  user.GetTag(),
					UUID: user.GetUuid(),
				})
			}

			inbounds = append(inbounds, connection)
		} else if connection.Hysteria2 != nil {
			users := make([]model.InboundUser, 0, len(entry.GetHysteria2Users()))
			for _, user := range entry.GetHysteria2Users() {
				users = append(users, model.InboundUser{
					Name:     user.GetName(),
					Password: user.GetPassword(),
				})
			}

			connection.Hysteria2.Users = users
			inbounds = append(inbounds, connection)
		}
	}

	return inbounds, nil
}

func (n *Node) addUser(ctx context.Context, req *api.AddUserRequest) (model.Inbound, error) {
	resp, err := n.users.AddUser(ctx, req)
	if err != nil {
		return model.Inbound{}, err
	}

	created := inboundModel(resp.GetInbound())

	if h := created.Hysteria2; h != nil && resp.GetHysteria2() != nil {
		h.Users = []model.InboundUser{
			{Name: resp.GetHysteria2().GetName(), Password: resp.GetHysteria2().GetPassword()},
		}
	} else if v := created.Vless; v != nil && resp.GetVless() != nil {
		v.Users = []string{resp.GetVless().GetUuid()}
	}

	if created.Tag == "" {
		return model.Inbound{}, errors.New("node returned no inbound")
	}

	return created, nil
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
