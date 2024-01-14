package main

import (
	"log"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{
		Type:              pinggy.HTTP,
		AltType:           pinggy.UDP,
		Server:            "l:7878",
		Token:             "noscreen",
		UdpForwardingAddr: "127.0.0.1:4000",
		TcpForwardingAddr: "127.0.0.1:4000",
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Addrs: ", pl.RemoteUrls())
	// err = pl.InitiateWebDebug("l:3424")
	log.Println(err)
	pl.StartForwarding()
	// _, err = pl.Accept()
	// log.Println(err)
}
