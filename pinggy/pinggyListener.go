package pinggy

import (
	"bufio"
	"bytes"
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

// {"ConfigTcp":4,"UrlTcp":8,"UsageContinuousTcp":5,"UsageOnceLongPollTcp":6,"UsageTcp":7}
type pinggyPortConfig struct {
	ConfigTcp            int `json:"ConfigTcp"`            //4
	UsageContinuousTcp   int `json:"UsageContinuousTcp"`   // 5
	UsageOnceLongPollTcp int `json:"UsageOnceLongPollTcp"` // 6
	UsageTcp             int `json:"UsageTcp"`             //7
	UrlTcp               int `json:"UrlTcp"`               //8
	StatusPort           int `json:"StatusPort"`           //12
}

type connectionStatus struct {
	Success       bool   `json:"Success"`
	Authenticated bool   `json:"Authenticated"`
	Error         string `json:"Error"`
}

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

	udpHandler     *packetForwardingHandler
	portConfig     *pinggyPortConfig
	updateListener PinggyUsagesUpdateListener

	status connectionStatus
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

func (pl *pinggyListener) checkConnectionStatus() error {
	if pl.portConfig == nil || pl.portConfig.StatusPort == 0 {
		// pl.conf.Logger.Println("noport")
		return nil
	}
	// logger := pl.conf.Logger
	pl.status.Success = false
	conn, err := pl.DialAddr(fmt.Sprintf("localhost:%d", pl.portConfig.StatusPort))
	if err != nil {
		// logger.Println("Error while localhost:4, ", err)
		return err
	}

	defer conn.Close()

	conn.Write([]byte("hello"))
	data, err := io.ReadAll(conn)
	if err != nil {
		// logger.Println("Could not read: ", err)
		return nil
	}

	err = json.Unmarshal(data, &pl.status)
	if err != nil {
		// logger.Println("Error while parsing: ", err, string(data))
		return err
	}
	if !pl.status.Success {
		// logger.Println("Could not read: ", pl.status.Error, string(data))
		return fmt.Errorf(pl.status.Error)
	}
	// logger.Println("done")
	return nil
}

func (pl *pinggyListener) preparePinggyPort() error {
	logger := pl.conf.Logger
	pl.status.Success = true //this is just to makesure old core would not create a problem.

	conn, err := pl.DialAddr("primaryHost:4")
	if err != nil {
		logger.Println("Error while localhost:4, ", err)
		return err
	}

	defer conn.Close()

	conn.Write([]byte("hello"))
	data, err := io.ReadAll(conn)
	if err != nil {
		logger.Println("Could not read: ", err)
		return nil
	}

	var portConf pinggyPortConfig
	err = json.Unmarshal(data, &portConf)
	if err != nil {
		logger.Println("Error while parsing: ", err, len(data), string(data))
		return nil
	}

	pl.portConfig = &portConf

	return pl.checkConnectionStatus()
}

func (pl *pinggyListener) updateUsage(conn net.Conn, bufReader *bufio.Reader) {
	defer conn.Close()
	defer pl.conf.Logger.Println("Ended update usages")
	for {
		line, _, err := bufReader.ReadLine()
		if err != nil {
			pl.updateListener = nil
			return
		}
		str := string(line)
		if pl.updateListener == nil {
			break
		}
		pl.updateListener.Update(str)
	}
}

func (pl *pinggyListener) SetUsagesUpdateListener(usageUpdate PinggyUsagesUpdateListener) error {
	if pl.portConfig == nil {
		return fmt.Errorf("pinggy does not support this")
	}

	if usageUpdate == nil {
		pl.updateListener = nil
		return nil
	}

	if pl.updateListener != nil {
		pl.updateListener = usageUpdate
		return nil
	}

	conn, err := pl.DialAddr(fmt.Sprintf("localhost:%d", pl.portConfig.UsageContinuousTcp))
	if err != nil {
		return err
	}

	reader := bufio.NewReader(conn)
	if reader == nil {
		conn.Close()
		return fmt.Errorf("cannot make reader")
	}

	pl.updateListener = usageUpdate

	go pl.updateUsage(conn, reader)

	return nil
}

func (pl *pinggyListener) readUsages(port int) (string, error) {
	conn, err := pl.DialAddr(fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return "", err
	}

	reader := bufio.NewReader(conn)
	if reader == nil {
		conn.Close()
		return "", fmt.Errorf("cannot make reader")
	}

	line, _, err := reader.ReadLine()

	conn.Close()

	return string(line), err
}

func (pl *pinggyListener) LongPollUsages() (string, error) {
	if pl.portConfig == nil {
		return "", fmt.Errorf("pinggy does not support this")
	}

	return pl.readUsages(pl.portConfig.UsageOnceLongPollTcp)
}

func (pl *pinggyListener) GetCurUsages() (string, error) {
	if pl.portConfig == nil {
		return "", fmt.Errorf("pinggy does not support this")
	}

	return pl.readUsages(pl.portConfig.UsageTcp)
}

func (pl *pinggyListener) getConnectionUrl() []string {
	logger := pl.conf.Logger

	conn, err := pl.DialAddr("localhost:4300")
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
	if pl.debugListener != nil {
		pl.debugListener.Close()
		pl.debugListener = nil
	}
	err := pl.clientConn.Close()
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
		err := pl.initiateSession()
		if err != nil {
			return err
		}
		err = pl.session.Shell()
		if err != nil {
			pl.conf.Logger.Println("Cannot initiate WebDebug")
			return err
		}
	}
	if pl.debugListener != nil {
		return fmt.Errorf("webDebugging is already running at %v", pl.debugListener.Addr().String())
	}
	webListener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	go func() {
		defer pl.conf.Logger.Println("Web listener close")
		for {
			conn, err := webListener.Accept()
			if err != nil {
				pl.conf.Logger.Println(err)
				return
			}
			conn2, err := pl.DialAddr("localhost:4300")
			if err != nil {
				conn.Close()
				pl.conf.Logger.Println(err)
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

func (pl *pinggyListener) initiateSession() error {
	if pl.session != nil {
		return nil
	}
	session, err := pl.clientConn.NewSession()
	if err != nil {
		pl.conf.Logger.Println("Cannot initiate WebDebug")
		return err
	}

	session.Stdout = pl.conf.Stdout
	session.Stderr = pl.conf.Stderr

	pl.session = session

	return nil
}

func (pl *pinggyListener) startSession() error {
	command := ""
	for _, ip := range pl.conf.IpWhiteList {
		command += " w:" + ip.String()
	}

	err := pl.initiateSession()
	if err != nil {
		return err
	}
	if command == "" {
		err = pl.session.Shell()
	} else {
		err = pl.session.Start(command)
	}
	if err != nil {
		pl.conf.Logger.Println("Cannot initiate WebDebug")
		return err
	}

	if pl.conf.HeaderManipulationAndAuth != nil {
		jsonBytes, err := json.Marshal(pl.conf.HeaderManipulationAndAuth)
		if err != nil {
			pl.conf.Logger.Printf("Failed to marshal JSON data: %v\n", err)
			return err
		}
		request, err := http.NewRequest("PUT", "http://localhost:4300/headerman", bytes.NewBuffer(jsonBytes))
		if err != nil {
			pl.conf.Logger.Printf("Failed to create HTTP request: %v\n", err)
			return err
		}
		request.Header.Set("Content-Type", "application/json")

		conn, err := pl.Dial()
		if err != nil {
			pl.conf.Logger.Println(err)
			return err
		}

		defer conn.Close()

		err = request.Write(conn)
		if err != nil {
			return fmt.Errorf("failed to write HTTP request to connection: %v", err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), request)
		if err != nil {
			pl.conf.Logger.Printf("Failed to read HTTP response: %v\n", err)
			return err
		}
		defer response.Body.Close()

		// Print the HTTP response
		pl.conf.Logger.Printf("HTTP Status Code: %d\n", response.StatusCode)
	}
	return nil
}

func setupPinggyTunnel(conf Config) (list *pinggyListener, err error) {
	clientConn, err := dialWithConfig(&conf)
	if err != nil {
		conf.Logger.Printf("Error in ssh connection initiation: %v\n", err)
		return
	}

	conf.Logger.Println("Ssh connection initiated. Setting up reverse tunnel")
	listener, err := clientConn.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		clientConn.Close()
		conf.Logger.Printf("Error in ssh tunnel initiation: %v\n", err)
		return
	}

	list = &pinggyListener{
		listener:    listener,
		udpListener: listener,
		clientConn:  clientConn,
		conf:        &conf,
		tcpChannel:  conf.Type != "",
		udpChannel:  conf.AltType != "",
		closed:      false,

		tcpDialer: nil,
		udpDialer: nil,
	}

	err = list.preparePinggyPort()
	if err != nil {
		conf.Logger.Println("Something wrong", err)
		return nil, err
	}

	if conf.Type != "" && conf.AltType != "" {
		socksListener := socks.InitiatateSocks5u(listener)
		udpListener := &udpListenerWrapper{udpListener: socksListener}
		listener = socksListener
		go socksListener.Start()

		list.listener = listener
		list.udpListener = udpListener
	}

	if conf.TcpForwardingAddr != "" {
		var addr *net.TCPAddr = nil
		addr, err = net.ResolveTCPAddr("tcp", conf.TcpForwardingAddr)
		if err != nil {
			list.clientConn.Close()
			return
		}
		list.tcpDialer = tunnel.NewTcpDialer(addr)
	}

	if conf.UdpForwardingAddr != "" {
		var addr *net.UDPAddr = nil
		addr, err = net.ResolveUDPAddr("udp", conf.UdpForwardingAddr)
		if err != nil {
			list.clientConn.Close()
			return
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

	if conf.startSession {
		err = list.startSession()
		return
	}

	return
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

func (pl *pinggyListener) DialAddr(addr string) (net.Conn, error) {
	conn, err := pl.clientConn.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	pconn := pinggyConn{logger: pl.conf.Logger, conn: conn, pl: pl}
	return &pconn, nil
}

func (pl *pinggyListener) Dial() (net.Conn, error) {
	return pl.DialAddr("localhost:4300")
}
