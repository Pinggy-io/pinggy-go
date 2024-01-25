package pinggy

import (
	"io"
	"io/fs"
	"log"
	"net"
	"time"
)

type TunnelType string
type UDPTunnelType string

const (
	TCP    TunnelType = "tcp"
	TLS    TunnelType = "tls"
	HTTP   TunnelType = "http"
	TLSTCP TunnelType = "tlstcp"
)

const (
	UDP UDPTunnelType = "udp"
)

type PinggyHttpHeaderInfo struct {
	/*
		Header name. Case insensitive
		Key can be any header name. However, host is not allowed here.
	*/
	Key string `json:"headerName"`

	/*
		Whether or not to remove existing headers
	*/
	Remove bool `json:"remove"`

	/*
		New Values for the header. If Remove is false, new headers
		would be added again.
	*/
	NewValues []string `json:"values"`
}

type HttpHeaderManipulationAndAuthConfig struct {
	/*
		New value for the `Host` Header. It is special header.
	*/
	HostName string `json:"hostName"`

	/*
		Request Header modification info.
	*/
	Headers map[string]*PinggyHttpHeaderInfo `json:"headers"`

	/*
		List of base64 encoded basic auth info.
	*/
	BasicAuths map[string]bool `json:"basicAuths"`

	/*
		List of keys for bearer authentication
	*/
	BearerAuths map[string]bool `json:"bearerAuths"`
}

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
	AltType UDPTunnelType

	/*
		This module log several thing. We use the Logger for this task. If Logger is `nil`, we use the default Logger.
	*/
	Logger *log.Logger

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

	/*
		Automatically forward connection to this address. Keep empty to disable it.
	*/
	TcpForwardingAddr string

	/*
		Automatically forward udp packet to this address. Keep empty to disable it.
	*/
	UdpForwardingAddr string

	/*
		IP Whitelist
	*/
	IpWhiteList []*net.IPNet

	/*
		Configure Header Manipulation, Basic auth, and Bearer auth for HTTP tunnels.
		The configuration will be ignored for tunnels other than HTTP tunnels.
	*/
	HeaderManipulationAndAuth *HttpHeaderManipulationAndAuthConfig

	/*
		Remote command output writer. By default it would be a instance of io.Discard.

		One need to be carefull while using these file. There is a fixed amount of
		buffering that is shared for the two streams. If either blocks it may
		eventually cause the remote command to block.
	*/
	Stdout io.Writer
	Stderr io.Writer

	// Timeout is the maximum amount of time for the TCP connection to establish.
	//
	// A Timeout of zero means no timeout.
	Timeout time.Duration

	startSession bool

	port int
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

	/*
		Forward tcp tunnel to this new addr
	*/
	UpdateTcpForwarding(addr string) error

	/*
		Forward tcp tunnel to this new addr
	*/
	UpdateUdpForwarding(addr string) error

	/*
		Start forwarding. It would work only
		Forwarding address present
	*/
	StartForwarding() error

	/*
		Dial a connection to tunnel server. It can be useful get various infomation without starting webdebugger.
		One can acheive exact same result with a webdebugger as well.
	*/
	Dial() (net.Conn, error)
}

/*
Connect to pinggy service and receive a PinggyListener object.
This function does not take any argument. So, it creates an annonymous
tunnel with HTTP.
*/
func Connect(typ TunnelType) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: "", Type: typ})
}

/*
Same as Connect() func, however it require a token. Token can be found at
Pinggy Dashboard (dashboard.pinggy.io). One can pass empty string as token
as well.
*/
func ConnectWithToken(token string, typ TunnelType) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token, Type: typ})
}

/*
Create a UDP Tunnel. One have to use ReadFrom and WriteTo method to receive
and send datagram. This tunnel is unreliable.

One can not send to any arbitary address. One can only reply to a address when
it receives an datagram from that address.
*/
func ConnectUdp(token string) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token, Type: "", AltType: UDP})
}

/*
Create tunnel with config.
*/
func ConnectWithConfig(conf Config) (PinggyListener, error) {
	conf.verify()
	return setupPinggyTunnel(conf)
}
