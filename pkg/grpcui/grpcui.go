package grpcui

import (
	"context"
	"net/http"

	"github.com/fullstorydev/grpcui/standalone"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Serve(address, target string) error {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	handler, err := standalone.HandlerViaReflection(context.Background(), conn, target)
	if err != nil {
		return err
	}

	return http.ListenAndServe(address, handler)
}
