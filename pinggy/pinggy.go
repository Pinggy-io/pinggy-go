package pinggy

import (
	"io/fs"
	"log"
	"net"
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

	/*
		It will automatically forward connection to this address. Keep empty to disable it.
	*/
	ForwardTcpTo string

	/*
		It will automatically forward udp packet to this address. Keep empty to disable it.
	*/
	ForwardUdpTo string

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
}

/*
Connect to pinggy service and receive a PinggyListener object.
This function does not take any argument. So, it creates an annonymous
tunnel with HTTP.
*/
func Connect() (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: ""})
}

/*
Same as Connect() func, however it require a token. Token can be found at
Pinggy Dashboard (dashboard.pinggy.io). One can pass empty string as token
as well.
*/
func ConnectToken(token string) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token})
}

/*
Creates a TCP tunnel.
*/
func ConnectTcp(token string) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token, Type: TCP})
}

/*
Creates a TLS tunnel
*/
func ConnectTls(token string) (PinggyListener, error) {
	return ConnectWithConfig(Config{Token: token, Type: TLS})
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
