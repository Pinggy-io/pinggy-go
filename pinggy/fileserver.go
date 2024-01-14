package pinggy

import (
	"fmt"
	"io/fs"
	"log"
	"os"
)

/*
Configuration for a simple http file server.
This struct can be used to create simple file server using pinggy.
*/
type FileServerConfig struct {
	/*
		Pinggy tunnel config. check pinggy config for more details.
	*/
	TunnelConf Config

	/*
		Local directory path. The server would list all the file inside the directory.
		The pathe would be used only if `Fs` is nil.
	*/
	Path string

	/*
		A file system object. It can initiated by `os.DirFS(path)`. However, it can be a
		simple object with implements the `fs.FS` interface. Pinggy util module implements
		one such object.
	*/
	Fs fs.FS

	/*
		Whether pinggy webdebug is enabled or not. Kindly provide valid port to enable it.
	*/
	WebDebugEnabled bool

	/*
		The port where pinggy webdebug would listen.
		If you provide port at 8080, you can access the debugg ui at localhost:8080
	*/
	WebDebugPort int
}

/*
Serve content of the `path` via pinggy as a http server.
*/
func ServeFile(path string) {
	ServeFileWithConfig(FileServerConfig{Path: path, TunnelConf: Config{Type: HTTP}})
}

/*
Serve content of the `path` via pinggy with token.
*/
func ServeFileWithToken(token string, path string) {
	ServeFileWithConfig(FileServerConfig{Path: path, TunnelConf: Config{Type: HTTP, Token: token}})
}

/*
Serve files as http.
*/
func ServeFileWithConfig(conf FileServerConfig) {
	path := conf.Path
	var fs fs.FS
	if conf.Fs != nil {
		fs = conf.Fs
	} else {
		fs = os.DirFS(path)
	}
	// http.Handle("/", http.FileServer(fs))
	l, e := ConnectWithConfig(conf.TunnelConf)
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
