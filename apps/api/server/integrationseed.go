package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/events"
	"github.com/myblueton-sw/hybrid_billing/modules/plugins"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

const (
	samplePublisher = "sample-publisher"
	samplePluginKey = "sample-plugin-key-1"
	samplePlugin    = "sample-ledger-export"
	sampleRuntime   = "sample-plugin-runtime"
	sampleNamespace = "sample-ledger-export-data"
	sampleSource    = "urn:sample-billing"
	sampleKeyLife   = 180 * 24 * time.Hour
	typeInvoice     = "invoice.issued"
	typePayment     = "payment.received"
	schemaVersion   = "1"
)

var errSeedLease = errors.New("the sample dispatch leased no delivery")

type pluginEntry struct {
	view      plugins.ProposalView
	created   time.Time
	rationale string
	closedBy  string
	closing   string
}

type pluginRecord struct {
	id       string
	versions []string
}

type actionRecord struct {
	action    string
	actor     string
	rationale string
}

type subscriptionEntry struct {
	id        string
	created   time.Time
	proposer  string
	rationale string
	history   []actionRecord
	letters   []string
}

type letterRequest struct {
	by        string
	rationale string
}

type typeRecord struct {
	name     string
	version  string
	fields   []string
	proposer string
	approver string
}

type integrationStore struct {
	publishers    []string
	plugins       []*pluginRecord
	workspace     string
	entries       []*pluginEntry
	subscriptions []*subscriptionEntry
	redeliver     map[string]letterRequest
	dismiss       map[string]letterRequest
	types         []typeRecord
	counter       int
}

func newIntegrationStore() *integrationStore {
	return &integrationStore{redeliver: map[string]letterRequest{}, dismiss: map[string]letterRequest{}}
}

func (c *integrationStore) entry(number uint64) *pluginEntry {
	for _, e := range c.entries {
		if e.view.Number == number {
			return e
		}
	}
	return nil
}

func (c *integrationStore) keep(e *pluginEntry) {
	c.entries = append(c.entries, e)
	if len(c.entries) <= installKept {
		return
	}
	for index, old := range c.entries {
		if old.view.State != plugins.ProposalPending {
			c.entries = append(c.entries[:index], c.entries[index+1:]...)
			return
		}
	}
}

func (c *integrationStore) plugin(id string) *pluginRecord {
	for _, p := range c.plugins {
		if p.id == id {
			return p
		}
	}
	return nil
}

func (c *integrationStore) subscription(id string) *subscriptionEntry {
	for _, e := range c.subscriptions {
		if e.id == id {
			return e
		}
	}
	return nil
}

func (c *integrationStore) letter(id string) *subscriptionEntry {
	for _, e := range c.subscriptions {
		for _, letter := range e.letters {
			if letter == id {
				return e
			}
		}
	}
	return nil
}

func (c *integrationStore) dropLetter(e *subscriptionEntry, id string) {
	for index, letter := range e.letters {
		if letter == id {
			e.letters = append(e.letters[:index:index], e.letters[index+1:]...)
			break
		}
	}
	delete(c.redeliver, id)
	delete(c.dismiss, id)
}

func (c *integrationStore) approvedType(name string) bool {
	for _, t := range c.types {
		if t.name == name {
			return true
		}
	}
	return false
}

func PluginLimits() plugins.Limits {
	return plugins.Limits{Publishers: 8, Keys: 4, KeyLife: 400 * 24 * time.Hour, Plugins: 16, Versions: 64, Workspaces: 4, Installations: 16, Tokens: 1,
		Fingerprints: 16, TokenLife: time.Hour, Endpoints: 4, Proposals: 64}
}

func WebhookLimits() webhooks.Limits {
	return webhooks.Limits{Workspaces: 2, Subscriptions: 16, Types: 8, Lease: time.Minute, Backoff: time.Minute, MaxBackoff: time.Hour, Attempts: 5,
		MaxAge: 24 * time.Hour, Dead: 100, Batch: 10, Work: 4, Keys: 4, KeyLife: 400 * 24 * time.Hour}
}

func EventLimits() events.Limits {
	return events.Limits{Workspaces: 2, Types: 16, Events: 1000, Retention: 30 * 24 * time.Hour, Page: 100, Sources: 1, Seen: 1, InboxRetention: time.Hour,
		Revisions: 1, Reservation: time.Minute}
}

func (s *Server) moduleCall(name string, fault error, call func() error) error {
	if s.faulty(name) {
		return fault
	}
	return call()
}

func (s *Server) pluginCall(name string, call func() error) error {
	return s.moduleCall("plugins-"+name, plugins.ErrInvalid, call)
}

func (s *Server) webhookCall(name string, call func() error) error {
	return s.moduleCall("webhooks-"+name, webhooks.ErrInvalid, call)
}

func (s *Server) eventCall(name string, call func() error) error {
	return s.moduleCall("events-"+name, events.ErrInvalid, call)
}

func (s *Server) pluginPropose(c plugins.Change, by plugins.Actor, rationale string) (*pluginEntry, error) {
	var view plugins.ProposalView
	if err := s.pluginCall("propose", func() (err error) { view, err = s.plugins.Propose(c, by); return }); err != nil {
		return nil, err
	}
	e := s.integ.entry(view.Number)
	if e == nil {
		e = &pluginEntry{view: view, created: s.clock(), rationale: rationale}
		s.integ.keep(e)
	}
	return e, nil
}

func (s *Server) pluginApprove(e *pluginEntry, by plugins.Actor, rationale string) error {
	var view plugins.ProposalView
	if err := s.pluginCall("approve", func() (err error) { view, err = s.plugins.Approve(e.view.Number, by); return }); err != nil {
		return err
	}
	if e.view.State == plugins.ProposalPending {
		e.closedBy, e.closing = by.ID, rationale
		switch c := view.Change; c.Kind {
		case plugins.ChangePublisher:
			s.integ.publishers = append(s.integ.publishers, c.Publisher.ID)
		case plugins.ChangeVersion:
			p := s.integ.plugin(c.Manifest.Plugin)
			if p == nil {
				p = &pluginRecord{id: c.Manifest.Plugin}
				s.integ.plugins = append(s.integ.plugins, p)
			}
			p.versions = append(p.versions, c.Manifest.Version)
		case plugins.ChangeWorkspace:
			s.integ.workspace = c.Workspace.ID
		}
	}
	e.view = view
	return nil
}

func (s *Server) pluginWithdraw(e *pluginEntry, by plugins.Actor, rationale string) error {
	var view plugins.ProposalView
	if err := s.pluginCall("withdraw", func() (err error) { view, err = s.plugins.Withdraw(e.view.Number, by); return }); err != nil {
		return err
	}
	e.view, e.closedBy, e.closing = view, by.ID, rationale
	return nil
}

func (s *Server) subscriptionPropose(sub webhooks.Subscription, by webhooks.Actor, rationale string) (*subscriptionEntry, webhooks.SubscriptionView, error) {
	var view webhooks.SubscriptionView
	err := s.webhookCall("propose", func() (err error) { view, err = s.webhooks.ProposeSubscription(s.bootWorkspace, sub, by); return })
	if err != nil {
		return nil, view, err
	}
	e := s.integ.subscription(sub.ID)
	if e == nil {
		e = &subscriptionEntry{id: sub.ID, created: s.clock(), proposer: by.ID, rationale: rationale}
		s.integ.subscriptions = append(s.integ.subscriptions, e)
	}
	return e, view, nil
}

func (s *Server) subscriptionStep(e *subscriptionEntry, action string, by webhooks.Actor, rationale string,
	call func(string, string, webhooks.Actor) (webhooks.SubscriptionView, error)) (webhooks.SubscriptionView, error) {
	var view webhooks.SubscriptionView
	if err := s.webhookCall(action, func() (err error) { view, err = call(s.bootWorkspace, e.id, by); return }); err != nil {
		return view, err
	}
	e.history = append(e.history, actionRecord{action: action, actor: by.ID, rationale: rationale})
	return view, nil
}

func (s *Server) approveType(name string, fields []string, first, second events.Actor) error {
	t := events.Type{Name: name, SchemaVersion: schemaVersion, Fields: fields}
	return run(
		func() error {
			return s.eventCall("propose", func() (err error) { _, err = s.events.ProposeType(s.bootWorkspace, t, first); return })
		},
		func() error {
			return s.eventCall("approve", func() (err error) {
				_, err = s.events.ApproveType(s.bootWorkspace, name, schemaVersion, second)
				return
			})
		},
		func() error {
			s.integ.types = append(s.integ.types, typeRecord{name: name, version: schemaVersion, fields: fields, proposer: first.ID, approver: second.ID})
			return nil
		},
	)
}

func sampleHash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func sampleManifest(version string) plugins.Manifest {
	return plugins.Manifest{Plugin: samplePlugin, Version: version, Publisher: samplePublisher, Key: samplePluginKey, Digest: sampleHash(samplePlugin + "-" + version),
		Core: plugins.CoreRange{Min: "1.0.0", Max: "1.9.9"}, Capabilities: []plugins.Capability{{Kind: plugins.CapabilityReadEvents, Target: typeInvoice},
			{Kind: plugins.CapabilityServeScreen, Target: "overview"}}, Egress: []string{sampleEgress}, Endpoints: []string{"api"},
		Limits:  plugins.RuntimeLimits{Timeout: 30 * time.Second, MemoryMiB: 128, Concurrency: 2},
		Storage: plugins.Storage{Namespace: sampleNamespace, Owner: samplePublisher, RetentionDays: 30, Deletion: plugins.RetainForEvidence, Migration: plugins.MigrationAdditive},
		Screens: []plugins.Screen{{ID: "overview", Component: plugins.ComponentTable, Label: "Overview", Fields: []plugins.Field{
			{Name: "total", Label: "Total", Source: sampleNamespace}, {Name: "event", Label: "Event", Source: typeInvoice}}}}}
}

func (s *Server) seedPlugins(now time.Time) error {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	key := plugins.Change{Kind: plugins.ChangeKey, Key: plugins.PublisherKey{Publisher: samplePublisher, Version: samplePluginKey, Public: public, NotBefore: now,
		NotAfter: now.Add(sampleKeyLife), Proof: ed25519.Sign(private, plugins.KeyChallenge(samplePublisher, samplePluginKey))}}
	versions := []plugins.Change{}
	for _, version := range []string{"1.0.0", "1.1.0"} {
		m := sampleManifest(version)
		versions = append(versions, plugins.Change{Kind: plugins.ChangeVersion, Manifest: m, Signature: ed25519.Sign(private, m.SigningMessage())})
	}
	clear(private)
	first := plugins.Actor{ID: SampleAdmin, Kind: plugins.KindHuman}
	second := plugins.Actor{ID: SampleAdmin2, Kind: plugins.KindHuman}
	ws := s.Workspace()
	approved := func(c plugins.Change) func() error {
		return func() error {
			e, err := s.pluginPropose(c, first, installSeedReason)
			if err != nil {
				return err
			}
			return s.pluginApprove(e, second, installSeedReason)
		}
	}
	pending := func(c plugins.Change, by plugins.Actor) func() error {
		return func() error {
			_, err := s.pluginPropose(c, by, installSeedReason)
			return err
		}
	}
	at := func(kind, version string) plugins.Change {
		return plugins.Change{Kind: kind, Workspace: plugins.Workspace{ID: ws}, Plugin: samplePlugin, Version: version}
	}
	endpoint := at(plugins.ChangeEndpoint, "")
	endpoint.Endpoint = plugins.Endpoint{ID: "api", Fingerprint: sampleHash("sample-certificate-api"), Evidence: sampleHash("sample-evidence-api")}
	return run(
		approved(plugins.Change{Kind: plugins.ChangePublisher, Publisher: plugins.Publisher{ID: samplePublisher, Name: "Sample Publisher"}}),
		approved(key),
		approved(versions[0]),
		approved(versions[1]),
		approved(plugins.Change{Kind: plugins.ChangeWorkspace, Workspace: plugins.Workspace{ID: ws, Runtime: sampleRuntime}}),
		approved(at(plugins.ChangeInstall, "1.0.0")),
		approved(at(plugins.ChangeEnable, "")),
		pending(at(plugins.ChangeUpgrade, "1.1.0"), second),
		pending(endpoint, first),
	)
}

func (s *Server) seedIntegrations(now time.Time) error {
	ws := s.Workspace()
	first, second := events.Actor{ID: SampleAdmin, Kind: events.KindHuman}, events.Actor{ID: SampleAdmin2, Kind: events.KindHuman}
	billing := events.Actor{ID: "sample-billing", Kind: events.KindSystem}
	relay := webhooks.Actor{ID: "sample-relay", Kind: webhooks.KindSystem}
	ann, bob := webhooks.Actor{ID: SampleAdmin, Kind: webhooks.KindHuman}, webhooks.Actor{ID: SampleAdmin2, Kind: webhooks.KindHuman}
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	defer clear(private)
	defer s.signer.disarm()
	var feed *subscriptionEntry
	publish := func(n string) func() error {
		return func() error {
			e := events.Event{ID: "sample-invoice-" + n, Source: sampleSource, Type: typeInvoice, SchemaVersion: schemaVersion, Data: map[string]string{"invoice_id": "sample-invoice-" + n}}
			return s.eventCall("publish", func() (err error) { _, err = s.events.Publish(ws, e, billing); return })
		}
	}
	round := func(status string, last bool) func() error {
		return func() error {
			var leases []webhooks.Lease
			if err := s.webhookCall("dispatch", func() (err error) { leases, err = s.webhooks.Dispatch(ws, 1, relay); return }); err != nil {
				return err
			}
			if len(leases) != 1 {
				return errSeedLease
			}
			lease := leases[0]
			if err := s.webhookCall("confirm", func() (err error) { _, err = s.webhooks.Confirm(ws, lease.Delivery, lease.Token, relay); return }); err != nil {
				return err
			}
			if last {
				s.signer.disarm()
			}
			if err := s.webhookCall("report", func() (err error) {
				_, err = s.webhooks.Report(ws, lease.Delivery, lease.Token, webhooks.Outcome{Status: status}, relay)
				return
			}); err != nil {
				return err
			}
			if status == webhooks.StatusRejected {
				feed.letters = append(feed.letters, lease.Delivery)
			}
			return nil
		}
	}
	return run(
		func() error { return s.approveType(typeInvoice, []string{"invoice_id"}, first, second) },
		func() error { return s.approveType(typePayment, []string{"payment_id"}, first, second) },
		func() error { return s.seedPlugins(now) },
		func() (err error) {
			feed, _, err = s.subscriptionPropose(webhooks.Subscription{ID: sampleSubscription, Subscriber: sampleSubscriber, Destination: "sample-erp-webhook",
				Types: []string{typeInvoice}}, ann, installSeedReason)
			return err
		},
		func() error {
			_, err := s.subscriptionStep(feed, "approve", bob, installSeedReason, s.webhooks.ApproveSubscription)
			return err
		},
		func() error {
			s.signer.arm(ws, sampleSubscription, sampleWebhookKey, private)
			k := webhooks.Key{Version: sampleWebhookKey, Public: public, NotBefore: now, NotAfter: now.Add(sampleKeyLife)}
			return s.webhookCall("propose_key", func() (err error) { _, err = s.webhooks.ProposeKey(ws, sampleSubscription, k, ann); return })
		},
		func() error {
			return s.webhookCall("approve_key", func() (err error) {
				_, err = s.webhooks.ApproveKey(ws, sampleSubscription, sampleWebhookKey, bob)
				return
			})
		},
		publish("1"),
		publish("2"),
		publish("3"),
		round(webhooks.StatusDelivered, false),
		round(webhooks.StatusRejected, false),
		round(webhooks.StatusFailed, true),
		func() error {
			letter := feed.letters[0]
			if err := s.webhookCall("request_redeliver", func() error { return s.webhooks.RequestRedeliver(ws, sampleSubscription, letter, ann) }); err != nil {
				return err
			}
			s.integ.redeliver[letter] = letterRequest{by: ann.ID, rationale: installSeedReason}
			return nil
		},
		func() error {
			_, _, err := s.subscriptionPropose(webhooks.Subscription{ID: samplePayments, Subscriber: sampleSubscriber, Destination: "sample-payments-webhook",
				Types: []string{typePayment}}, bob, installSeedReason)
			return err
		},
	)
}
