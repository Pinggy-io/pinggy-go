package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

type UdpDialer interface {
	Dialer
	Dial() (*net.UDPConn, error)
}
type udpTunnel struct {
	packetConn *net.UDPConn
	streamConn net.Conn
	toAddr     net.Addr
}

func (c *udpTunnel) close() {
	c.packetConn.Close()
	c.streamConn.Close()
}

func (c *udpTunnel) copyToTcp() {
	defer c.close()
	buffer := make([]byte, 2048)
	for {
		n, _, err := c.packetConn.ReadFrom(buffer)
		if err != nil {
			break
		}
		if n <= 0 {
			break
		}
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(n))
		packet := append(lengthBytes, buffer[:n]...)
		fmt.Println("Writing ", n+2, "bytes to TCP")
		_, err = c.streamConn.Write(packet)
		if err != nil {
			log.Println("Error while writing packet to tcp, ", err)
			break
		}
	}
}

func (c *udpTunnel) copyToUdp() {
	defer c.close()
	buffer := make([]byte, 2048)
	for {
		// Read the length of the UDP packet
		_, err := io.ReadFull(c.streamConn, buffer[:2])
		if err != nil {
			break
		}

		// Extract the length information
		length := binary.BigEndian.Uint16(buffer[:2])

		// Read the rest of the UDP packet
		_, err = io.ReadFull(c.streamConn, buffer[:length])
		if err != nil {
			break
		}

		fmt.Println("Writing ", length, "bytes to UDP", c.toAddr.String())

		// Write the data to the TCP connection
		_, err = c.packetConn.Write(buffer[:length])
		if err != nil {
			log.Println("Error while writing packet to udp, ", err)
			break
		}
	}
}

type udpDialer struct {
	udpAddr *net.UDPAddr
}

func (u *udpDialer) Dial() (*net.UDPConn, error) {
	return net.DialUDP("udp", nil, u.udpAddr)
}

func (u *udpDialer) GetAddr() net.Addr {
	return u.udpAddr
}

func (u *udpDialer) UpdateAddr(addr net.Addr) {
	if addr == nil {
		return
	}
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok {
		return
	}
	u.udpAddr = udpAddr
}

type udpTunnelManager struct {
	dialer       UdpDialer
	connListener net.Listener
}

func (t *udpTunnelManager) StartTunnel(streamConn net.Conn) {
	packetConn, err := t.dialer.Dial()
	if err != nil {
		streamConn.Close()
		return
	}
	tun := udpTunnel{packetConn: packetConn, streamConn: streamConn, toAddr: t.dialer.GetAddr()}
	fmt.Println("Fowarding new con")
	go tun.copyToTcp()
	tun.copyToUdp()
}

func (t *udpTunnelManager) AcceptAndForward() error {
	conn, err := t.connListener.Accept()
	if err != nil {
		return err
	}

	go t.StartTunnel(conn)
	return nil
}

func (t *udpTunnelManager) StartForwarding() {
	for {
		err := t.AcceptAndForward()
		if err != nil {
			break
		}
	}
}

func (u *udpTunnelManager) GetDialer() Dialer {
	return u.dialer
}

func NewUdpDialer(forwardAddr *net.UDPAddr) UdpDialer {
	return &udpDialer{udpAddr: forwardAddr}
}

func NewUdpTunnelMangerWithDialer(listener net.Listener, dialer UdpDialer) TunnelManager {
	tunMan := &udpTunnelManager{connListener: listener, dialer: dialer}
	return tunMan
}

func NewUdpTunnelMangerAddr(listener net.Listener, forwardAddr *net.UDPAddr) TunnelManager {
	return NewUdpTunnelMangerWithDialer(listener, &udpDialer{udpAddr: forwardAddr})
}

func NewUdpTunnelManger(listener net.Listener, forwardAddr string) (TunnelManager, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", forwardAddr)
	if err != nil {
		return nil, err
	}
	return NewUdpTunnelMangerAddr(listener, udpAddr), nil
}

func NewUdpTunnelMangerListen(listeningPort int, forwardAddr string) (TunnelManager, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", forwardAddr)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", listeningPort))
	if err != nil {
		return nil, err
	}
	fmt.Println("Listening: ", listeningPort)
	return NewUdpTunnelMangerAddr(listener, udpAddr), nil
}
