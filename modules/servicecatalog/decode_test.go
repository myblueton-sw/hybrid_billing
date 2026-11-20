package servicecatalog

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestCatalogDecodeRejectsAmbiguousConfiguration(t *testing.T) {
	v := validator(t)
	encoded, _ := json.Marshal(sample())
	valid := string(encoded)
	for name, input := range map[string]string{
		"case alias": `{"AUDIENCES":[]}`, "case duplicate": `{"audiences":[],"AUDIENCES":[]}`, "nested case alias": strings.Replace(valid, `"id":"read"`, `"ID":"read"`, 1), "null": "null", "missing audiences": "{}", "null audiences": `{"audiences":null}`, "wrong audiences": `{"audiences":{}}`, "unknown property": `{"audiences":[],"unknown":true}`, "duplicate audiences": `{"audiences":[],"audiences":[]}`, "escaped duplicate": `{"audiences":[],"\u0061udiences":[]}`, "extra object": `{"audiences":[]} {"audiences":[]}`, "scope null": strings.Replace(valid, `"scopes":[`, `"scopes":null,"discard":[`, 1), "scalar scope": strings.Replace(valid, `"scopes":[`, `"scopes":[true,`, 1), "wrong boolean": strings.Replace(valid, `"any":true`, `"any":"true"`, 1), "duplicate scope key": strings.Replace(valid, `"id":"read"`, `"id":"read","id":"other"`, 1), "null field": strings.Replace(valid, `"fields":["quantity"]`, `"fields":null`, 1), "deep object": strings.Repeat("[", 65) + strings.Repeat("]", 65),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Decode(strings.NewReader(input), v); !errors.Is(err, ErrInvalid) {
				t.Fatal("ambiguous configuration accepted", name, err)
			}
		})
	}
	catalog, err := Decode(strings.NewReader(valid), v)
	if err != nil || catalog.Version() == "" {
		t.Fatal("valid explicit file refused", err)
	}
	empty, err := Decode(strings.NewReader(`{"audiences":[]}`), v)
	if err != nil {
		t.Fatal(err)
	}
	programmatic, _ := New(Config{}, v)
	if empty.Version() != programmatic.Version() {
		t.Fatal("empty representations differ")
	}
	if _, err := Decode(strings.NewReader(strings.Repeat(" ", MaxBytes+1)), v); !errors.Is(err, ErrInvalid) {
		t.Fatal("input byte limit ignored", err)
	}
}

func TestCatalogRecipientAndProgrammaticBounds(t *testing.T) {
	v := validator(t)
	for _, resource := range []string{"", "/relative", "https://user:secret@example.invalid", "https://example.invalid?", "https://example.invalid?token=synthetic", "https://example.invalid#fragment", "urn:", "urn:example", "urn::value", "urn:example:", "javascript:alert(1)", "https://example.invalid/" + strings.Repeat("a", MaxResourceBytes)} {
		cfg := sample()
		cfg.Audiences[0].Resource = resource
		if _, err := New(cfg, v); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid recipient admitted", resource, err)
		}
	}
	for _, resource := range []string{"https://example.invalid/api", "http://127.0.0.1:8894/api", "urn:example:usage"} {
		cfg := sample()
		cfg.Audiences[0].Resource = resource
		if _, err := New(cfg, v); err != nil {
			t.Fatal("valid explicit recipient refused", resource, err)
		}
	}
	if _, err := New(Config{Audiences: make([]AudienceSpec, MaxAudiences+1)}, v); !errors.Is(err, ErrInvalid) {
		t.Fatal("programmatic audience bound bypassed", err)
	}
	cfg := sample()
	cfg.Audiences[0].Scopes = make([]ScopeSpec, MaxScopes+1)
	if _, err := New(cfg, v); !errors.Is(err, ErrInvalid) {
		t.Fatal("programmatic scope bound bypassed", err)
	}
}

func TestCatalogDecodeRequiresExactSelectorAndPeriodVariants(t *testing.T) {
	v := validator(t)
	encoded, _ := json.Marshal(sample())
	for _, variant := range []string{`{"any":false,"id":"customer"}`, `{"any":true,"id":""}`, `{"any":false,"id":""}`} {
		input := strings.Replace(string(encoded), `"customer":{"any":true}`, `"customer":`+variant, 1)
		if _, err := Decode(strings.NewReader(input), v); !errors.Is(err, ErrInvalid) {
			t.Fatal("mixed selector members accepted", variant, err)
		}
	}
	input := strings.Replace(string(encoded), `"period":{"any":true}`, `"period":{"any":true,"from":"","until":""}`, 1)
	if _, err := Decode(strings.NewReader(input), v); !errors.Is(err, ErrInvalid) {
		t.Fatal("mixed period members accepted", err)
	}
}

func TestCatalogDecodePreservesRecipientUnicodeWithoutReplacement(t *testing.T) {
	v := validator(t)
	encoded, _ := json.Marshal(sample())
	for _, resource := range []string{`urn:example:\ud800`, `urn:example:\udc00`, `urn:example:\ud800\u0041`, "urn:example:" + string([]byte{0xff})} {
		input := strings.Replace(string(encoded), "https://usage.invalid/api", resource, 1)
		if _, err := Decode(strings.NewReader(input), v); !errors.Is(err, ErrInvalid) {
			t.Fatal("malformed recipient Unicode was replaced and accepted", err)
		}
	}
	for _, resource := range []string{`urn:example:\ud83d\ude80`, "urn:example:�"} {
		input := strings.Replace(string(encoded), "https://usage.invalid/api", resource, 1)
		catalog, err := Decode(strings.NewReader(input), v)
		if err != nil {
			t.Fatal("valid Unicode recipient refused", resource, err)
		}
		row, _ := catalog.Audience("w1", "usage-api")
		if row.Resource != "urn:example:🚀" && row.Resource != "urn:example:�" {
			t.Fatal("valid recipient identity changed", row.Resource)
		}
	}
}

func TestCatalogProgrammaticByteBudgetCannotBypassFileLimit(t *testing.T) {
	v := validator(t)
	for _, resource := range []string{"urn:example:" + strings.Repeat("a", 1800), "urn:example:" + strings.Repeat("&", 1800)} {
		cfg := Config{Audiences: []AudienceSpec{}}
		count := 600
		if strings.Contains(resource, "&") {
			count = 100
		}
		for i := 0; i < count; i++ {
			row := sample().Audiences[0]
			row.ID = fmt.Sprintf("audience-%03d", i)
			row.Resource = resource
			cfg.Audiences = append(cfg.Audiences, row)
		}
		if _, err := New(cfg, v); !errors.Is(err, ErrInvalid) {
			t.Fatal("programmatic retained byte budget bypassed", err)
		}
	}
	if _, err := Decode(nil, v); !errors.Is(err, ErrInvalid) {
		t.Fatal("absent file reader admitted", err)
	}
}
