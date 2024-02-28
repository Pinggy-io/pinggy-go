package pinggy

import (
	"log"
	"net"
	"time"
)

type pinggyConn struct {
	logger *log.Logger
	conn   net.Conn
	pl     *pinggyListener
}

func (pc *pinggyConn) Read(b []byte) (n int, err error)   { return pc.conn.Read(b) }
func (pc *pinggyConn) Write(b []byte) (n int, err error)  { return pc.conn.Write(b) }
func (pc *pinggyConn) Close() error                       { return pc.conn.Close() }
func (pc *pinggyConn) LocalAddr() net.Addr                { return pc.conn.LocalAddr() }
func (pc *pinggyConn) RemoteAddr() net.Addr               { return pc.conn.RemoteAddr() }
func (pc *pinggyConn) SetDeadline(t time.Time) error      { return pc.conn.SetDeadline(t) }
func (pc *pinggyConn) SetReadDeadline(t time.Time) error  { return pc.conn.SetReadDeadline(t) }
func (pc *pinggyConn) SetWriteDeadline(t time.Time) error { return pc.conn.SetWriteDeadline(t) }
