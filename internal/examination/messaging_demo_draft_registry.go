package examination

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/jeevanism/visionopus/internal/episodes"
)

const messagingDemoEventType = "ophthalmology.messaging_demo"

type MessagingDemoDraftRegistry struct{}

func NewMessagingDemoDraftRegistry(environment string) (*MessagingDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration messaging is enabled")
	}
	return &MessagingDemoDraftRegistry{}, nil
}

type messagingDemoPayload struct {
	RecordMode       string   `json:"recordMode"`
	ProfileCode      string   `json:"profileCode"`
	MessageType      string   `json:"messageType"`
	PrimaryRecipient string   `json:"primaryRecipient"`
	CCRecipients     []string `json:"ccRecipients"`
	Subject          string   `json:"subject"`
	Body             string   `json:"body"`
	ReadState        string   `json:"readState"`
}

func (MessagingDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != messagingDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p messagingDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_messaging" || p.ProfileCode != "demo_messaging_v1" ||
		!allowedMessagingType(p.MessageType) || !allowedRecipient(p.PrimaryRecipient) || !validMessageText(p.Subject, 160) || !validMessageText(p.Body, 4000) ||
		(p.ReadState != "" && p.ReadState != "demo_unread" && p.ReadState != "demo_read") || len(p.CCRecipients) > 5 {
		return episodes.ErrInvalidRequest
	}
	seen := map[string]bool{p.PrimaryRecipient: true}
	for _, recipient := range p.CCRecipients {
		if !allowedRecipient(recipient) || seen[recipient] {
			return episodes.ErrInvalidRequest
		}
		seen[recipient] = true
	}
	return nil
}

func allowedMessagingType(value string) bool {
	return value == "demo_clinic_update" || value == "demo_review_request" || value == "demo_task_note"
}

func allowedRecipient(value string) bool {
	return value == "demo_gp" || value == "demo_optometrist" || value == "demo_consultant" || value == "demo_clinic_staff"
}

func validMessageText(value string, max int) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && trimmed == value && len(value) <= max
}
