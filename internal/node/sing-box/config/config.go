package config

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

type Config struct {
	file fileConfig
}

type fileConfig struct {
	Log       log             `json:"log,omitzero"`
	DNS       dns             `json:"dns,omitzero"`
	Inbounds  []inboundConfig `json:"inbounds,omitempty"`
	Outbounds []outbound      `json:"outbounds,omitzero"`
	Route     route           `json:"route,omitzero"`
}

// Log section

// ENUM(debug, info, warn, error, fatal, panic)
type LogLevel string

type log struct {
	Level LogLevel `json:"level,omitzero"`
}

// DNS section

type dns struct {
	Servers []server `json:"servers,omitempty"`
}

// ENUM(udp, tcp, tls, https)
type DNSServerType string

type server struct {
	Type   DNSServerType `json:"type,omitzero"`
	Tag    string        `json:"tag,omitzero"`
	Server string        `json:"server,omitzero"`
}

// Outbound section

// ENUM(direct, block)
type OutboundType string

type outbound struct {
	Type OutboundType `json:"type,omitzero"`
}

// Route section

type route struct {
	Rules   []routeRule `json:"rules,omitempty"`
	RuleSet []ruleSet   `json:"rule_set,omitempty"`
}

// ENUM(inline, local, remote)
type RuleSetType string

// ENUM(source, binary)
type RuleSetFormat string

type ruleSet struct {
	Type   RuleSetType   `json:"type"`
	Tag    string        `json:"tag"`
	Format RuleSetFormat `json:"format,omitempty"`
	URL    string        `json:"url,omitempty"`
}

type routeRule struct {
	AuthUser []string    `json:"auth_user,omitempty"`
	RuleSet  []string    `json:"rule_set,omitempty"`
	Action   RouteAction `json:"action"`
}

// ENUM(reject=1, route, hijack-dns, sniff)
type RouteAction int32
