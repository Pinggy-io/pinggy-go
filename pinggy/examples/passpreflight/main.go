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
	hman.SetPassPreflight(true)

	config := pinggy.Config{
		Server:                    "t.pinggy.io:443",
		TcpForwardingAddr:         "127.0.0.1:4000",
		Stdout:                    os.Stderr,
		Stderr:                    os.Stderr,
		HeaderManipulationAndAuth: hman,
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
