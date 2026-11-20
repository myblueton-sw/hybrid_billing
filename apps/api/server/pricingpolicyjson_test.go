package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

func TestPricingPolicyJSONRefusesAmbiguityAndMissingTerms(t *testing.T) {
	valid := policySpecText(t, policyHTTPSpec())
	for name, raw := range map[string]string{
		"duplicate":             strings.Replace(valid, `"Mode":"synthetic"`, `"Mode":"synthetic","Mode":"synthetic"`, 1),
		"case":                  strings.Replace(valid, `"Mode":"synthetic"`, `"mode":"synthetic"`, 1),
		"case-alias":            strings.Replace(valid, `"Mode":"synthetic"`, `"Mode":"synthetic","mode":"synthetic"`, 1),
		"unknown":               strings.Replace(valid, `"Mode":"synthetic"`, `"Mode":"synthetic","extra":1`, 1),
		"nested-duplicate":      strings.Replace(valid, `"Lines":1000`, `"Lines":1000,"Lines":1000`, 1),
		"nested-case":           strings.Replace(valid, `"Lines":1000`, `"lines":1000`, 1),
		"fractional-integer":    strings.Replace(valid, `"Lines":1000`, `"Lines":1000.0`, 1),
		"exponent-integer":      strings.Replace(valid, `"Lines":1000`, `"Lines":1e3`, 1),
		"string-integer":        strings.Replace(valid, `"Lines":1000`, `"Lines":"1000"`, 1),
		"overflow-integer":      strings.Replace(valid, `"Lines":1000`, `"Lines":99999999999999999999999`, 1),
		"null":                  strings.Replace(valid, `"Discounts":[]`, `"Discounts":null`, 1),
		"missing-mode":          strings.Replace(valid, `"Mode":"synthetic",`, "", 1),
		"missing-interval":      strings.Replace(valid, `"From":"2026-10-01T00:00:00Z",`, "", 1),
		"missing-customer":      strings.Replace(valid, `"Customer":"sample-le-north",`, "", 1),
		"missing-profile-bound": strings.Replace(valid, `"InputScale":4,`, "", 1),
		"foreign-workspace":     strings.Replace(valid, `"Workspace":"`+SampleWorkspace+`"`, `"Workspace":"other"`, 1),
		"trailing":              valid + `{}`,
		"null-root":             "null",
		"over-limit":            strings.Repeat(" ", pricingPolicyFieldBytes) + valid,
		"invalid-utf8":          strings.Replace(valid, "synthetic", "synthetic"+string([]byte{0xff}), 1),
		"high-surrogate":        strings.Replace(valid, "synthetic", `synthetic\ud800`, 1),
		"low-surrogate":         strings.Replace(valid, "synthetic", `synthetic\udfff`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if raw == valid {
				t.Fatal("fixture did not change input")
			}
			if _, err := parsePricingSpec(raw, SampleWorkspace); err == nil {
				t.Fatal("invalid spec accepted")
			}
		})
	}
	if _, err := parsePricingSpec(valid, SampleWorkspace); err != nil {
		t.Fatal(err)
	}
}

func TestPricingPolicyJSONOptionalFieldsAndInspectExpansion(t *testing.T) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(policySpecText(t, policyHTTPSpec())), &payload); err != nil {
		t.Fatal(err)
	}
	policy := payload["Policy"].(map[string]any)
	policy["Rules"] = []map[string]any{{"ID": "pass", "Level": "default", "Type": "pass_through"}}
	body, _ := json.Marshal(payload)
	parsed, err := parsePricingSpec(string(body), SampleWorkspace)
	if err != nil {
		t.Fatal("valid optional zero fields refused", err)
	}
	if len(parsed.Policy.Rules) != 1 || parsed.Policy.Rules[0].Rate != "" || parsed.Policy.Rules[0].Price.Per != 0 {
		t.Fatal("optional pricing defaults invented")
	}
	rules := []map[string]any{}
	for i := 0; i < 19; i++ {
		rules = append(rules, map[string]any{"ID": fmt.Sprintf("r%d", i), "Level": "sku", "Key": fmt.Sprintf("sku%d", i), "Type": "pass_through"})
	}
	rules = append(rules, map[string]any{"ID": "pass", "Level": "default", "Type": "pass_through"})
	policy["Rules"] = rules
	body, _ = json.Marshal(payload)
	if len(body) >= pricingPolicyFieldBytes {
		t.Fatal("input expansion fixture already too large")
	}
	var expanded pricingpolicy.Spec
	if err := json.Unmarshal(body, &expanded); err != nil {
		t.Fatal(err)
	}
	if err := pricingpolicy.ValidateSpec(expanded); err != nil {
		t.Fatal("expansion fixture has invalid pricing", err)
	}
	if _, err := pricingSpecJSON(expanded); err == nil {
		t.Fatal("fixture does not expand past output bound")
	}
	if _, err := parsePricingSpec(string(body), SampleWorkspace); err == nil {
		t.Fatal("uninspectable expanded spec admitted")
	}
}

func TestPricingPolicyUnicodeEscapes(t *testing.T) {
	for _, raw := range []string{`"\ud800"`, `"\udfff"`, `"\ud800\u0041"`, `"\ud800\ud800"`, `"\u12"`, `"\uxxxx"`} {
		if pricingJSONUnicode(raw) {
			t.Fatal("lossy Unicode accepted", raw)
		}
	}
	for _, raw := range []string{`"\ud83d\ude00"`, `"\\ud800"`, `"\u0041"`, `"한글"`, `"العربية"`} {
		if !pricingJSONUnicode(raw) {
			t.Fatal("valid Unicode refused", raw)
		}
	}
}
