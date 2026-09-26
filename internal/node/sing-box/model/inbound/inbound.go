package inbound

import (
	"vpn/pkg/cert"
	"vpn/pkg/env"
)

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(hysteria2)
type InboundType string

type Inbound struct {
	Type   InboundType `json:"type"`
	Tag    string      `json:"tag,omitempty"`
	Listen string      `json:"listen,omitempty"`
	Port   int32       `json:"listen_port,omitzero"`

	Users []user `json:"users,omitempty"`
	TLS   *tls   `json:"tls,omitempty"`
}

type user struct {
	Name string `json:"name"`

	// Hysteria2
	Password string `json:"password,omitempty"`

	// VLESS
	UUID string `json:"uuid,omitempty"`
	Flow string `json:"flow,omitempty"`
}

type tls struct {
	Enabled         bool   `json:"enabled"`
	CertificatePath string `json:"certificate_path,omitempty"`
	KeyPath         string `json:"key_path,omitempty"`
	ServerName      string `json:"server_name,omitempty"`
	Insecure        bool   `json:"insecure,omitempty"`
	PinSHA256       string `json:"-"`
}

func (i *Inbound) GetType() string {
	return i.Type.String()
}

func (i *Inbound) GetHost() string {
	return env.GetServerHost()
}

func (i *Inbound) LoadCertificatePin() error {
	pin, err := cert.CertificatePin(i.TLS.CertificatePath)
	if err != nil {
		return err
	}
	i.TLS.PinSHA256 = pin
	return nil
}
