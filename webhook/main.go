package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/time/rate"
)

func main() {
	server := http.NewServeMux()
	limiter := rate.NewLimiter(1, 1)
	phMonitor := NewPosthogMonitor(os.Getenv("POSTHOG_API_KEY"), os.Getenv("POSTHOG_ENDPOINT"))

	server.HandleFunc("POST /webhook", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		if !limiter.Allow() {
			errPh := phMonitor.SendEvent("tooManyRequests", "rateLimitExceeded", "", 0, true, "Exceeded rate limit of 1 request per second")
			if errPh != nil {
				log.Printf("Failed to send event to PostHog because of %v\n", errPh)
			}
			log.Println("Too many requests, bounced")
			w.WriteHeader(429)
			return
		}
		var emailEvent EmailReceivedEvent
		err := json.NewDecoder(r.Body).Decode(&emailEvent)
		if err != nil {
			errPh := phMonitor.SendEvent("emailReceivedFailure", "failedToRead", "decodeError", time.Since(start).Milliseconds(), true, "Failed to unmarshal incoming request body")
			if errPh != nil {
				log.Printf("Failed to send event to PostHog because of %v\n", errPh)
			}
			log.Printf("Failed to unmarshal incoming request body")
			w.WriteHeader(400)
			return
		}
		inptEvent := InputEvent{Sender: emailEvent.Data.From, Subject: emailEvent.Data.Subject, EmailId: emailEvent.Data.EmailID}
		errPh := phMonitor.SendEvent(inptEvent.EmailId, "emailReceived", fmt.Sprintf("emailBy%s", inptEvent.Sender), time.Since(start).Milliseconds(), false, "")
		if errPh != nil {
			log.Printf("Failed to send event to PostHog because of %v\n", err)
		}
		reqBody := RequestBody{StartEvent: inptEvent, Context: map[string]any{}, HandlerId: ""}
		reqBodyData, err := json.Marshal(reqBody)
		if err != nil {
			errPh := phMonitor.SendEvent("requestToLlamaAgent", "requestFailed", "marshalError", time.Since(start).Milliseconds(), true, "Failed to marshal request body")
			if errPh != nil {
				log.Printf("Failed to send event to PostHog because of %v\n", errPh)
			}
			w.WriteHeader(500)
			log.Println("Failed to marshal request body")
			return
		}
		go processEmail(reqBodyData, phMonitor)
		w.WriteHeader(204)
	})

	if err := http.ListenAndServe(":8080", server); err != nil {
		log.Fatalf("There was an error while running the server: %s", err.Error())
	}
}

func processEmail(reqBodyData []byte, phMonitor *PosthogMonitor) {
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Second)
	defer cancel()

	start := time.Now()
	endpoint := os.Getenv("LLAMA_CLOUD_API_ENDPOINT")
	apiKey := os.Getenv("LLAMA_CLOUD_API_KEY")

	if endpoint == "" || apiKey == "" {
		errPh := phMonitor.SendEvent("requestToLlamaAgent", "requestFailed", "missingEnvVariables", time.Since(start).Milliseconds(), true, "Missing required environment variables")
		if errPh != nil {
			log.Printf("Failed to send event to PostHog because of %v\n", errPh)
		}
		log.Println("Missing required environment variables")
		return
	}

	request, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(reqBodyData))
	if err != nil {
		errPh := phMonitor.SendEvent("requestToLlamaAgent", "requestFailed", "failedToCreateRequest", time.Since(start).Milliseconds(), true, err.Error())
		if errPh != nil {
			log.Printf("Failed to send event to PostHog because of %v\n", err)
		}
		log.Printf("Failed to create request: %v", err)
		return
	}

	request.Header.Set("Authorization", "Bearer "+apiKey)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 800 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		errPh := phMonitor.SendEvent("requestToLlamaAgent", "requestFailed", "failedToSendRequest", time.Since(start).Milliseconds(), true, err.Error())
		if errPh != nil {
			log.Printf("Failed to send event to PostHog because of %v\n", err)
		}
		log.Printf("Failed to send request to LlamaCloud: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errPh := phMonitor.SendEvent("responseFromLlamaAgent", "responseError", fmt.Sprintf("status%d", resp.StatusCode), time.Since(start).Milliseconds(), true, "Server sent an error status code")
		if errPh != nil {
			log.Printf("Failed to send event to PostHog because of %v\n", err)
		}
		log.Printf("LlamaCloud API returned error status: %d", resp.StatusCode)
		return
	}

	errPh := phMonitor.SendEvent("responseFromLlamaAgent", "responseSuccess", fmt.Sprintf("status%d", resp.StatusCode), time.Since(start).Milliseconds(), false, "")
	if errPh != nil {
		log.Printf("Failed to send event to PostHog because of %v\n", err)
	}
	log.Printf("Successfully processed email and sent to LlamaCloud")
}
