FROM golang:1.27-alpine AS build

RUN apk add --no-cache protobuf ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12 \
    && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

COPY api ./api
COPY cmd/control ./cmd/control
COPY internal/control ./internal/control
COPY pkg ./pkg
RUN mkdir -p gen \
    && protoc -I api --go_out=gen --go_opt=paths=source_relative \
        --go-grpc_out=gen --go-grpc_opt=paths=source_relative \
        api/control/*.proto api/vpn/common/*.proto \
    && go generate ./internal/control/node/model ./internal/control/api/interceptors \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /control ./cmd/control \
    && CGO_ENABLED=0 go install -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.0

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /control /control
COPY --from=build /go/bin/migrate /migrate
ENTRYPOINT ["/control"]
