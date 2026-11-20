package main

import (
	"bytes"
	"errors"
	"flag"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/apps/api/server"
	"github.com/myblueton-sw/hybrid_billing/modules/servicecatalog"
)

func cliCatalog() string {
	return `{"audiences":[{"workspace":"workspace","id":"usage-api","resource":"https://usage.invalid/api","scopes":[{"id":"read","action":"view","kind":"` + server.ReaderKind + `","fields":["` + server.ReaderField + `"],"customer":{"any":true},"legal_entity":{"any":true},"contract":{"any":true},"period":{"any":true}}]}]}`
}

func TestServiceCatalogCLIStartsWithActualConfiguration(t *testing.T) {
	for _, mode := range []string{"configured", "empty", "omitted"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "catalog.json")
			contents := cliCatalog()
			if mode == "empty" {
				contents = `{"audiences":[]}`
			}
			if err := os.WriteFile(file, []byte(contents), 0600); err != nil {
				t.Fatal(err)
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			address := listener.Addr().String()
			listener.Close()
			args := []string{"--listen", address, "--seed", "none", "--data", filepath.Join(dir, "data")}
			if mode != "omitted" {
				args = append(args, "--service-catalog", file)
			}
			stop := make(chan os.Signal, 1)
			done := make(chan error, 1)
			go func() { done <- run(args, stop) }()
			finished := false
			t.Cleanup(func() {
				if !finished {
					stop <- os.Interrupt
					select {
					case err := <-done:
						if err != nil {
							t.Error(err)
						}
					case <-time.After(10 * time.Second):
						t.Error("CLI did not stop")
					}
				}
			})
			client := &http.Client{Timeout: time.Second}
			healthy := false
			for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
				select {
				case err := <-done:
					finished = true
					t.Fatal("catalog CLI stopped before health", err)
				default:
				}
				response, err := client.Get("http://" + address + "/api/v1/health")
				if err == nil {
					healthy = response.StatusCode == http.StatusOK
					response.Body.Close()
					if healthy {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
			}
			if !healthy {
				t.Fatal("catalog CLI never became healthy")
			}
			if err := os.WriteFile(file, []byte("changed-invalid-catalog"), 0600); err != nil {
				t.Fatal(err)
			}
			response, err := client.Get("http://" + address + "/api/v1/health")
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatal("running installation reread changed file")
			}
			stop <- os.Interrupt
			select {
			case err := <-done:
				finished = true
				if err != nil {
					t.Fatal(err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("catalog CLI shutdown timed out")
			}
		})
	}
}

func TestServiceCatalogCLIFailurePrecedesDataAndListener(t *testing.T) {
	for _, input := range []string{"", "null", "{", strings.Replace(cliCatalog(), server.ReaderKind, "unknown-kind", 1)} {
		for _, occupied := range []bool{false, true} {
			t.Run("invalid", func(t *testing.T) {
				dir := t.TempDir()
				file := filepath.Join(dir, "catalog.json")
				data := filepath.Join(dir, "absent")
				if err := os.WriteFile(file, []byte(input), 0600); err != nil {
					t.Fatal(err)
				}
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				address := listener.Addr().String()
				defer listener.Close()
				if !occupied {
					listener.Close()
				}
				stop := make(chan os.Signal, 1)
				stop <- os.Interrupt
				err = run([]string{"--listen", address, "--seed", "none", "--data", data, "--service-catalog", file}, stop)
				if !errors.Is(err, errServiceCatalog) && !errors.Is(err, server.ErrStart) {
					t.Fatal("catalog refusal did not precede listener/startup", err)
				}
				if strings.Contains(err.Error(), file) || strings.Contains(err.Error(), "unknown-kind") {
					t.Fatal("catalog failure exposed path or contents", err)
				}
				if _, err := os.Stat(data); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("catalog failure initialized installation data", err)
				}
				if !occupied {
					probe, err := net.Listen("tcp", address)
					if err != nil {
						t.Fatal("invalid configuration retained a listener", err)
					}
					probe.Close()
				}
			})
		}
	}
}

func TestServiceCatalogFileReadingIsBoundedAndDoesNotExposePaths(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "catalog.json")
	if err := os.WriteFile(file, []byte(cliCatalog()), 0600); err != nil {
		t.Fatal(err)
	}
	flag := catalogFileFlag{}
	flag.Set(file)
	data, err := flag.load()
	if err != nil || string(data) != cliCatalog() {
		t.Fatal("catalog file was not read exactly", err)
	}
	if err := os.WriteFile(file, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if string(data) != cliCatalog() {
		t.Fatal("file replacement changed already loaded configuration")
	}
	large := filepath.Join(dir, "large.json")
	if err := os.WriteFile(large, []byte(strings.Repeat(" ", servicecatalog.MaxBytes+1)), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(dir, "missing.json"), dir, large} {
		flag := catalogFileFlag{}
		flag.Set(path)
		data, err := flag.load()
		if data != nil || !errors.Is(err, errServiceCatalog) || strings.Contains(err.Error(), path) {
			t.Fatal("invalid local catalog exposed details or was admitted", err)
		}
	}
}

func TestServiceCatalogFlagRejectsDuplicateAndEmptyPathsWithoutParserEcho(t *testing.T) {
	dir := t.TempDir()
	first, second := filepath.Join(dir, "private-first.json"), filepath.Join(dir, "private-second.json")
	for _, path := range []string{first, second} {
		if err := os.WriteFile(path, []byte(cliCatalog()), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"--service-catalog", ""}, {"--service-catalog", first, "--service-catalog", second}} {
		var output bytes.Buffer
		var value catalogFileFlag
		fs := flag.NewFlagSet("catalog-test", flag.ContinueOnError)
		fs.SetOutput(&output)
		fs.Var(&value, "service-catalog", "catalog")
		if err := fs.Parse(args); err != nil {
			t.Fatal("flag parser exposed rejected path", err)
		}
		if _, err := value.load(); !errors.Is(err, errServiceCatalog) {
			t.Fatal("invalid flag did not fail after parsing", err)
		}
		if output.Len() != 0 {
			t.Fatal("parser printed configured paths", output.String())
		}
	}
	value := catalogFileFlag{}
	data, err := value.load()
	if err != nil || data != nil {
		t.Fatal("omitted catalog was not distinguished", err)
	}
}
