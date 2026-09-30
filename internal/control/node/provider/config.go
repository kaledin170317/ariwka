package provider

import (
	"context"
	"errors"
	"net"
	"slices"

	"vpn/internal/control/node/model"
	"vpn/pkg/link"
)

func (n *Node) Config(ctx context.Context, configID, tag string) (string, error) {
	inbounds, err := n.Inbounds(ctx)
	if err != nil {
		return "", err
	}

	for _, inbound := range inbounds {
		if inbound.Tag == tag {
			return n.Link(inbound, configID)
		}
	}

	return "", errors.New("config is missing on node " + n.Name)
}

func (n *Node) Link(inbound model.Inbound, userTag string) (string, error) {
	if v := inbound.Vless; v != nil && slices.Contains(v.Users, userTag) {
		return (link.Vless{
			Tag:        inbound.Tag,
			Address:    n.inboundAddress(v.Address),
			Port:       v.Port,
			UUID:       userTag,
			ServerName: v.ServerName,
			PublicKey:  v.PublicKey,
			ShortID:    v.ShortID,
			Transport: link.Transport{
				Type:        v.Transport.Type.String(),
				ServiceName: v.Transport.ServiceName,
				Host:        v.Transport.Host,
				Path:        v.Transport.Path,
			},
		}).String(), nil
	}

	if h := inbound.Hysteria2; h != nil {
		for _, user := range h.Users {
			if user.Name == userTag {
				return (link.Hysteria2{
					Tag:        inbound.Tag,
					Address:    n.inboundAddress(h.Address),
					Port:       h.Port,
					Password:   user.Password,
					ServerName: h.TLS.ServerName,
					Insecure:   h.TLS.Insecure,
					PinSHA256:  h.TLS.PinSHA256,
				}).String(), nil
			}
		}
	}

	return "", errors.New("config is missing on node " + n.Name)
}

func (n *Node) inboundAddress(address string) string {
	if address == "" {
		address, _, _ = net.SplitHostPort(n.Address)
	}

	return address
}
