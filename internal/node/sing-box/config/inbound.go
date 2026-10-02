package config

import (
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/curve25519"

	"vpn/internal/node/sing-box/model/inbound"
	"vpn/pkg/cert"
	"vpn/pkg/env"
)

//go:generate go tool go-enum --marshal --output-suffix=.enum.gen

// ENUM(hysteria2=1, vless)
type inboundType int32

const (
	certPath = "/etc/sing-box/hy2_cert.pem"
	keyPath  = "/etc/sing-box/hy2_key.pem"
)

// inboundConfig contains only fields written to the sing-box config file.
type inboundConfig struct {
	Type      inboundType  `json:"type"`
	Tag       string       `json:"tag,omitempty"`
	Listen    string       `json:"listen,omitempty"`
	Port      int32        `json:"listen_port,omitzero"`
	Users     []userConfig `json:"users,omitempty"`
	TLS       *inboundTLS  `json:"tls,omitempty"`
	Transport *transport   `json:"transport,omitempty"`
}

type inboundTLS struct {
	Enabled         bool     `json:"enabled"`
	CertificatePath string   `json:"certificate_path,omitempty"`
	KeyPath         string   `json:"key_path,omitempty"`
	ServerName      string   `json:"server_name,omitempty"`
	Insecure        bool     `json:"insecure,omitempty"`
	Reality         *reality `json:"reality,omitempty"`
}

type reality struct {
	Enabled    bool      `json:"enabled"`
	Handshake  handshake `json:"handshake"`
	PrivateKey string    `json:"private_key"`
	ShortID    []string  `json:"short_id"`
}

type handshake struct {
	Server string `json:"server"`
	Port   int32  `json:"server_port"`
}

type transport struct {
	Type        string   `json:"type"`
	ServiceName string   `json:"service_name,omitempty"`
	Host        []string `json:"host,omitempty"`
	Path        string   `json:"path,omitempty"`
}

type InboundOption interface {
	inboundOption()
}

type PortOption int32

func (PortOption) inboundOption() {}

type ServerNameOption string

func (ServerNameOption) inboundOption() {}

type TransportOption struct {
	Type, ServiceName, Host, Path string
}

func (TransportOption) inboundOption() {}

type PrivateKeyOption string

func (PrivateKeyOption) inboundOption() {}

type ShortIDOption string

func (ShortIDOption) inboundOption() {}

func (c *Config) AddHysteria2Inbound(tag string, portNumber int32) (*inbound.Hysteria2, error) {
	connection, err := newHysteria2(tag, portNumber)
	if err != nil {
		return nil, err
	}

	created, err := c.addInbound(connection)
	if err != nil {
		return nil, err
	}

	return created.(*inbound.Hysteria2), nil
}

func (c *Config) AddVlessInbound(tag string, portNumber int32, serverName string, options inbound.VlessTransportOptions, privateKey, shortID, serviceName string) (*inbound.Vless, error) {
	connection, err := newVless(tag, portNumber, serverName, options, privateKey, shortID, serviceName)
	if err != nil {
		return nil, err
	}

	created, err := c.addInbound(connection)
	if err != nil {
		return nil, err
	}

	return created.(*inbound.Vless), nil
}

func (c *Config) GetInbounds() ([]inbound.Connection, error) {
	result := make([]inbound.Connection, 0, len(c.Inbounds))
	for i := range c.Inbounds {
		model, err := c.Inbounds[i].model()
		if err != nil {
			return nil, err
		}

		result = append(result, model)
	}

	return result, nil
}

func (c *Config) RemoveInbound(tag string) error {
	for i, connection := range c.Inbounds {
		if connection.Tag == tag {
			c.Inbounds = append(c.Inbounds[:i], c.Inbounds[i+1:]...)

			return c.Apply()
		}
	}

	return fmt.Errorf("failed to find inbound %q", tag)
}

func (c *Config) ModifyInbound(tag string, options []InboundOption) error {
	for i := range c.Inbounds {
		current := &c.Inbounds[i]
		if current.Tag == tag {
			for _, option := range options {
				switch value := option.(type) {
				case PortOption:
					current.Port = int32(value)
				case ServerNameOption:
					current.TLS.ServerName = string(value)
					if current.TLS.Reality != nil {
						current.TLS.Reality.Handshake.Server = string(value)
					}
				case TransportOption:
					current.Transport = &transport{
						Type:        value.Type,
						ServiceName: value.ServiceName,
						Path:        value.Path,
					}
					if value.Host != "" {
						current.Transport.Host = []string{value.Host}
					}
				case PrivateKeyOption:
					if current.TLS.Reality != nil {
						current.TLS.Reality.PrivateKey = string(value)
					}
				case ShortIDOption:
					if current.TLS.Reality != nil {
						current.TLS.Reality.ShortID = []string{string(value)}
					}
				}
			}

			return c.Apply()
		}
	}

	return fmt.Errorf("inbound %q not found", tag)
}

func (c *Config) addInbound(connection inboundConfig) (inbound.Connection, error) {
	if connection.Tag == "" {
		return nil, errors.New("inbound tag required")
	}

	for i := range c.Inbounds {
		if c.Inbounds[i].Tag == connection.Tag {
			return nil, fmt.Errorf("inbound %q already exists", connection.Tag)
		}

		if c.Inbounds[i].Port == connection.Port {
			return nil, fmt.Errorf("port %d already in use", connection.Port)
		}
	}

	model, err := connection.model()
	if err != nil {
		return nil, err
	}

	c.Inbounds = append(c.Inbounds, connection)

	err = c.Apply()
	if err != nil {
		return nil, err
	}

	return model, nil
}

func newHysteria2(tag string, port int32) (inboundConfig, error) {
	certificate, err := cert.Read(certPath)
	if err != nil {
		return inboundConfig{}, err
	}

	return inboundConfig{
		Type:   InboundTypeHysteria2,
		Tag:    tag,
		Listen: "::",
		Port:   port,
		Users:  []userConfig{},
		TLS: &inboundTLS{
			Enabled:         true,
			CertificatePath: certPath,
			KeyPath:         keyPath,
			ServerName:      certificate.ServerName,
		},
	}, nil
}

func newVless(tag string, port int32, serverName string, options inbound.VlessTransportOptions, privateKey, shortID, serviceName string) (inboundConfig, error) {
	var t transport

	switch options.Type {
	case inbound.VlessTransportTypeGrpc:
		t = transport{
			Type:        "grpc",
			ServiceName: serviceName,
		}
	case inbound.VlessTransportTypeHttp:
		t = transport{
			Type: "http",
			Path: options.Path,
			Host: []string{options.Host},
		}
	default:
		return inboundConfig{}, fmt.Errorf("unsupported VLESS transport type %d", options.Type)
	}

	return inboundConfig{
		Type:   InboundTypeVless,
		Tag:    tag,
		Listen: "::",
		Port:   port,
		Users:  []userConfig{},
		TLS: &inboundTLS{
			Enabled:    true,
			ServerName: serverName,
			Reality: &reality{
				Enabled:    true,
				Handshake:  handshake{Server: serverName, Port: 443},
				PrivateKey: privateKey,
				ShortID:    []string{shortID},
			},
		},
		Transport: &t,
	}, nil
}

// model adds client-facing details derived from the stored certificate or REALITY key.
func (i *inboundConfig) model() (inbound.Connection, error) {
	switch i.Type {
	case InboundTypeHysteria2:
		pin, err := cert.CertificatePin(i.TLS.CertificatePath)
		if err != nil {
			return nil, err
		}

		users := make([]inbound.Hysteria2User, 0, len(i.Users))
		for _, user := range i.Users {
			users = append(users, inbound.Hysteria2User{Name: user.Name, Password: user.Password})
		}

		return &inbound.Hysteria2{
			Tag:     i.Tag,
			Address: env.GetServerHost(),
			Port:    i.Port,
			TLS: inbound.TLS{
				ServerName: i.TLS.ServerName,
				Insecure:   i.TLS.Insecure,
				PinSHA256:  pin,
			},
			Users: users,
		}, nil
	case InboundTypeVless:
		private, err := base64.RawURLEncoding.DecodeString(i.TLS.Reality.PrivateKey)
		if err != nil {
			return nil, err
		}

		public, err := curve25519.X25519(private, curve25519.Basepoint)
		if err != nil {
			return nil, err
		}

		t, _ := inbound.ParseVlessTransportType(i.Transport.Type)

		transport := inbound.VlessTransport{
			Type:        t,
			ServiceName: i.Transport.ServiceName,
			Path:        i.Transport.Path,
		}
		if len(i.Transport.Host) > 0 {
			transport.Host = i.Transport.Host[0]
		}

		users := make([]inbound.VlessUser, 0, len(i.Users))
		for _, user := range i.Users {
			users = append(users, inbound.VlessUser{Tag: user.Name, UUID: user.UUID})
		}

		return &inbound.Vless{
			Tag:        i.Tag,
			Address:    env.GetServerHost(),
			Port:       i.Port,
			ServerName: i.TLS.ServerName,
			PublicKey:  base64.RawURLEncoding.EncodeToString(public),
			ShortID:    i.TLS.Reality.ShortID[0],
			Transport:  transport,
			Users:      users,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported inbound type %q", i.Type)
	}
}
