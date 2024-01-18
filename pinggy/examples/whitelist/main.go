package main

import (
	"log"
	"net"
	"os"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	_, ipnet, err := net.ParseCIDR("0.0.0.0/24")
	if err != nil {
		log.Println(err)
		return
	}
	config := pinggy.Config{
		Server:            "a.pinggy.io:443",
		TcpForwardingAddr: "127.0.0.1:4000",
		IpWhiteList:       []*net.IPNet{ipnet},
		Stdout:            os.Stderr,
		Stderr:            os.Stderr,
	}

	pl, err := pinggy.ConnectWithConfig(config)
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
