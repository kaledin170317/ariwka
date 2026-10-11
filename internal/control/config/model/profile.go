package model

type Profile struct {
	ID       string
	UserID   string
	NodeID   string
	Inbound  string
	Password string
	DeviceID string
}

type Device struct {
	ID             string
	UserID         string
	Name           string
	HWID           string
	DefaultNetwork bool
}

type Connection struct {
	ID   string
	Link string
}
