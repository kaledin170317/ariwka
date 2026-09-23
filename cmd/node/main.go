package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	api "vpn/gen/node"
	"vpn/internal/node/sing-box/handlers"
	"vpn/internal/node/sing-box/service"
	"vpn/pkg/grpcui"
)

func main() {
	singBox := service.NewSingBoxService()
	err := singBox.Sync()
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer()
	api.RegisterVPNServiceServer(server, handlers.NewVPNHandler(singBox))
	reflection.Register(server)

	listener, err := net.Listen("tcp", ":9091")
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		err = server.Serve(listener)
		if err != nil {
			log.Fatal(err)
		}
	}()

	log.Println("gRPC :9091, grpcui http://localhost:8081")
	log.Fatal(grpcui.Serve(":8081", "127.0.0.1:9091"))
}
