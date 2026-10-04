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
	nodehandlers "vpn/internal/control/node/handlers"
	"vpn/internal/control/node/provider"
	noderepo "vpn/internal/control/node/repo"
	nodeservice "vpn/internal/control/node/service"
	"vpn/internal/control/subscription"
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
	nodes := nodeservice.Service{Repo: noderepo.Postgres{DB: db}, Mu: stateMu}

	nodeProvider, err := provider.New(context.Background(), nodes.Repo)
	if err != nil {
		log.Fatal(err)
	}
	defer nodeProvider.Close()

	nodes.Provider = nodeProvider
	configs := configservice.Service{
		Repo:  configrepo.Postgres{DB: db},
		Nodes: nodes,
		Users: users,
		Mu:    stateMu,
	}

	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			inboundErr := nodes.SyncInbounds(ctx)
			meshErr := nodes.SyncMesh(ctx)
			userErr := configs.SyncUsers(ctx)

			cancel()

			syncErr := errors.Join(inboundErr, meshErr, userErr)
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
	api.RegisterConfigServiceServer(grpcServer, confighandlers.Configs{Service: configs})
	reflection.Register(grpcServer)

	go func() {
		log.Fatal(subscription.Serve(os.Getenv("SUBSCRIPTION_DOMAIN"), configs, users))
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
