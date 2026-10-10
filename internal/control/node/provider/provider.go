package provider

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/xtls/xray-core/app/proxyman/command"
	routercommand "github.com/xtls/xray-core/app/router/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"vpn/internal/control/api/interceptors"
	"vpn/internal/control/node/model"
)

type Repository interface {
	List(context.Context) ([]model.Node, error)
}

type Node struct {
	model.Node
	xray      command.HandlerServiceClient
	xrayRoute routercommand.RoutingServiceClient
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
	host, _, err := net.SplitHostPort(node.Address)
	if err != nil {
		return err
	}

	xrayConn, err := grpc.NewClient(net.JoinHostPort(host, "9093"),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(interceptors.Timeout(30*time.Second)),
	)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if old, ok := p.nodes[node.ID]; ok {
		_ = old.xrayConn.Close()
	}

	p.nodes[node.ID] = &Node{
		Node:      node,
		xray:      command.NewHandlerServiceClient(xrayConn),
		xrayRoute: routercommand.NewRoutingServiceClient(xrayConn),
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
		_ = node.xrayConn.Close()

		delete(p.nodes, id)
	}
}

func (p *Provider) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, node := range p.nodes {
		_ = node.xrayConn.Close()

		delete(p.nodes, id)
	}
}
