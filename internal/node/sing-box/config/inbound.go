package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
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

type userConfig struct {
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
	UUID     string `json:"uuid,omitempty"`
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

func newHysteria2(port int32) (inboundConfig, error) {
	certificate, err := cert.Read(certPath)
	if err != nil {
		return inboundConfig{}, err
	}

	return inboundConfig{
		Type:   InboundTypeHysteria2,
		Tag:    uuid.NewString(),
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

func newVless(port int32, serverName string, options inbound.VlessTransportOptions) (inboundConfig, error) {
	key := make([]byte, 32)

	_, err := rand.Read(key)
	if err != nil {
		return inboundConfig{}, err
	}

	short := make([]byte, 8)

	_, err = rand.Read(short)
	if err != nil {
		return inboundConfig{}, err
	}

	var t transport

	switch options.Type {
	case inbound.VlessTransportTypeGrpc:
		t = transport{
			Type:        "grpc",
			ServiceName: rand.Text(),
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
		Tag:    uuid.NewString(),
		Listen: "::",
		Port:   port,
		Users:  []userConfig{},
		TLS: &inboundTLS{
			Enabled:    true,
			ServerName: serverName,
			Reality: &reality{
				Enabled:    true,
				Handshake:  handshake{Server: serverName, Port: 443},
				PrivateKey: base64.RawURLEncoding.EncodeToString(key),
				ShortID:    []string{hex.EncodeToString(short)},
			},
		},
		Transport: &t,
	}, nil
}

func newHysteria2User(name string) userConfig {
	return userConfig{Name: name, Password: rand.Text()}
}

func newVlessUser(tag string) userConfig {
	return userConfig{Name: tag, UUID: tag}
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
