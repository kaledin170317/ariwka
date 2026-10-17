FROM golang:1.27-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY cmd/node ./cmd/node
COPY internal/node/xray/proxy ./internal/node/xray/proxy
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /node ./cmd/node

FROM scratch
COPY --from=build /node /node
USER 65532:65532

ENTRYPOINT ["/node"]
