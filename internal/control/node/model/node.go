package model

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(hysteria2=1, vless)
type InboundType int32

// ENUM(grpc=1, http)
type VlessTransportType int32

// ENUM(users=1, internal, default)
type InboundNetwork int32

type VlessTransport struct {
	Type        VlessTransportType
	ServiceName string
	Host        string
	Path        string
}

type VlessTransportOptions struct {
	Type VlessTransportType
	Host string
	Path string
}

type Node struct {
	ID          string
	Name        string
	Address     string
	Description string
	Country     string
}

type InboundConfig struct {
	Tag          string
	NodeID       string
	Type         InboundType
	Port         int32
	ServerName   string
	Transport    VlessTransportOptions
	ServiceName  string
	PrivateKey   string
	ShortID      string
	Network      InboundNetwork
	InternalUUID string
}

type Inbound struct {
	Tag       string
	Network   InboundNetwork
	Hysteria2 *Hysteria2
	Vless     *Vless
}

type Vless struct {
	Address     string
	Port        int32
	ServerName  string
	PublicKey   string
	ShortID     string
	Transport   VlessTransport
	Users       []string
	UserDetails []VlessUser
}

type VlessUser struct {
	Tag  string
	UUID string
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
