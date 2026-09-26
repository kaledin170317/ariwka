package inbound

import (
	"crypto/rand"
	"fmt"

	"vpn/pkg/cert"
)

const (
	certPath = "/etc/sing-box/hy2_cert.pem"
	keyPath  = "/etc/sing-box/hy2_key.pem"
)

func NewHysteria2(tag string, port int32) (Inbound, error) {
	certificate, err := cert.Read(certPath)
	if err != nil {
		return Inbound{}, err
	}
	i := Inbound{
		Type:   InboundTypeHysteria2,
		Tag:    tag,
		Listen: "::",
		Port:   port,

		Users: []user{},

		TLS: &tls{
			Enabled:         true,
			CertificatePath: certPath,
			KeyPath:         keyPath,
			ServerName:      certificate.ServerName,
		},
	}
	return i, nil
}

type hysteria2Strategy struct{}

func (hysteria2Strategy) AddUser(inbound *Inbound, name string) error {
	for _, user := range inbound.Users {
		if user.Name == name {
			return fmt.Errorf("user %q already exists", name)
		}
	}

	inbound.Users = append(inbound.Users, user{Name: name, Password: rand.Text()})

	return nil
}

func (hysteria2Strategy) RemoveUser(inbound *Inbound, name string) error {
	for i, user := range inbound.Users {
		if user.Name == name {
			inbound.Users = append(inbound.Users[:i], inbound.Users[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("user %q not found", name)
}
