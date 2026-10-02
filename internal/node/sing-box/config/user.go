package config

import (
	"errors"
	"fmt"

	"vpn/internal/node/sing-box/model/inbound"
)

type userConfig struct {
	Name     string `json:"name"`
	Password string `json:"password,omitempty"`
	UUID     string `json:"uuid,omitempty"`
}

type User interface {
	user()
}

type Hysteria2User struct {
	Name     string
	Password string
}

func (Hysteria2User) user() {}

type VlessUser struct {
	Tag  string
	UUID string
}

func (VlessUser) user() {}

type UserOption interface {
	userOption()
}

type PasswordOption string

func (PasswordOption) userOption() {}

type UUIDOption string

func (UUIDOption) userOption() {}

func (c *Config) AddUser(inboundTag string, option User) (inbound.Connection, error) {
	for i := range c.Inbounds {
		if c.Inbounds[i].Tag == inboundTag {
			var user userConfig

			switch value := option.(type) {
			case Hysteria2User:
				if c.Inbounds[i].Type != InboundTypeHysteria2 || value.Name == "" || value.Password == "" {
					return nil, errors.New("invalid Hysteria2 user")
				}

				user = userConfig{Name: value.Name, Password: value.Password}
			case VlessUser:
				if c.Inbounds[i].Type != InboundTypeVless || value.Tag == "" || value.UUID == "" {
					return nil, errors.New("invalid VLESS user")
				}

				user = userConfig{Name: value.Tag, UUID: value.UUID}
			default:
				return nil, errors.New("unsupported user type")
			}

			for _, existing := range c.Inbounds[i].Users {
				if existing.Name == user.Name {
					return nil, fmt.Errorf("user %q already exists", user.Name)
				}
			}

			c.Inbounds[i].Users = append(c.Inbounds[i].Users, user)

			err := c.Apply()
			if err != nil {
				return nil, err
			}

			return c.Inbounds[i].model()
		}
	}

	return nil, fmt.Errorf("failed to find inbound %q", inboundTag)
}

func (c *Config) RemoveUser(user, tag string) error {
	for i := range c.Inbounds {
		if c.Inbounds[i].Tag == tag {
			for j, existing := range c.Inbounds[i].Users {
				if existing.Name == user {
					users := c.Inbounds[i].Users
					c.Inbounds[i].Users = append(users[:j], users[j+1:]...)

					return c.Apply()
				}
			}

			return fmt.Errorf("user %q not found", user)
		}
	}

	return fmt.Errorf("failed to find inbound %q", tag)
}

func (c *Config) ModifyUser(inboundTag, tag string, options []UserOption) error {
	for i := range c.Inbounds {
		if c.Inbounds[i].Tag == inboundTag {
			for j := range c.Inbounds[i].Users {
				if c.Inbounds[i].Users[j].Name == tag {
					current := &c.Inbounds[i].Users[j]

					for _, option := range options {
						switch value := option.(type) {
						case PasswordOption:
							current.Password = string(value)
						case UUIDOption:
							current.UUID = string(value)
						}
					}

					return c.Apply()
				}
			}
		}
	}

	return errors.New("user not found")
}
