package inbound

type Info struct {
	Type       InboundType
	Tag        string
	Server     string
	ServerPort int
	Users      []User
	TLS        TLS
}

type User struct {
	Name     string
	Password string
}

type TLS struct {
	Enabled    bool
	ServerName string
	Insecure   bool
}
