package http

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/yas1nshah/ssh-webhook-tunnel/ssh"
)

type HTTPHandler struct {
}

func (h *HTTPHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	value, ok := ssh.Clients.Load(id)
	if !ok {
		http.Error(w, "client id not found", http.StatusBadRequest)
		return
	}

	fmt.Println("This is the id:", id)

	session := value.(ssh.Session)
	defer r.Body.Close()

	req, err := http.NewRequest(r.Method, session.Destination, r.Body)
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

func StartHTTPServer() error {
	httpPort := ":5000"
	handler := &HTTPHandler{}

	router := chi.NewRouter()
	router.HandleFunc("/{id}", handler.handleWebhook)
	router.HandleFunc("/{id}/*", handler.handleWebhook)

	return http.ListenAndServe(httpPort, router)
}
