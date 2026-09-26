package model

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(hysteria2=1)
type InboundType int32

// ENUM(reject=1, route, hijack-dns, sniff)
type RouteAction int32

type Node struct {
	ID      string
	Name    string
	Address string
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
