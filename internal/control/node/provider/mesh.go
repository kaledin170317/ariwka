package provider

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"

	"github.com/xtls/xray-core/app/proxyman"
	"github.com/xtls/xray-core/app/proxyman/command"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/proxy/vless"
	vlessout "github.com/xtls/xray-core/proxy/vless/outbound"
	"github.com/xtls/xray-core/transport/internet"
	grpcstream "github.com/xtls/xray-core/transport/internet/grpc"
	"github.com/xtls/xray-core/transport/internet/reality"

	"vpn/internal/control/node/model"
)

func (n *Node) XrayOutbounds(ctx context.Context) ([]*core.OutboundHandlerConfig, error) {
	response, err := n.xray.ListOutbounds(ctx, &command.ListOutboundsRequest{})
	if err != nil {
		return nil, err
	}

	return response.Outbounds, nil
}

func (n *Node) XrayRemoveOutbound(ctx context.Context, tag string) error {
	_, err := n.xray.RemoveOutbound(ctx, &command.RemoveOutboundRequest{Tag: tag})
	return err
}

func (n *Node) XrayAddOutbound(ctx context.Context, outbound *core.OutboundHandlerConfig) error {
	_, err := n.xray.AddOutbound(ctx, &command.AddOutboundRequest{Outbound: outbound})
	return err
}

func MeshOutbound(address string, inbound model.Inbound, uuid string) (*core.OutboundHandlerConfig, error) {
	if inbound.Vless == nil {
		return nil, errors.New("mesh inbound is not VLESS")
	}

	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	key, err := base64.RawURLEncoding.DecodeString(inbound.Vless.PublicKey)
	if err != nil {
		return nil, err
	}

	short, err := hex.DecodeString(inbound.Vless.ShortID)
	if err != nil {
		return nil, err
	}

	stream := &internet.StreamConfig{ProtocolName: "grpc",
		TransportSettings: []*internet.TransportConfig{{ProtocolName: "grpc", Settings: serial.ToTypedMessage(&grpcstream.Config{ServiceName: inbound.Vless.Transport.ServiceName})}},
	}
	security := serial.ToTypedMessage(&reality.Config{ServerName: inbound.Vless.ServerName, PublicKey: key, ShortId: short, Fingerprint: "chrome"})
	stream.SecurityType = security.Type
	stream.SecuritySettings = []*serial.TypedMessage{security}

	addressValue := &xnet.IPOrDomain{Address: &xnet.IPOrDomain_Domain{Domain: host}}
	if ip := net.ParseIP(host); ip != nil {
		addressValue.Address = &xnet.IPOrDomain_Ip{Ip: ip}
	}

	server := &protocol.ServerEndpoint{Address: addressValue,
		Port: uint32(inbound.Vless.Port), User: &protocol.User{Account: serial.ToTypedMessage(&vless.Account{Id: uuid})}}

	return &core.OutboundHandlerConfig{Tag: inbound.Tag,
		SenderSettings: serial.ToTypedMessage(&proxyman.SenderConfig{StreamSettings: stream}),
		ProxySettings:  serial.ToTypedMessage(&vlessout.Config{Vnext: server}),
	}, nil
}
