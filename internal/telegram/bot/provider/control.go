package provider

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	api "vpn/gen/control"
	common "vpn/gen/vpn/common"
)

var transportNames = map[common.VlessTransportType]string{
	common.VlessTransportType_VLESS_TRANSPORT_TYPE_GRPC: "grpc",
	common.VlessTransportType_VLESS_TRANSPORT_TYPE_HTTP: "http",
}

type Control struct {
	conn     *grpc.ClientConn
	auth     api.AuthServiceClient
	configs  api.ConfigServiceClient
	nodes    api.NodeServiceClient
	username string
	password string
}

type User struct {
	control Control
	token   string
}

type Node struct {
	ID   string
	Name string
}

type Inbound struct {
	Tag  string
	Type string
}

type Profile struct {
	ID         string
	Connection string
}

func Open(address, username, password string) (Control, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return Control{}, err
	}

	return Control{
		conn: conn,
		auth: api.NewAuthServiceClient(conn), configs: api.NewConfigServiceClient(conn),
		nodes: api.NewNodeServiceClient(conn), username: username, password: password,
	}, nil
}

func (c Control) Close() error { return c.conn.Close() }

func (c Control) ForUser(ctx context.Context, telegramID int64) (User, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	admin, err := c.auth.Login(ctx, &api.LoginRequest{Username: c.username, Password: c.password})
	if err != nil {
		return User{}, err
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+admin.GetToken())

	resp, err := c.auth.LoginTelegramUser(ctx, &api.LoginTelegramUserRequest{TelegramId: telegramID})
	if err != nil {
		return User{}, err
	}

	return User{control: c, token: resp.GetToken()}, nil
}

func (u User) Profiles(ctx context.Context) ([]Profile, error) {
	resp, err := u.control.configs.GetConfig(u.context(ctx), &api.GetConfigRequest{})
	if err != nil {
		return nil, err
	}

	profiles := make([]Profile, 0, len(resp.GetProfiles()))
	for _, profile := range resp.GetProfiles() {
		profiles = append(profiles, Profile{
			ID:         profile.GetId(),
			Connection: profile.GetConnection(),
		})
	}

	return profiles, nil
}

func (u User) SubscriptionLink(ctx context.Context) (string, error) {
	resp, err := u.control.configs.GetSubscriptionLink(u.context(ctx), &api.GetSubscriptionLinkRequest{})
	if err != nil {
		return "", err
	}

	return resp.GetUrl(), nil
}

func (u User) Nodes(ctx context.Context) ([]Node, error) {
	resp, err := u.control.nodes.GetNodes(u.context(ctx), &api.GetNodesRequest{})
	if err != nil {
		return nil, err
	}

	nodes := make([]Node, 0, len(resp.GetNodes()))
	for _, node := range resp.GetNodes() {
		nodes = append(nodes, Node{ID: node.GetId(), Name: node.GetName()})
	}

	return nodes, nil
}

func (u User) Inbounds(ctx context.Context, nodeID string) ([]Inbound, error) {
	resp, err := u.control.nodes.GetNodeInbounds(u.context(ctx), &api.GetNodeInboundsRequest{NodeId: nodeID})
	if err != nil {
		return nil, err
	}

	inbounds := make([]Inbound, 0, len(resp.GetInbounds()))
	for _, inbound := range resp.GetInbounds() {
		if inbound.GetNetwork() == common.InboundNetwork_INBOUND_NETWORK_INTERNAL {
			continue
		}

		if inbound.GetHysteria2() != nil {
			inbounds = append(inbounds, Inbound{Tag: inbound.GetTag(), Type: "hysteria2"})
		}

		if v := inbound.GetVless(); v != nil {
			kind := "vless"
			if name := transportNames[v.GetTransport().GetType()]; name != "" {
				kind += "/" + name
			}

			inbounds = append(inbounds, Inbound{Tag: inbound.GetTag(), Type: kind})
		}
	}

	return inbounds, nil
}

func (u User) CreateProfile(ctx context.Context, nodeID, tag string) (string, error) {
	resp, err := u.control.configs.CreateMyProfile(u.context(ctx), &api.CreateMyProfileRequest{NodeId: nodeID, Inbound: tag})
	if err != nil {
		return "", err
	}

	return resp.GetConnection(), nil
}

func (u User) DeleteProfile(ctx context.Context, id string) error {
	_, err := u.control.configs.DeleteMyProfile(u.context(ctx), &api.DeleteMyProfileRequest{Id: id})
	return err
}

func (u User) Profile(ctx context.Context, id string) (string, error) {
	resp, err := u.control.configs.GetProfile(u.context(ctx), &api.GetProfileRequest{Id: id})
	if err != nil {
		return "", err
	}

	return resp.GetConnection(), nil
}

func (u User) context(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+u.token)
}
