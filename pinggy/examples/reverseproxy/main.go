package main

import (
	"log"
	"os"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)

	// Configure pass preflight
	hman := pinggy.CreateHeaderManipulationAndAuthConfig()
	hman.SetReverseProxy("localhost:8443")

	tllconfig := pinggy.ForwardedConnectionConf{}
	tllconfig.TlsLocalServer = true

	config := pinggy.Config{
		Server:                    "t.pinggy.io:443",
		TcpForwardingAddr:         "127.0.0.1:8443",
		Stdout:                    os.Stderr,
		Stderr:                    os.Stderr,
		HeaderManipulationAndAuth: hman,
		ForwardedConnectionConf:   &tllconfig,
	}

	pl, err := pinggy.ConnectWithConfig(config)
	if err != nil {
		log.Panicln(err)
	}
	log.Println("Addrs: ", pl.RemoteUrls())
	err = pl.InitiateWebDebug("localhost:3424")
	log.Println(err)
	pl.StartForwarding()
}
