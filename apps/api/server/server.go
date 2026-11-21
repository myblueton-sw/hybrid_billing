package server

import (
	"crypto/rand"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/contracts"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/documentauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/events"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/installer"
	"github.com/myblueton-sw/hybrid_billing/modules/jobs"
	"github.com/myblueton-sw/hybrid_billing/modules/organizations"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
	"github.com/myblueton-sw/hybrid_billing/modules/relationships"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
	"github.com/myblueton-sw/hybrid_billing/modules/resultauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
	"github.com/myblueton-sw/hybrid_billing/modules/trustchange"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
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
	Origins   []string
	Data      string
	Seed      string
	Clock     func() time.Time
	Elapsed   func() time.Duration
	Log       *slog.Logger
	Kinds     []string
	Fields    map[string]gate.FieldClass
	Identity  identity.Limits
	Commands  commands.Limits
	Passwords passwords.Params
	Readers   map[string]string
	Access    AccessSettings
	Org       OrgSettings
	Plugins   PluginSettings
}

type Server struct {
	financialCharges           map[string]financialChargeBinding
	financialPolicies          map[string]financialPolicyBinding
	financialAuthorities       map[string]financialAuthorityBinding
	documentAuthority          *serverGrantAuthority
	resultAuthority            *serverGrantAuthority
	financialPermission        *serverGrantAuthority
	financialRequestPermission *serverGrantAuthority

	approvals        *approvals.Registry
	approvalIndex    []approvalIndex
	approvalKey      []byte
	inputAuthority   *serverInputAuthority
	pricingAuthority *serverGrantAuthority
	inputReviews     *candidateInputReviews
	cfg              Config
	clock            func() time.Time
	time             *serverClock
	log              *slog.Logger
	identity         *identity.Registry
	gate             *gate.Gate
	admin            *gate.Administration
	commands         *commands.Registry
	policy           *authpolicy.Registry
	domain           sync.RWMutex
	counter          uint64
	mutating         map[string]bool
	sessions         *sessions
	origins          map[string]bool
	hosts            map[string]bool
	hostMu           sync.RWMutex
	seed             []step
	passwords        *passwords.Registry
	orgs             *organizations.Registry
	chargeProducer   *normalizedCharges
	documentERP      *financialERP
	rels             *relationships.Registry
	plans            *planStore
	drafts           *draftStore
	relLimits        relationships.Limits
	lifecycles       *contracts.Registry
	contracts        *contractStore
	lifecycleLimits  contracts.Limits
	installer        *installer.Registry
	install          *installStore
	plugins          *plugins.Registry
	webhooks         *webhooks.Registry
	events           *events.Registry
	integ            *integrationStore
	signer           *seedSigner
	recovery         *recovery.Registry
	rec              *recoveryStore
	retention        *retention.Registry
	ret              *retentionStore
	trace            *releasetrace.Registry
	perf             *perfgate.Registry
	ev               *evidenceStore
	changes          *accesschange.Registry
	ledger           *decisionledger.Ledger
	history          *accesshistory.History
	holdPlaced       sync.Map
	workers          sync.WaitGroup
	startedAt        time.Time
	limit            *limiter
	slots            chan struct{}
	boot             activation
	bootWorkspace    string
	bootUser         string
	pause            func()
	readerSecret     string
	dir              *directory
	failures         map[identity.Verifier]int
	caller           identity.Verifier
	fault            func(string) bool
	trust            *trustchange.Registry
	trustOrigin      trustOrigin
	hold             *trustHold
	progress         map[string]trustProgress
	random           io.Reader
	memberLedger     map[string]string
	roleEnds         uint64
	bulk             *bulkrevoke.Registry
	jobs             *jobs.Registry
	answer           gate.Answer
	stranded         []uint64
	clockChoice      clockChoice
	replayNotes      map[string]string
	stop             chan struct{}
	wake             chan struct{}
	stopOnce         sync.Once
	manual           bool
	awaiting         map[string]awaitingReplay
	selfEnds         map[string][]*selfEnd
	selfSeq          uint64
	retries          map[identity.Verifier]*selfRetry
	selfEndHook      func()
	accessHook       func(identity.Verifier)
}

func IdentityLimits() identity.Limits {
	return identity.Limits{BootstrapValidity: time.Hour, InvitationValidity: 7 * 24 * time.Hour, CredentialValidity: 30 * 24 * time.Hour,
		SessionIdle: sessionIdle, SessionAbsolute: sessionAbsolute, Reauthentication: 15 * time.Minute, Attempts: 5, PendingInvitations: 1000,
		SessionsPerUser: 10, CredentialsPerServiceUser: 10, PageSize: 100}
}

func CommandLimits() commands.Limits {
	return commands.Limits{Fields: 64, Value: 4096, Rationale: 2000, Operations: 1200000, Journal: 1200000, Workspaces: 2, Users: 20,
		WorkspaceJournal: 600000, UserJournal: 30000, WorkspaceOperations: 600000, UserOperations: 30000, Replay: time.Hour, Retention: 8 * time.Hour,
		Burst: 60, Refill: time.Second}
}

func RelationshipLimits() relationships.Limits {
	return relationships.Limits{Depth: relationships.MaxDepth, EntriesPerTimeline: 64, TimelinesPerChunk: 1000, ChunksPerPlan: 100, TimelinesPerPlan: 10000,
		EntriesPerPlan: 50000, OpenPlans: 16, Violations: 100, PageSize: 100}
}

func Catalog(kinds []string, fields map[string]gate.FieldClass) gate.Catalog {
	c := gate.Catalog{AuthorClient: ConsoleClient, Kinds: append([]string{gate.InstallationKind, ReaderKind, candidateKind, approvalKind, inputauthority.ResourceKind, pricingauthority.ResourceKind, financialpermission.ResourceKind, documentauthority.ResourceKind, resultauthority.ResourceKind}, kinds...), Fields: map[string]gate.FieldClass{ReaderField: gate.General, candidateField: gate.Cost, approvalField: gate.General, inputauthority.EvidenceField: gate.Cost, pricingauthority.EvidenceField: gate.Cost}}
	for name, class := range fields {
		c.Fields[name] = class
	}
	c.Fields[pricingauthority.EvidenceField] = gate.Cost
	c.Fields[documentauthority.EvidenceField] = gate.Cost
	c.Fields[resultauthority.EvidenceField] = gate.Cost
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
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.DiscardHandler)
	}
	if len(cfg.Origins) == 0 {
		cfg.Origins = DefaultOrigins
	}
	if len(cfg.Origins) > maxOrigins || cfg.Data == "" || (cfg.Seed != SeedSample && cfg.Seed != SeedNone) {
		return nil, ErrConfig
	}
	s := &Server{cfg: cfg, log: cfg.Log, mutating: map[string]bool{}, sessions: newSessions(), origins: map[string]bool{}, hosts: map[string]bool{},
		limit: &limiter{}, slots: make(chan struct{}, derivations), dir: newDirectory(), failures: map[identity.Verifier]int{}, selfEnds: map[string][]*selfEnd{}, retries: map[identity.Verifier]*selfRetry{}, plans: newPlanStore(), drafts: newDraftStore(), relLimits: RelationshipLimits(),
		contracts: newContractStore(), install: newInstallStore(), integ: newIntegrationStore(), signer: newSeedSigner(), rec: newRecoveryStore(),
		ret: newRetentionStore(), ev: newEvidenceStore(), progress: map[string]trustProgress{}, random: rand.Reader, memberLedger: map[string]string{},
		replayNotes: map[string]string{}, awaiting: map[string]awaitingReplay{}, stop: make(chan struct{}), wake: make(chan struct{}, 1)}
	s.time = newServerClock(cfg.Clock, cfg.Elapsed, cfg.Log)
	s.clock = s.time.now
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
	if cfg.Readers == nil {
		cfg.Readers = DefaultReaders()
	}
	if !validReaders(cfg.Readers) {
		return nil, ErrConfig
	}
	if cfg.Passwords == (passwords.Params{}) {
		cfg.Passwords = passwords.DefaultParams()
	}
	cfg.Access = cfg.Access.withDefaults()
	if err := cfg.Access.validate(); err != nil {
		return nil, err
	}
	cfg.Org = cfg.Org.withDefaults()
	if err := cfg.Org.validate(); err != nil {
		return nil, err
	}
	licenses, err := cfg.Plugins.licenses()
	if err != nil {
		return nil, err
	}
	if err := sampleLicensed(cfg.Seed, licenses); err != nil {
		return nil, err
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
		func() (err error) { s.passwords, err = passwords.New(s.clock, cfg.Passwords); return },
		func() (err error) { s.orgs, err = organizations.NewRegistry(Installation); return },
		func() (err error) { s.rels, err = relationships.NewRegistry(s.relLimits, s.clock); return },
		func() (err error) {
			s.lifecycleLimits = ContractLimits()
			s.lifecycles, err = contracts.NewRegistry(s.lifecycleLimits, s.clock)
			return
		},
		func() (err error) { s.chargeProducer, err = s.newChargeProducer(); return },
		func() (err error) {
			s.installer, err = installer.NewRegistry(installer.Config{Random: rand.Reader}, InstallerLimits(), s.clock)
			return
		},
		func() (err error) { s.events, err = events.NewRegistry(EventLimits(), s.clock); return },
		func() (err error) {
			s.plugins, err = plugins.NewRegistry(plugins.Config{Core: pluginCore, Destinations: integrationDestinations{s: s}, Random: rand.Reader, Licenses: licenses}, PluginLimits(), s.clock)
			return
		},
		func() (err error) {
			s.webhooks, err = webhooks.NewRegistry(webhooks.Config{Outbox: newOutboxReader(s), Destinations: integrationDestinations{s: s}, Permits: integrationPermits{s: s},
				Signer: s.signer, Redirects: noRedirects{}}, WebhookLimits(), s.clock)
			return
		},
		func() (err error) { s.recovery, err = recovery.NewRegistry(s.clock); return },
		func() (err error) { s.retention, err = retention.NewRegistry(s.clock); return },
		func() (err error) { s.trace, err = releasetrace.NewRegistry(s.clock); return },
		func() (err error) { s.perf, err = perfgate.NewRegistry(s.clock); return },
		func() (err error) { s.changes, err = accesschange.NewRegistry(s.clock, cfg.Access.limits()); return },
		func() (err error) { s.bulk, err = bulkrevoke.New(cfg.Access.bulkLimits(), s.clock); return },
		func() (err error) {
			s.jobs, err = jobs.NewRegistry(cleanupLimits(cfg.Access.BulkMaxMatching), s.clock)
			return
		},
		s.initApprovals,

		func() (err error) {
			if cfg.Seed == SeedSample {
				s.inputAuthority, err = newServerInputAuthority(s, inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 1000})
				if err == nil {
					s.pricingAuthority, err = newServerGrantAuthority(s, inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 1000}, pricingGrantAuthority)
				}
				if err == nil {
					s.financialPermission, err = newServerGrantAuthority(s, inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 1000}, financialPermissionDomain)
				}
				if err == nil {
					s.financialRequestPermission, err = newServerGrantAuthority(s, inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 1000}, financialRequestDomain)
				}
				if err == nil {
					s.documentAuthority, err = newServerGrantAuthority(s, inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 1000}, documentGrantAuthority)
				}
				if err == nil {
					s.resultAuthority, err = newServerGrantAuthority(s, inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 1000}, resultGrantAuthority)
				}
				if err == nil {
					s.inputReviews, err = newCandidateInputReviews(s.chargeProducer, serverInputReviewAuthority{s: s}, s.clock)
				}
				if err == nil {
					_, err = s.chargeProducer.newPricingPolicies(serverPricingPolicyAuthority{s: s}, s.clock)
				}
			}
			return
		},
		s.registerUsers,
	}
	for _, build := range steps {
		if build() != nil {
			return nil, ErrStart
		}
	}
	s.cfg = cfg
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

func (s *Server) Passwords() *passwords.Registry { return s.passwords }

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
