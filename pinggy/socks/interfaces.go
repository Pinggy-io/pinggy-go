package socks

import "net"

type ConnType int

const (
	ConnType_NONE ConnType = 0
	ConnType_TCP  ConnType = 1
	ConnType_UDP  ConnType = 2
)

type SocksCmd byte

const (
	SocksCmd_Connect    SocksCmd = 1
	SocksCmd_UdpConnect SocksCmd = 4
)

type ReplyType byte

const (
	ReplyType_Success                 ReplyType = 0
	ReplyType_GeneralFailure          ReplyType = 1
	ReplyType_NotAllowed              ReplyType = 2
	ReplyType_NetworkUnreachable      ReplyType = 3
	ReplyType_HostUnreachable         ReplyType = 4
	ReplyType_ConnectionRefused       ReplyType = 5
	ReplyType_TtlExpired              ReplyType = 6
	ReplyType_CommandNotSupported     ReplyType = 7
	ReplyType_AddressTypeNotSupported ReplyType = 8
)

type Socks5u interface {
	net.Listener

	AcceptTcp() (net.Conn, net.Addr, error)
	AcceptUdp() (net.Conn, net.Addr, error)
	StripSockFromConn(net.Conn) (net.Addr, ConnType, error)
	AcceptAndStripSock(net.Listener) (net.Conn, net.Addr, ConnType, error)
	Start()
}
