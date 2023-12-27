package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Pinggy-io/pinggy-go/lib/pinggy/udptunnel"
)

func main() {
	if len(os.Args) <= 2 {
		fmt.Println(len(os.Args), os.Args)
		os.Exit(3)
	}

	tcpPort, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	forwarder, err := udptunnel.NewTunnelMangerListen(tcpPort, os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	forwarder.StartForwarding()
}
