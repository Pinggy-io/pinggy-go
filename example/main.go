package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/abhimp/pinggy"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir("/tmp")))
	l, e := pinggy.Connect()
	if e != nil {
		log.Fatal(e)
	}
	fmt.Println(l.RemoteUrls())
	err := l.InitiateWebDebug("0.0.0.0:4300")
	fmt.Println(err)
	go func() {
		time.Sleep(time.Second * 7)
		l.Close()
	}()
	log.Fatal(http.Serve(l, nil))
}
