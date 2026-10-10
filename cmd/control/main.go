package main

import (
	"context"
	"errors"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	api "vpn/gen/control"
	"vpn/internal/control/api/interceptors"
	authhandlers "vpn/internal/control/auth/handlers"
	authrepo "vpn/internal/control/auth/repo"
	authservice "vpn/internal/control/auth/service"
	confighandlers "vpn/internal/control/config/handlers"
	configrepo "vpn/internal/control/config/repo"
	configservice "vpn/internal/control/config/service"
	"vpn/internal/control/happ"
	nodehandlers "vpn/internal/control/node/handlers"
	inboundhandlers "vpn/internal/control/node/inbound/handlers"
	inboundrepo "vpn/internal/control/node/inbound/repo"
	inboundservice "vpn/internal/control/node/inbound/service"
	"vpn/internal/control/node/provider"
	noderepo "vpn/internal/control/node/repo"
	nodeservice "vpn/internal/control/node/service"
	userhandlers "vpn/internal/control/user/handlers"
	userrepo "vpn/internal/control/user/repo"
	userservice "vpn/internal/control/user/service"
	"vpn/pkg/auth"
	"vpn/pkg/database"
	"vpn/pkg/env"
	"vpn/pkg/grpcui"
)

func main() {
	db, err := database.OpenControl()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	jwt := auth.JWT{Secret: []byte(env.GetJWTSecret())}
	usersDB := userrepo.Postgres{DB: db}
	users := userservice.Service{Repo: usersDB}
	stateMu := &sync.Mutex{}
	nodesDB := noderepo.Postgres{DB: db}
	inboundsDB := inboundrepo.Postgres{DB: db}
	nodes := nodeservice.Service{Repo: nodesDB, Inbounds: inboundsDB, Mu: stateMu}

	nodeProvider, err := provider.New(context.Background(), nodes.Repo)
	if err != nil {
		log.Fatal(err)
	}
	defer nodeProvider.Close()

	nodes.Provider = nodeProvider
	inbounds := inboundservice.InboundService{Repo: inboundsDB, Nodes: nodes, Mu: stateMu}
	configs := configservice.Service{
		Repo:     configrepo.Postgres{DB: db},
		Nodes:    nodes,
		Inbounds: inbounds,
		Users:    users,
		Mu:       stateMu,
	}

	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defaultErr := inbounds.SyncDefault(ctx)
			inboundErr := inbounds.SyncInbounds(ctx)
			userErr := configs.SyncUsers(ctx)

			cancel()

			syncErr := errors.Join(defaultErr, inboundErr, userErr)
			if syncErr != nil {
				log.Printf("node sync: %v", syncErr)
			}

			time.Sleep(time.Minute)
		}
	}()

	login := authservice.Service{
		Repo:  authrepo.Postgres{DB: db},
		JWT:   jwt,
		Users: users,
	}
	access := interceptors.Auth{JWT: jwt, Users: users}
	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(access.Intercept))
	api.RegisterUserServiceServer(grpcServer, userhandlers.Users{Service: users})
	api.RegisterAuthServiceServer(grpcServer, authhandlers.Auth{Service: login})
	api.RegisterNodeServiceServer(grpcServer, nodehandlers.Nodes{Service: nodes})
	api.RegisterNodeInboundServiceServer(grpcServer, inboundhandlers.Inbounds{Service: inbounds})
	api.RegisterConfigServiceServer(grpcServer, confighandlers.Configs{Service: configs})
	reflection.Register(grpcServer)

	go func() {
		log.Fatal(happ.Serve(os.Getenv("SUBSCRIPTION_DOMAIN"), configs, users))
	}()

	listener, err := net.Listen("tcp", ":9092")
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err = grpcServer.Serve(listener)
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("gRPC :9092, grpcui http://localhost:8082")
	log.Fatal(grpcui.Serve(":8082", "127.0.0.1:9092"))
}
