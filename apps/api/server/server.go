package server

import (
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	Installation    = "prelaunch"
	ConsoleClient   = "console"
	SeedSample      = "sample"
	SeedNone        = "none"
	maxOrigins      = 4
	maxBodyBytes    = 1 << 20
	sessionIdle     = 30 * time.Minute
	sessionAbsolute = 12 * time.Hour
)

const MaxHeaderBytes = 16 << 10

var (
	ErrConfig = errors.New("configuration is invalid")
	ErrStart  = errors.New("service could not start")
)

var DefaultOrigins = []string{"http://127.0.0.1:5173", "http://localhost:5173", "http://127.0.0.1:4173", "http://localhost:4173"}

type Config struct {
	Origins  []string
	Data     string
	Seed     string
	Clock    func() time.Time
	Log      *slog.Logger
	Kinds    []string
	Fields   map[string]gate.FieldClass
	Identity identity.Limits
	Commands commands.Limits
}

type Server struct {
	cfg      Config
	clock    func() time.Time
	log      *slog.Logger
	identity *identity.Registry
	gate     *gate.Gate
	admin    *gate.Administration
	commands *commands.Registry
	policy   *authpolicy.Registry
	domain   sync.RWMutex
	counter  uint64
	mutating map[string]bool
	sessions *sessions
	origins  map[string]bool
	hosts    map[string]bool
	hostMu   sync.RWMutex
	seed     []step
}

func IdentityLimits() identity.Limits {
	return identity.Limits{BootstrapValidity: time.Hour, InvitationValidity: 7 * 24 * time.Hour, CredentialValidity: 30 * 24 * time.Hour,
		SessionIdle: sessionIdle, SessionAbsolute: sessionAbsolute, Reauthentication: 15 * time.Minute, Attempts: 5, PendingInvitations: 1000,
		SessionsPerUser: 10, CredentialsPerServiceUser: 10, PageSize: 100}
}

func CommandLimits() commands.Limits {
	return commands.Limits{Fields: 64, Value: 4096, Rationale: 2000, Operations: 1200000, Journal: 1200000, Workspaces: 2, Users: 20,
		WorkspaceJournal: 600000, UserJournal: 30000, WorkspaceOperations: 600000, UserOperations: 30000, Replay: time.Hour, Retention: 2 * time.Hour,
		Burst: 60, Refill: 250 * time.Millisecond}
}

func Catalog(kinds []string, fields map[string]gate.FieldClass) gate.Catalog {
	c := gate.Catalog{Kinds: append([]string{"installation"}, kinds...), Fields: map[string]gate.FieldClass{}}
	for name, class := range fields {
		c.Fields[name] = class
	}
	return c
}

func validOrigin(origin string) bool {
	host, ok := cutScheme(origin)
	if !ok {
		return false
	}
	h, port, err := net.SplitHostPort(host)
	if err != nil || h == "" {
		return false
	}
	_, err = strconv.ParseUint(port, 10, 16)
	return err == nil
}

func cutScheme(origin string) (string, bool) {
	for _, scheme := range []string{"http://", "https://"} {
		if rest, ok := strings.CutPrefix(origin, scheme); ok {
			return rest, true
		}
	}
	return "", false
}

func New(cfg Config) (*Server, error) {
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	if len(cfg.Origins) == 0 {
		cfg.Origins = DefaultOrigins
	}
	if len(cfg.Origins) > maxOrigins || cfg.Data == "" || (cfg.Seed != SeedSample && cfg.Seed != SeedNone) {
		return nil, ErrConfig
	}
	s := &Server{cfg: cfg, log: cfg.Log, mutating: map[string]bool{}, sessions: newSessions(), origins: map[string]bool{}, hosts: map[string]bool{}}
	s.clock = func() time.Time { return cfg.Clock().UTC() }
	for _, origin := range cfg.Origins {
		if !validOrigin(origin) {
			return nil, ErrConfig
		}
		s.origins[origin] = true
	}
	if cfg.Identity == (identity.Limits{}) {
		cfg.Identity = IdentityLimits()
	}
	if cfg.Commands == (commands.Limits{}) {
		cfg.Commands = CommandLimits()
	}
	steps := []func() error{
		func() (err error) { s.identity, err = identity.New(Installation, cfg.Identity, s.clock); return },
		func() (err error) {
			s.gate, s.admin, err = gate.New(s.identity, Catalog(cfg.Kinds, cfg.Fields))
			return
		},
		func() (err error) {
			_, err = s.admin.PutClient(gate.Client{ID: ConsoleClient, State: gate.Active}, 0)
			return
		},
		func() (err error) { s.commands, err = commands.New(s.gate, cfg.Commands); return },
		func() (err error) { s.policy, err = authpolicy.NewRegistry(s.clock); return },
	}
	for _, build := range steps {
		if build() != nil {
			return nil, ErrStart
		}
	}
	return s, nil
}

func Loopback(listen string) bool {
	host, _, err := net.SplitHostPort(listen)
	if err != nil {
		return false
	}
	if host == "localhost" {
		return true
	}
	addr, err := netip.ParseAddr(host)
	return err == nil && addr.IsLoopback() && (addr == netip.MustParseAddr("127.0.0.1") || addr == netip.IPv6Loopback())
}

func (s *Server) Bind(addr net.Addr) error {
	tcp, ok := addr.(*net.TCPAddr)
	if !ok || !tcp.IP.IsLoopback() || tcp.Port == 0 {
		return ErrConfig
	}
	port := strconv.Itoa(tcp.Port)
	hosts := map[string]bool{"localhost:" + port: true}
	if tcp.IP.To4() != nil {
		hosts["127.0.0.1:"+port] = true
	} else {
		hosts["[::1]:"+port] = true
	}
	s.hostMu.Lock()
	s.hosts = hosts
	s.hostMu.Unlock()
	return nil
}

func (s *Server) allowedHost(host string) bool {
	s.hostMu.RLock()
	defer s.hostMu.RUnlock()
	return s.hosts[host]
}

func (s *Server) Identity() *identity.Registry { return s.identity }

func (s *Server) Gate() *gate.Gate { return s.gate }

func (s *Server) Administration() *gate.Administration { return s.admin }

func (s *Server) Commands() *commands.Registry { return s.commands }

func (s *Server) Policy() *authpolicy.Registry { return s.policy }

func (s *Server) Register(d commands.Definition) error {
	if err := s.commands.Register(d); err != nil {
		return err
	}
	s.domain.Lock()
	s.mutating[d.ID] = d.Mutating
	s.domain.Unlock()
	return nil
}

func (s *Server) Revision() string { return "r" + strconv.FormatUint(s.counter, 10) }

func (s *Server) Next() string { return "r" + strconv.FormatUint(s.counter+1, 10) }

func (s *Server) Change(name string, step func() (bool, error)) error {
	s.domain.Lock()
	defer s.domain.Unlock()
	changed, err := step()
	if changed {
		s.counter++
	}
	if err != nil {
		s.log.Error("step failed", "step", name)
		return err
	}
	return nil
}

func (s *Server) Read(read func(revision string)) {
	s.domain.RLock()
	defer s.domain.RUnlock()
	read(s.Revision())
}
