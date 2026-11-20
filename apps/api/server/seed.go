package server

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

const (
	ActivationFile  = "activation.txt"
	SampleWorkspace = "sample-workspace"
	EmptyWorkspace  = "workspace"
	SampleAdmin     = "sample-admin"
	SampleAdmin2    = "sample-admin-2"
	SampleAdmin3    = "sample-admin-3"
	AdminLogin      = "admin@example.invalid"
	Admin2Login     = "admin2@example.invalid"
	Admin3Login     = "admin3@example.invalid"
	SampleMember    = "sample-member"
	MemberLogin     = "member@example.invalid"
	EmptyAdmin      = "admin"
	PasswordLink    = "local-password"
)

type step struct {
	name string
	run  func() (bool, error)
}

func (s *Server) AddStep(name string, run func() (bool, error)) {
	s.seed = append(s.seed, step{name: name, run: run})
}

func (s *Server) Workspace() string {
	if s.cfg.Seed == SeedSample {
		return SampleWorkspace
	}
	return EmptyWorkspace
}

func AdminTriple() identity.Triple { return LoginTriple(AdminLogin) }

func checkData(dir string) error {
	info, err := os.Lstat(dir)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return os.MkdirAll(dir, 0o700)
	case err != nil:
		return err
	case info.Mode()&fs.ModeSymlink != 0 || !info.IsDir():
		return ErrConfig
	case runtime.GOOS != "windows" && (info.Mode().Perm()&0o077 != 0 || !owned(info)):
		return ErrConfig
	}
	return nil
}

func writeExclusive(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.Write(data)
	return errors.Join(werr, f.Close())
}

func (s *Server) Start() error {
	if err := checkData(s.cfg.Data); err != nil {
		return err
	}
	if err := s.setClockBase(); err != nil {
		return err
	}
	link := filepath.Join(s.cfg.Data, ActivationFile)
	if err := os.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	secret := random(tokenBytes)
	verifier := verifierOf(secret)
	now := s.clock()
	s.bootWorkspace, s.bootUser = s.Workspace(), EmptyAdmin
	steps := []step{{"installation-workspace", func() (bool, error) {
		return true, s.admin.SetInstallationWorkspace(s.Workspace())
	}}, {"organizations-workspace", func() (bool, error) {
		return true, s.orgWorkspace()
	}}, {"arm-bootstrap", func() (bool, error) {
		return true, s.identity.Installation().ArmBootstrap(verifier, identity.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}, s.Workspace())
	}}}
	var lines []string
	if s.cfg.Seed == SeedSample {
		s.bootUser = SampleAdmin
		steps = append(steps, step{"bootstrap-admin", func() (bool, error) {
			s.dir.add(SampleAdmin, AdminLogin, AdminLogin)
			return true, s.identity.Boundary().Bootstrap(verifier, SampleAdmin, AdminTriple(), AdminLogin)
		}}, step{"second-admin", func() (bool, error) { return true, s.seedAdmin(SampleAdmin2, Admin2Login, "sample-invite", now) }},
			step{"third-admin", func() (bool, error) { return true, s.seedAdmin(SampleAdmin3, Admin3Login, "sample-invite-3", now) }},
			step{"sample-member", func() (bool, error) { return true, s.seedMember(now) }},
			step{"reader-grants", func() (bool, error) { return true, s.seedReaderGrants(now) }},
			step{"organizations", func() (bool, error) { return true, s.seedOrganizations() }},
			step{"contracts", func() (bool, error) { return true, s.seedContracts() }},
			step{"charge-candidates", func() (bool, error) { return true, s.seedChargeCandidates(now) }},
			step{"installation", func() (bool, error) { return true, s.seedInstallation(now) }},
			step{"installation-grants", func() (bool, error) { return true, s.seedInstallationGrants(now) }},
			step{"trust-grants", func() (bool, error) { return true, s.seedTrustGrants(now) }},
			step{"bulk-grants", func() (bool, error) { return true, s.seedBulkGrants(now) }},
			step{"integration-grants", func() (bool, error) { return true, s.seedIntegrationGrants(now) }},
			step{"integrations", func() (bool, error) { return true, s.seedIntegrations(now) }},
			step{"recovery", func() (bool, error) { return true, s.seedRecovery(now) }},
			step{"retention", func() (bool, error) { return true, s.seedRetention(now) }},
			step{"approval-requests", func() (bool, error) { return true, s.seedApprovals(now) }},
			step{"release-evidence", func() (bool, error) { return true, s.seedEvidence(now) }},
			step{"password-links", func() (bool, error) {
				var err error
				lines, err = s.seedLinks(now)
				return true, err
			}})
		steps = append(steps, s.seed...)
	}
	s.startedAt = now
	steps = append(steps, step{"decision-ledger", s.openLedger}, step{"access-history", s.openHistory}, step{"trust-replay", s.replayTrust}, step{"role-sweep", func() (bool, error) { return s.sweepRoles(), nil }})
	for _, st := range steps {
		if err := s.Change(st.name, st.run); err != nil {
			return err
		}
	}
	if s.cfg.Seed == SeedNone {
		s.boot = activation{verifier: verifier, expires: now.Add(time.Hour), live: true, file: link}
		lines = []string{s.cfg.Origins[0] + "/activate#" + encode(secret)}
	}
	if err := writeExclusive(link, []byte(strings.Join(lines, "\n")+"\n")); err != nil {
		return err
	}
	if len(s.cfg.Readers) > 0 {
		if err := s.writeReaderSecret(); err != nil {
			return err
		}
	}
	s.log.Info("activation links written", "file", link)
	s.runWorker()
	return nil
}

func run(steps ...func() error) error {
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) seedAdmin(user, login, invitation string, now time.Time) error {
	ws, b := s.Workspace(), s.identity.Boundary()
	first, second, invite := verifierOf(random(tokenBytes)), verifierOf(random(tokenBytes)), verifierOf(random(tokenBytes))
	defer s.endProbe(first)
	defer s.endProbe(second)
	return run(
		func() error {
			return s.identity.Installation().CreateUser(identity.Operator{Reference: "seed", Reason: "sample data"}, user, LoginTriple(login), login)
		},
		func() error { return b.OpenSession(AdminTriple(), first, time.Minute, time.Minute) },
		func() error {
			return s.identity.Workspace().Invite(first, ws, invitation, identity.Administrator, invite, identity.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}, user)
		},
		func() error { return b.OpenSession(LoginTriple(login), second, time.Minute, time.Minute) },
		func() error { return s.identity.Workspace().Accept(ws, invitation, invite, second) },
		func() error { s.dir.add(user, login, login); return nil },
	)
}

func (s *Server) seedMember(now time.Time) error {
	b := s.identity.Boundary()
	admin := verifierOf(random(tokenBytes))
	defer s.endProbe(admin)
	return run(
		func() error {
			return s.identity.Installation().CreateUser(identity.Operator{Reference: "seed", Reason: "sample data"}, SampleMember, LoginTriple(MemberLogin), MemberLogin)
		},
		func() error { s.dir.add(SampleMember, MemberLogin, MemberLogin); return nil },
		func() error { return b.OpenSession(AdminTriple(), admin, time.Minute, time.Minute) },
		func() error { return s.inviteUser(admin, MemberLogin, SampleMember, identity.Member, now, time.Time{}) },
	)
}

func (s *Server) seedLinks(now time.Time) ([]string, error) {
	lines := []string{}
	issue := func(login string) func() error {
		return func() error {
			token, err := s.recordLink(login, passwords.PurposeSet, now)
			lines = append(lines, login+" "+s.cfg.Origins[0]+"/activate#"+token)
			return err
		}
	}
	err := run(issue(AdminLogin), issue(Admin2Login), issue(Admin3Login), issue(MemberLogin))
	return lines, err
}

func (s *Server) seedReaderGrants(now time.Time) error {
	terms := gate.Terms{Action: gate.View, Kind: ReaderKind, Fields: []string{ReaderField}, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: s.Workspace(), Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	validity := gate.Validity{NotBefore: now, NotAfter: now.Add(readerValidity)}
	steps := []func() error{func() error {
		return s.admin.AddGrant(gate.Grant{ID: "reader-console", ClientID: ConsoleClient, Terms: terms, Validity: validity})
	}}
	for _, user := range []string{SampleAdmin, SampleAdmin2} {
		steps = append(steps, func() error {
			return s.admin.AddGrant(gate.Grant{ID: "reader-" + user, SubjectID: user, Terms: terms, Validity: validity})
		})
	}
	return run(steps...)
}

func (s *Server) seedInstallationGrants(now time.Time) error {
	return run(func() error { return s.grantInstallation(SampleAdmin, now) }, func() error { return s.grantInstallation(SampleAdmin2, now) },
		func() error { return s.grantInstallation(SampleAdmin3, now) })
}

var trustRights = []gate.Action{gate.TrustPropose, gate.TrustValidate, gate.TrustApprove, gate.TrustActivate}

var trustGrantUntil = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)

func (s *Server) seedTrustGrants(now time.Time) error {
	return run(func() error { return s.grantTrust(SampleAdmin, now) }, func() error { return s.grantTrust(SampleAdmin2, now) },
		func() error { return s.grantTrust(SampleAdmin3, now) })
}

func (s *Server) grantTrust(user string, now time.Time) error {
	if s.faulty("trust-grant") {
		return errPartial
	}
	validity := gate.Validity{NotBefore: now, NotAfter: trustGrantUntil}
	steps := []func() error{}
	for _, right := range trustRights {
		terms := gate.Terms{Action: right, Kind: gate.InstallationKind, Period: gate.AnyPeriod(),
			Scope: gate.Scope{WorkspaceID: s.Workspace(), Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
		steps = append(steps, func() error {
			if err := s.admin.AddGrant(gate.Grant{ID: "trust-" + string(right) + "-console", ClientID: ConsoleClient, Terms: terms, Validity: validity}); !errors.Is(err, gate.ErrConflict) {
				return err
			}
			return nil
		}, func() error {
			return s.admin.AddGrant(gate.Grant{ID: "trust-" + string(right) + "-" + user, SubjectID: user, Terms: terms, Validity: validity})
		})
	}
	return run(steps...)
}

func (s *Server) seedBulkGrants(now time.Time) error {
	return run(func() error { return s.grantBulk(SampleAdmin, now) }, func() error { return s.grantBulk(SampleAdmin2, now) })
}

func (s *Server) grantBulk(user string, now time.Time) error {
	if s.faulty("bulk-grant") {
		return errPartial
	}
	terms := gate.Terms{Action: gate.RevokeAccessInBulk, Kind: gate.InstallationKind, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: s.Workspace(), Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	validity := gate.Validity{NotBefore: now, NotAfter: now.Add(installationGrants)}
	return run(func() error {
		if err := s.admin.AddGrant(gate.Grant{ID: "bulk-right-console", ClientID: ConsoleClient, Terms: terms, Validity: validity}); !errors.Is(err, gate.ErrConflict) {
			return err
		}
		return nil
	}, func() error {
		return s.admin.AddGrant(gate.Grant{ID: "bulk-right-" + user, SubjectID: user, Terms: terms, Validity: validity})
	})
}

func (s *Server) grantInstallation(user string, now time.Time) error {
	if s.faulty("grant") {
		return errPartial
	}
	terms := gate.Terms{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Period: gate.AnyPeriod(),
		Scope: gate.Scope{WorkspaceID: s.Workspace(), Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
	validity := gate.Validity{NotBefore: now, NotAfter: now.Add(installationGrants)}
	err := run(func() error {
		if err := s.admin.AddGrant(gate.Grant{ID: "installation-console", ClientID: ConsoleClient, Terms: terms, Validity: validity}); !errors.Is(err, gate.ErrConflict) {
			return err
		}
		return nil
	}, func() error {
		return s.admin.AddGrant(gate.Grant{ID: "installation-" + user, SubjectID: user, Terms: terms, Validity: validity})
	})
	if err == nil {
		s.dir.grant(user, now.Add(installationGrants))
	}
	return err
}
