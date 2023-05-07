package pinggy

import (
	"fmt"
	"log"
	"net"

	"golang.org/x/crypto/ssh"
)

type Config struct {
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
	return listener, nil
}
