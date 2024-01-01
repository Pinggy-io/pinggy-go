package tunnel

import (
	"io"
	"log"
	"net"
)

type TcpDialer interface {
	Dialer
	Dial() (net.Conn, error)
}

type tcpDialer struct {
	addr *net.TCPAddr
}

type tcpTunnelManager struct {
	dialer       TcpDialer
	connListener net.Listener
}

func (t *tcpDialer) Dial() (net.Conn, error) {
	return net.DialTCP("tcp", nil, t.addr)
}

func (t *tcpDialer) GetAddr() net.Addr {
	return t.addr
}

func (t *tcpTunnelManager) copy(dst, src net.Conn) {
	defer src.Close()
	defer dst.Close()
	io.Copy(dst, src)
}

func (t *tcpTunnelManager) StartTunnel(streamConn net.Conn) {
	conn, err := t.dialer.Dial()
	if err != nil {
		streamConn.Close()
		log.Println("Error: could not connect to ", t.dialer.GetAddr().String())
		return
	}
	go t.copy(streamConn, conn)
	t.copy(conn, streamConn)
}

func (t *tcpDialer) UpdateAddr(addr net.Addr) {
	if addr == nil {
		return
	}
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return
	}
	t.addr = tcpAddr
}

func NewTcpDialer(tcpAddr *net.TCPAddr) TcpDialer {
	return &tcpDialer{addr: tcpAddr}
}

func (t *tcpTunnelManager) AcceptAndForward() error {
	conn, err := t.connListener.Accept()
	if err != nil {
		return err
	}
	go t.StartTunnel(conn)
	return nil
}

func (t *tcpTunnelManager) StartForwarding() {
	for {
		err := t.AcceptAndForward()
		if err != nil {
			log.Println("Error: could not Accept and forward ", t.dialer.GetAddr().String())
			break
		}
	}
}

func (t *tcpTunnelManager) GetDialer() Dialer {
	return t.dialer
}

func NewTcpTunnelMangerDialer(listener net.Listener, dialer TcpDialer) TunnelManager {
	return &tcpTunnelManager{connListener: listener, dialer: dialer}
}

func NewTcpTunnelMangerAddr(listener net.Listener, forwardAddr *net.TCPAddr) TunnelManager {
	return &tcpTunnelManager{connListener: listener, dialer: NewTcpDialer(forwardAddr)}
}

func NewTcpTunnelManger(listener net.Listener, forwardAddr string) (TunnelManager, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", forwardAddr)
	if err != nil {
		return nil, err
	}
	return NewTcpTunnelMangerAddr(listener, tcpAddr), nil
}
