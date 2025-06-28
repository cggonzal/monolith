package jobs

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"monolith/config"
)

type EmailPayload struct {
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Sender  string   `json:"sender"`
	To      []string `json:"to"`
}

// EmailJob sends an email via Mailgun using the REST API.
func EmailJob(payload []byte) error {
	var p EmailPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return err
	}
	if config.MAILGUN_API_KEY == "" || config.MAILGUN_DOMAIN == "" {
		return errors.New("mailgun not configured")
	}
	values := url.Values{}
	values.Set("from", p.Sender)
	values.Set("to", strings.Join(p.To, ","))
	values.Set("subject", p.Subject)
	values.Set("text", p.Body)

	apiURL := fmt.Sprintf("%s/%s/messages", config.MAILGUN_API_BASE, config.MAILGUN_DOMAIN)
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth("api", config.MAILGUN_API_KEY)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("mailgun error: %s", body)
	}
	return nil
}
