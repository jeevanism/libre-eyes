package examination

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

func TestMessagingDemoDraftRegistryValidate(t *testing.T) {
	registry := MessagingDemoDraftRegistry{}
	valid := map[string]any{"recordMode": "demo_messaging", "profileCode": "demo_messaging_v1", "messageType": "demo_clinic_update", "primaryRecipient": "demo_gp", "ccRecipients": []string{"demo_consultant"}, "subject": "Review", "body": "Please review this demo draft.", "readState": "demo_unread"}
	payload, _ := json.Marshal(valid)
	if err := registry.Validate(context.Background(), messagingDemoEventType, episodes.DraftIntentCreate, 1, payload); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(map[string]any){
		"duplicate recipient": func(p map[string]any) { p["ccRecipients"] = []string{"demo_gp"} },
		"too many cc": func(p map[string]any) {
			p["ccRecipients"] = []string{"demo_optometrist", "demo_consultant", "demo_clinic_staff", "demo_gp", "demo_review", "demo_extra"}
		},
		"empty body": func(p map[string]any) { p["body"] = "" },
	} {
		t.Run(name, func(t *testing.T) {
			copy := map[string]any{}
			for k, v := range valid {
				copy[k] = v
			}
			mutate(copy)
			bad, _ := json.Marshal(copy)
			if err := registry.Validate(context.Background(), messagingDemoEventType, episodes.DraftIntentCreate, 1, bad); err != episodes.ErrInvalidRequest {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestMessagingDemoDraftRegistryProductionGuard(t *testing.T) {
	if _, err := NewMessagingDemoDraftRegistry("production"); err == nil {
		t.Fatal("expected production guard")
	}
}
