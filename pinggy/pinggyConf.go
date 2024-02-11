package pinggy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
)

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
			conf.Logger.Fatal(err)
		}
		conf.port = p
	}
	if conf.Logger == nil {
		conf.Logger = log.Default()
	}

	ctype := conf.Type
	switch ctype {
	case HTTP, TCP, TLS, TLSTCP:
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

	// if conf.Type != "" && conf.AltType != "" {
	// 	conf.AltType = ""
	// }

	if conf.UdpForwardingAddr != "" && conf.AltType == "" {
		conf.AltType = UDP
	}

	if conf.TcpForwardingAddr != "" && conf.Type == "" {
		conf.Type = HTTP //this is default behaviour
	}

	if conf.Type == "" && conf.AltType == "" {
		conf.Type = HTTP
	}

	conf.startSession = false
	if len(conf.IpWhiteList) > 0 {
		conf.startSession = true
	}
	if conf.HeaderManipulationAndAuth != nil {
		for _, hman := range conf.HeaderManipulationAndAuth.Headers {
			if strings.ToLower(hman.Key) == "host" {
				conf.Logger.Fatalln("host header is not allowed here")
			}
		}

		conf.startSession = true
	}
}

func dialWithConnectProxy(conf *Config, addr string) (net.Conn, error) {
	proxyAddr := fmt.Sprintf("%s:%s", conf.Proxy.Hostname(), conf.Proxy.Port())

	conn, err := net.DialTimeout("tcp", proxyAddr, conf.Timeout)
	if err != nil {
		return conn, err
	}

	req, err := http.NewRequest("CONNECT", "", nil)
	if err != nil {
		conn.Close()
		return nil, err
	}

	req.Host = addr

	userInfo := conf.Proxy.User
	if userInfo != nil && userInfo.Username() != "" {
		userPass := userInfo.String()
		encString := base64.StdEncoding.EncodeToString([]byte(userPass))
		req.Header.Add("Proxy-Authorization", fmt.Sprintf("basic %s", encString))
	}

	err = req.WriteProxy(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	buf := make([]byte, 2048)
	offset := 0
	for {
		n, err := conn.Read(buf[offset:])
		if err != nil {
			conn.Close()
			return nil, err
		}
		offset += n
		// log.Println(offset, string(buf), len(string(buf[offset-4:offset])), len(string(buf[offset-2:offset])), err)
		if offset > 4 && (string(buf[offset-4:offset]) == "\r\n\r\n" || string(buf[offset-2:offset]) == "\n\n") {
			res, err := http.ReadResponse(bufio.NewReader(bytes.NewBuffer(buf)), req)
			if err != nil {
				conn.Close()
				return nil, err
			}
			if res.StatusCode == 200 {
				return conn, nil
			}
			conn.Close()
			return nil, fmt.Errorf("proxy connection error: status.code: %d", res.StatusCode)
		}
	}
}

func connectToServer(conf *Config, addr string) (net.Conn, error) {
	if conf.ServerConnection != nil {
		return conf.ServerConnection, nil
	}

	if conf.Proxy == nil {
		return net.DialTimeout("tcp", addr, conf.Timeout)
	}

	switch conf.Proxy.Scheme {
	case "http":
		return dialWithConnectProxy(conf, addr)
	default:
		return nil, fmt.Errorf("unknown scheme in proxy address")
	}
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
	if conf.Force {
		user += "+force"
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
	conf.Logger.Printf("Initiating ssh connection %s to server: %s:%d\n", usingToken, conf.Server, conf.port)

	addr := fmt.Sprintf("%s:%d", conf.Server, conf.port)
	conn, err := connectToServer(conf, addr)
	if err != nil {
		conf.Logger.Printf("Error in ssh connection initiation: %v\n", err)
		return nil, err
	}
	if conf.SshOverSsl {
		tlsConn := tls.Client(conn, &tls.Config{ServerName: conf.Server})
		err := tlsConn.Handshake()
		if err != nil {
			conf.Logger.Printf("Error in ssh connection initiation: %v\n", err)
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
