package main

import (
	"context"
	"log"
	"net"

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
	usersDB := userrepo.SQLite{DB: db}
	users := userservice.Service{Repo: usersDB}
	nodes := nodeservice.Service{Repo: noderepo.SQLite{DB: db}}
	nodeProvider, err := provider.New(context.Background(), nodes.Repo)
	if err != nil {
		log.Fatal(err)
	}
	defer nodeProvider.Close()
	nodes.Provider = nodeProvider
	configs := configservice.Service{Repo: configrepo.SQLite{DB: db}, Nodes: nodes, Users: users}
	login := authservice.Service{
		Repo:  authrepo.SQLite{DB: db},
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
