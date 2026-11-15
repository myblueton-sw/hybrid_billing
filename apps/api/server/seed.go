package server

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	ActivationFile  = "activation.txt"
	SampleWorkspace = "sample-workspace"
	EmptyWorkspace  = "workspace"
	SampleAdmin     = "sample-admin"
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

func AdminTriple() identity.Triple {
	return identity.Triple{Connection: PasswordLink, Issuer: Installation, Subject: "admin"}
}

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
	link := filepath.Join(s.cfg.Data, ActivationFile)
	if err := os.Remove(link); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	secret := random(tokenBytes)
	verifier := verifierOf(secret)
	now := s.clock()
	steps := []step{{"arm-bootstrap", func() (bool, error) {
		return true, s.identity.Installation().ArmBootstrap(verifier, identity.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}, s.Workspace())
	}}}
	if s.cfg.Seed == SeedSample {
		steps = append(steps, step{"bootstrap-admin", func() (bool, error) {
			return true, s.identity.Boundary().Bootstrap(verifier, SampleAdmin, AdminTriple(), "admin@example.invalid")
		}})
		steps = append(steps, s.seed...)
	}
	for _, st := range steps {
		if err := s.Change(st.name, st.run); err != nil {
			return err
		}
	}
	if s.cfg.Seed == SeedNone {
		if err := writeExclusive(link, []byte(s.cfg.Origins[0]+"/activate#"+encode(secret)+"\n")); err != nil {
			return err
		}
		s.log.Info("activation link written", "file", link)
	}
	return nil
}
