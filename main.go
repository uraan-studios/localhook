package main

import (
	"github.com/yas1nshah/ssh-webhook-tunnel/http"
	"github.com/yas1nshah/ssh-webhook-tunnel/ssh"
)

func main() {
	go ssh.StartSSHServer()
	http.StartHTTPServer()
}
