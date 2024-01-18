package main

import (
	"log"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{Server: "l:7878", Token: "noscreen", TcpForwardingAddr: "127.0.0.1:4000"})
	if err != nil {
		log.Panicln(err)
	}
	log.Println("Addrs: ", pl.RemoteUrls())
	// err = pl.InitiateWebDebug("l:3424")
	log.Println(err)
	pl.StartForwarding()
	// _, err = pl.Accept()
	// log.Println(err)
}
