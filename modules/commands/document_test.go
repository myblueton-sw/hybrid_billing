package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"math/rand"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

var update = flag.Bool("update", false, "rewrite the golden OpenAPI document")

func TestOpenAPIDocument(t *testing.T) {
	f := setup(t)
	document, err := f.c.OpenAPI("Synthetic commands", "0.0.0-provisional")
	f.must(err)
	const golden = "testdata/openapi.golden.json"
	if *update {
		f.must(os.MkdirAll("testdata", 0o755))
		f.must(os.WriteFile(golden, append(document, '\n'), 0o644))
	}
	want, err := os.ReadFile(golden)
	f.must(err)
	if !bytes.Equal(append(document, '\n'), want) {
		t.Errorf("the document differs from %s; run the test with -update after checking the change", golden)
	}
	again, _ := f.c.OpenAPI("Synthetic commands", "0.0.0-provisional")
	if !bytes.Equal(document, again) {
		t.Errorf("two renderings differ")
	}
	reversed := setupWith(t, testLimits)
	other, err := New(reversed.g, testLimits)
	f.must(err)
	entries := f.c.Catalogue()
	f.c.mu.Lock()
	definitions := []Definition{}
	for index := len(entries) - 1; index >= 0; index-- {
		definitions = append(definitions, f.c.definitions[entries[index].ID])
	}
	f.c.mu.Unlock()
	for _, definition := range definitions {
		f.must(other.Register(definition))
	}
	if reordered, _ := other.OpenAPI("Synthetic commands", "0.0.0-provisional"); !bytes.Equal(document, reordered) {
		t.Errorf("registration order changes the document")
	}

	var parsed map[string]any
	f.must(json.Unmarshal(document, &parsed))
	if parsed["openapi"] != "3.1.0" || parsed["x-provisional"] != true || parsed["x-anonymous-access"] != false {
		t.Errorf("markers %v %v %v", parsed["openapi"], parsed["x-provisional"], parsed["x-anonymous-access"])
	}
	if _, declared := parsed["security"]; declared {
		t.Errorf("the document declares a security requirement")
	}
	if _, declared := parsed["components"].(map[string]any)["securitySchemes"]; declared {
		t.Errorf("the document declares a security scheme")
	}
	description := parsed["info"].(map[string]any)["description"].(string)
	for _, phrase := range []string{"not a published contract", "no operation is anonymous", "No HTTP path is deployed"} {
		if !strings.Contains(strings.ToLower(description), strings.ToLower(phrase)) {
			t.Errorf("the description lacks %q", phrase)
		}
	}
	paths := parsed["paths"].(map[string]any)
	if len(paths) != len(entries) {
		t.Errorf("%d paths for %d commands", len(paths), len(entries))
	}
	for _, entry := range entries {
		operation := paths["/commands/"+entry.ID].(map[string]any)["post"].(map[string]any)
		if operation["operationId"] != entry.ID || operation["x-supported"] != entry.Supported || operation["x-mutating"] != entry.Mutating {
			t.Errorf("%s: operation %v", entry.ID, operation)
		}
		responses := operation["responses"].(map[string]any)
		wanted := []string{"401", "500", "501", "503"}
		if entry.Supported {
			wanted = append(wanted, "200", "403", "409", "422")
		}
		if entry.Supported && entry.Mutating {
			wanted = append(wanted, "202")
		}
		if len(responses) != len(wanted) {
			t.Errorf("%s: %d statuses; want %v", entry.ID, len(responses), wanted)
		}
		for _, status := range wanted {
			if _, listed := responses[status]; !listed {
				t.Errorf("%s: status %s is missing", entry.ID, status)
			}
		}
		if _, listed := responses["404"]; listed {
			t.Errorf("%s: 404 belongs to no operation", entry.ID)
		}
		if _, listed := responses["201"]; listed {
			t.Errorf("%s: 201 is not used", entry.ID)
		}
		request := operation["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
		properties := request["properties"].(map[string]any)
		_, key := properties["idempotency_key"]
		_, revision := properties["expected_revision"]
		_, rationale := properties["rationale"]
		if key != entry.Mutating || revision != entry.Mutating || rationale != entry.Rationale || request["additionalProperties"] != false {
			t.Errorf("%s: request members %v", entry.ID, properties)
		}
		permission := operation["x-permission"].(map[string]any)
		if permission["action"] != string(entry.Permission.Action) || permission["kind"] != entry.Permission.Kind || permission["approval"] != entry.Approval {
			t.Errorf("%s: permission %v", entry.ID, permission)
		}
	}
	amount := paths["/commands/statement.adjust"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)["body"].(map[string]any)
	decimalSchema := amount["properties"].(map[string]any)["amount"].(map[string]any)
	if decimalSchema["type"] != "string" || decimalSchema["pattern"] != decimalForm.String() || amount["required"].([]any)[0] != "amount" {
		t.Errorf("the decimal field %v required %v", decimalSchema, amount["required"])
	}
	described := func(command, status string) string {
		return paths["/commands/"+command].(map[string]any)["post"].(map[string]any)["responses"].(map[string]any)[status].(map[string]any)["description"].(string)
	}
	for name, c := range map[string][3]string{
		"a reading command cannot conflict on a key":     {"statement.read", "409", "revision_conflict"},
		"a mutating command can":                         {"statement.adjust", "409", "idempotency_conflict, revision_conflict"},
		"only an approval can be a self approval":        {"statement.approve", "403", "permission_denied, self_approval_denied"},
		"an ordinary command cannot":                     {"statement.adjust", "403", "permission_denied"},
		"a command without a handler is not implemented": {"statement.planned", "501", "unsupported_capability"},
	} {
		if got := described(c[0], c[1]); got != c[2] {
			t.Errorf("%s: %s %s lists %q; want %q", name, c[0], c[1], got, c[2])
		}
	}
	schemas := parsed["components"].(map[string]any)["schemas"].(map[string]any)
	failed := schemas["Error"].(map[string]any)
	if len(failed["properties"].(map[string]any)["next_action"].(map[string]any)["enum"].([]any)) != 6 {
		t.Errorf("the next actions are not a closed list: %v", failed["properties"])
	}
	for name, schema := range schemas {
		if _, closed := schema.(map[string]any)["additionalProperties"]; closed {
			t.Errorf("%s forbids members a later version may add", name)
		}
	}
	body := paths["/commands/statement.adjust"].(map[string]any)["post"].(map[string]any)["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)
	if body["body"].(map[string]any)["x-max-value-bytes"] != float64(testLimits.Value) || body["rationale"].(map[string]any)["x-max-bytes"] != float64(testLimits.Rationale) {
		t.Errorf("the byte limits are not in the document: %v", body["rationale"])
	}
	if strings.Contains(string(document), "maxLength") {
		t.Errorf("the document states a limit in characters while the executor counts bytes")
	}
	listed := []string{}
	for _, item := range parsed["x-errors"].([]any) {
		listed = append(listed, item.(map[string]any)["code"].(string))
		if _, reading := item.(map[string]any)["retry_for_reading_command"]; reading != (item.(map[string]any)["code"] == string(RevisionConflict)) {
			t.Errorf("the reading rule of %v", item)
		}
	}
	codes := []string{}
	for code := range errorRules {
		codes = append(codes, string(code))
	}
	sort.Strings(codes)
	if strings.Join(listed, ",") != strings.Join(codes, ",") || len(codes) != 13 {
		t.Errorf("error catalogue %v", listed)
	}
	empty, err := New(f.g, testLimits)
	f.must(err)
	blank, err := empty.OpenAPI("Synthetic commands", "0")
	if err != nil || !json.Valid(blank) || !strings.Contains(string(blank), `"paths": {}`) {
		t.Errorf("an empty catalogue: %v %s", err, blank)
	}
	for name, arguments := range map[string][2]string{"no title": {"", "1"}, "no version": {"t", ""}} {
		if _, err := f.c.OpenAPI(arguments[0], arguments[1]); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestTheErrorCatalogue(t *testing.T) {
	want := map[Code]errorRule{
		Unauthenticated: {401, false, NextAnswer}, PermissionDenied: {403, false, NextNone}, SelfApprovalDenied: {403, false, NextNone},
		RevisionConflict: {409, false, NextInspect}, IdempotencyConflict: {409, false, NextInspect},
		InvalidRequest: {422, false, NextCorrect}, InvalidStage: {422, false, NextCorrect}, PrerequisiteFailed: {422, false, NextCorrect},
		UnsupportedCommand: {404, false, NextNone}, UnsupportedCapability: {501, false, NextNone},
		AuthorityUnavailable: {503, true, NextRetry}, DependencyUnavailable: {503, true, NextRetry}, InternalError: {500, false, NextNone},
	}
	if len(errorRules) != len(want) {
		t.Fatalf("%d codes", len(errorRules))
	}
	for code, rule := range want {
		if errorRules[code] != rule {
			t.Errorf("%s: %+v; want %+v", code, errorRules[code], rule)
		}
	}
	for code := range handlerCodes {
		if _, listed := errorRules[code]; !listed {
			t.Errorf("handler code %s is not in the catalogue", code)
		}
	}
	if len(handlerCodes) != 7 || handlerCodes[Unauthenticated] || handlerCodes[InternalError] || handlerCodes[IdempotencyConflict] {
		t.Errorf("handler codes %v", handlerCodes)
	}
}

func TestTheDecimalFormAgreesWithTheDecimalPrimitive(t *testing.T) {
	profile, err := decimal.NewProfile(decimal.ProfileSpec{ID: "synthetic-wide", Revision: "1", InputIntegerDigits: 40, InputScale: 40, ResultIntegerDigits: 60, ResultScale: 40, IntermediateDigits: 200})
	if err != nil {
		t.Fatal(err)
	}
	literal := []string{"0", "-0", "1", "-1", "10", "0.5", "-0.5", "1.0", "1.00", "123456789.000000001", "01", "00", "1.", ".5", "-", "", "+1", "1e3", "1E3", "NaN", "Infinity",
		"-Infinity", "1,5", "1 000", " 1", "1 ", "0x10", "--1", "1.2.3", "１", "1_000", "-.5", "-01", "0.", "٣"}
	r := rand.New(rand.NewSource(176))
	alphabet := []string{"0", "1", "9", ".", "-", "+", "e", " ", ",", "5", "00"}
	for index := 0; index < 20000; index++ {
		text := ""
		for length := r.Intn(7); length >= 0; length-- {
			text += alphabet[r.Intn(len(alphabet))]
		}
		literal = append(literal, text)
	}
	accepted, refused := 0, 0
	for _, text := range literal {
		_, parseErr := decimal.Parse(profile, text)
		form := decimalForm.MatchString(text)
		if form != (parseErr == nil) {
			t.Fatalf("%q: the form accepts %v, the primitive %v", text, form, parseErr == nil)
		}
		if form {
			accepted++
		} else {
			refused++
		}
	}
	if accepted < 500 || refused < 500 {
		t.Errorf("%d accepted and %d refused", accepted, refused)
	}
}

func TestThePayloadDigestIsInjective(t *testing.T) {
	r := rand.New(rand.NewSource(1760))
	parts := []string{"", "a", "b", "ab", "a\x00", "1", "1.0", "amount", "note"}
	seen := map[string]string{}
	pick := func() string { return parts[r.Intn(len(parts))] }
	for index := 0; index < 20000; index++ {
		call := Call{Command: pick(), Target: pick(), Expected: pick(), Rationale: pick()}
		values := map[string]string{}
		for count := r.Intn(3); count > 0; count-- {
			values[pick()] = pick()
		}
		description := call.Command + "\x01" + call.Target + "\x01" + call.Expected + "\x01" + call.Rationale
		names := []string{}
		for name := range values {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			description += "\x02" + name + "\x03" + values[name]
		}
		encoded := string(payload(call, values))
		if earlier, found := seen[encoded]; found && earlier != description {
			t.Fatalf("two payloads share one encoding: %q and %q", earlier, description)
		}
		seen[encoded] = description
	}
	distinct := map[string]bool{}
	for _, description := range seen {
		distinct[description] = true
	}
	if len(distinct) != len(seen) || len(seen) < 3000 {
		t.Errorf("%d encodings for %d payloads", len(seen), len(distinct))
	}
	absent := payload(Call{Command: "c"}, map[string]string{})
	present := payload(Call{Command: "c"}, map[string]string{"note": ""})
	if bytes.Equal(absent, present) {
		t.Errorf("an absent field and a present empty one have the same encoding")
	}
}
