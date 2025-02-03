package ssh

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
	"github.com/teris-io/shortid"
	"github.com/yas1nshah/ssh-webhook-tunnel/ui"
	gossh "golang.org/x/crypto/ssh"
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
	input := session
	output := session
	p := tea.NewProgram(ui.InitialModel(), tea.WithInput(input), tea.WithOutput(output))

	// Run the program and capture the model state
	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(output, "Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}

	// Retrieve the user's choice from the model
	var choice int
	if model, ok := m.(ui.Model); ok {
		fmt.Println(model.Choice())
		choice = model.Choice()
		// return model.Choice()
	}

	p.ReleaseTerminal()
	session.Write([]byte(fmt.Sprintf("Terminal Killed %v", choice)))

	// Create a buffer for reading user input
	buf := make([]byte, 1024)

	for {
		time.Sleep(time.Millisecond)

		n, err := session.Read(buf)
		if err != nil {
			if err == io.EOF {
				break // Exit loop on session close
			}
			session.Write([]byte(fmt.Sprintf("Read error: %v\n", err)))
			break
		}

		// Echo the user input
		session.Write(buf[:n])
	}

}

func CreateNewHook(localURL string, session ssh.Session) (string, error) {
	generatedPort := randomPort()
	id := shortid.MustGenerate()
	destination, err := url.Parse(localURL)
	if err != nil {
		return "", err
	}

	host := destination.Host
	internalSession := Session{
		Sesssion:    session,
		Destination: destination.String(),
	}
	Clients.Store(id, internalSession)

	webhookURL := fmt.Sprintf("http://localhost:5000/%s\n", id)
	command := fmt.Sprintf("Generate webhook: %s\n\nCommand to copy:\nssh -R 127.0.0.1:%d:%s localhost -p 2222 tunnel\n", webhookURL, generatedPort, host)
	return command, err
}

// func (h *SSHHandler) HandleSSHSession(session ssh.Session) {

// 	if session.RawCommand() == "tunnel" {
// 		session.Write([]byte("Tunneling traffic...\n"))
// 		<-session.Context().Done()
// 		return
// 	}

// 	term := term.NewTerminal(session, "$ ")
// 	msg := fmt.Sprintf("%s\n\nWelcome to LocalWeb!\n\nenter the webhook destination:\n", logo)
// 	term.Write([]byte(msg))
// 	for {
// 		input, err := term.ReadLine()
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		generatedPort := randomPort()
// 		id := shortid.MustGenerate()
// 		destination, err := url.Parse(input)
// 		if err != nil {
// 			log.Fatal(err)
// 		}

// 		host := destination.Host
// 		internalSession := Session{
// 			Sesssion:    session,
// 			Destination: destination.String(),
// 		}
// 		Clients.Store(id, internalSession)

// 		webhookURL := fmt.Sprintf("http://localhost:5000/%s\n", id)
// 		command := fmt.Sprintf("Generate webhook: %s\n\nCommand to copy:\nssh -R 127.0.0.1:%d:%s localhost -p 2222 tunnel\n", webhookURL, generatedPort, host)
// 		term.Write([]byte(command))
// 		return

// 	}

// }

var logo = `

_                 _ _                 _    
| |               | | |               | |   
| | ___   ___ __ _| | |__   ___   ___ | | __
| |/ _ \ / __/ _' | | '_ \ / _ \ / _ \| |/ /
| | (_) | (_| (_| | | | | | (_) | (_) |   < 
|_|\___/ \___\__,_|_|_| |_|\___/ \___/|_|\_\                                            
                            by Uraan Studios        
                                              											
`
