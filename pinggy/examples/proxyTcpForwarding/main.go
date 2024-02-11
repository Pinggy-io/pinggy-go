package main

import (
	"fmt"
	"log"
	"net/url"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	url, _ := url.Parse("http://localhost:8080")
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{
		Server:            "a.pinggy.io:443",
		Token:             "noscreen",
		TcpForwardingAddr: "127.0.0.1:4000",
		Proxy:             url,
	})
	if err != nil {
		log.Panicln(err)
	}

	fmt.Println("Hello world")

	log.Println("Addrs: ", pl.RemoteUrls())
	err = pl.InitiateWebDebug("l:3424")
	log.Println(err)
	pl.StartForwarding()
	// _, err = pl.Accept()
	// log.Println(err)
}
