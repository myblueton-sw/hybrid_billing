package decisionledger

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var errInjected = errors.New("injected")

type faultyFile struct {
	handle
	writes   *int
	writeErr bool
	fullErr  bool
	short    bool
	syncErr  bool
	seekErr  bool
	statErr  bool
	truncErr bool
}

func (f *faultyFile) Write(b []byte) (int, error) {
	*f.writes++
	switch {
	case f.writeErr:
		return 0, errInjected
	case f.short:
		return f.handle.Write(b[:len(b)/2])
	case f.fullErr:
		n, _ := f.handle.Write(b)
		return n, errInjected
	}
	return f.handle.Write(b)
}

func (f *faultyFile) Sync() error {
	if f.syncErr {
		return errInjected
	}
	return f.handle.Sync()
}

func (f *faultyFile) Seek(offset int64, whence int) (int64, error) {
	if f.seekErr {
		return 0, errInjected
	}
	return f.handle.Seek(offset, whence)
}

func (f *faultyFile) Stat() (fs.FileInfo, error) {
	if f.statErr {
		return nil, errInjected
	}
	return f.handle.Stat()
}

func (f *faultyFile) Truncate(size int64) error {
	if f.truncErr {
		return errInjected
	}
	return f.handle.Truncate(size)
}

type fakeSys struct {
	platform
	synced    []string
	openErr   bool
	lockErr   bool
	syncErr   bool
	renameErr bool
	file      *faultyFile
	writes    int
}

func (s *fakeSys) openFile(path string, create bool) (handle, error) {
	if s.openErr {
		return nil, errInjected
	}
	h, err := s.platform.openFile(path, create)
	if err != nil {
		return nil, err
	}
	if s.file == nil {
		s.file = &faultyFile{}
	}
	s.file.handle, s.file.writes = h, &s.writes
	return s.file, nil
}

func (s *fakeSys) lock(h handle) error {
	if s.lockErr {
		return errInjected
	}
	return s.platform.lock(h)
}

func (s *fakeSys) syncDir(path string) error {
	s.synced = append(s.synced, path)
	if s.syncErr {
		return errInjected
	}
	return s.platform.syncDir(path)
}

func (s *fakeSys) rename(from, to string) error {
	if s.renameErr {
		return errInjected
	}
	return s.platform.rename(from, to)
}

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errInjected }

func faulted(t *testing.T, o Options, s *fakeSys) *Ledger {
	t.Helper()
	l, err := open(o, s)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func TestCommitFaultsStopTheLedger(t *testing.T) {
	for name, c := range map[string]struct {
		set    func(*faultyFile)
		reason string
	}{
		"write error":              {func(f *faultyFile) { f.writeErr = true }, ReasonCommitFailed},
		"short write":              {func(f *faultyFile) { f.short = true }, ReasonCommitFailed},
		"full write with an error": {func(f *faultyFile) { f.fullErr = true }, ReasonCommitFailed},
		"sync error":               {func(f *faultyFile) { f.syncErr = true }, ReasonCommitUnknown},
		"seek error":               {func(f *faultyFile) { f.seekErr = true }, ReasonCommitFailed},
	} {
		t.Run(name, func(t *testing.T) {
			s := &fakeSys{}
			l := faulted(t, options(t), s)
			defer l.Close()
			c.set(s.file)
			if _, err := l.Commit(sample("1")); err.(*CommitError).CommitReason() != c.reason || l.Status().State != StateStopped {
				t.Fatalf("%v %+v", err, l.Status())
			}
			writes := s.writes
			if _, err := l.Commit(sample("2")); err.(*CommitError).Reason != ReasonStopped || s.writes != writes {
				t.Errorf("a stopped ledger touched the file %v", err)
			}
			if l.Status().Records != 1 || len(l.Decisions()) != 0 {
				t.Error("a failed commit was counted")
			}
		})
	}
}

func TestDirectorySyncs(t *testing.T) {
	o := options(t)
	s := &fakeSys{}
	l := faulted(t, o, s)
	l.Close()
	want := []string{o.Dir, filepath.Join(o.Dir, DirName), o.Dir}
	if strings.Join(s.synced, "|") != strings.Join(want, "|") {
		t.Errorf("synced %v", s.synced)
	}
}

func TestOpenFaults(t *testing.T) {
	cases := map[string]struct {
		prepare func(t *testing.T, o *Options)
		set     func(*fakeSys)
	}{
		"create open":      {nil, func(s *fakeSys) { s.openErr = true }},
		"create lock":      {nil, func(s *fakeSys) { s.lockErr = true }},
		"create dir sync":  {nil, func(s *fakeSys) { s.syncErr = true }},
		"create file sync": {withDir, func(s *fakeSys) { s.syncErr = true }},
		"genesis random":   {func(t *testing.T, o *Options) { o.Random = brokenReader{} }, nil},
		"genesis write":    {nil, func(s *fakeSys) { s.file = &faultyFile{writeErr: true} }},
		"marker rename":    {nil, func(s *fakeSys) { s.renameErr = true }},
		"check open":       {existing, func(s *fakeSys) { s.openErr = true }},
		"check lock":       {existing, func(s *fakeSys) { s.lockErr = true }},
		"check stat":       {existing, func(s *fakeSys) { s.file = &faultyFile{statErr: true} }},
		"empty truncate":   {emptyUnmarked, func(s *fakeSys) { s.file = &faultyFile{truncErr: true} }},
		"empty sync":       {emptyUnmarked, func(s *fakeSys) { s.file = &faultyFile{syncErr: true} }},
		"torn truncate":    {torn, func(s *fakeSys) { s.file = &faultyFile{truncErr: true} }},
		"unmarked rename":  {unmarked, func(s *fakeSys) { s.renameErr = true }},
		"replace rename":   {damaged, func(s *fakeSys) { s.renameErr = true }},
		"replace create":   {damaged, func(s *fakeSys) { s.lockErr = true }},
		"marker temp file": {readOnlyData, nil},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			o := options(t)
			if c.prepare != nil {
				c.prepare(t, &o)
			}
			s := &fakeSys{}
			if c.set != nil {
				c.set(s)
			}
			if _, err := open(o, s); err == nil {
				t.Error("opened")
			}
		})
	}
}

func withDir(t *testing.T, o *Options) { os.Mkdir(filepath.Join(o.Dir, DirName), 0o700) }

func existing(t *testing.T, o *Options) {
	l := must(t, *o)
	l.Close()
}

func emptyUnmarked(t *testing.T, o *Options) {
	existing(t, o)
	os.WriteFile(ledgerPath(*o), nil, 0o600)
	os.Remove(filepath.Join(o.Dir, MarkerName))
}

func torn(t *testing.T, o *Options) {
	existing(t, o)
	f, _ := os.OpenFile(ledgerPath(*o), os.O_WRONLY|os.O_APPEND, 0o600)
	f.Write([]byte("{"))
	f.Close()
}

func unmarked(t *testing.T, o *Options) {
	existing(t, o)
	os.Remove(filepath.Join(o.Dir, MarkerName))
}

func damaged(t *testing.T, o *Options) {
	existing(t, o)
	os.WriteFile(ledgerPath(*o), []byte("x\n"), 0o600)
	o.AcceptNew = true
}

func readOnlyData(t *testing.T, o *Options) {
	os.Mkdir(filepath.Join(o.Dir, DirName), 0o700)
	os.Chmod(o.Dir, 0o500)
	t.Cleanup(func() { os.Chmod(o.Dir, 0o700) })
}

func TestRecoveryAppendFailureStops(t *testing.T) {
	o := options(t)
	torn(t, &o)
	s := &fakeSys{file: &faultyFile{writeErr: true}}
	l := faulted(t, o, s)
	defer l.Close()
	if st := l.Status(); st.State != StateStopped || st.Recovery == nil {
		t.Errorf("%+v", st)
	}
}

func TestUnreadableMarker(t *testing.T) {
	o := options(t)
	existing(t, &o)
	path := filepath.Join(o.Dir, MarkerName)
	os.Chmod(path, 0)
	t.Cleanup(func() { os.Chmod(path, 0o600) })
	if _, err := Open(o); err == nil || errors.Is(err, ErrUnsafe) {
		t.Errorf("unreadable marker %v", err)
	}
}

func TestMissingIsNil(t *testing.T) {
	if missingIsNil(fs.ErrNotExist) != nil || missingIsNil(errInjected) != errInjected {
		t.Error("missingIsNil")
	}
	var _ io.ReaderAt = (*faultyFile)(nil)
}

func TestASyncFailureAtStartStopsButKeepsEveryVerifiedRecord(t *testing.T) {
	for name, prepare := range map[string]func(t *testing.T, o *Options){
		"verified ledger": func(t *testing.T, o *Options) {
			l := must(t, *o)
			commit(t, l, "1")
			l.Close()
		},
		"torn tail": func(t *testing.T, o *Options) {
			l := must(t, *o)
			commit(t, l, "1")
			l.Close()
			appendRaw(t, *o, []byte(`{"v":1`))
		},
	} {
		t.Run(name, func(t *testing.T) {
			o := options(t)
			prepare(t, &o)
			before, _ := os.ReadFile(ledgerPath(o))
			s := &fakeSys{file: &faultyFile{syncErr: true}}
			l := faulted(t, o, s)
			defer l.Close()
			st := l.Status()
			if st.State != StateStopped || !st.NotDurable || len(l.Decisions()) != 1 || st.Recovery != nil || st.Records != 2 {
				t.Errorf("%+v %d", st, len(l.Decisions()))
			}
			if after, _ := os.ReadFile(ledgerPath(o)); !bytes.Equal(after, before) {
				t.Error("the file changed")
			}
			if _, err := l.Commit(sample("2")); err.(*CommitError).Reason != ReasonStopped || s.writes != 0 {
				t.Errorf("commit %v writes %d", err, s.writes)
			}
		})
	}
	o := options(t)
	l := must(t, o)
	commit(t, l, "1")
	commit(t, l, "2")
	l.Close()
	flipLine := lines(t, o)
	flipLine[2] = bytes.Replace(flipLine[2], []byte("grant-2"), []byte("grant-x"), 1)
	rewrite(t, o, flipLine)
	s := &fakeSys{file: &faultyFile{syncErr: true}}
	iso := faulted(t, o, s)
	defer iso.Close()
	if st := iso.Status(); st.State != StateIsolated || st.ProblemLine != 3 || !st.NotDurable || len(iso.Decisions()) != 1 {
		t.Errorf("an isolated prefix that could not be synced is still replayed and labelled %+v %d", st, len(iso.Decisions()))
	}
}

func TestTheStartSyncOption(t *testing.T) {
	o := options(t)
	l := must(t, o)
	commit(t, l, "1")
	l.Close()
	o.FailStartSync = true
	stopped := must(t, o)
	defer stopped.Close()
	if st := stopped.Status(); st.State != StateStopped || !st.NotDurable || len(stopped.Decisions()) != 1 {
		t.Errorf("%+v", st)
	}
}
