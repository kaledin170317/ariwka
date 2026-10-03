package config

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

type OutboundTransport struct {
	Type        string
	ServiceName string
	Host        string
	Path        string
}

type Outbound struct {
	Tag        string
	Type       OutboundType
	Server     string
	ServerPort uint32
	UUID       string
	Password   string
	ServerName string
	PublicKey  string
	ShortID    string
	Insecure   bool
	Transport  OutboundTransport
}

type OutboundOption interface {
	outboundOption()
}

type OutboundServerOption string

func (OutboundServerOption) outboundOption() {}

type OutboundPortOption uint32

func (OutboundPortOption) outboundOption() {}

type OutboundUUIDOption string

func (OutboundUUIDOption) outboundOption() {}

type OutboundPasswordOption string

func (OutboundPasswordOption) outboundOption() {}

type OutboundVlessOption struct {
	ServerName string
	PublicKey  string
	ShortID    string
	Transport  OutboundTransport
}

func (OutboundVlessOption) outboundOption() {}

type OutboundHysteria2Option struct {
	ServerName string
	Insecure   bool
}

func (OutboundHysteria2Option) outboundOption() {}

func (c *Config) GetOutbounds() []Outbound {
	result := make([]Outbound, 0, len(c.Outbounds))
	for _, item := range c.Outbounds {
		value := Outbound{
			Tag:        item.Tag,
			Type:       item.Type,
			Server:     item.Server,
			ServerPort: item.ServerPort,
			UUID:       item.UUID,
			Password:   item.Password,
		}

		if item.TLS != nil {
			value.ServerName = item.TLS.ServerName
			value.Insecure = item.TLS.Insecure

			if item.TLS.Reality != nil {
				value.PublicKey = item.TLS.Reality.PublicKey
				value.ShortID = item.TLS.Reality.ShortID
			}
		}

		if item.Transport != nil {
			value.Transport = OutboundTransport{
				Type:        item.Transport.Type,
				ServiceName: item.Transport.ServiceName,
				Path:        item.Transport.Path,
			}
			if len(item.Transport.Host) > 0 {
				value.Transport.Host = item.Transport.Host[0]
			}
		}

		result = append(result, value)
	}

	return result
}

func (c *Config) AddOutbound(value Outbound) error {
	if value.Tag == "" || !value.Type.IsValid() {
		return errors.New("outbound tag and valid type required")
	}

	err := validateOutbound(value)
	if err != nil {
		return err
	}

	for _, item := range c.Outbounds {
		if item.Tag == value.Tag {
			return fmt.Errorf("outbound %q already exists", value.Tag)
		}
	}

	item := outbound{
		Type:       value.Type,
		Tag:        value.Tag,
		Server:     value.Server,
		ServerPort: value.ServerPort,
		UUID:       value.UUID,
		Password:   value.Password,
	}

	if value.Type == OutboundTypeVless || value.Type == OutboundTypeHysteria2 {
		item.TLS = &outboundTLS{
			Enabled:    true,
			ServerName: value.ServerName,
			Insecure:   value.Insecure,
		}
	}

	if value.Type == OutboundTypeVless {
		item.TLS.Reality = &outboundReality{
			Enabled:   true,
			PublicKey: value.PublicKey,
			ShortID:   value.ShortID,
		}
		item.Transport = outboundTransport(value.Transport)
	}

	c.Outbounds = append(c.Outbounds, item)

	return c.Apply()
}

func (c *Config) RemoveOutbound(tag string) error {
	for _, rule := range c.Route.Rules {
		if rule.Outbound == tag {
			return fmt.Errorf("outbound %q is used by a route rule", tag)
		}
	}

	for i, item := range c.Outbounds {
		if item.Tag == tag {
			c.Outbounds = append(c.Outbounds[:i], c.Outbounds[i+1:]...)

			return c.Apply()
		}
	}

	return fmt.Errorf("outbound %q not found", tag)
}

func (c *Config) ModifyOutbound(tag string, options []OutboundOption) error {
	for i := range c.Outbounds {
		item := &c.Outbounds[i]
		if item.Tag == tag {
			if item.Type != OutboundTypeVless && item.Type != OutboundTypeHysteria2 {
				return errors.New("outbound type cannot be modified")
			}

			for _, option := range options {
				err := validateOutboundOption(item.Type, option)
				if err != nil {
					return err
				}
			}

			if item.TLS == nil {
				item.TLS = &outboundTLS{Enabled: true}
			}

			if item.Type == OutboundTypeVless && item.TLS.Reality == nil {
				item.TLS.Reality = &outboundReality{Enabled: true}
			}

			for _, option := range options {
				switch value := option.(type) {
				case OutboundServerOption:
					item.Server = string(value)
				case OutboundPortOption:
					item.ServerPort = uint32(value)
				case OutboundUUIDOption:
					item.UUID = string(value)
				case OutboundPasswordOption:
					item.Password = string(value)
				case OutboundVlessOption:
					item.TLS.ServerName = value.ServerName
					item.TLS.Reality.PublicKey = value.PublicKey
					item.TLS.Reality.ShortID = value.ShortID
					item.Transport = outboundTransport(value.Transport)
				case OutboundHysteria2Option:
					item.TLS.ServerName = value.ServerName
					item.TLS.Insecure = value.Insecure
				}
			}

			return c.Apply()
		}
	}

	return fmt.Errorf("outbound %q not found", tag)
}

func outboundTransport(value OutboundTransport) *transport {
	result := &transport{
		Type:        value.Type,
		ServiceName: value.ServiceName,
		Path:        value.Path,
	}
	if value.Host != "" {
		result.Host = []string{value.Host}
	}

	return result
}

func validateOutbound(value Outbound) error {
	if value.Type == OutboundTypeDirect || value.Type == OutboundTypeBlock {
		return nil
	}

	if value.Server == "" || value.ServerPort < 1 || value.ServerPort > 65535 {
		return errors.New("outbound server and valid port required")
	}

	if value.Type == OutboundTypeHysteria2 {
		if value.Password == "" {
			return errors.New("Hysteria2 password required")
		}

		return nil
	}

	if value.ServerName == "" || !validUUID(value.UUID) || !validPublicKey(value.PublicKey) || !validShortID(value.ShortID) {
		return errors.New("invalid VLESS REALITY credentials")
	}

	if !validTransport(value.Transport) {
		return errors.New("invalid VLESS transport")
	}

	return nil
}

func validateOutboundOption(kind OutboundType, option OutboundOption) error {
	switch value := option.(type) {
	case OutboundServerOption:
		if value == "" {
			return errors.New("outbound server required")
		}
	case OutboundPortOption:
		if value < 1 || value > 65535 {
			return errors.New("invalid outbound port")
		}
	case OutboundUUIDOption:
		if kind != OutboundTypeVless || !validUUID(string(value)) {
			return errors.New("invalid VLESS UUID")
		}
	case OutboundPasswordOption:
		if kind != OutboundTypeHysteria2 || value == "" {
			return errors.New("invalid Hysteria2 password")
		}
	case OutboundVlessOption:
		if kind != OutboundTypeVless || value.ServerName == "" || !validPublicKey(value.PublicKey) || !validShortID(value.ShortID) || !validTransport(value.Transport) {
			return errors.New("invalid VLESS REALITY settings")
		}
	case OutboundHysteria2Option:
		if kind != OutboundTypeHysteria2 {
			return errors.New("Hysteria2 settings require Hysteria2 outbound")
		}
	}

	return nil
}

func validUUID(value string) bool {
	_, err := uuid.Parse(value)

	return err == nil
}

func validPublicKey(value string) bool {
	key, err := base64.RawURLEncoding.DecodeString(value)
	return err == nil && len(key) == 32
}

func validShortID(value string) bool {
	if len(value) > 16 || len(value)%2 != 0 {
		return false
	}

	_, err := hex.DecodeString(value)

	return err == nil
}

func validTransport(value OutboundTransport) bool {
	return value.Type == "grpc" && value.ServiceName != "" || value.Type == "http"
}
