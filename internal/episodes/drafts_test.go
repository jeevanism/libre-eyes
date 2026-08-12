package episodes

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateDraftPayloadBounds(t *testing.T) {
	if err := validateDraftPayload(json.RawMessage(`{"top":{"leaf":1}}`)); err != nil {
		t.Fatalf("valid payload error = %v", err)
	}
	if err := validateDraftPayload(json.RawMessage(`[]`)); err == nil {
		t.Fatal("array payload accepted")
	}
	deep := "1"
	for range maximumDraftDepth + 1 {
		deep = `{"x":` + deep + `}`
	}
	if err := validateDraftPayload(json.RawMessage(deep)); err == nil {
		t.Fatal("depth-17 payload accepted")
	}
	keys := make([]string, maximumDraftKeys+1)
	for index := range keys {
		keys[index] = `"k` + string(rune('a'+index%26)) + strings.Repeat("x", index/26) + `":1`
	}
	if err := validateDraftPayload(json.RawMessage(`{` + strings.Join(keys, ",") + `}`)); err == nil {
		t.Fatal("payload with too many keys accepted")
	}
	oversized := json.RawMessage(`{"field":"` + strings.Repeat("x", 64*1024) + `"}`)
	if err := validateDraftPayload(oversized); err == nil {
		t.Fatal("payload larger than 64 KiB accepted")
	}
}
