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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/myblueton-sw/hybrid_billing/apps/api/server"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
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
	var catalogFile catalogFileFlag
	fs.Var(&catalogFile, "service-catalog", "optional nonsecret service audience/scope catalog file; at most 1 MiB; supplied once")
	var list, readers origins
	fs.Var(&list, "console-origin", "allowed console origin")
	fs.Var(&readers, "reader", "reader path=http://127.0.0.1:port, repeatable; replaces the defaults")
	access, given := accessFlags(fs)
	org, orgGiven := orgFlags(fs)
	licenses, licensesGiven := pluginFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	given()
	orgGiven()
	licensesGiven()
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
	catalog, err := catalogFile.load()
	if err != nil {
		return err
	}
	s, err := server.New(server.Config{Origins: list, Data: *data, Seed: *seed, Log: log, Readers: routes, Access: *access, Org: *org, Plugins: *licenses, ServiceCatalog: catalog})
	if err != nil {
		return err
	}
	defer s.Close()
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

func accessFlags(fs *flag.FlagSet) (*server.AccessSettings, func()) {
	access := server.DefaultAccessSettings()
	byFlag := map[string]server.AccessSetting{}
	for _, setting := range server.AllAccessSettingList() {
		byFlag[setting.Flag] = setting
		numberFlag(fs, setting.Flag, setting.Name, setting.Unit, setting.Min, setting.Max, setting.Default, server.SettingNote(setting.Name), func(n int) { setting.Set(&access, n) })
	}
	fs.Func("clock-base", "one-time: ledger or wall; accepted only when the decision ledger's latest time is beyond the clock skew tolerance, or when its latest clock base choice cannot be verified", func(raw string) error {
		if raw != server.ClockBaseLedger && raw != server.ClockBaseWall {
			return fmt.Errorf("clock-base must be %s or %s", server.ClockBaseLedger, server.ClockBaseWall)
		}
		access.ClockBase = raw
		return nil
	})
	fs.Func("accept-access-history-shortening", "one-time: apply a shorter access_history_days; the value must equal the new access_history_days", func(raw string) error {
		n, ok := settingNumber(raw)
		if !ok || n < 1 {
			return errors.New("accept-access-history-shortening must be a whole number of days")
		}
		access.AcceptShortening = n
		return nil
	})
	fs.StringVar(&access.AcceptHistoryClock, "accept-access-history-clock", "", "one-time: accept the server time logged when access history deletion was deferred after a clock jump (RFC 3339)")
	fs.StringVar(&access.ReleaseMarker, "release-access-history-marker", "", "one-time: release one held access history marker from an earlier run, with release-access-history-reason")
	fs.StringVar(&access.ReleaseReason, "release-access-history-reason", "", "one-time: the reason recorded for release-access-history-marker")
	fs.BoolVar(&access.AcceptNewLedger, "accept-new-decision-ledger", false, "one-time: keep the isolated decision ledger under a new name and start a new one")
	fs.BoolVar(&access.AcceptTrustReset, "accept-trust-reset", false, "one-time, with accept-new-decision-ledger: the replaced ledger has no carryable trust record; the installation default sign-in rules become active")
	fs.StringVar(&access.CarryTrustFrom, "carry-trust-from", "", "one-time, with accept-new-decision-ledger: carry the latest trust record of the replaced ledger with this genesis ledger ID")
	fs.Func("trust-generation", "one-time, with accept-new-decision-ledger when the marker and the ledger are both unusable: the new ledger generation, above every retained one and below 4294967295", func(raw string) error {
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || strconv.FormatUint(n, 10) != raw || n == 0 || n >= decisionledger.MaxGeneration {
			return fmt.Errorf("trust-generation must be a whole number from 1 to %d", uint64(decisionledger.MaxGeneration)-1)
		}
		access.TrustGeneration = n
		return nil
	})
	return &access, func() {
		fs.Visit(func(f *flag.Flag) {
			if setting, ok := byFlag[f.Name]; ok {
				access.Given[setting.Name] = true
			}
		})
	}
}

func orgFlags(fs *flag.FlagSet) (*server.OrgSettings, func()) {
	org := server.DefaultOrgSettings()
	byFlag := map[string]server.OrgSetting{}
	for _, setting := range server.OrgSettingList() {
		byFlag[setting.Flag] = setting
		numberFlag(fs, setting.Flag, setting.Name, setting.Unit, setting.Min, setting.Max, setting.Default, "", func(n int) { setting.Set(&org, n) })
	}
	return &org, func() {
		fs.Visit(func(f *flag.Flag) {
			if setting, ok := byFlag[f.Name]; ok {
				org.Given[setting.Name] = true
			}
		})
	}
}

func pluginFlags(fs *flag.FlagSet) (*server.PluginSettings, func()) {
	p := &server.PluginSettings{}
	fs.Func("plugin-licenses", "permitted plugin licenses, comma-separated SPDX identifiers or LicenseRef- identifiers without spaces; empty permits none (default "+
		strings.Join(server.DefaultPluginLicenses(), ",")+")", func(raw string) error {
		p.Raw = raw
		p.Uses++
		return nil
	})
	return p, func() {
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "plugin-licenses" {
				p.Given = true
			}
		})
	}
}

func numberFlag(fs *flag.FlagSet, name, setting, unit string, low, high, initial int, note string, set func(int)) {
	fs.Func(name, fmt.Sprintf("%s in %s, %d to %d (default %d)%s", setting, unit, low, high, initial, note), func(raw string) error {
		n, ok := settingNumber(raw)
		if !ok {
			return fmt.Errorf("%s must be a whole number without sign or leading zero", name)
		}
		set(n)
		return nil
	})
}

func settingNumber(raw string) (int, bool) {
	if raw == "" || len(raw) > 8 || (len(raw) > 1 && raw[0] == '0') || strings.TrimLeft(raw, "0123456789") != "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	return n, err == nil
}

func main() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	if err := run(os.Args[1:], stop); err != nil && !errors.Is(err, http.ErrServerClosed) {
		fmt.Fprintln(os.Stderr, "hb-api:", err)
		os.Exit(1)
	}
}
