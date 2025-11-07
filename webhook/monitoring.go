package main

import (
	"github.com/posthog/posthog-go"
)

type PosthogClient interface {
	GetClient() (posthog.Client, error)
	SendEvent(string, string, string, int64, bool, string) error
}

type PosthogMonitor struct {
	ApiKey   string
	Endpoint string
}

func (p *PosthogMonitor) GetClient() (posthog.Client, error) {
	client, err := posthog.NewWithConfig(p.ApiKey, posthog.Config{Endpoint: p.Endpoint})
	return client, err
}

func (p *PosthogMonitor) SendEvent(uniqueId, eventCategory, eventType string, latency int64, failed bool, failReason string) error {
	client, err := p.GetClient()
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.Enqueue(
		posthog.Capture{
			DistinctId: uniqueId,
			Event:      eventCategory,
			Properties: posthog.NewProperties().
				Set("eventType", eventType).
				Set("latentcy", latency).
				Set("isError", failed),
		},
	)
	return err
}

func NewPosthogMonitor(apiKey, endpoint string) *PosthogMonitor {
	return &PosthogMonitor{
		ApiKey:   apiKey,
		Endpoint: endpoint,
	}
}
