package http

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi"
	"github.com/yas1nshah/ssh-webhook-tunnel/ssh"
)

type HTTPHandler struct {
}

// func (h *HTTPHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
// 	id := chi.URLParam(r, "id")
// 	value, ok := ssh.Clients.Load(id)
// 	if !ok {
// 		http.Error(w, "client id not found", http.StatusBadRequest)
// 		return
// 	}

// 	fmt.Println("This is the id:", id)

// 	session := value.(ssh.Session)
// 	defer r.Body.Close()

// 	var req *http.Request
// 	var err error

// 	if session.IsWebhook {
// 		req, err = http.NewRequest(r.Method, session.Destination, r.Body)
// 	} else {
// 		destination := strings.TrimPrefix(r.URL.RawPath, "/"+id)
// 		req, err = http.NewRequest(r.Method, destination, r.Body)

// 	}

// 	if err != nil {
// 		log.Println("Error creating request:", err)
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}

// 	// Copy original request headers
// 	for key, values := range r.Header {
// 		for _, value := range values {
// 			req.Header.Add(key, value)
// 		}
// 	}

// 	resp, err := http.DefaultClient.Do(req)
// 	if err != nil {
// 		log.Println("No response from server:", err)
// 		http.Error(w, "No response from destination", http.StatusNotFound) // Return 404
// 		return
// 	}
// 	defer resp.Body.Close()

// 	// Copy response headers
// 	for key, values := range resp.Header {
// 		for _, value := range values {
// 			w.Header().Add(key, value)
// 		}
// 	}

// 	// Write status code
// 	w.WriteHeader(resp.StatusCode)

// 	// If response is empty, send 404
// 	if resp.ContentLength == 0 {
// 		http.Error(w, "No content received", http.StatusNotFound)
// 		return
// 	}

// 	// Copy response body
// 	if _, err := io.Copy(w, resp.Body); err != nil {
// 		log.Println("Error copying response body:", err)
// 	}
// }

func (h *HTTPHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	value, ok := ssh.Clients.Load(id)
	if !ok {
		http.Error(w, "Client ID not found", http.StatusBadRequest)
		return
	}

	// Safe type assertion for session
	session, ok := value.(ssh.Session)
	if !ok {
		http.Error(w, "Invalid session type", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	var req *http.Request
	var err error
	var reqURL *url.URL

	if session.IsWebhook {
		if session.Destination == "" {
			http.Error(w, "Invalid destination URL", http.StatusBadRequest)
			return
		}
		req, err = http.NewRequest(r.Method, session.Destination, r.Body)
	} else {
		reqURL, err = url.Parse(session.Destination)
		if err != nil {
			http.Error(w, "Invalid destination URL", http.StatusBadRequest)
		}
		path := strings.TrimPrefix(r.URL.Path, "/"+id)
		req, err = http.NewRequest(r.Method, "http://"+reqURL.Hostname()+":"+reqURL.Port()+path, r.Body)
	}

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

	// Send request
	fmt.Println(req.URL)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("No response from server:", err)
		http.Error(w, "No response from destination", http.StatusNotFound)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Check if response is HTML
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		// Read full response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response body", http.StatusInternalServerError)
			return
		}
		bodyStr := string(bodyBytes)

		// Inject the top bar if <body> tag exists
		if strings.Contains(bodyStr, "<body>") {
			topBar := `<div style="position: fixed; top: 0; left: 0; width: 100%;
							background: black; color: white; text-align: center;
							padding: 5px; font-size: 12px; z-index: 9999;">
						  Disclaimer: The Content is being tunneled by <a href="https://uraanstudios.com" 
						  style="color: white; text-decoration: underline;">LocalTunnel</a> 
					  </div>`

			// Ensure the body has padding at the top to avoid content being hidden
			bodyStr = strings.Replace(bodyStr, "<body>", "<body style='padding-top: 25px;'>"+topBar, 1)
		}

		// Remove Content-Length to avoid incorrect content length errors
		w.Header().Del("Content-Length")

		// Write modified response
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(resp.StatusCode)
		_, err = w.Write([]byte(bodyStr))
		if err != nil {
			log.Println("Error writing modified response:", err)
		}
	} else {
		// If not HTML, just stream the response as is
		w.WriteHeader(resp.StatusCode)
		_, err := io.Copy(w, resp.Body)
		if err != nil {
			log.Println("Error copying response body:", err)
		}
	}
}

func StartHTTPServer(httpPort string) error {
	// httpPort := ":5000"
	handler := &HTTPHandler{}

	router := chi.NewRouter()
	router.HandleFunc("/{id}", handler.handleWebhook)
	router.HandleFunc("/{id}/*", handler.handleWebhook)

	return http.ListenAndServe(httpPort, router)
}
