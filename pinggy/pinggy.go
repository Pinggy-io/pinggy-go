package pinggy

import (
	"io"
	"io/fs"
	"log"
	"net"
	"net/url"
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

type HeaderManipulationInterface interface {
	AddBasicAuth(username, password string)
	AddBearerAuth(key string)
	SetHostname(hostname string)
	RemoveHeader(headerName string) bool
	RemoveHeaderValue(headerName, headerValue string)
	AppendHeaderValue(headerName, headerValue string)
	RemoveHeaderManipulation(headerName string)
	ListHeaderManipulations() []byte
	ReconstructHeaderManipulationDataFromJson([]byte) error
	SetXFFHeader(xff string)
	SetHttpsOnly(val bool)
	SetFullUrl(val bool)
}

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

	/*
		The XFF header name. The server would set the header with value containing
		original source. It is expected to set X-Forwarded-For header. However, users
		allowed to use any header they want.
	*/
	XFF string `json:"xff"` //header name. empty means not do not set

	/*
		Enable https only mode. You will keep getting http url. However, those url would redirected to
		https counter part via 301.
	*/
	HttpsOnly bool `json:"httpsOnly"` //All the http would be redirected

	/*
		In case user wants to know the original url of the request, pinggy can provide the same if user
		enable this option. Pinggy would add a new header `X-Pinggy-Url` which contains the original
		url. It may not contain the query string part of the url.
	*/
	FullRequestUrl bool `json:"fullRequestUrl"` //Will add X-Pinggy-Url to add entire url

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
		Pinggy server to connect to. Default value `a.pinggy.io`
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

	/*
		Force login.
	*/
	Force bool

	/*
		ServerConnection to be used to setup ssh connection. Proxy configuration
		won't be effective here.

		Keep it nil unless you know what you are doing.
	*/
	ServerConnection net.Conn

	/*
		Proxy url. it wii be used to connect to the server.
	*/
	Proxy *url.URL

	sni string

	startSession bool

	port int
}

type PinggyUsagesUpdateListener interface {
	Update(line string)
}

type PinggyListener interface {
	net.Listener
	net.PacketConn

	/*
		Return the remote urls to access the tunnels.
	*/
	RemoteUrls() []string

	/*
		Start webdebugger. This can not be called more than once.
		Once the debugger started, it cannot be closed.
		The webdebugger only available for `http` tunnels.
	*/
	InitiateWebDebug(addr string) error

	/*
		Start a webserver.
	*/
	ServeHttp(fs fs.FS) error

	/*
		Forward tcp tunnel to this new address.
	*/
	UpdateTcpForwarding(addr string) error

	/*
		Forward tcp tunnel to this new address.
	*/
	UpdateUdpForwarding(addr string) error

	/*
		Start forwarding. It would work only if
		Forwarding address is present
	*/
	StartForwarding() error

	/*
		Dial a connection to tunnel server. It can be useful to get various infomation without starting webdebugger.
		One can acheive exact same result with a webdebugger as well.
	*/
	Dial() (net.Conn, error)

	/*
		Receive usages update. Server would provide updates when it has any. You can set only one update listener.
		Set update listener with nil to stop listening
	*/
	SetUsagesUpdateListener(usagesUpdate PinggyUsagesUpdateListener) error

	/*
		It would wait till the server has any update. It would provide only one update.
	*/
	LongPollUsages() (string, error)

	/*
		This would provide the current usages without waiting.
	*/
	GetCurUsages() (string, error)
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
