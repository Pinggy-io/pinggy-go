package main

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

func setupCopyFile(conn net.Conn) {
	defer conn.Close()
	localConn, err := net.Dial("tcp", "localhost:4000")
	if err != nil {
		conn.Close()
		return
	}
	defer localConn.Close()

	// fmt.Println("remotConn: ", conn.LocalAddr().String(), " <-> ", conn.RemoteAddr().String())
	// fmt.Println("localConn: ", localConn.LocalAddr().String(), " <-> ", localConn.RemoteAddr().String())

	go io.Copy(conn, localConn)
	io.Copy(localConn, conn)
}

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	// pinggy.ServeFileWithConfig(pinggy.FileServerConfi?g{Path: "/tmp/", Conf: pinggy.Config{Type: pinggy.HTTP}, WebDebugEnabled: true})
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{})
	if err != nil {
		log.Fatal(err)
	}
	pl.InitiateWebDebug("0.0.0.0:4300")
	fmt.Println(pl.RemoteUrls())
	// pl.ServeHttp(os.DirFS("/tmp"))
	for {
		fmt.Println("asdas")
		con, err := pl.Accept()
		if err != nil {
			break
		}
		go setupCopyFile(con)
	}
}
