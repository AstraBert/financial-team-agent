package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	server := http.NewServeMux()

	server.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		var emailEvent EmailReceivedEvent
		err := json.NewDecoder(r.Body).Decode(&emailEvent)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		inptEvent := InputEvent{Sender: emailEvent.Data.From, Subject: emailEvent.Data.Subject, EmailId: emailEvent.Data.EmailID}
		reqBody := RequestBody{StartEvent: inptEvent, Context: map[string]any{}, HandlerId: ""}
		reqBodyData, err := json.Marshal(reqBody)
		if err != nil {
			w.WriteHeader(500)
			return
		}
		go processEmail(reqBodyData)
		w.WriteHeader(204)
	})

	if err := http.ListenAndServe(":8080", server); err != nil {
		log.Fatalf("There was an error while running the server: %s", err.Error())
	}
}

func processEmail(reqBodyData []byte) {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Second)
	defer cancel()

	endpoint := os.Getenv("LLAMA_CLOUD_API_ENDPOINT")
	apiKey := os.Getenv("LLAMA_CLOUD_API_KEY")

	if endpoint == "" || apiKey == "" {
		log.Println("Missing required environment variables")
		return
	}

	request, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBodyData))
	if err != nil {
		log.Printf("Failed to create request: %v", err)
		return
	}

	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 800 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		log.Printf("Failed to send request to LlamaCloud: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("LlamaCloud API returned error status: %d", resp.StatusCode)
		return
	}

	log.Printf("Successfully processed email and sent to LlamaCloud")
}
