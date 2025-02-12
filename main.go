package main

import (
	"flag"
	"fmt"

	"github.com/yas1nshah/ssh-webhook-tunnel/http"
	"github.com/yas1nshah/ssh-webhook-tunnel/ssh"
)

func main() {
	var domainFlag = flag.String("domain", "localhost", "used to create live links")
	var httpPortFlag = flag.Int("httpPort", 5000, "used to run http server")
	var sshPortFlag = flag.Int("sshPort", 2222, "used to run ssh server")

	flag.Parse()

	go ssh.StartSSHServer(fmt.Sprintf(":%d", *sshPortFlag), *httpPortFlag, *domainFlag)
	http.StartHTTPServer(fmt.Sprintf(":%d", *httpPortFlag))
}
