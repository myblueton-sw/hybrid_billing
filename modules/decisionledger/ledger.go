package decisionledger

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type handle interface {
	io.ReaderAt
	io.Writer
	io.Seeker
	Sync() error
	Truncate(size int64) error
	Close() error
	Stat() (fs.FileInfo, error)
	Fd() uintptr
}

type system interface {
	openFile(path string, create bool) (handle, error)
	lock(h handle) error
	checkDir(info fs.FileInfo) error
	syncDir(path string) error
	rename(from, to string) error
}

type marker struct {
	LedgerID  string `json:"ledger_id"`
	CreatedAt string `json:"created_at"`
}

type Ledger struct {
	mu        sync.Mutex
	o         Options
	sys       system
	dir       string
	path      string
	marker    string
	file      handle
	size      int64
	seq       uint64
	last      string
	oldID     string
	decisions []Record
	status    Status
}

func Open(o Options) (*Ledger, error) { return open(o, platform{}) }

func open(o Options, sys system) (*Ledger, error) {
	if o.Dir == "" || o.MaxBytes <= 0 || o.Clock == nil || o.Random == nil {
		return nil, ErrInvalid
	}
	l := &Ledger{o: o, sys: sys, dir: filepath.Join(o.Dir, DirName), marker: filepath.Join(o.Dir, MarkerName), last: genesisPrev}
	l.path = filepath.Join(l.dir, FileName)
	l.status = Status{State: StateOK, CheckedAt: l.now(), File: RelativeFile}
	if err := l.ensureDir(); err != nil {
		return nil, err
	}
	mark, marked, err := readMarker(l.marker)
	if err != nil {
		return nil, err
	}
	_, statErr := os.Lstat(l.path)
	switch {
	case errors.Is(statErr, fs.ErrNotExist) && !marked:
		if o.AcceptNew {
			return nil, ErrNothingToReplace
		}
		if err := l.create("", ""); err != nil {
			l.Close()
			return nil, err
		}
		return l, nil
	case errors.Is(statErr, fs.ErrNotExist):
		l.isolate(IsolationMissing, 0)
	default:
		if err := l.check(mark, marked); err != nil {
			l.Close()
			return nil, err
		}
	}
	switch {
	case l.status.State == StateIsolated && o.AcceptNew:
		if err := l.replace(); err != nil {
			l.Close()
			return nil, err
		}
	case o.AcceptNew:
		l.Close()
		return nil, ErrNothingToReplace
	}
	return l, nil
}

func (l *Ledger) now() time.Time { return l.o.Clock().UTC() }

func (l *Ledger) ensureDir() error {
	info, err := os.Lstat(l.dir)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.Mkdir(l.dir, 0o700); err != nil {
			return err
		}
		return l.sys.syncDir(l.o.Dir)
	}
	if err != nil {
		return err
	}
	return l.sys.checkDir(info)
}

func readMarker(path string) (marker, bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return marker{}, false, missingIsNil(err)
	}
	if !info.Mode().IsRegular() || info.Size() > LineLimit {
		return marker{}, false, ErrUnsafe
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return marker{}, false, err
	}
	var m marker
	if json.Unmarshal(data, &m) != nil {
		return marker{}, true, nil
	}
	return m, true, nil
}

func missingIsNil(err error) error {
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func (l *Ledger) writeMarker(id string, created time.Time) error {
	data, _ := json.Marshal(marker{LedgerID: id, CreatedAt: stamp(created)})
	f, err := os.CreateTemp(l.o.Dir, MarkerName+".*")
	if err != nil {
		return err
	}
	_, werr := f.Write(append(data, '\n'))
	err = errors.Join(werr, f.Sync(), f.Close())
	if err == nil {
		err = l.sys.rename(f.Name(), l.marker)
	}
	if err != nil {
		os.Remove(f.Name())
		return err
	}
	return l.sys.syncDir(l.o.Dir)
}

func (l *Ledger) isolate(reason Isolation, at int) {
	l.status.State, l.status.Isolation, l.status.ProblemLine = StateIsolated, reason, at
}

func (l *Ledger) check(mark marker, marked bool) error {
	h, err := l.sys.openFile(l.path, false)
	if err != nil {
		return err
	}
	if err := l.sys.lock(h); err != nil {
		h.Close()
		return err
	}
	l.file = h
	info, err := h.Stat()
	if err != nil {
		return err
	}
	s := verify(h, info.Size())
	switch {
	case s.problem > 0 && marked && s.ledgerID != "" && mark.LedgerID != s.ledgerID:
		l.oldID = s.ledgerID
		l.isolate(IsolationMismatch, 0)
		return nil
	case s.problem > 0:
		l.oldID, l.decisions = s.ledgerID, s.decisions
		l.status.NotDurable = l.startSync() != nil
		l.isolate(IsolationIntegrity, s.problem)
		return nil
	case s.records == 0 && marked:
		l.isolate(IsolationIntegrity, 1)
		return nil
	case s.records == 0:
		if err := l.truncate(0); err != nil {
			return err
		}
		return l.genesis("", "")
	case marked && mark.LedgerID != s.ledgerID:
		l.oldID = s.ledgerID
		l.isolate(IsolationMismatch, 0)
		return nil
	case s.size > l.o.MaxBytes:
		return ErrBound
	}
	l.adopt(s)
	if l.startSync() != nil {
		l.status.State, l.status.NotDurable = StateStopped, true
		return nil
	}
	if s.torn > 0 {
		if err := l.recover(s.torn); err != nil {
			return err
		}
	}
	if !marked {
		if err := l.writeMarker(s.ledgerID, s.createdAt); err != nil {
			return err
		}
	}
	l.fullAtStart()
	return nil
}

func (l *Ledger) startSync() error {
	if l.o.FailStartSync {
		return errStartSync
	}
	return l.file.Sync()
}

func (l *Ledger) adopt(s scan) {
	l.size, l.seq, l.last, l.decisions = s.size, s.records, s.last, s.decisions
	l.status.LedgerID, l.status.CreatedAt, l.status.Replaces, l.status.ReplaceReason = s.ledgerID, s.createdAt, s.replaces, s.reason
	l.status.Records, l.status.Watermark = s.records, l.watermark()
}

func (l *Ledger) truncate(size int64) error {
	if err := l.file.Truncate(size); err != nil {
		return err
	}
	return l.file.Sync()
}

func (l *Ledger) recover(torn int64) error {
	if err := l.truncate(l.size); err != nil {
		return err
	}
	at := l.now()
	l.status.Recovery = &Recovery{At: at, TruncatedBytes: torn}
	l.status.State = StateRecovered
	if l.o.MaxBytes-l.size < RecordLimit {
		l.status.State = StateFull
		return nil
	}
	l.append(line{Kind: KindRecovery, TruncatedBytes: torn, RecoveredAt: stamp(at)}, at)
	return nil
}

func (l *Ledger) fullAtStart() {
	if l.status.State != StateStopped && l.o.MaxBytes-l.size < RecordLimit {
		l.status.State = StateFull
	}
}

func (l *Ledger) create(replaces, reason string) error {
	h, err := l.sys.openFile(l.path, true)
	if err != nil {
		return err
	}
	if err := l.sys.lock(h); err != nil {
		h.Close()
		return err
	}
	l.file = h
	if err := l.sys.syncDir(l.dir); err != nil {
		return err
	}
	return l.genesis(replaces, reason)
}

func (l *Ledger) genesis(replaces, reason string) error {
	var raw [16]byte
	if _, err := io.ReadFull(l.o.Random, raw[:]); err != nil {
		return err
	}
	id, at := hex.EncodeToString(raw[:]), l.now()
	l.size, l.seq, l.last, l.decisions = 0, 0, genesisPrev, nil
	l.status.State = StateOK
	if failed := l.append(line{Kind: KindGenesis, LedgerID: id, CreatedAt: stamp(at), Replaces: replaces, ReplaceReason: reason}, at); failed != "" {
		return &CommitError{Reason: failed}
	}
	l.status.LedgerID, l.status.CreatedAt, l.status.Replaces, l.status.ReplaceReason = id, at, replaces, reason
	if err := l.writeMarker(id, at); err != nil {
		return err
	}
	l.fullAtStart()
	return nil
}

func (l *Ledger) replace() error {
	replaces := l.oldID
	if replaces == "" {
		replaces = UnknownLedger
	}
	reason := string(l.status.Isolation)
	if l.file != nil {
		l.file.Close()
		l.file = nil
		name := FileName + ReplacedTag + l.now().Format("20060102T150405.000000000Z")
		if err := l.sys.rename(l.path, filepath.Join(l.dir, name)); err != nil {
			return err
		}
		l.status.ReplacedFile = name
	}
	l.status.Isolation, l.status.ProblemLine = "", 0
	return l.create(replaces, reason)
}

func (l *Ledger) watermark() string {
	return strconv.FormatUint(l.seq, 10) + ":" + l.last
}

func (l *Ledger) append(rec line, at time.Time) string {
	rec.V, rec.Seq, rec.Prev, rec.At = FormatV1, l.seq+1, l.last, stamp(at)
	data := encode(rec)
	if len(data) > RecordLimit {
		return ReasonCommitFailed
	}
	if l.size+int64(len(data))+1 > l.o.MaxBytes {
		l.status.State = StateFull
		return ReasonFull
	}
	if _, err := l.file.Seek(0, io.SeekEnd); err != nil {
		l.status.State = StateStopped
		return ReasonCommitFailed
	}
	n, err := l.file.Write(append(append([]byte{}, data...), '\n'))
	if err != nil || n != len(data)+1 {
		l.status.State = StateStopped
		return ReasonCommitFailed
	}
	if l.file.Sync() != nil {
		l.status.State = StateStopped
		return ReasonCommitUnknown
	}
	l.size += int64(n)
	l.seq, l.last = rec.Seq, hashOf(data)
	l.status.Records, l.status.Watermark = l.seq, l.watermark()
	return ""
}

func (l *Ledger) Commit(d Decision) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch l.status.State {
	case StateIsolated:
		return Receipt{}, &CommitError{Reason: ReasonIsolated}
	case StateStopped:
		return Receipt{}, &CommitError{Reason: ReasonStopped}
	case StateFull:
		return Receipt{}, &CommitError{Reason: ReasonFull}
	}
	at := l.now()
	rec := line{Kind: KindDecision, Workspace: d.Workspace, Decision: d.Decision, Target: d.Target, TargetVersion: d.TargetVersion, Revocation: d.Revocation,
		Request: d.Request, RequestDigest: d.RequestDigest, RequestedBy: d.RequestedBy, ApprovedBy: d.ApprovedBy, DenyGeneration: d.DenyGeneration,
		EndedDelegations: append([]string(nil), d.EndedDelegations...)}
	if !rec.shaped() {
		return Receipt{}, &CommitError{Reason: ReasonCommitFailed}
	}
	if reason := l.append(rec, at); reason != "" {
		return Receipt{}, &CommitError{Reason: reason}
	}
	d.EndedDelegations = rec.EndedDelegations
	l.decisions = append(l.decisions, Record{Sequence: l.seq, Watermark: l.watermark(), At: at, Decision: d})
	return Receipt{Sequence: l.seq, Watermark: l.watermark()}, nil
}

func (l *Ledger) Status() Status {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.status
	if out.Recovery != nil {
		r := *out.Recovery
		out.Recovery = &r
	}
	return out
}

func (l *Ledger) Decisions() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Record, len(l.decisions))
	for i, r := range l.decisions {
		r.Decision.EndedDelegations = append([]string(nil), r.Decision.EndedDelegations...)
		out[i] = r
	}
	return out
}

func (l *Ledger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	if l.status.State != StateIsolated {
		l.status.State = StateStopped
	}
	return err
}
