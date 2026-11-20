package server

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestServiceDeclarationPurposeUsesDecodedCommandText(t *testing.T) {
	for name, wire := range map[string]string{"surrogate": `"\ud800"`, "invalid UTF8": "\"" + string([]byte{0xff}) + "\""} {
		t.Run(name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			fields := f.declaredFields(map[string]string{"name": "decoded", "owner": SampleAdmin, "purpose": "PURPOSE_SENTINEL"})
			body := serviceDeclarationBody("original", "original-key", servicesTarget, f.s.Revision(), fields)
			body = strings.Replace(body, `"PURPOSE_SENTINEL"`, wire, 1)
			got := f.call("service.create", body, root.cookie, root.csrf)
			if got.status != 200 {
				t.Fatal("inherited decoded-text boundary changed", got.status, got.body["code"])
			}
			record, known := f.s.serviceDeclaration(SampleWorkspace, "svc-decoded", 0)
			if !known || record.Purpose != "\uFFFD" {
				t.Fatal("decoded replacement not recorded truthfully")
			}
		})
	}
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.run(root, "service.create", servicesTarget, f.declaredFields(map[string]string{"name": "control", "owner": SampleAdmin, "purpose": "a\u202Eb"}))
	f.failed(t, "decoded override refused", got, commands.InvalidRequest, "purpose")
}

func TestServiceDeclarationOpenAPIRequiresActualCommandFields(t *testing.T) {
	f := setup(t, SeedSample).ready()
	var doc map[string]any
	if err := json.Unmarshal(f.mustOpenAPI(), &doc); err != nil {
		t.Fatal(err)
	}
	paths := doc["paths"].(map[string]any)
	for command, required := range map[string][]any{
		"service.create":  {"catalog_version", "name", "organization", "owner", "purpose", "scopes"},
		"service.declare": {"catalog_version", "declaration_version", "organization", "purpose", "scopes"},
	} {
		post := paths["/commands/"+command].(map[string]any)["post"].(map[string]any)
		if post["x-supported"] != true || post["x-mutating"] != true || post["x-step-up"] != true {
			t.Fatal("declaration command lacks boundary contract", command)
		}
		schema := post["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
		body := schema["properties"].(map[string]any)["body"].(map[string]any)
		if !reflect.DeepEqual(body["required"], required) || body["additionalProperties"] != false {
			t.Fatal("published declaration fields differ", command, body["required"])
		}
		for _, field := range required {
			property := body["properties"].(map[string]any)[field.(string)].(map[string]any)
			if property["type"] != "string" {
				t.Fatal("declaration escaped string-valued command contract", command, field)
			}
		}
	}
}
