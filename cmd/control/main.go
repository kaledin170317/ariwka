package main

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	swaggerdocs "vpn/docs"
	api "vpn/gen/control"
	swaggerui "vpn/pkg/swagger"
)

func main() {
	gatewayMux := runtime.NewServeMux()

	if err := api.RegisterUserServiceHandlerServer(
		context.Background(),
		gatewayMux,
		api.UnimplementedUserServiceServer{},
	); err != nil {
		log.Fatal(err)
	}

	httpMux := http.NewServeMux()

	swaggerui.Register(httpMux, swaggerdocs.Control)

	httpMux.Handle("/", gatewayMux)

	log.Println("Listening on :8082")
	log.Println("Swagger: http://localhost:8082/swagger/control/")

	if err := http.ListenAndServe(":8082", httpMux); err != nil {
		log.Fatal(err)
	}
}
