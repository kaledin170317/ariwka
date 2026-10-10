package proxy

import (
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Codec forwards protobuf bytes without decoding Xray's messages.
type Codec struct{}

func (Codec) Name() string { return "proto" }

func (Codec) Marshal(value any) ([]byte, error) {
	message, ok := value.(*[]byte)
	if !ok {
		return nil, fmt.Errorf("unsupported gRPC message %T", value)
	}

	return *message, nil
}

func (Codec) Unmarshal(data []byte, value any) error {
	message, ok := value.(*[]byte)
	if !ok {
		return fmt.Errorf("unsupported gRPC message %T", value)
	}

	*message = append((*message)[:0], data...)

	return nil
}

type Proxy struct{ Connection *grpc.ClientConn }

func (p Proxy) Handle(_ any, stream grpc.ServerStream) error {
	method, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		return errors.New("gRPC method missing")
	}

	var request []byte

	err := stream.RecvMsg(&request)
	if err != nil {
		return err
	}

	ctx := stream.Context()
	if headers, found := metadata.FromIncomingContext(ctx); found {
		ctx = metadata.NewOutgoingContext(ctx, headers.Copy())
	}

	var response []byte

	err = p.Connection.Invoke(ctx, method, &request, &response, grpc.ForceCodec(Codec{}))
	if err != nil {
		return err
	}

	return stream.SendMsg(&response)
}
