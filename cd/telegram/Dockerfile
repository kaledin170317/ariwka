FROM golang:1.27-alpine AS build

RUN apk add --no-cache protobuf ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

COPY api ./api
COPY cmd/telegram ./cmd/telegram
COPY internal/telegram ./internal/telegram
RUN mkdir -p gen \
    && protoc -I api --go_out=gen --go_opt=paths=source_relative \
        --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
        api/control/*.proto api/vpn/common/*.proto \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /telegram ./cmd/telegram

FROM scratch AS binary
COPY --from=build /telegram /bin/telegram

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /telegram /telegram
USER 65532:65532
ENTRYPOINT ["/telegram"]
