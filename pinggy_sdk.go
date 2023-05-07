package pinggy

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"

	"golang.org/x/crypto/ssh"
)

type Config struct {
}

func printConnectionUrl(conn net.Conn) {

	req, err := http.NewRequest("GET", "http://localhost:4300/urls", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	err = req.Write(conn)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}

	// Read the HTTP response
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}
	defer resp.Body.Close()

	// Print the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
		return
	}
	fmt.Println(string(body))
}

func Connect(token string) (net.Listener, error) {
	pinggyServer := "a.pinggy.io"
	clientConfig := &ssh.ClientConfig{
		User: token + "+auth",
		Auth: []ssh.AuthMethod{
			ssh.Password("yourpassword"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	log.Printf("Initiating ssh connection using token: %s to server: %s\n", token, pinggyServer)
	clientConn, err := ssh.Dial("tcp", fmt.Sprintf("%s:443", pinggyServer), clientConfig)
	if err != nil {
		log.Printf("Error in ssh connection initiation: %v\n", err)
		return nil, err
	}

	log.Println("Ssh connection initiated. Setting up reverse tunnel")
	listener, err := clientConn.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		clientConn.Close()
		log.Printf("Error in ssh tunnel initiation: %v\n", err)
		return nil, err
	}
	conn, err := clientConn.Dial("tcp", "localhost:4300")
	if err != nil {
		log.Println(err)
	} else {
		printConnectionUrl(conn)
	}
	return listener, nil
}
