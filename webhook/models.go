package main

type EmailReceivedEvent struct {
	Type      string    `json:"type"`
	CreatedAt string    `json:"created_at"`
	Data      EmailData `json:"data"`
}

type EmailData struct {
	EmailID     string       `json:"email_id"`
	CreatedAt   string       `json:"created_at"`
	From        string       `json:"from"`
	To          []string     `json:"to"`
	BCC         []string     `json:"bcc"`
	CC          []string     `json:"cc"`
	MessageID   string       `json:"message_id"`
	Subject     string       `json:"subject"`
	Attachments []Attachment `json:"attachments"`
}

type Attachment struct {
	ID                 string `json:"id"`
	Filename           string `json:"filename"`
	ContentType        string `json:"content_type"`
	ContentDisposition string `json:"content_disposition"`
	ContentID          string `json:"content_id"`
}

type RequestBody struct {
	StartEvent InputEvent     `json:"start_event"`
	Context    map[string]any `json:"context"`
	HandlerId  string         `json:"handler_id"`
}

type InputEvent struct {
	Sender  string `json:"sender"`
	Subject string `json:"subject"`
	EmailId string `json:"email_id"`
}
