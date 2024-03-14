package main

import (
	"log"
	"net"
	"os"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	_, ipnet1, _ := net.ParseCIDR("0.0.0.0/0")
	_, ipnet2, _ := net.ParseCIDR("::0/0")

	config := pinggy.Config{
		Server: "t.pinggy.io:443",
		// Server:            "l:7878",
		TcpForwardingAddr: "127.0.0.1:4000",
		IpWhiteList:       []*net.IPNet{ipnet1, ipnet2},
		Stdout:            os.Stderr,
		Stderr:            os.Stderr,
		HeaderManipulationAndAuth: &pinggy.HttpHeaderManipulationAndAuthConfig{
			HttpsOnly: true,
			XFF:       "X-Forwarded-For",
		},
	}

	pl, err := pinggy.ConnectWithConfig(config)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("Addrs: ", pl.RemoteUrls())
	err = pl.InitiateWebDebug("l:3424")
	log.Println(err)
	pl.StartForwarding()
	// _, err = pl.Accept()
	// log.Println(err)
}
