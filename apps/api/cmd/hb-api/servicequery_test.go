package main

import (
	"flag"
	"io"
	"os"
	"strings"
	"testing"
)

func TestServiceQueryCLISettingsReachActualDefaultsAndStartValidation(t *testing.T) {
	fs := flag.NewFlagSet("service-query", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	access, given := accessFlags(fs)
	if err := fs.Parse([]string{"--service-query-page-size", "7", "--service-query-scan", "100", "--service-query-lifetime-seconds", "60", "--credential-warning-days", "3"}); err != nil {
		t.Fatal(err)
	}
	given()
	if access.ServiceQueryPage != 7 || access.ServiceQueryScan != 100 || access.ServiceQuerySeconds != 60 || access.CredentialWarningDays != 3 || !access.Given["service_query_page_size"] || !access.Given["credential_warning_days"] || access.Given["service_declaration_scopes"] {
		t.Fatal("query flags did not preserve exact values/source/default separation")
	}
	var usage strings.Builder
	fs.SetOutput(&usage)
	fs.PrintDefaults()
	for _, name := range []string{"service-query-page-size", "service-query-scan", "service-query-lifetime-seconds", "credential-warning-days"} {
		if !strings.Contains(usage.String(), name) {
			t.Fatal("operator query setting missing", name)
		}
	}
	for name, args := range map[string][]string{"page below": {"--service-query-page-size", "0"}, "page above": {"--service-query-page-size", "101"}, "budget below": {"--service-query-scan", "99"}, "budget above": {"--service-query-scan", "10001"}, "lifetime below": {"--service-query-lifetime-seconds", "29"}, "lifetime above": {"--service-query-lifetime-seconds", "3601"}, "warning above": {"--credential-warning-days", "91"}, "noncanonical": {"--service-query-page-size", "07"}} {
		t.Run(name, func(t *testing.T) {
			data := t.TempDir() + "/unopened"
			full := append([]string{"--listen", "127.0.0.1:0", "--data", data}, args...)
			if err := run(full, make(chan os.Signal)); err == nil {
				t.Fatal("invalid query setting started API")
			}
			if _, err := os.Stat(data); !os.IsNotExist(err) {
				t.Fatal("invalid query configuration initialized data", err)
			}
		})
	}
}
