package examination

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jeevanism/libre-eyes/internal/episodes"
)

const catpromDemoEventType = "ophthalmology.catprom_demo"

type CatpromDemoDraftRegistry struct{}

func NewCatpromDemoDraftRegistry(environment string) (*CatpromDemoDraftRegistry, error) {
	if environment == "production" {
		return nil, errors.New("production startup refused: demonstration cataract PROM is enabled")
	}
	return &CatpromDemoDraftRegistry{}, nil
}

type catpromDemoAnswer struct {
	QuestionCode string `json:"questionCode"`
	AnswerCode   string `json:"answerCode"`
}
type catpromDemoPayload struct {
	RecordMode  string              `json:"recordMode"`
	ProfileCode string              `json:"profileCode"`
	Laterality  string              `json:"laterality"`
	Answers     []catpromDemoAnswer `json:"answers"`
	Comment     string              `json:"comment"`
}

func (CatpromDemoDraftRegistry) Validate(_ context.Context, eventTypeCode string, intent episodes.DraftIntent, schemaVersion int64, payload json.RawMessage) error {
	if eventTypeCode != catpromDemoEventType || (intent != episodes.DraftIntentCreate && intent != episodes.DraftIntentUpdate) || schemaVersion != 1 || len(payload) == 0 || len(payload) > 16*1024 {
		return episodes.ErrInvalidRequest
	}
	var p catpromDemoPayload
	if decodeStrictJSON(payload, &p) != nil || p.RecordMode != "demo_catprom" || p.ProfileCode != "demo_catprom_v1" || len(p.Answers) != 6 || len(p.Comment) > 500 {
		return episodes.ErrInvalidRequest
	}
	if p.Laterality != "" && p.Laterality != "not_applicable" && p.Laterality != "right" && p.Laterality != "left" && p.Laterality != "bilateral" {
		return episodes.ErrInvalidRequest
	}
	seen := map[string]bool{}
	for _, answer := range p.Answers {
		if !validCatpromCode(answer.QuestionCode, answer.AnswerCode) || seen[answer.QuestionCode] {
			return episodes.ErrInvalidRequest
		}
		seen[answer.QuestionCode] = true
	}
	for i := 1; i <= 6; i++ {
		if !seen[fmt.Sprintf("demo_q%d", i)] {
			return episodes.ErrInvalidRequest
		}
	}
	return nil
}

func validCatpromCode(question, answer string) bool {
	if len(question) != 7 || question[:6] != "demo_q" || question[6] < '1' || question[6] > '6' {
		return false
	}
	return len(answer) == 10 && answer[:9] == question+"_a" && answer[9] >= '1' && answer[9] <= '4'
}
