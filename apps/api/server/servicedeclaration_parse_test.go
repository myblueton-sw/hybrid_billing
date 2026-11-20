package server

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestServiceDeclarationScopeParsingPreservesPairsAndRejectsAmbiguity(t *testing.T) {
	valid := `[{"audience":"a:b","scope":"c"},{"audience":"a","scope":"b:c"}]`
	rows, ok := parseServiceScopes(valid, 2, 4096)
	if !ok || !reflect.DeepEqual(rows, []serviceScopeRef{{"a", "b:c"}, {"a:b", "c"}}) {
		t.Fatal("scope pair identity or canonical order changed", rows, ok)
	}
	for _, raw := range []string{"", `null`, `{}`, `[]`, `[null]`, `[{"audience":"a"}]`, `[{"audience":"a","scope":""}]`, `[{"Audience":"a","scope":"b"}]`, `[{"audience":"a","scope":"b","unknown":"x"}]`, `[{"audience":"a","audience":"b","scope":"c"}]`, `[{"audience":"a","\u0061udience":"b","scope":"c"}]`, `[{"audience":"a","scope":"b"},{"audience":"a","scope":"b"}]`, `[{"audience":"a","scope":true}]`, `[{"audience":"bad id","scope":"b"}]`, valid + `[]`} {
		if _, ok := parseServiceScopes(raw, 2, 4096); ok {
			t.Fatal("ambiguous scope selection accepted", raw)
		}
	}
	if _, ok := parseServiceScopes(valid, 1, 4096); ok {
		t.Fatal("scope count cap ignored")
	}
	if _, ok := parseServiceScopes(valid, 2, len(valid)-1); ok {
		t.Fatal("scope byte cap ignored")
	}
	for _, purpose := range []string{"Batch export", "동기화 작업", "مهمة معالجة"} {
		if !validServicePurpose(purpose, len(purpose)) {
			t.Fatal("valid multilingual purpose refused")
		}
	}
	for _, purpose := range []string{"", " leading", "trailing ", "a\nb", "a\u202Eb"} {
		if validServicePurpose(purpose, 512) {
			t.Fatal("invalid purpose accepted")
		}
	}
	if validServicePurpose("동기화", 8) {
		t.Fatal("purpose byte limit treated as character count")
	}
}

func TestServiceDeclarationSettingsAreBoundedAndExplicit(t *testing.T) {
	for _, setting := range ServiceDeclarationSettingList() {
		for _, value := range []int{setting.Min - 1, setting.Max + 1} {
			cfg := DefaultAccessSettings()
			setting.Set(&cfg, value)
			if !errors.Is(cfg.validate(), ErrConfig) {
				t.Fatal("declaration setting limit ignored", setting.Name, value)
			}
		}
	}
	settings := ServiceDeclarationSettingList()
	settings[0].Name = "changed"
	if strings.HasPrefix(ServiceDeclarationSettingList()[0].Name, "changed") {
		t.Fatal("setting list aliases source")
	}
}
