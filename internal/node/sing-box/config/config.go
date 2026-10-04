package config

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

type Config struct {
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
	Level     LogLevel `json:"level,omitzero"`
	Timestamp bool     `json:"timestamp,omitzero"`
}

// DNS section

type dns struct {
	Servers []server `json:"servers,omitempty"`
	Final   string   `json:"final,omitempty"`
}

// ENUM(udp, tcp, tls, https)
type DNSServerType string

type server struct {
	Type       DNSServerType `json:"type,omitzero"`
	Tag        string        `json:"tag,omitzero"`
	Server     string        `json:"server,omitzero"`
	ServerPort uint16        `json:"server_port,omitzero"`
}

// Outbound section

// ENUM(direct, block, vless, hysteria2)
type OutboundType string

type outbound struct {
	Type       OutboundType `json:"type,omitzero"`
	Tag        string       `json:"tag,omitempty"`
	Server     string       `json:"server,omitempty"`
	ServerPort uint32       `json:"server_port,omitzero"`
	UUID       string       `json:"uuid,omitempty"`
	Password   string       `json:"password,omitempty"`
	TLS        *outboundTLS `json:"tls,omitempty"`
	Transport  *transport   `json:"transport,omitempty"`
}

type outboundTLS struct {
	Enabled    bool             `json:"enabled"`
	ServerName string           `json:"server_name,omitempty"`
	Insecure   bool             `json:"insecure,omitempty"`
	Reality    *outboundReality `json:"reality,omitempty"`
	UTLS       *outboundUTLS    `json:"utls,omitempty"`
}

type outboundUTLS struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint"`
}

type outboundReality struct {
	Enabled   bool   `json:"enabled"`
	PublicKey string `json:"public_key"`
	ShortID   string `json:"short_id"`
}

// Route section

type route struct {
	Rules   []routeRule `json:"rules,omitempty"`
	RuleSet []ruleSet   `json:"rule_set,omitempty"`
	Final   string      `json:"final,omitempty"`
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
	Port     uint16      `json:"port,omitzero"`
	Action   RouteAction `json:"action"`
	Outbound string      `json:"outbound,omitempty"`
}

// ENUM(reject=1, route, hijack-dns, sniff)
type RouteAction int32
