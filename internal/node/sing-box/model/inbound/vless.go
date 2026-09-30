package inbound

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(grpc=1, http)
type VlessTransportType int32

type Vless struct {
	Tag        string
	Address    string
	Port       int32
	ServerName string
	PublicKey  string
	ShortID    string
	Transport  VlessTransport
	Users      []VlessUser
}

func (v *Vless) GetTag() string { return v.Tag }
func (v *Vless) GetPort() int32 { return v.Port }

type VlessUser struct {
	Tag  string
	UUID string
}

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
