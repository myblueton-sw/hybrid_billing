package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/myblueton-sw/hybrid_billing/apps/api/server"
)

type origins []string

func (o *origins) String() string { return strings.Join(*o, ",") }

func (o *origins) Set(v string) error {
	*o = append(*o, v)
	return nil
}

func run(args []string, stop <-chan os.Signal) error {
	fs := flag.NewFlagSet("hb-api", flag.ContinueOnError)
	listen := fs.String("listen", "127.0.0.1:8820", "loopback address")
	data := fs.String("data", ".hb-api", "data directory")
	seed := fs.String("seed", server.SeedSample, "sample or none")
	var list, readers origins
	fs.Var(&list, "console-origin", "allowed console origin")
	fs.Var(&readers, "reader", "reader path=http://127.0.0.1:port, repeatable; replaces the defaults")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !server.Loopback(*listen) {
		return errors.New("listen address must be loopback")
	}
	log := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	var routes map[string]string
	if len(readers) > 0 {
		routes = map[string]string{}
		for _, r := range readers {
			path, target, ok := strings.Cut(r, "=")
			if !ok {
				return errors.New("reader must be path=url")
			}
			routes[path] = target
		}
	}
	s, err := server.New(server.Config{Origins: list, Data: *data, Seed: *seed, Log: log, Readers: routes})
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		return err
	}
	defer ln.Close()
	if err := s.Bind(ln.Addr()); err != nil {
		return err
	}
	if err := s.Start(); err != nil {
		return err
	}
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second,
		IdleTimeout: 60 * time.Second, MaxHeaderBytes: server.MaxHeaderBytes, ErrorLog: slog.NewLogLogger(log.Handler(), slog.LevelError)}
	errs := make(chan error, 1)
	go func() { errs <- srv.Serve(ln) }()
	log.Info("listening", "address", ln.Addr().String(), "seed", *seed)
	select {
	case err := <-errs:
		return err
	case <-stop:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
	}
}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	if err := run(os.Args[1:], stop); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, "hb-api:", err)
		os.Exit(1)
	}
}
