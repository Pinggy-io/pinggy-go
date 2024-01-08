package socks

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type strippedConn struct {
	conn net.Conn
	addr net.Addr
	err  error
}

type socksStriper struct {
	listener net.Listener

	udpConnections chan *strippedConn
	tcpConnections chan *strippedConn
}

func (s *socksStriper) StripSockFromConn(clientConn net.Conn) (addr net.Addr, cType ConnType, err error) {
	// defer clientConn.Close()
	addr, cType, err = nil, ConnType_NONE, nil

	// Perform handshake
	version, nmethods, err := readHandshake(clientConn)
	if err != nil {
		log.Println("Error during handshake:", err)
		return
	}

	// Only support SOCKS5
	if version != 5 {
		err = fmt.Errorf("unsupported socks version")
		log.Println("Unsupported SOCKS version:", version)
		return
	}

	methods := make([]byte, nmethods)
	// Read and discard methods
	_, err = io.ReadFull(clientConn, methods)
	if err != nil {
		log.Println("Error reading methods:", err)
		return
	}
	acceptedMethod := byte(255)
	for m := range methods {
		if m == 0 {
			acceptedMethod = 0
		}
	}
	// Respond to the client with a "no authentication required" message
	_, err = clientConn.Write([]byte{5, acceptedMethod})
	if err != nil {
		log.Println("Error responding to client:", err)
		return
	}

	if acceptedMethod == 255 {
		err = fmt.Errorf("no acceptable authentication found")
		return
	}

	// Read the request
	cmd, addrStr, err := readRequest(clientConn)
	if err != nil {
		log.Println("Error reading request:", err)
		return
	}

	reply := ReplyType_Success

	if SocksCmd_Connect == SocksCmd(cmd) {
		cType = ConnType_TCP
		addr, err = net.ResolveTCPAddr("tcp", addrStr)
		if err != nil {
			reply = ReplyType_AddressTypeNotSupported
		}
	} else if SocksCmd_UdpConnect == SocksCmd(cmd) {
		cType = ConnType_UDP
		addr, err = net.ResolveUDPAddr("udp", addrStr)
		if err != nil {
			reply = ReplyType_AddressTypeNotSupported
		}
	} else {
		err = fmt.Errorf("unsupported command. Ignoring")
		reply = ReplyType_CommandNotSupported
	}

	// Respond to the client that the connection is established
	_, err1 := clientConn.Write([]byte{5, byte(reply), 0, 1, 0, 0, 0, 0, 0, 0})
	if err1 != nil {
		err = err1
		log.Println("Error responding to client:", err)
		return
	}

	log.Println("Striping done")
	return
}

func (s *socksStriper) AcceptAndStripSock(listener net.Listener) (clientConn net.Conn, addr net.Addr, cType ConnType, err error) {
	clientConn, addr, cType, err = nil, nil, ConnType_NONE, nil

	clientConn, err = listener.Accept()
	if err != nil {
		log.Println("Error while accepting a connection: ", err)
		return
	}

	addr, cType, err = s.StripSockFromConn(clientConn)
	if err != nil {
		clientConn.Close()
		clientConn = nil
		return
	}

	return
}

func (s *socksStriper) Start() {
	for {
		clientConn, err := s.listener.Accept()
		if err != nil {
			log.Println("Error while accepting a connection: ", err)
			s.udpConnections <- &strippedConn{err: err}
			s.tcpConnections <- &strippedConn{err: err}
			return
		}

		go func(clientConn net.Conn) {
			log.Println("Connection accepted")
			addr, cType, err := s.StripSockFromConn(clientConn)
			if err != nil {
				clientConn.Close()
				clientConn = nil
				log.Println("Error while striping: ", err)
				return
			}
			log.Println("Connection striped, ", addr, " ", cType, " ")
			if ConnType_UDP == cType {
				s.udpConnections <- &strippedConn{conn: clientConn, addr: addr}
			} else if ConnType_TCP == cType {
				s.tcpConnections <- &strippedConn{conn: clientConn, addr: addr}
			}
		}(clientConn)
	}
}

func (s *socksStriper) AcceptTcp() (net.Conn, net.Addr, error) {
	log.Println("Trying to accept tcp")
	sock := <-s.tcpConnections
	return sock.conn, sock.addr, sock.err
}

func (s *socksStriper) AcceptUdp() (net.Conn, net.Addr, error) {
	log.Println("Trying to accept Udp")
	sock := <-s.udpConnections
	return sock.conn, sock.addr, sock.err
}

func (s *socksStriper) Accept() (net.Conn, error) {
	c, _, err := s.AcceptTcp()
	return c, err
}

func (s *socksStriper) Close() error {
	return s.listener.Close()
}

func (s *socksStriper) Addr() net.Addr {
	return s.listener.Addr()
}

func InitiatateSocks5u(listener net.Listener) Socks5u {
	return &socksStriper{
		listener:       listener,
		udpConnections: make(chan *strippedConn, 5),
		tcpConnections: make(chan *strippedConn, 5),
	}
}

func readHandshake(conn net.Conn) (version, nmethods byte, err error) {
	handshake := make([]byte, 2)
	_, err = io.ReadFull(conn, handshake)
	if err != nil {
		return
	}
	version = handshake[0]
	nmethods = handshake[1]
	return
}

func readRequest(conn net.Conn) (cmd byte, addr string, err error) {
	request := make([]byte, 5)
	_, err = io.ReadFull(conn, request)
	if err != nil {
		return
	}

	cmd = request[1]

	switch request[3] {
	case 1: // IPv4 address
		ip := make([]byte, 5) //1 byte already read
		_, err = io.ReadFull(conn, ip)
		if err != nil {
			return
		}
		addr = fmt.Sprintf("%d.%d.%d.%d:%d", request[4], ip[0], ip[1], ip[2], (int(request[3])<<8)+int(request[4]))
	case 3: // Domain name
		domainLen := int(request[4])
		domain := make([]byte, domainLen+2)
		_, err = io.ReadFull(conn, domain)
		if err != nil {
			return
		}
		addr = fmt.Sprintf("%s:%d", domain[:domainLen], (int(domain[domainLen])<<8)+int(domain[domainLen+1]))
	case 4: // IPv6 address
		ip := make([]byte, 18) //1byte already read
		ip[0] = request[4]
		_, err = io.ReadFull(conn, ip[:17])
		if err != nil {
			return
		}

		addr = fmt.Sprintf("[%s]:%d", formatIPv6(ip), (int(ip[16])<<8)+int(ip[17]))
	default:
		err = fmt.Errorf("unsupported address type: %d", request[3])
	}
	return
}

func formatIPv6(ip []byte) string {
	var sections []string
	for i := 0; i < 16; i += 2 {
		sections = append(sections, fmt.Sprintf("%x%x", ip[i], ip[i+1]))
	}
	return strings.Join(sections, ":")
}
