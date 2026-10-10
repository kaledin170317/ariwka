package provider

import (
	"context"
	"errors"
	"net"
	"slices"
	"strings"

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
	name := inbound.Tag
	if flag := countryFlag(n.Country); flag != "" {
		name = flag + " " + n.Name
	}

	if strings.HasPrefix(inbound.Tag, "default:") {
		name += " default"
	}

	if v := inbound.Vless; v != nil && slices.Contains(v.Users, userTag) {
		return (link.Vless{
			Tag:        name,
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
					Tag:        name,
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

func countryFlag(code string) string {
	code = strings.ToUpper(code)
	if len(code) != 2 || code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
		return ""
	}

	return string([]rune{
		'🇦' + rune(code[0]-'A'),
		'🇦' + rune(code[1]-'A'),
	})
}
