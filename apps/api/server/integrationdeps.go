package server

import (
	"crypto/ed25519"
	"errors"
	"sync"

	"github.com/myblueton-sw/hybrid_billing/modules/events"
	"github.com/myblueton-sw/hybrid_billing/modules/webhooks"
)

const (
	pluginCore         = "1.0.0"
	outboxIdentity     = "integration-outbox"
	sampleSubscriber   = "sample-erp-connector"
	sampleSubscription = "sample-erp-invoices"
	samplePayments     = "sample-payments-feed"
	sampleWebhookKey   = "sample-webhook-key-1"
	sampleEgress       = "sample-erp"
	allowlistSource    = "sample_allowlist"
)

var (
	errStandIn  = errors.New("an integration stand-in is unavailable")
	errDisarmed = errors.New("the sample signer holds no key")
	errRedirect = errors.New("redirects are not connected")
)

type sampleDestination struct {
	id    string
	state string
}

var webhookDestinations = []sampleDestination{
	{"sample-erp-webhook", webhooks.DestinationActive},
	{"sample-legacy-webhook", webhooks.DestinationInactive},
	{"sample-payments-webhook", webhooks.DestinationActive},
}

func destinationState(id string) (string, bool) {
	for _, d := range webhookDestinations {
		if d.id == id {
			return d.state, true
		}
	}
	return webhooks.DestinationInactive, false
}

type integrationDestinations struct{ s *Server }

func (d integrationDestinations) State(workspace, destination string) (string, error) {
	if d.s.faulty("destinations") {
		return "", errStandIn
	}
	state, _ := destinationState(destination)
	if workspace != d.s.bootWorkspace {
		state = webhooks.DestinationInactive
	}
	return state, nil
}

func (d integrationDestinations) Active(workspace, destination string) (bool, error) {
	if d.s.faulty("destinations") {
		return false, errStandIn
	}
	return workspace == d.s.bootWorkspace && destination == sampleEgress, nil
}

type integrationPermits struct{ s *Server }

func (p integrationPermits) admits(workspace, subscriber string, types []string) (bool, error) {
	if p.s.faulty("permits") {
		return false, errStandIn
	}
	if workspace != p.s.bootWorkspace || subscriber != sampleSubscriber {
		return false, nil
	}
	for _, name := range types {
		if !p.s.integ.approvedType(name) {
			return false, nil
		}
	}
	return true, nil
}

func (p integrationPermits) Subscribe(workspace, subscriber string, types []string) (bool, error) {
	return p.admits(workspace, subscriber, types)
}

func (p integrationPermits) Allowed(workspace, subscriber string, event events.Stored) (bool, error) {
	return p.admits(workspace, subscriber, []string{event.Event.Type})
}

type outboxReader struct {
	s      *Server
	system events.Actor
}

func newOutboxReader(s *Server) outboxReader {
	return outboxReader{s: s, system: events.Actor{ID: outboxIdentity, Kind: events.KindSystem}}
}

func (o outboxReader) Read(workspace string, after uint64, limit int) (events.Page, error) {
	if o.s.faulty("events-read") {
		return events.Page{}, events.ErrInvalid
	}
	return o.s.events.Read(workspace, after, limit, o.system)
}

func (o outboxReader) Bounds(workspace string) (uint64, uint64, error) {
	if o.s.faulty("events-bounds") {
		return 0, 0, events.ErrInvalid
	}
	return o.s.events.Bounds(workspace, o.system)
}

type signerTriple struct {
	workspace    string
	subscription string
	version      string
}

type seedSigner struct {
	mu      sync.Mutex
	current map[signerTriple]string
	keys    map[signerTriple]ed25519.PrivateKey
}

func newSeedSigner() *seedSigner {
	return &seedSigner{current: map[signerTriple]string{}}
}

func (g *seedSigner) arm(workspace, subscription, version string, key ed25519.PrivateKey) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.current[signerTriple{workspace: workspace, subscription: subscription}] = version
	g.keys = map[signerTriple]ed25519.PrivateKey{{workspace, subscription, version}: key}
}

func (g *seedSigner) disarm() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, key := range g.keys {
		clear(key)
	}
	g.keys = nil
}

func (g *seedSigner) Current(workspace, subscription string) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	version, ok := g.current[signerTriple{workspace: workspace, subscription: subscription}]
	if !ok {
		return "", errDisarmed
	}
	return version, nil
}

func (g *seedSigner) Sign(workspace, subscription, version string, message []byte) ([]byte, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	key, ok := g.keys[signerTriple{workspace, subscription, version}]
	if !ok {
		return nil, errDisarmed
	}
	return ed25519.Sign(key, message), nil
}

type noRedirects struct{}

func (noRedirects) Check(string, string, string, int, string) (string, string, int, error) {
	return "", "", 0, errRedirect
}
