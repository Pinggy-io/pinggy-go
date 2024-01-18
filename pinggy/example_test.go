package pinggy_test

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/Pinggy-io/pinggy-go/pinggy"
	"github.com/Pinggy-io/pinggy-go/pinggy/util"
)

func TestConnection(t *testing.T) {
	l, err := pinggy.Connect(pinggy.HTTP)
	if err != nil {
		t.Fatalf("Test failed: %v\n", err)
	}
	fmt.Println(l.Addr())
	l.Close()
}

func TestFileServing(t *testing.T) {
	log.SetFlags(log.Llongfile | log.LstdFlags)
	fname := "hello"
	fdata := []byte("This is data")

	var fs fs.FS = util.NewMapFS(map[string][]byte{fname: fdata})
	pl, err := pinggy.ConnectWithConfig(pinggy.Config{Server: "a.pinggy.online"})
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}
	urls := pl.RemoteUrls()
	pl.InitiateWebDebug("0.0.0.0:4300")
	fmt.Println("Connected, ", urls)
	// fs = os.DirFS("/tmp/")
	go func() { fmt.Println("Error: ", pl.ServeHttp(fs)) }()
	for _, url := range urls {
		url += "/" + fname
		response, err := http.Get(url)
		if err != nil {
			log.Println("Error:", err)
		} else {
			if response.StatusCode != 200 {
				log.Println("Status mismatch: ", response.StatusCode, " "+url)
			} else {
				fmt.Println("Content-Length: ", response.Header.Get("Content-length"))
				body, _ := ioutil.ReadAll(response.Body)
				if string(body) != string(fdata) {
					fmt.Println("Not matching")
				} else {
					fmt.Println("Matching for: ", url)
				}
			}
		}
	}
	// time.Sleep(time.Second * 20)
	pl.Close()
}
