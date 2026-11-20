package main

import (
	"flag"
	"io"
	"os"
	"strings"
	"testing"
)

func TestServiceDeclarationCLIFlagsPreserveDefaultsAndExplicitSource(t *testing.T) {
	fs := flag.NewFlagSet("declarations", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	access, given := accessFlags(fs)
	if err := fs.Parse([]string{"--service-declaration-scopes", "3"}); err != nil {
		t.Fatal(err)
	}
	given()
	if access.ServicePurposeBytes != 512 || access.ServiceDeclarationScopes != 3 || access.ServiceDeclarationVersions != 5 || !access.Given["service_declaration_scopes"] || access.Given["service_declaration_versions"] {
		t.Fatal("declaration flags lost defaults or source")
	}
	var usage strings.Builder
	fs.SetOutput(&usage)
	fs.PrintDefaults()
	for _, want := range []string{"service-declaration-purpose-bytes", "service-declaration-scopes", "service-declaration-versions"} {
		if !strings.Contains(usage.String(), want) {
			t.Fatal("missing operator setting", want)
		}
	}
	for name, args := range map[string][]string{
		"purpose zero":      {"--service-declaration-purpose-bytes", "0"},
		"purpose too large": {"--service-declaration-purpose-bytes", "2049"},
		"scope too large":   {"--service-declaration-scopes", "65"},
		"history zero":      {"--service-declaration-versions", "0"},
		"history too large": {"--service-declaration-versions", "21"},
		"noncanonical":      {"--service-declaration-scopes", "03"},
	} {
		t.Run(name, func(t *testing.T) {
			data := t.TempDir() + "/unopened"
			full := append([]string{"--listen", "127.0.0.1:0", "--data", data}, args...)
			if err := run(full, make(chan os.Signal)); err == nil {
				t.Fatal("invalid declaration setting started API")
			}
			if _, err := os.Stat(data); !os.IsNotExist(err) {
				t.Fatal("invalid setting initialized installation data", err)
			}
		})
	}
}
