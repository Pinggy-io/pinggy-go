package tunnel

import "net"

type Dialer interface {
	GetAddr() net.Addr
	UpdateAddr(net.Addr)
}

type TunnelManager interface {
	StartForwarding()
	AcceptAndForward() error
	GetDialer() Dialer
}
