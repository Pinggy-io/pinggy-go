package pinggy

import (
	"io"
	"io/fs"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/Pinggy-io/pinggy-go/pinggy/internal/headermanipulation"
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

type HttpHeaderManipulationAndAuthConfig interface {
	/*
		Add username password for besic authentication.
		One can add multiple basic authentications without any issues.
		It can be added along with Bearer authentication.
	*/
	AddBasicAuth(username, password string)

	/*
		Add bearer authention key.
	*/
	AddBearerAuth(key string)

	/*
		Change the Host header in request.
	*/
	SetHostname(hostname string)

	/*
		Get Host modified header value.
	*/
	GetHostname() string

	/*
		Add the header name you want to remove from the request header.
		All the header with the provided name from in coming request header.
		AddHeader would be effective even after removal or a header.
	*/
	RemoveHeader(headerName string) error

	/*
		New request header. It would add a new header along with exsting.
	*/
	AddHeader(headerName, headerValue string) error

	/*
		This is combination of remove and add header.
	*/
	UpdateHeader(headerName, headerValue string) error

	/*
		Get a json dump of header manipulation
	*/
	ListHeaderManipulations() ([]byte, error)

	/*
		One can reconstract header manipulation from a json.
	*/
	ReconstructHeaderManipulationDataFromJson([]byte) error

	/*
		Set XFF header name that would be added to request header.
		It would contain original source address.
	*/
	SetXFFHeader(xff string)

	/*
		Set X-Forwarded-For header in the request.
	*/
	SetXFF()

	/*
		No http request would be allowed on this tunnel. http requests would be
		redirected qith 301 status.
	*/
	SetHttpsOnly(val bool)

	/*
		Pinggy would pass original url in the X-Pinggy-Url request header
	*/
	SetFullUrl(val bool)

	/*
		By default, enabling key authentication / password authentication will block
		all unauthenticated requests to Pinggy URLs. But sometimes the CORS preflight
		requests are required to be sent through to enable CORS.
		Enabling this option will make Pinggy allow CORS preflight requests to pass through
	*/
	SetPassPreflight(val bool)

	/*
		Enable reverse proxy mode. Pass the upstream server hostname. Eg. localhost:5000 or example.com
		This enables X-Forwarded-Host, X-Forwarded-Proto, and Forwarded headers.
		It also manipulates Host header and SNI in case of TLS.
	*/
	SetReverseProxy(hostname string)

	/*
		Enable reverse proxy mode. Pass the upstream server hostname. Eg. localhost:5000 or example.com
		This enables X-Forwarded-Host, X-Forwarded-Proto, and Forwarded headers.
		It also manipulates Host header and SNI in case of TLS.
	*/
	SetNoReverseProxy()

	/*
		Set X-Forwarded-Host header in the requests. We recommend using SetReverseProxy instead.
	*/
	SetXFH(val bool)

	/*
		Set X-Forwarded-Proto header in the requests.  We recommend using SetReverseProxy instead.
	*/
	SetXFP(val bool)

	/*
		Set Forwarded header in the requests.  We recommend using SetReverseProxy instead.
	*/
	SetForwarded(val bool)
}

type ForwardedConnectionConf struct {
	// Whether or not local server tls
	TlsLocalServer bool `json:"tlsLocalServer"`

	// What is the SNI to be used in case of local server TLS
	TlsLocalServerSNI string `json:"tlsLocalServerSNI"`
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
		Forwarded local server connection configuration
	*/
	ForwardedConnectionConf *ForwardedConnectionConf

	/*
		Configure Header Manipulation, Basic auth, and Bearer auth for HTTP tunnels.
		The configuration will be ignored for tunnels other than HTTP tunnels.
	*/
	HeaderManipulationAndAuth HttpHeaderManipulationAndAuthConfig

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

	// SshTimeout is the maximum amount of time allowed for SSH to complete the handshake.
	// It includes the time required for ssl to established in case of ssh-over-ssh.
	//
	// A SshTimeout of zero means no timeout.
	SshTimeout time.Duration

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
	RemoteUrls2() ([]string, error)

	/*
		Start webdebugger. This can not be called more than once.
		Once the debugger started, it cannot be closed.
		The webdebugger only available for `http` tunnels.
	*/
	InitiateWebDebug(addr string) error

	/*
		Start forwarding requests to webdebugger port without starting debug session.
		Useful for gettin the URLs, IP whitelists of TCP / UDP tunnels.
	*/
	InitiateDebugForward(addr string) error

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
		Start additional forwarding. It would work only with
		http tunnel.

		domain needs to be valid domain for the given tunnel.
	*/
	AddAdditionalForwarding(domain, addr string) error

	/*
		Update additional forwarding.
	*/
	UpdateAdditionalForwarding(domain, addr string) error

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

	/*
		This would provide the greeting msg. Not usefull most of the cases
	*/
	GetGreetingMsg() ([]string, error)
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

/*
Get header manipulation and auth
*/
func CreateHeaderManipulationAndAuthConfig() HttpHeaderManipulationAndAuthConfig {
	return headermanipulation.NewHeaderManipulationAndAuthConfig()
}
