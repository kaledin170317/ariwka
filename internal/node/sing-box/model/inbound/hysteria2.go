package inbound

type Hysteria2 struct {
	Tag     string
	Address string
	Port    int32
	TLS     TLS
	Users   []Hysteria2User
}

func (h *Hysteria2) GetTag() string { return h.Tag }
func (h *Hysteria2) GetPort() int32 { return h.Port }

type Hysteria2User struct {
	Name     string
	Password string
}

type TLS struct {
	ServerName string
	Insecure   bool
	PinSHA256  string
}
