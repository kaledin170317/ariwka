package model

import common "vpn/gen/vpn/common/v1"

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(reject=1, route, hijack-dns, sniff)
type RouteAction int32

type Node struct {
	ID      string
	Name    string
	Address string
}

type InboundConfig struct {
	Tag    string
	NodeID string
	Type   common.InboundType
	Port   int32
}

type Inbound struct {
	Tag       string
	Hysteria2 *Hysteria2
}

type Hysteria2 struct {
	Address string
	Port    int32
	TLS     TLS
	Users   []InboundUser
}

type TLS struct {
	ServerName string
	Insecure   bool
	PinSHA256  string
}

type InboundUser struct {
	Name     string
	Password string
}

type RouteRule struct {
	User    *string
	RuleSet *string
	Action  RouteAction
}
