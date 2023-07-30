package main

import (
	"fmt"
	"log"
	"os"

	"github.com/abhimp/pinggy"
)

// func main() {
// 	http.Handle("/", http.FileServer(http.Dir("/tmp")))
// 	l, e := pinggy.Connect()
// 	if e != nil {
// 		log.Fatal(e)
// 	}
// 	fmt.Println(l.RemoteUrls())
// 	err := l.InitiateWebDebug("0.0.0.0:4300")
// 	fmt.Println(err)
// 	go func() {
// 		time.Sleep(time.Second * 7)
// 		l.Close()
// 	}()
// 	log.Fatal(http.Serve(l, nil))
// }

func main() {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	// pinggy.ServeFileWithConfig(pinggy.FileServerConfi?g{Path: "/tmp/", Conf: pinggy.Config{Type: pinggy.HTTP}, WebDebugEnabled: true})
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{Server: "t.pinggy.io"})
	if err != nil {
		log.Fatal(err)
	}
	pl.InitiateWebDebug("0.0.0.0:4300")
	fmt.Println(pl.RemoteUrls())
	pl.ServeHttp(os.DirFS("/tmp"))
}
