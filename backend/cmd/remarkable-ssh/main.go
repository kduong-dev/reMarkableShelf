package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/kqvd/reMarkableShelf/backend/internal/config"
	"github.com/kqvd/reMarkableShelf/backend/internal/fatal"
	"golang.org/x/crypto/ssh"
)

func main() {
	host := config.EnvStringOrFatal("SSH_HOST")
	port := config.EnvStringOrFatal("SSH_PORT")
	addr := fmt.Sprintf("%s:%s", host, port)
	command := flag.String("cmd", "hostname && cat /etc/version", "command to run on the tablet")
	flag.Parse()
	config := &ssh.ClientConfig{
		User: config.EnvString("SSH_USER", "root"),
		Auth: []ssh.AuthMethod{
			ssh.Password(config.EnvStringOrFatal("SSH_PASSWORD")),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	client, err := ssh.Dial("tcp", addr, config)
	fatal.OnError(err)
	defer client.Close()
	session, err := client.NewSession()
	fatal.OnError(err)
	defer session.Close()
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr
	err = session.Run(*command)
	if err != nil {
		fmt.Fprint(os.Stderr, stderr.String())
		fatal.OnError(err, "running command: ")
	}
	fmt.Print(stdout.String())
}
