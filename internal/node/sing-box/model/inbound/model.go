package inbound

type Connection interface {
	GetTag() string
	GetPort() int32
}
