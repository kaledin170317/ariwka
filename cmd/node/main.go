package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"vpn/internal/node/xray/proxy"
)

func main() {
	xray, err := grpc.NewClient("127.0.0.1:10085", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer xray.Close()

	xrayServer := grpc.NewServer(
		grpc.ForceServerCodec(proxy.Codec{}),
		grpc.UnknownServiceHandler((proxy.Proxy{Connection: xray}).Handle),
	)

	xrayListener, err := net.Listen("tcp", ":9093")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Xray gRPC proxy :9093")
	log.Fatal(xrayServer.Serve(xrayListener))
}
