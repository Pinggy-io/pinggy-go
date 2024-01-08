package pinggy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/Pinggy-io/pinggy-go/pinggy/socks"
	"github.com/Pinggy-io/pinggy-go/pinggy/tunnel"
	"golang.org/x/crypto/ssh"
)

type pinggyListener struct {
	conf          *Config
	clientConn    *ssh.Client
	listener      net.Listener
	udpListener   net.Listener
	session       *ssh.Session
	debugListener net.Listener
	udpChannel    bool
	tcpChannel    bool
	closed        bool

	tcpDialer tunnel.TcpDialer
	udpDialer tunnel.UdpDialer

	udpHandler *packetForwardingHandler
}

type udpListenerWrapper struct {
	udpListener socks.Socks5u
}

func (ul *udpListenerWrapper) Accept() (net.Conn, error) {
	conn, _, err := ul.udpListener.AcceptUdp()
	return conn, err
}

func (ul *udpListenerWrapper) Close() error {
	return ul.udpListener.Close()
}

func (ul *udpListenerWrapper) Addr() net.Addr {
	return ul.udpListener.Addr()
}

// func (pl *pinggyListener) isSocks() bool { return pl.udpChannel && pl.tcpChannel }

func (pl *pinggyListener) getConnectionUrl() []string {
	logger := pl.conf.logger

	conn, err := pl.clientConn.Dial("tcp", "localhost:4300")
	if err != nil {
		logger.Println("Error connecting the server:", err)
		return nil
	}

	req, err := http.NewRequest("GET", "http://localhost:4300/urls", nil)
	if err != nil {
		logger.Println("Error creating request:", err)
		return nil
	}
	err = req.Write(conn)
	if err != nil {
		logger.Println("Error sending request:", err)
		return nil
	}

	// Read the HTTP response
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		logger.Println("Error reading response:", err)
		return nil
	}
	defer resp.Body.Close()

	// Print the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Println("Error reading body:", err)
		return nil
	}

	urls := make(map[string][]string)
	err = json.Unmarshal(body, &urls)

	if err != nil {
		logger.Println("Error parsing body:", err)
		return nil
	}
	logger.Println(urls)
	return urls["urls"]
}
func (pl *pinggyListener) Accept() (net.Conn, error) {
	if pl.udpHandler != nil {
		return nil, fmt.Errorf("not allowed")
	}

	if pl.tcpDialer != nil || pl.udpDialer != nil {
		return nil, fmt.Errorf("automatic tcp forwarding enabled")
	}

	return pl.listener.Accept()
}

func (pl *pinggyListener) Close() error {
	err := pl.listener.Close()
	if pl.debugListener != nil {
		pl.debugListener.Close()
		pl.debugListener = nil
	}
	if pl.session != nil {
		pl.session.Close()
		pl.session = nil
	}
	pl.clientConn.Close()
	return err
}

func (pl *pinggyListener) Addr() net.Addr { return pl.listener.Addr() }

func (pl *pinggyListener) RemoteUrls() []string {
	urls := pl.getConnectionUrl()
	if urls == nil {
		return make([]string, 0)
	}
	return urls
}

func (pl *pinggyListener) InitiateWebDebug(addr string) error {
	if pl.conf.Type != HTTP {
		return fmt.Errorf("webDebugging is available only with %v mode", HTTP)
	}
	if pl.session == nil {
		session, err := pl.clientConn.NewSession()
		if err != nil {
			pl.conf.logger.Println("Cannot initiate WebDebug")
			return err
		}
		err = session.Shell()
		if err != nil {
			pl.conf.logger.Println("Cannot initiate WebDebug")
			return err
		}
		pl.session = session
	}
	if pl.debugListener != nil {
		return fmt.Errorf("webDebugging is already running at %v", pl.debugListener.Addr().String())
	}
	webListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		for {
			conn, err := webListener.Accept()
			if err != nil {
				pl.conf.logger.Println(err)
				return
			}
			conn2, err := pl.clientConn.Dial("tcp", "localhost:4300")
			if err != nil {
				conn.Close()
				pl.conf.logger.Println(err)
				return
			}
			go io.Copy(conn, conn2)
			go io.Copy(conn2, conn)
		}
	}()
	pl.debugListener = webListener
	return nil
}

func (pl *pinggyListener) ServeHttp(fs fs.FS) error {
	httpfs := http.FS(fs)

	server := http.Server{}
	server.Handler = http.FileServer(httpfs)
	return server.Serve(pl.listener)
}

// net.PacketConn
func (pl *pinggyListener) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	if pl.udpHandler == nil {
		return -1, nil, fmt.Errorf("not allowed")
	}
	if pl.closed {
		return 0, nil, io.EOF
	}
	pkt := <-pl.udpHandler.readChannel
	if pkt.closed {
		pl.closed = true
		return 0, nil, io.EOF
	}
	l := copy(p, pkt.bytes)
	return l, pkt.addr, nil
}

func (pl *pinggyListener) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	if pl.udpHandler == nil {
		return -1, fmt.Errorf("not allowed")
	}
	pl.udpHandler.writeTo(p, addr)
	return n, nil
}

func (pl *pinggyListener) LocalAddr() net.Addr {
	if pl.udpHandler == nil {
		return nil
	}
	return pl.Addr()
}

func (pl *pinggyListener) SetDeadline(t time.Time) error {
	if pl.udpHandler == nil {
		return fmt.Errorf("not allowed")
	}
	return fmt.Errorf("not implemented")
}

func (pl *pinggyListener) SetReadDeadline(t time.Time) error {
	if pl.udpHandler == nil {
		return fmt.Errorf("not allowed")
	}
	return fmt.Errorf("not implemented")
}

func (pl *pinggyListener) SetWriteDeadline(t time.Time) error {
	if pl.udpHandler == nil {
		return fmt.Errorf("not allowed")
	}
	return fmt.Errorf("not implemented")
}

func (pl *pinggyListener) UpdateTcpForwarding(addr string) error {
	if pl.tcpDialer == nil {
		return fmt.Errorf("this function can be used only to chenge the target address")
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return nil
	}

	pl.tcpDialer.UpdateAddr(tcpAddr)
	return nil
}

func (pl *pinggyListener) UpdateUdpForwarding(addr string) error {
	if pl.udpDialer == nil {
		return fmt.Errorf("this function can be used only to chenge the target address")
	}
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil
	}

	pl.udpDialer.UpdateAddr(udpAddr)
	return nil
}

func setupPinggyTunnel(conf Config) (*pinggyListener, error) {
	clientConn, err := dialWithConfig(&conf)
	if err != nil {
		conf.logger.Printf("Error in ssh connection initiation: %v\n", err)
		return nil, err
	}

	conf.logger.Println("Ssh connection initiated. Setting up reverse tunnel")
	listener, err := clientConn.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		clientConn.Close()
		conf.logger.Printf("Error in ssh tunnel initiation: %v\n", err)
		return nil, err
	}

	var udpListener net.Listener = listener

	if conf.Type != "" && conf.AltType != "" {
		socksListener := socks.InitiatateSocks5u(listener)
		udpListener = &udpListenerWrapper{udpListener: socksListener}
		listener = socksListener
		go socksListener.Start()
	}

	list := &pinggyListener{
		listener:    listener,
		udpListener: udpListener,
		clientConn:  clientConn,
		conf:        &conf,
		tcpChannel:  conf.Type != "",
		udpChannel:  conf.AltType != "",
		closed:      false,

		tcpDialer: nil,
		udpDialer: nil,
	}

	if conf.ForwardTcpTo != "" {
		addr, err := net.ResolveTCPAddr("tcp", conf.ForwardTcpTo)
		if err != nil {
			list.clientConn.Close()
			return nil, err
		}
		list.tcpDialer = tunnel.NewTcpDialer(addr)
	}

	if conf.ForwardUdpTo != "" {
		addr, err := net.ResolveUDPAddr("udp", conf.ForwardUdpTo)
		if err != nil {
			list.clientConn.Close()
			return nil, err
		}
		list.udpDialer = tunnel.NewUdpDialer(addr)
	}

	if list.udpChannel && list.udpDialer == nil {
		list.udpHandler = &packetForwardingHandler{
			list:        list.udpListener,
			readChannel: make(chan *packet, 50),
			tunnels:     make(map[string]udpTunnel),
		}
		go list.udpHandler.startForwarding()
	}

	return list, nil
}

func (pl *pinggyListener) StartForwarding() error {
	var wg sync.WaitGroup
	forwarding := false
	//add socks here
	if pl.udpChannel && pl.udpDialer != nil {
		forwarding = true
		wg.Add(1)
		go func(pl *pinggyListener, wg *sync.WaitGroup) {
			defer wg.Done()
			udpTunnelMan := tunnel.NewUdpTunnelMangerWithDialer(pl.udpListener, pl.udpDialer)
			udpTunnelMan.StartForwarding()
		}(pl, &wg)
	}
	if pl.tcpChannel && pl.tcpDialer != nil {
		forwarding = true
		wg.Add(1)
		go func(pl *pinggyListener, wg *sync.WaitGroup) {
			defer wg.Done()
			tcpTunnelMan := tunnel.NewTcpTunnelMangerDialer(pl.listener, pl.tcpDialer)
			tcpTunnelMan.StartForwarding()
		}(pl, &wg)
	}
	if !forwarding {
		return fmt.Errorf("nothing to forward")
	}
	wg.Wait()
	return nil
}
