package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	"golang.org/x/crypto/ssh"
)

func getConnectionUrl(sshConn *ssh.Client) []string {
	conn, err := sshConn.Dial("tcp", "localhost:4300")
	if err != nil {
		log.Println("Error connecting the server:", err)
		return nil
	}

	req, err := http.NewRequest("GET", "http://localhost:4300/urls", nil)
	if err != nil {
		log.Println("Error creating request:", err)
		return nil
	}
	err = req.Write(conn)
	if err != nil {
		log.Println("Error sending request:", err)
		return nil
	}

	// Read the HTTP response
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		log.Println("Error reading response:", err)
		return nil
	}
	defer resp.Body.Close()

	// Print the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Error reading body:", err)
		return nil
	}

	urls := make(map[string][]string)
	err = json.Unmarshal(body, &urls)

	if err != nil {
		log.Println("Error parsing body:", err)
		return nil
	}
	log.Println(urls)
	return urls["urls"]
}

func setupCopyFile(conn net.Conn) {
	defer conn.Close()
	localConn, err := net.Dial("tcp", "localhost:4000")
	if err != nil {
		return
	}

	fmt.Println("remotConn: ", conn.LocalAddr().String(), " <-> ", conn.RemoteAddr().String())
	fmt.Println("localConn: ", localConn.LocalAddr().String(), " <-> ", localConn.RemoteAddr().String())
	defer localConn.Close()
	go io.Copy(conn, localConn)
	io.Copy(localConn, conn)
}

func main() {
	clientConfig := &ssh.ClientConfig{
		User: "somename+tcp",
		Auth: []ssh.AuthMethod{
			ssh.Password("nopass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := "l:7878" //"t.pinggy.io:443"

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	c, chans, reqs, err := ssh.NewClientConn(conn, addr, clientConfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	sshConn := ssh.NewClient(c, chans, reqs)

	list, _ := sshConn.Listen("tcp", "0.0.0.0:0")

	session, err := sshConn.NewSession()
	if err != nil {
		log.Println("Cannot initiate WebDebug")
		return
	}
	err = session.Shell()
	if err != nil {
		log.Println("Cannot initiate WebDebug")
		return
	}

	fmt.Println(list)

	fmt.Println(getConnectionUrl(sshConn))

	for {
		fmt.Println("asdas")
		con, err := list.Accept()
		if err != nil {
			break
		}
		go setupCopyFile(con)
	}
}
