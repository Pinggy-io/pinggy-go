package pinggy

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

type packet struct {
	bytes  []byte
	addr   net.Addr
	closed bool
}

type udpTunnel struct {
	addr net.Addr
	conn net.Conn

	writeChannel chan []byte
	closeChannel chan bool
	closed       bool
	pfh          *packetForwardingHandler
}

type packetForwardingHandler struct {
	list        net.Listener
	port        uint16
	readChannel chan *packet
	tunnels     map[string]udpTunnel
}

func (t *udpTunnel) close() {
	t.closed = true
	t.closeChannel <- true
	t.conn.Close()
}

func (t *udpTunnel) copyToTcp() {
	defer t.close()
	for {
		select {
		case buffer := <-t.writeChannel:
			n := len(buffer)
			if n <= 0 {
				log.Println("Error")
				return
			}
			lengthBytes := make([]byte, 2)
			binary.BigEndian.PutUint16(lengthBytes, uint16(n))
			packet := append(lengthBytes, buffer[:n]...)
			fmt.Println("Writing ", n+2, "bytes to TCP")
			_, err := t.conn.Write(packet)
			if err != nil {
				log.Println("Error")
				return
			}
		case <-t.closeChannel:
			log.Println("Closed")
			return
		}
	}
}

func (t *udpTunnel) copyToUdp() {
	defer t.close()
	buffer := make([]byte, 2048)
	for {
		_, err := io.ReadFull(t.conn, buffer[:2])
		if err != nil {
			log.Println("Error")
			return
		}

		// Extract the length information
		length := binary.BigEndian.Uint16(buffer[:2])

		// Read the rest of the UDP packet
		_, err = io.ReadFull(t.conn, buffer[:length])
		if err != nil {
			log.Println("Error")
			return
		}

		fmt.Println("Writing ", length, "bytes to UDP")

		if t.closed {
			return
		}

		t.pfh.readChannel <- &packet{buffer[:length], t.addr, false} //FIXME
	}
}

func (pfh *packetForwardingHandler) startTunnel(conn net.Conn) {
	pfh.port += 1
	if pfh.port == 0 {
		pfh.port += 1
	}
	tun := udpTunnel{
		conn:         conn,
		addr:         &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: int(pfh.port)}, //FIXME
		pfh:          pfh,
		writeChannel: make(chan []byte, 20),
		closeChannel: make(chan bool, 3),
		closed:       false,
	}
	pfh.tunnels[tun.addr.String()] = tun
	log.Println("Starting tunnel")
	go tun.copyToTcp()
	tun.copyToUdp()
}

func (pfh *packetForwardingHandler) startForwarding() error {
	log.Println("starting forwarding")
	for {
		conn, err := pfh.list.Accept()
		if err != nil {
			log.Println("Error occured")
			pfh.readChannel <- &packet{nil, nil, true}
			return err
		}
		go pfh.startTunnel(conn)
	}
}

func (pfh *packetForwardingHandler) writeTo(b []byte, addr net.Addr) {
	tun, ok := pfh.tunnels[addr.String()]
	if !ok {
		return
	}
	if tun.closed {
		return
	}
	tun.writeChannel <- b
}
