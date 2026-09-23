package main

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	swaggerdocs "vpn/docs"
	api "vpn/gen/node"
	"vpn/internal/node/sing-box/handlers"
	"vpn/internal/node/sing-box/service"
	swaggerui "vpn/pkg/swagger"
)

func main() {
	singBox := service.NewSingBoxService()

	if err := singBox.Sync(); err != nil {
		log.Fatal(err)
	}

	vpnHandler := handlers.NewVPNHandler(singBox)

	// grpc-gateway
	gatewayMux := runtime.NewServeMux()

	if err := api.RegisterVPNServiceHandlerServer(
		context.Background(),
		gatewayMux,
		vpnHandler,
	); err != nil {
		log.Fatal(err)
	}

	// Обычный HTTP mux
	httpMux := http.NewServeMux()

	// Swagger
	swaggerui.Register(httpMux, swaggerdocs.Node)

	// Все остальные запросы -> grpc-gateway
	httpMux.Handle("/", gatewayMux)

	log.Println("Listening on :8081")
	log.Println("Swagger: http://localhost:8081/swagger/node/")

	if err := http.ListenAndServe(":8081", httpMux); err != nil {
		log.Fatal(err)
	}
}
