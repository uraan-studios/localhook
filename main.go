package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"

	"github.com/gliderlabs/ssh"
	"github.com/go-chi/chi"
	gossh "golang.org/x/crypto/ssh"

	tea "github.com/charmbracelet/bubbletea"
)

type Session struct {
	sesssion    ssh.Session
	destination string
}

var clients sync.Map

type HTTPHandler struct {
}

func (h *HTTPHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	value, ok := clients.Load(id)
	if !ok {
		http.Error(w, "client id not found", http.StatusBadRequest)
		return
	}

	fmt.Println("This is the id:", id)

	session := value.(Session)
	defer r.Body.Close()

	req, err := http.NewRequest(r.Method, session.destination, r.Body)
	if err != nil {
		log.Println("Error creating request:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Copy original request headers
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("No response from server:", err)
		http.Error(w, "No response from destination", http.StatusNotFound) // Return 404
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// If response is empty, send 404
	if resp.ContentLength == 0 {
		http.Error(w, "No content received", http.StatusNotFound)
		return
	}

	// Copy response body
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Println("Error copying response body:", err)
	}
}

func startHTTPServer() error {
	httpPort := ":5000"
	handler := &HTTPHandler{}

	router := chi.NewRouter()
	router.HandleFunc("/{id}", handler.handleWebhook)
	router.HandleFunc("/{id}/*", handler.handleWebhook)

	return http.ListenAndServe(httpPort, router)
}

func startSSHServer() error {
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

func main() {
	go startSSHServer()
	startHTTPServer()
}

type model struct {
	choices  []string         // items on the to-do list
	cursor   int              // which to-do list item our cursor is pointing at
	selected map[int]struct{} // which to-do items are selected
}

func initialModel() model {
	return model{
		// Our to-do list is a grocery list
		choices: []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},

		// A map which indicates which choices are selected. We're using
		// the  map like a mathematical set. The keys refer to the indexes
		// of the `choices` slice, above.
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	// Just return `nil`, which means "no I/O right now, please."
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Is it a key press?
	case tea.KeyMsg:

		// Cool, what was the actual key pressed?
		switch msg.String() {

		// These keys should exit the program.
		case "ctrl+c", "q":
			return m, tea.Quit

		// The "up" and "k" keys move the cursor up
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		// The "down" and "j" keys move the cursor down
		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		// The "enter" key and the spacebar (a literal space) toggle
		// the selected state for the item that the cursor is pointing at.
		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	return m, nil
}

func (m model) View() string {
	// The header
	s := "What should we buy at the market?\n\n"

	// Iterate over our choices
	for i, choice := range m.choices {

		// Is the cursor pointing at this choice?
		cursor := " " // no cursor
		if m.cursor == i {
			cursor = ">" // cursor!
		}

		// Is this choice selected?
		checked := " " // not selected
		if _, ok := m.selected[i]; ok {
			checked = "x" // selected!
		}

		// Render the row
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
	}

	// The footer
	s += "\nPress q to quit.\n"

	// Send the UI for rendering
	return s
}

type SSHHandler struct {
}

func NewSSHHandler() *SSHHandler {
	return &SSHHandler{}
}

func (h *SSHHandler) HandleSSHSession(session ssh.Session) {
	// Get input and output streams from the SSH session
	input := session
	output := session

	// Create a new Bubble Tea program with the SSH session streams
	p := tea.NewProgram(initialModel(), tea.WithInput(input), tea.WithOutput(output))

	// Run the program and handle errors
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(output, "Alas, there's been an error: %v\n", err)
		os.Exit(1)
	}
}

// func (h *SSHHandler) HandleSSHSession(session ssh.Session) {

// 	if session.RawCommand() == "tunnel" {
// 		session.Write([]byte("Tunneling traffic...\n"))
// 		<-session.Context().Done()
// 		return
// 	}

// 	term := term.NewTerminal(session, "$ ")
// 	msg := fmt.Sprintf("\n\nWelcome to LocalWeb!\n\nenter the webhook destination:\n")
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
// 			sesssion:    session,
// 			destination: destination.String(),
// 		}
// 		clients.Store(id, internalSession)

// 		webhookURL := fmt.Sprintf("http://localhost:5000/%s\n", id)
// 		command := fmt.Sprintf("Generate webhook: %s\n\nCommand to copy:\nssh -R 127.0.0.1:%d:%s localhost -p 2222 tunnel\n", webhookURL, generatedPort, host)
// 		term.Write([]byte(command))
// 		return

// 	}

// }

func randomPort() int {
	min := 49152
	max := 65535
	return min + rand.Intn(max-min+1)
}
