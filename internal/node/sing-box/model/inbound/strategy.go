package inbound

type inboundStrategy interface {
	AddUser(inbound *Inbound, name string) error
	RemoveUser(inbound *Inbound, name string) error
}

func (i *Inbound) strategy() inboundStrategy {
	switch i.Type {
	case InboundTypeHysteria2:
		return hysteria2Strategy{}

	default:
		return nil
	}
}

func (i *Inbound) AddUser(name string) error {
	return i.strategy().AddUser(i, name)
}

func (i *Inbound) RemoveUser(name string) error {
	return i.strategy().RemoveUser(i, name)
}
