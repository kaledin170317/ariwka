package provider

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/xtls/xray-core/app/proxyman/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	api "vpn/gen/node"
	"vpn/internal/control/api/interceptors"
	"vpn/internal/control/node/model"
)

type Repository interface {
	List(context.Context) ([]model.Node, error)
}

type Node struct {
	model.Node
	inbounds  api.InboundServiceClient
	users     api.UserServiceClient
	routes    api.RouteServiceClient
	outbounds api.OutboundServiceClient
	conn      *grpc.ClientConn
	xray      command.HandlerServiceClient
	xrayConn  *grpc.ClientConn
}

type Provider struct {
	mu    sync.RWMutex
	nodes map[string]*Node
}

func New(ctx context.Context, repo Repository) (*Provider, error) {
	nodes, err := repo.List(ctx)
	if err != nil {
		return nil, err
	}

	p := &Provider{nodes: make(map[string]*Node, len(nodes))}
	for _, node := range nodes {
		err = p.Add(node)
		if err != nil {
			p.Close()
			return nil, err
		}
	}

	return p, nil
}

func (p *Provider) Add(node model.Node) error {
	conn, err := grpc.NewClient(
		node.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptors.Timeout(30*time.Second)),
	)
	if err != nil {
		return err
	}

	host, _, err := net.SplitHostPort(node.Address)
	if err != nil {
		_ = conn.Close()
		return err
	}

	xrayConn, err := grpc.NewClient(net.JoinHostPort(host, "9093"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptors.Timeout(30*time.Second)),
	)
	if err != nil {
		_ = conn.Close()
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if old, ok := p.nodes[node.ID]; ok {
		_ = old.conn.Close()
		_ = old.xrayConn.Close()
	}

	p.nodes[node.ID] = &Node{
		Node:      node,
		inbounds:  api.NewInboundServiceClient(conn),
		users:     api.NewUserServiceClient(conn),
		routes:    api.NewRouteServiceClient(conn),
		outbounds: api.NewOutboundServiceClient(conn),
		conn:      conn,
		xray:      command.NewHandlerServiceClient(xrayConn),
		xrayConn:  xrayConn,
	}

	return nil
}

func (p *Provider) Get(id string) (*Node, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	node, ok := p.nodes[id]
	if !ok {
		return nil, errors.New("node not found")
	}

	return node, nil
}

func (p *Provider) Remove(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if node, ok := p.nodes[id]; ok {
		_ = node.conn.Close()
		_ = node.xrayConn.Close()

		delete(p.nodes, id)
	}
}

func (p *Provider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, node := range p.nodes {
		_ = node.conn.Close()
		_ = node.xrayConn.Close()

		delete(p.nodes, id)
	}
}
