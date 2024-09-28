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

	go io.Copy(conn, localConn)
	io.Copy(localConn, conn)
}

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{})
	if err != nil {
		log.Fatal(err)
	}
	pl.InitiateWebDebug("0.0.0.0:4300")
	fmt.Println(pl.RemoteUrls())
	fmt.Println(pl.GetGreetingMsg())
	for {
		con, err := pl.Accept()
		if err != nil {
			break
		}
		go setupCopyFile(con)
	}
}
