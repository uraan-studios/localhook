package ssh

import (
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/teris-io/shortid"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Session struct {
	Sesssion    ssh.Session
	Destination string
}

var Clients sync.Map

type SSHHandler struct {
}

func NewSSHHandler() *SSHHandler {
	return &SSHHandler{}
}

func StartSSHServer() error {
	sshPort := ":2222"
	// respCh := make(chan string)
	handler := NewSSHHandler()

	fwHandler := &ssh.ForwardedTCPHandler{}
	server := ssh.Server{
		Addr:    sshPort,
		Handler: handler.HandleSSHSession,
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			return password == ""
		},
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			cfg := &gossh.ServerConfig{
				ServerVersion: "SSH-2.0-sendit",
			}

			cfg.Ciphers = []string{"chacha20-poly1305@openssh.com"}
			return cfg
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		},
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			log.Println("Accepted forward", dhost, dport)
			// todo: auth validation
			return true
		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			log.Println("Accepted reverse forward", host, port, "granted")
			// todo: auth validation
			return true
		}),
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        fwHandler.HandleSSHRequest,
			"cancel-tcpip-forward": fwHandler.HandleSSHRequest,
		},
	}

	b, err := os.ReadFile("keys/privatekey")
	if err != nil {
		log.Fatal(err)
	}

	privateKey, err := gossh.ParsePrivateKey(b)
	if err != nil {
		log.Fatal("Failed to parse private key ", err)
	}

	server.AddHostKey(privateKey)

	return server.ListenAndServe()
}

func randomPort() int {
	min := 49152
	max := 65535
	return min + rand.Intn(max-min+1)
}

func (h *SSHHandler) HandleSSHSession(session ssh.Session) {

	if session.RawCommand() == "tunnel" {
		session.Write([]byte("Tunneling traffic...\n"))
		<-session.Context().Done()
		return
	}

	term := term.NewTerminal(session, "$ ")
	msg := fmt.Sprintf("\n\nWelcome to LocalWeb!\n\nenter the webhook destination:\n")
	term.Write([]byte(msg))
	for {
		input, err := term.ReadLine()
		if err != nil {
			log.Fatal(err)
		}

		generatedPort := randomPort()
		id := shortid.MustGenerate()
		destination, err := url.Parse(input)
		if err != nil {
			log.Fatal(err)
		}

		host := destination.Host
		internalSession := Session{
			Sesssion:    session,
			Destination: destination.String(),
		}
		Clients.Store(id, internalSession)

		webhookURL := fmt.Sprintf("http://localhost:5000/%s\n", id)
		command := fmt.Sprintf("Generate webhook: %s\n\nCommand to copy:\nssh -R 127.0.0.1:%d:%s localhost -p 2222 tunnel\n", webhookURL, generatedPort, host)
		term.Write([]byte(command))
		return

	}

}
