package pinggy

import (
	"fmt"
	"io/fs"
	"log"
	"os"
)

type FileServerConfig struct {
	Conf            Config
	Path            string
	Fs              fs.FS
	WebDebugEnabled bool
	WebDebugPort    int
}

func ServeFile(path string) {
	ServeFileWithConfig(FileServerConfig{Path: path, Conf: Config{Type: HTTP}})
}

func ServeFileWithToken(token string, path string) {
	ServeFileWithConfig(FileServerConfig{Path: path, Conf: Config{Type: HTTP, Token: token}})
}

func ServeFileWithConfig(conf FileServerConfig) {
	path := conf.Path
	var fs fs.FS
	if conf.Fs != nil {
		fs = conf.Fs
	} else {
		fs = os.DirFS(path)
	}
	// http.Handle("/", http.FileServer(fs))
	l, e := ConnectWithConfig(conf.Conf)
	if e != nil {
		log.Fatal(e)
	}
	// fmt.Println(l.RemoteUrls())
	fmt.Println("The file server is ready. Use following url to browse the file.")
	for _, u := range l.RemoteUrls() {
		fmt.Println("\t", u)
	}
	if conf.WebDebugEnabled {
		port := conf.WebDebugPort
		if port <= 0 {
			port = 4300
		}
		err := l.InitiateWebDebug(fmt.Sprintf("0.0.0.0:%d", port))
		if err != nil {
			log.Println(err)
			l.Close()
			os.Exit(1)
		}
		fmt.Printf("WebDebugUI running at http://0.0.0.0:%d/\n", port)
	}
	// log.Fatal(http.Serve(l, nil))
	log.Fatal(l.ServeHttp(fs))
}
