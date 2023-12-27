package pinggy

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type TunnelType string
type AltTunnelType string

const (
	TCP  TunnelType = "tcp"
	TLS  TunnelType = "tls"
	HTTP TunnelType = "http"
)

const (
	UDP AltTunnelType = "udp"
)

type Config struct {
	/*
		Token is a string. It identify an user. You can find a token at the https://dashboard.pinggy.io.
		Token is required to connect in TCP and TLS tunnel.
	*/
	Token string

	/*
		Tunnel type. It can be one of TCP or TLS or HTTP or empty.
		Both type and altType cannot be empty.
	*/
	Type TunnelType

	/*
		Alternate AltTunnelType. It can be UDP or empty. However,
		both type and altType cannot be empty. As of now only one of
		them can be populated.
	*/
	AltType AltTunnelType

	/*
		This module log several thing. We use the logger for this task. If logger is `nil`, we use the default logger.
	*/
	logger *log.Logger

	/*
		Pinggy supports ssh over ssl when user is behind a firewall which does not allow anything but ssl.
		Simply enable this flag and this package would take care of this problem.
	*/
	SshOverSsl bool
	/*
		Pinggy server to connect to. Default value `a.pinggy.io`.
		Users are discouraged to use this.
	*/
	Server string

	port int
}

func (conf *Config) verify() {
	if conf.Server == "" {
		conf.Server = "a.pinggy.io"
	}
	addr := strings.Split(conf.Server, ":")
	conf.port = 443
	conf.Server = addr[0]
	if len(addr) > 1 {
		p, err := strconv.Atoi(addr[1])
		if err != nil {
			conf.logger.Fatal(err)
		}
		conf.port = p
	}
	if conf.logger == nil {
		conf.logger = log.Default()
	}

	ctype := conf.Type
	switch ctype {
	case HTTP, TCP, TLS:
		conf.Type = ctype
	default:
		conf.Type = ""
	}
	atype := conf.AltType
	conf.AltType = ""
	switch atype {
	case UDP:
		conf.AltType = UDP
	default:
		conf.AltType = ""
	}

	if conf.Type != "" && conf.AltType != "" {
		conf.AltType = ""
	}

	if conf.Type == "" && conf.AltType == "" {
		conf.Type = HTTP
	}
}

type PinggyListener interface {
	net.Listener
	net.PacketConn

	/*
		Return the remote urls to access the tunnels.
	*/
	RemoteUrls() []string

	/*
		Start webdebugger. This can not be call multiple time. Once the debugger started, it cannot be closed.
		Also, the debugger is not available in case of `tls` and `tcp` tunnel
	*/
	InitiateWebDebug(addr string) error

	/*
		Start a webserver.
	*/
	ServeHttp(fs fs.FS) error
}

type pinggyListener struct {
	conf          *Config
	clientConn    *ssh.Client
	listener      net.Listener
	session       *ssh.Session
	debugListener net.Listener
	udpChannel    bool
	tcpChannel    bool
	closed        bool

	udpHandler *packetForwardingHandler
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

func Connect() (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: ""})
}
func ConnectToken(token string) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token})
}
func ConnectTcp(token string) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token, Type: TCP})
}
func ConnectTls(token string) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token, Type: TLS})
}

func dialWithConfig(conf *Config) (*ssh.Client, error) {
	user := "auth"
	if conf.Type != "" {
		user += "+" + string(conf.Type)
	}
	if conf.AltType != "" {
		user += "+" + string(conf.AltType)
	}
	if conf.Token != "" {
		user = conf.Token + "+" + user
	}
	clientConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password("nopass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	usingToken := "without using any token"
	if conf.Token != "" {
		usingToken = fmt.Sprintf("using token: %s", conf.Token)
	}
	conf.logger.Printf("Initiating ssh connection %s to server: %s:%d\n", usingToken, conf.Server, conf.port)

	addr := fmt.Sprintf("%s:%d", conf.Server, conf.port)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		conf.logger.Printf("Error in ssh connection initiation: %v\n", err)
		return nil, err
	}
	if conf.SshOverSsl {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: conf.Server})
		err := tlsConn.Handshake()
		if err != nil {
			conf.logger.Printf("Error in ssh connection initiation: %v\n", err)
			return nil, err
		}
		conn = tlsConn
	}
	c, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		return nil, err
	}

	return ssh.NewClient(c, chans, reqs), nil
}

func ConnectWithConfig(conf Config) (PinggyListener, error) {
	conf.verify()

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

	list := &pinggyListener{
		listener:   listener,
		clientConn: clientConn,
		conf:       &conf,
		tcpChannel: conf.Type != "",
		udpChannel: conf.AltType != "",
		closed:     false,
	}
	// log.Println("udp:", list.udpChannel, " tcpChannel: ", list.tcpChannel)
	if list.udpChannel {
		list.udpHandler = &packetForwardingHandler{
			list:        list.listener,
			readChannel: make(chan *packet, 50),
			tunnels:     make(map[string]tunnel),
		}
		go list.udpHandler.startForwarding()
	}

	return list, nil
}

func (pl *pinggyListener) Accept() (net.Conn, error) {
	if pl.udpHandler != nil {
		return nil, fmt.Errorf("not allowed")
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
