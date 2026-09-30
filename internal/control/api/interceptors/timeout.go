package interceptors

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func Timeout(duration time.Duration) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		requestCtx, cancel := context.WithTimeout(ctx, duration)
		defer cancel()

		return invoke(requestCtx, method, req, reply, cc, opts...)
	}
}
