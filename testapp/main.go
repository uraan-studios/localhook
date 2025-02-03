package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
)

func main() {
	router := chi.NewRouter()
	router.Post("/payment/webhook", handlePamentWebhook)

	http.ListenAndServe(":3000", router)
}

type WebhookRequest struct {
	Amount  int    `json:"amount"`
	Message string `json:"message"`
}

func handlePamentWebhook(w http.ResponseWriter, r *http.Request) {
	var req WebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Fatal(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	fmt.Println("we got our request", req)
	// SET HEADERS TO 404
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Payment webhook received"))
}
