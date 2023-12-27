package udptunnel

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type DatagramDialer interface {
	Dial() (net.PacketConn, error)
	GetAddr() *net.UDPAddr
}

type TunnelManager interface {
	StartForwarding()
	AcceptAndForward() error
}

type tunnel struct {
	packetConn net.PacketConn
	streamConn net.Conn
	toAddr     net.Addr
}

func (c *tunnel) close() {
	c.packetConn.Close()
	c.streamConn.Close()
}

func (c *tunnel) copyToTcp() {
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
			break
		}
	}
}

func (c *tunnel) copyToUdp() {
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

		fmt.Println("Writing ", length, "bytes to UDP")

		// Write the data to the TCP connection
		_, err = c.packetConn.WriteTo(buffer[:length], c.toAddr)
		if err != nil {
			break
		}
	}
}

type udpDialer struct {
	udpAddr *net.UDPAddr
}

func (u *udpDialer) Dial() (net.PacketConn, error) {
	return net.DialUDP("udp", nil, u.udpAddr)
}

func (u *udpDialer) GetAddr() *net.UDPAddr {
	return u.udpAddr
}

type tunnelManager struct {
	dialer       DatagramDialer
	connListener net.Listener
}

func (t *tunnelManager) StartTunnel(streamConn net.Conn) {
	packetConn, err := t.dialer.Dial()
	if err != nil {
		streamConn.Close()
		return
	}
	tun := tunnel{packetConn: packetConn, streamConn: streamConn, toAddr: t.dialer.GetAddr()}
	fmt.Println("Fowarding new con")
	go tun.copyToTcp()
	tun.copyToUdp()
}

func (t *tunnelManager) AcceptAndForward() error {
	conn, err := t.connListener.Accept()
	if err != nil {
		return err
	}

	go t.StartTunnel(conn)
	return nil
}

func (t *tunnelManager) StartForwarding() {
	for {
		err := t.AcceptAndForward()
		if err != nil {
			break
		}
	}
}

func NewTunnelManger(listener net.Listener, forwardAddr *net.UDPAddr) TunnelManager {
	tunMan := &tunnelManager{connListener: listener, dialer: &udpDialer{udpAddr: forwardAddr}}
	return tunMan
}

func NewTunnelMangerAddr(listener net.Listener, forwardAddr string) (TunnelManager, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", forwardAddr)
	if err != nil {
		return nil, err
	}
	return NewTunnelManger(listener, udpAddr), nil
}

func NewTunnelMangerListen(listeningPort int, forwardAddr string) (TunnelManager, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", forwardAddr)
	if err != nil {
		return nil, err
	}
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", listeningPort))
	if err != nil {
		return nil, err
	}
	fmt.Println("Listening: ", listeningPort)
	return NewTunnelManger(listener, udpAddr), nil
}
