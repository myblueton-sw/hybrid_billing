package main

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	for name, args := range map[string][]string{
		"flag":     {"--nope"},
		"listen":   {"--listen", "0.0.0.0:0", "--data", t.TempDir() + "/d"},
		"seed":     {"--listen", "127.0.0.1:0", "--seed", "bad", "--data", t.TempDir() + "/d"},
		"origin":   {"--listen", "127.0.0.1:0", "--console-origin", "nope", "--data", t.TempDir() + "/d"},
		"busy":     {"--listen", "127.0.0.1:1", "--data", t.TempDir() + "/d"},
		"bad data": {"--listen", "127.0.0.1:0", "--data", "/dev/null/x"},
	} {
		if err := run(args, make(chan os.Signal)); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	stop := make(chan os.Signal, 1)
	done := make(chan error, 1)
	go func() {
		done <- run([]string{"--listen", addr, "--data", t.TempDir() + "/d", "--console-origin", "http://127.0.0.1:5173"}, stop)
	}()
	var resp *http.Response
	for i := 0; i < 50; i++ {
		resp, err = http.Get("http://" + addr + "/api/v1/health")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("health %v", err)
	}
	resp.Body.Close()
	req, _ := http.NewRequest("GET", "http://"+addr+"/api/v1/health", nil)
	req.Header.Set("X-Big", strings.Repeat("a", 40<<10))
	if big, err := http.DefaultClient.Do(req); err == nil {
		if big.StatusCode != 431 {
			t.Errorf("header limit %d", big.StatusCode)
		}
		big.Body.Close()
	}
	stop <- os.Interrupt
	if err := <-done; err != nil {
		t.Errorf("shutdown %v", err)
	}
	var o origins
	o.Set("a")
	o.Set("b")
	if o.String() != "a,b" {
		t.Error("origins flag")
	}
}
