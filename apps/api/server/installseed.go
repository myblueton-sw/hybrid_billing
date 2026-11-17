package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/installer"
)

const (
	installSite       = "sample-site"
	installProbe      = "sample-probe"
	installKey        = "sample-key-1"
	installKeyLife    = 180 * 24 * time.Hour
	installKept       = 200
	installShown      = 50
	installSeedReason = "Sample data"
)

var installPlatform = installer.Platform{OS: "linux", Arch: "amd64"}

type installEntry struct {
	view      installer.ProposalView
	created   time.Time
	rationale string
	closedBy  string
	closing   string
}

type installReport struct {
	destination string
	result      installer.Result
	guidance    installer.Guidance
}

type installStore struct {
	site         string
	observed     time.Time
	releases     []string
	entries      []*installEntry
	destinations []installer.Destination
	reports      []installReport
}

func newInstallStore() *installStore { return &installStore{} }

func (c *installStore) entry(number uint64) *installEntry {
	for _, e := range c.entries {
		if e.view.Number == number {
			return e
		}
	}
	return nil
}

func (c *installStore) keep(e *installEntry) {
	c.entries = append(c.entries, e)
	if len(c.entries) <= installKept {
		return
	}
	for index, old := range c.entries {
		if old.view.State != installer.ProposalPending {
			c.entries = append(c.entries[:index], c.entries[index+1:]...)
			return
		}
	}
}

func InstallerLimits() installer.Limits {
	return installer.Limits{Keys: 4, KeyLife: 400 * 24 * time.Hour, Releases: 32, Sites: 4, Destinations: 16, Reports: 200, Runs: 16,
		ObservationLife: 24 * time.Hour, BootstrapLife: time.Hour, Proposals: 64}
}

func (s *Server) installCall(name string, call func() error) error {
	if s.faulty("installer-" + name) {
		return installer.ErrInvalid
	}
	return call()
}

func (s *Server) installPropose(c installer.Change, by installer.Actor, rationale string) (*installEntry, error) {
	var view installer.ProposalView
	if err := s.installCall("propose", func() (err error) { view, err = s.installer.Propose(c, by); return }); err != nil {
		return nil, err
	}
	e := s.install.entry(view.Number)
	if e == nil {
		e = &installEntry{view: view, created: s.clock(), rationale: rationale}
		s.install.keep(e)
	}
	return e, nil
}

func (s *Server) installApprove(e *installEntry, by installer.Actor, rationale string) error {
	var view installer.ProposalView
	if err := s.installCall("approve", func() (err error) { view, err = s.installer.Approve(e.view.Number, by); return }); err != nil {
		return err
	}
	if e.view.State == installer.ProposalPending {
		e.closedBy, e.closing = by.ID, rationale
		switch c := view.Change; c.Kind {
		case installer.ChangeRelease:
			s.install.releases = append(s.install.releases, c.Release.Version)
		case installer.ChangeSite:
			s.install.site = c.Site.ID
		case installer.ChangeDestination:
			s.install.destinations = append(s.install.destinations, c.Destination)
		}
	}
	e.view = view
	return nil
}

func (s *Server) installWithdraw(e *installEntry, by installer.Actor, rationale string) error {
	var view installer.ProposalView
	if err := s.installCall("withdraw", func() (err error) { view, err = s.installer.Withdraw(e.view.Number, by); return }); err != nil {
		return err
	}
	e.view, e.closedBy, e.closing = view, by.ID, rationale
	return nil
}

func sampleDigest(name, version string) string {
	sum := sha256.Sum256([]byte("sample-" + name + "-" + version))
	return hex.EncodeToString(sum[:])
}

func sampleRelease(version string, from ...string) installer.Release {
	return installer.Release{Version: version, Key: installKey, UpgradeFrom: from,
		Platforms: []installer.Platform{{OS: "darwin", Arch: "arm64"}, installPlatform, {OS: "linux", Arch: "arm64"}, {OS: "windows", Arch: "amd64"}},
		Artifacts: []installer.Artifact{{Name: "api", Kind: installer.ArtifactContainer, Digest: sampleDigest("api", version)},
			{Name: "worker", Kind: installer.ArtifactArchive, Digest: sampleDigest("worker", version)}},
		Services: []installer.Service{
			{Name: "api", Artifact: "api", Ports: []installer.Port{{Number: 8443, Protocol: installer.ProtocolTCP}}, Volumes: []string{"config", "ledger"}},
			{Name: "worker", Artifact: "worker", Ports: []installer.Port{{Number: 9000, Protocol: installer.ProtocolTCP}}, Volumes: []string{"ledger"}},
		},
		Volumes: []installer.Volume{{Name: "config"}, {Name: "ledger", Data: true}}}
}

func (s *Server) seedInstallation(now time.Time) error {
	public, private, _ := ed25519.GenerateKey(rand.Reader)
	first := installer.Actor{ID: SampleAdmin, Kind: installer.KindHuman}
	second := installer.Actor{ID: SampleAdmin2, Kind: installer.KindHuman}
	probe := installer.Actor{ID: installProbe, Kind: installer.KindSystem}
	approved := func(c installer.Change) func() error {
		return func() error {
			e, err := s.installPropose(c, first, installSeedReason)
			if err != nil {
				return err
			}
			return s.installApprove(e, second, installSeedReason)
		}
	}
	release := func(rel installer.Release) func() error {
		return approved(installer.Change{Kind: installer.ChangeRelease, Release: rel, Signature: ed25519.Sign(private, rel.SigningMessage())})
	}
	erp := installer.Destination{Site: installSite, ID: "sample-erp-endpoint", Host: "erp.example.invalid", Port: 443, Ranges: []string{"192.0.2.0/24"}}
	export := installer.Destination{Site: installSite, ID: "sample-billing-export", Host: "export.example.invalid", Port: 443, Ranges: []string{"198.51.100.0/24"}}
	return run(
		approved(installer.Change{Kind: installer.ChangeKey, Key: installer.ReleaseKey{Version: installKey, Public: public, NotBefore: now,
			NotAfter: now.Add(installKeyLife), Proof: ed25519.Sign(private, installer.KeyChallenge(installKey))}}),
		release(sampleRelease("1.0.0")),
		release(sampleRelease("1.1.0", "1.0.0")),
		approved(installer.Change{Kind: installer.ChangeSite, Site: installer.Site{ID: installSite, Tenant: s.Workspace(), Platform: installPlatform, Probe: installProbe}}),
		func() error {
			return s.installCall("observe", func() error {
				s.install.observed = s.clock()
				_, err := s.installer.Observe(installSite, installer.Observation{Platform: installPlatform, Complete: true}, probe)
				return err
			})
		},
		approved(installer.Change{Kind: installer.ChangeDestination, Destination: erp}),
		func() error {
			result := installer.Result{Kind: installer.DiagnoseTLS, Failure: "unknown_authority", Address: "192.0.2.10"}
			return s.installCall("report", func() error {
				guidance, err := s.installer.Report(installSite, erp.ID, result, probe)
				s.install.reports = append(s.install.reports, installReport{destination: erp.ID, result: result, guidance: guidance})
				return err
			})
		},
		func() error {
			var v installer.PreviewView
			if err := s.installCall("preview", func() (err error) { v, err = s.installer.Preview(installSite, "1.0.0"); return }); err != nil {
				return err
			}
			_, err := s.installPropose(installer.Change{Kind: installer.ChangePlan, Acceptance: installer.Acceptance{Site: installSite, Version: "1.0.0", Hash: v.Hash,
				Admin: SampleAdmin2}}, second, installSeedReason)
			return err
		},
		func() error {
			_, err := s.installPropose(installer.Change{Kind: installer.ChangeDestination, Destination: export}, first, installSeedReason)
			return err
		},
	)
}
