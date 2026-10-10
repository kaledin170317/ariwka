package provider

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/app/proxyman/command"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/proxy/hysteria"
	hyaccount "github.com/xtls/xray-core/proxy/hysteria/account"
	"github.com/xtls/xray-core/proxy/vless"
	vlessin "github.com/xtls/xray-core/proxy/vless/inbound"
	"github.com/xtls/xray-core/transport/internet"
	grpcstream "github.com/xtls/xray-core/transport/internet/grpc"
	hytransport "github.com/xtls/xray-core/transport/internet/hysteria"
	"github.com/xtls/xray-core/transport/internet/reality"
	"github.com/xtls/xray-core/transport/internet/splithttp"
	"github.com/xtls/xray-core/transport/internet/tls"
	"golang.org/x/crypto/curve25519"
	"google.golang.org/protobuf/proto"

	"vpn/internal/control/node/model"
)

func (n *Node) XrayInbounds(ctx context.Context, configs []model.InboundConfig) ([]model.Inbound, error) {
	resp, err := n.xray.ListInbounds(ctx, &command.ListInboundsRequest{IsOnlyTags: true})
	if err != nil {
		return nil, err
	}

	byTag := make(map[string]model.InboundConfig, len(configs))
	for _, c := range configs {
		byTag[c.Tag] = c
	}

	result := make([]model.Inbound, 0, len(resp.Inbounds))
	for _, entry := range resp.Inbounds {
		c, ok := byTag[entry.Tag]
		if !ok {
			continue
		}

		in, err := n.xrayInbound(ctx, c)
		if err != nil {
			return nil, fmt.Errorf("inbound %s: %w", c.Tag, err)
		}

		result = append(result, in)
	}

	return result, nil
}

func (n *Node) xrayInbound(ctx context.Context, c model.InboundConfig) (model.Inbound, error) {
	result := model.Inbound{Tag: c.Tag, Network: c.Network}

	users, err := n.xray.GetInboundUsers(ctx, &command.GetInboundUserRequest{Tag: c.Tag})
	if err != nil {
		return result, err
	}

	switch c.Type {
	case model.InboundTypeVless:
		key, err := base64.RawURLEncoding.DecodeString(c.PrivateKey)
		if err != nil {
			return result, err
		}

		public, err := curve25519.X25519(key, curve25519.Basepoint)
		if err != nil {
			return result, err
		}

		v := &model.Vless{Port: c.Port, ServerName: c.ServerName, PublicKey: base64.RawURLEncoding.EncodeToString(public), ShortID: c.ShortID,
			Transport: model.VlessTransport{Type: c.Transport.Type, ServiceName: c.ServiceName, Host: c.Transport.Host, Path: c.Transport.Path}}

		for _, u := range users.Users {
			account := new(vless.Account)

			err := proto.Unmarshal(u.GetAccount().GetValue(), account)
			if err != nil {
				return result, err
			}

			v.Users = append(v.Users, account.Id)
			v.UserDetails = append(v.UserDetails, model.VlessUser{Tag: u.Email, UUID: account.Id})
		}

		result.Vless = v
	case model.InboundTypeHysteria2:
		h := &model.Hysteria2{Port: c.Port, TLS: model.TLS{ServerName: "vpn", Insecure: true}}

		for _, u := range users.Users {
			account := new(hyaccount.Account)

			err := proto.Unmarshal(u.GetAccount().GetValue(), account)
			if err != nil {
				return result, err
			}

			h.Users = append(h.Users, model.InboundUser{Name: u.Email, Password: account.Auth})
		}

		result.Hysteria2 = h
	default:
		return result, fmt.Errorf("unsupported inbound type %d", c.Type)
	}

	return result, nil
}

func (n *Node) XrayAddInbound(ctx context.Context, c model.InboundConfig) (model.Inbound, error) {
	receiver := &proxyman.ReceiverConfig{PortList: &xnet.PortList{Range: []*xnet.PortRange{{From: uint32(c.Port), To: uint32(c.Port)}}}}

	var settings proto.Message

	switch c.Type {
	case model.InboundTypeVless:
		key, err := base64.RawURLEncoding.DecodeString(c.PrivateKey)
		if err != nil {
			return model.Inbound{}, err
		}

		short, err := hex.DecodeString(c.ShortID)
		if err != nil {
			return model.Inbound{}, err
		}

		security := serial.ToTypedMessage(&reality.Config{
			Dest: c.ServerName + ":443", Type: "tcp", ServerNames: []string{c.ServerName}, PrivateKey: key, ShortIds: [][]byte{short},
		})

		stream := &internet.StreamConfig{SecurityType: security.Type, SecuritySettings: []*serial.TypedMessage{security}}
		switch c.Transport.Type {
		case model.VlessTransportTypeGrpc:
			stream.ProtocolName = "grpc"
			stream.TransportSettings = []*internet.TransportConfig{{ProtocolName: "grpc", Settings: serial.ToTypedMessage(&grpcstream.Config{ServiceName: c.ServiceName})}}
		case model.VlessTransportTypeHttp:
			stream.ProtocolName = "splithttp"
			stream.TransportSettings = []*internet.TransportConfig{{ProtocolName: "splithttp", Settings: serial.ToTypedMessage(&splithttp.Config{Host: c.Transport.Host, Path: c.Transport.Path})}}
		default:
			return model.Inbound{}, errors.New("unsupported VLESS transport")
		}

		receiver.StreamSettings = stream
		settings = &vlessin.Config{Decryption: "none"}
	case model.InboundTypeHysteria2:
		// The certificate is mounted in the Xray container by Compose.
		tlsSettings := serial.ToTypedMessage(&tls.Config{Certificate: []*tls.Certificate{{CertificatePath: "/etc/xray/hy2_cert.pem", KeyPath: "/etc/xray/hy2_key.pem"}}})
		receiver.StreamSettings = &internet.StreamConfig{ProtocolName: "hysteria", SecurityType: tlsSettings.Type,
			TransportSettings: []*internet.TransportConfig{{ProtocolName: "hysteria", Settings: serial.ToTypedMessage(&hytransport.Config{})}},
			SecuritySettings:  []*serial.TypedMessage{tlsSettings},
		}
		settings = &hysteria.ServerConfig{}
	default:
		return model.Inbound{}, errors.New("unsupported inbound type")
	}

	_, err := n.xray.AddInbound(ctx, &command.AddInboundRequest{Inbound: &core.InboundHandlerConfig{Tag: c.Tag,
		ReceiverSettings: serial.ToTypedMessage(receiver), ProxySettings: serial.ToTypedMessage(settings)}})
	if err != nil {
		return model.Inbound{}, err
	}

	return n.xrayInbound(ctx, c)
}

func (n *Node) XrayRemoveInbound(ctx context.Context, tag string) error {
	_, err := n.xray.RemoveInbound(ctx, &command.RemoveInboundRequest{Tag: tag})
	return err
}

func (n *Node) XrayAddUser(ctx context.Context, inbound string, user model.VlessUser, password string) error {
	var account proto.Message = &vless.Account{Id: user.UUID}
	if password != "" {
		account = &hyaccount.Account{Auth: password}
	}

	_, err := n.xray.AlterInbound(ctx, &command.AlterInboundRequest{Tag: inbound,
		Operation: serial.ToTypedMessage(&command.AddUserOperation{User: &protocol.User{Email: user.Tag, Account: serial.ToTypedMessage(account)}})})

	return err
}

func (n *Node) XrayRemoveUser(ctx context.Context, inbound, email string) error {
	_, err := n.xray.AlterInbound(ctx, &command.AlterInboundRequest{Tag: inbound,
		Operation: serial.ToTypedMessage(&command.RemoveUserOperation{Email: email})})

	return err
}
