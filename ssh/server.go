package ssh

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gliderlabs/ssh"
	"github.com/teris-io/shortid"
	"github.com/yas1nshah/ssh-webhook-tunnel/ui"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type Session struct {
	Sesssion    ssh.Session
	Destination string
	IsWebhook   bool
}

var Clients sync.Map

type SSHHandler struct {
	domain   string
	httpPort int
}

func NewSSHHandler(liveDomain string, port int) *SSHHandler {
	return &SSHHandler{
		domain:   liveDomain,
		httpPort: port,
	}
}

func StartSSHServer(sshPort string, httpPort int, domain string) error {
	// respCh := make(chan string)
	handler := NewSSHHandler(domain, httpPort)

	fwHandler := &ssh.ForwardedTCPHandler{}
	server := ssh.Server{
		Addr:            sshPort,
		Handler:         handler.HandleSSHSession,
		PasswordHandler: nil,
		// PasswordHandler: func(ctx ssh.Context, password string) bool {
		// 	return true
		// },
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			cfg := &gossh.ServerConfig{
				ServerVersion: "SSH-2.0-sendit",
			}

			cfg.Ciphers = []string{"chacha20-poly1305@openssh.com"}
			return cfg
		},
		// PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
		// 	return true
		// },
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
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator
	min := 49150
	max := 49500

	for {
		port := min + rand.Intn(max-min+1)
		if isPortAvailable(port) {
			return port
		}
	}
}

func isPortAvailable(port int) bool {
	addr := net.JoinHostPort("localhost", fmt.Sprintf("%d", port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false // Port is in use
	}
	listener.Close() // Close the listener if it's available
	return true
}

func (h *SSHHandler) HandleSSHSession(session ssh.Session) {
	if session.RawCommand() == "tunnel" {
		session.Write([]byte("\nYour Local Tunnel is now Live...(30 mins)\n"))
		select {
		case <-session.Context().Done():
			// Context canceled, exit
		case <-time.After(30 * time.Minute):
			// Timeout reached, close session
			session.Close()
		}
	}

	input := session
	output := session
	p := tea.NewProgram(ui.InitWlcmTerminal(), tea.WithInput(input), tea.WithOutput(output))

	// Run the program and capture the model state
	m, err := p.Run()
	if err != nil {
		fmt.Fprintf(output, "Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}

	// Retrieve the user's choice from the model
	var choice int
	if model, ok := m.(ui.WelcomeModel); ok {
		choice = model.GetChoice()
		if model.CloseConn() {
			return
		}
		// return model.Choice()
	}

	p.ReleaseTerminal()

	switch choice {
	case 1:
		{
			p := tea.NewProgram(ui.InitWebhookTerminal(), tea.WithInput(input), tea.WithOutput(output))

			// Run the program and capture the model state
			m, err := p.Run()
			if err != nil {
				fmt.Fprintf(output, "Alas, there's been an error: %v\n", err)
				os.Exit(1)
			}

			// Retrieve the user's choice from the model
			var host string
			var port string
			var path string

			if model, ok := m.(ui.WebhookModel); ok {
				host, port, path = model.GetLocalURL()

				if model.CloseConn() {
					return
				}
				// return model.Choice()
			}

			p.ReleaseTerminal()

			term := term.NewTerminal(session, "$ ")
			for {
				generatedPort := randomPort()
				id := shortid.MustGenerate()
				// destination, err := url.Parse("http://" + host + port + path)
				// if err != nil {
				// 	log.Fatal(err)
				// }

				// host := destination.Host
				internalSession := Session{
					Sesssion:    session,
					Destination: "http://" + host + ":" + port + path,
					IsWebhook:   true,
				}
				Clients.Store(id, internalSession)

				var webhookURL string
				if h.domain == "localhost" {
					webhookURL = fmt.Sprintf("http://%s:%d/%s\n", h.domain, h.httpPort, id)
				} else {
					webhookURL = fmt.Sprintf("http://%s/%s\n", h.domain, id)

				}
				command := fmt.Sprintf("Live Webhook URL:\n\t%s\n\nRun Command (copy and run to tunnel traffic):\n\tssh -R 127.0.0.1:%d:%s:%s localhost -p 2222 tunnel\n\n\n-----------------\n", webhookURL, generatedPort, host, port)
				term.Write([]byte(command))
				return
			}
		}

	case 2:
		{
			p := tea.NewProgram(ui.InitWebhookTerminal(), tea.WithInput(input), tea.WithOutput(output))

			// Run the program and capture the model state
			m, err := p.Run()
			if err != nil {
				fmt.Fprintf(output, "Alas, there's been an error: %v\n", err)
				os.Exit(1)
			}

			// Retrieve the user's choice from the model
			var host string
			var port string

			if model, ok := m.(ui.WebhookModel); ok {
				host, port, _ = model.GetLocalURL()
				if model.CloseConn() {
					return
				}
				// return model.Choice()
			}

			p.ReleaseTerminal()

			term := term.NewTerminal(session, "$ ")
			for {
				generatedPort := randomPort()
				id := shortid.MustGenerate()
				internalSession := Session{
					Sesssion:    session,
					Destination: "http://" + host + ":" + port,
					IsWebhook:   false,
				}
				Clients.Store(id, internalSession)

				var webhookURL string
				if h.domain == "localhost" {
					webhookURL = fmt.Sprintf("http://%s:%d/%s\n", h.domain, h.httpPort, id)
				} else {
					webhookURL = fmt.Sprintf("http://%s/%s\n", h.domain, id)

				}
				command := fmt.Sprintf("Live Website URL:\n\t%s\n\nRun Command (copy and run to tunnel traffic):\n\tssh -R 127.0.0.1:%d:%s:%s localhost -p 2222 tunnel\n\n\n-----------------\n", webhookURL, generatedPort, host, port)
				term.Write([]byte(command))
				return
			}
		}
	}

}

// func CreateNewHook(localURL string, session ssh.Session) (string, error) {
// 	generatedPort := randomPort()
// 	id := shortid.MustGenerate()
// 	destination, err := url.Parse(localURL)
// 	if err != nil {
// 		return "", err
// 	}

// 	host := destination.Host
// 	internalSession := Session{
// 		Sesssion:    session,
// 		Destination: destination.String(),
// 		IsWebhook:   true,
// 	}
// 	Clients.Store(id, internalSession)

// 	webhookURL := fmt.Sprintf("http://localhost:%d/%s\n", id)
// 	command := fmt.Sprintf("Generate webhook: %s\n\nCommand to copy:\nssh -R 127.0.0.1:%d:%s localhost -p 2222 tunnel\n", webhookURL, generatedPort, host)
// 	return command, err
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
