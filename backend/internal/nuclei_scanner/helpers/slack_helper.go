package helpers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackPayload struct {
	Text string `json:"text"`
}

func SendSlackNotification(webhookURL, message string) error {
	payload := SlackPayload{Text: message}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send Slack notification, status code: %d", resp.StatusCode)
	}

	return nil
} 