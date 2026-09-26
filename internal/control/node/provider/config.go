package provider

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"

	"vpn/internal/control/node/model"
)

func (n *Node) Config(ctx context.Context, configID, tag string) (string, error) {
	inbounds, err := n.Inbounds(ctx)
	if err != nil {
		return "", err
	}
	for _, inbound := range inbounds {
		if inbound.Tag != tag || inbound.Hysteria2 == nil {
			continue
		}
		for _, user := range inbound.Hysteria2.Users {
			if user.Name == configID {
				return n.hysteria2Link(tag, inbound.Hysteria2, user), nil
			}
		}
	}
	return "", errors.New("config is missing on node " + n.Name)
}

func (n *Node) hysteria2Link(tag string, h *model.Hysteria2, user model.InboundUser) string {
	address := h.Address
	if address == "" {
		address, _, _ = net.SplitHostPort(n.Address)
	}
	params := url.Values{}
	if h.TLS.Insecure || h.TLS.PinSHA256 != "" {
		params.Set("insecure", "1")
	}
	if h.TLS.PinSHA256 != "" {
		params.Set("pinSHA256", h.TLS.PinSHA256)
	}
	if h.TLS.ServerName != "" {
		params.Set("sni", h.TLS.ServerName)
	}
	return (&url.URL{
		Scheme: "hysteria2", User: url.User(user.Password),
		Host: net.JoinHostPort(address, strconv.Itoa(int(h.Port))),
		Path: "/", RawQuery: params.Encode(), Fragment: tag,
	}).String()
}
