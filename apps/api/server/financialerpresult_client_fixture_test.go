package server

import (
	"encoding/json"
	"os"
	"testing"
)

func TestFinancialERPResultClientFixture(t *testing.T) {
	path := os.Getenv("HB_RESULT_CLIENT_FIXTURE")
	if path == "" {
		t.Skip("explicit synthetic client fixture output required")
	}
	w, p, digest := publicResult(t)
	actor := w.actors[Admin2Login]
	calls := map[string]reply{
		"preparations": documentRead(w.f, actor, "result_evidence.preparations", SampleWorkspace, map[string]string{}),
		"context":      documentRead(w.f, actor, "result_evidence.context", p.Preparation, map[string]string{"digest": p.PreparationDigest}),
		"history":      documentRead(w.f, actor, "result_evidence.history", p.Preparation, map[string]string{"digest": p.PreparationDigest}),
		"inspection":   documentRead(w.f, actor, "result_evidence.inspect", "public-result", map[string]string{"digest": digest}),
		"advice":       documentRead(w.f, actor, "result_evidence.advice", "public-result", map[string]string{"digest": digest}),
	}
	output := map[string]json.RawMessage{}
	for name, result := range calls {
		if result.status != 200 {
			t.Fatalf("%s response status %d", name, result.status)
		}
		output[name] = json.RawMessage(result.raw)
	}
	raw, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}
