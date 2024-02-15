package main

import (
	"fmt"
	"log"

	"github.com/Pinggy-io/pinggy-go/pinggy"
)

type pop1 struct {
	pl  pinggy.PinggyListener
	cnt int
}

func (p *pop1) Update(line string) {
	fmt.Println(p.cnt, line)
	p.cnt += 1

	if p.cnt == 20 {
		p.pl.SetUsagesUpdateListener(nil)
	}
}

type pop struct {
	pl  pinggy.PinggyListener
	cnt int
}

func (p *pop) Update(line string) {
	fmt.Println(p, line)
	p.cnt += 1

	if p.cnt == 20 {
		p.pl.SetUsagesUpdateListener(&pop1{p.pl, 0})
	}
}

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{Server: "t.pinggy.io:443", Token: "noscreen", TcpForwardingAddr: "127.0.0.1:4000"})
	if err != nil {
		log.Panicln(err)
	}
	log.Println("Addrs: ", pl.RemoteUrls())
	// err = pl.InitiateWebDebug("l:3424")
	p := pop{pl, 0}
	log.Println(err)
	pl.SetUsagesUpdateListener(&p)
	fmt.Println(pl.GetCurUsages())
	pl.StartForwarding()
}
