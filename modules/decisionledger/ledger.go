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
	G         uint64 `json:"g,omitempty"`
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
	oldG      uint64
	mark      marker
	marked    bool
	decisions []Record
	verified  []Record
	epochs    map[string]uint64
	changes   map[string]bool
	status    Status
}

func Open(o Options) (*Ledger, error) { return open(o, platform{}) }

func open(o Options, sys system) (*Ledger, error) {
	if o.Dir == "" || o.MaxBytes <= 0 || o.Clock == nil || o.Random == nil {
		return nil, ErrInvalid
	}
	l := &Ledger{o: o, sys: sys, dir: filepath.Join(o.Dir, DirName), marker: filepath.Join(o.Dir, MarkerName), last: genesisPrev, epochs: map[string]uint64{}, changes: map[string]bool{}}
	l.path = filepath.Join(l.dir, FileName)
	l.status = Status{State: StateOK, CheckedAt: l.now(), File: RelativeFile}
	if err := l.ensureDir(); err != nil {
		return nil, err
	}
	mark, marked, err := readMarker(l.marker)
	if err != nil {
		return nil, err
	}
	l.mark, l.marked = mark, marked
	if l.markerUsable() {
		l.status.Generation, l.status.MarkerUsable = mark.G, true
	}
	_, statErr := os.Lstat(l.path)
	switch {
	case errors.Is(statErr, fs.ErrNotExist) && !marked:
		if o.AcceptNew {
			return nil, ErrNothingToReplace
		}
		if err := l.create("", "", Successor{}); err != nil {
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

func (l *Ledger) markerUsable() bool {
	return l.marked && hexID(l.mark.LedgerID, 32) && l.mark.G < MaxGeneration
}

func (l *Ledger) writeMarker(id string, created time.Time, g uint64) error {
	data, _ := json.Marshal(marker{LedgerID: id, CreatedAt: stamp(created), G: g})
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
	if s.newer > 0 {
		return ErrFormatNewer
	}
	l.oldG, l.verified = s.g, s.decisions
	if s.ledgerID != "" && !l.markerUsable() {
		l.status.Generation = s.g
	}
	switch {
	case s.problem > 0 && marked && s.ledgerID != "" && mark.LedgerID != s.ledgerID:
		l.oldID = s.ledgerID
		l.isolate(IsolationMismatch, 0)
		return nil
	case s.problem > 0:
		l.oldID, l.decisions = s.ledgerID, s.decisions
		l.status.NotDurable = l.startSync() != nil
		reason := IsolationIntegrity
		if s.order {
			reason = IsolationOrder
		}
		l.isolate(reason, s.problem)
		return nil
	case s.records == 0 && marked:
		l.isolate(IsolationIntegrity, 1)
		return nil
	case s.records == 0:
		if err := l.truncate(0); err != nil {
			return err
		}
		return l.genesis("", "", Successor{})
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
		if err := l.writeMarker(s.ledgerID, s.createdAt, s.g); err != nil {
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
	l.size, l.seq, l.last, l.decisions, l.epochs, l.changes = s.size, s.records, s.last, s.decisions, s.epochs, s.changes
	l.status.Generation, l.status.Genesis = s.g, s.genesis
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

func (l *Ledger) create(replaces, reason string, next Successor) error {
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
	return l.genesis(replaces, reason, next)
}

func (l *Ledger) genesis(replaces, reason string, next Successor) error {
	var raw [16]byte
	if _, err := io.ReadFull(l.o.Random, raw[:]); err != nil {
		return err
	}
	id, at := hex.EncodeToString(raw[:]), l.now()
	l.size, l.seq, l.last, l.decisions, l.verified, l.epochs, l.changes = 0, 0, genesisPrev, nil, nil, map[string]uint64{}, map[string]bool{}
	l.status.State = StateOK
	carry := next.Carry
	next.Carry = nil
	head := line{Kind: KindGenesis, LedgerID: id, CreatedAt: stamp(at), Replaces: replaces, ReplaceReason: reason, G: next.Generation, TrustReset: next.TrustReset,
		GenerationStated: next.GenerationStated, CarriedLedger: next.CarriedFrom, ReplacedLine: next.ReplacedLine}
	if failed := l.append(head, at); failed != "" {
		return &CommitError{Reason: failed}
	}
	l.status.LedgerID, l.status.CreatedAt, l.status.Replaces, l.status.ReplaceReason = id, at, replaces, reason
	l.status.Generation, l.status.Genesis = next.Generation, next
	for _, d := range carry {
		if _, err := l.commit(d); err != nil {
			return err
		}
	}
	if err := l.writeMarker(id, at, next.Generation); err != nil {
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
	next, err := l.successor()
	if err != nil {
		return err
	}
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
	return l.create(replaces, reason, next)
}

func (l *Ledger) watermark() string {
	return strconv.FormatUint(l.seq, 10) + ":" + l.last
}

func (l *Ledger) append(rec line, at time.Time) string {
	return l.appendLines([]line{rec}, at)
}

func (l *Ledger) appendLines(recs []line, at time.Time) string {
	out, last, size := []byte{}, l.last, l.size
	for i := range recs {
		recs[i].V, recs[i].Seq, recs[i].Prev, recs[i].At = recs[i].format(), l.seq+uint64(i)+1, last, stamp(at)
		data := encode(recs[i])
		if len(data) > RecordLimit {
			return ReasonCommitFailed
		}
		size += int64(len(data)) + 1
		last = hashOf(data)
		out = append(append(out, data...), '\n')
	}
	if size > l.o.MaxBytes {
		l.status.State = StateFull
		return ReasonFull
	}
	if _, err := l.file.Seek(0, io.SeekEnd); err != nil {
		l.status.State = StateStopped
		return ReasonCommitFailed
	}
	n, err := l.file.Write(out)
	if err != nil || n != len(out) {
		l.status.State = StateStopped
		return ReasonCommitFailed
	}
	if l.file.Sync() != nil || l.o.FailSync != nil && l.o.FailSync() {
		l.status.State = StateStopped
		return ReasonCommitUnknown
	}
	l.size = size
	l.seq, l.last = l.seq+uint64(len(recs)), last
	l.status.Records, l.status.Watermark = l.seq, l.watermark()
	return ""
}

func (l *Ledger) Commit(d Decision) (Receipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.commit(d)
}

func (l *Ledger) refusal() error {
	switch l.status.State {
	case StateIsolated:
		return &CommitError{Reason: ReasonIsolated}
	case StateStopped:
		return &CommitError{Reason: ReasonStopped}
	case StateFull:
		return &CommitError{Reason: ReasonFull}
	}
	return nil
}

func (l *Ledger) commit(d Decision) (Receipt, error) {
	if err := l.refusal(); err != nil {
		return Receipt{}, err
	}
	at := l.now()
	rec := lineFor(d)
	rec.Seq = l.seq + 1
	if !rec.shaped() || rec.Trust != nil && (rec.Trust.NewEpoch <= l.epochs[d.Workspace] || rec.Trust.NewEpoch>>32 != l.status.Generation ||
		l.changes[d.Workspace+"/"+rec.Trust.Change]) {
		return Receipt{}, &CommitError{Reason: ReasonCommitFailed}
	}
	if reason := l.append(rec, at); reason != "" {
		return Receipt{}, &CommitError{Reason: reason}
	}
	d.EndedDelegations = rec.EndedDelegations
	if rec.Trust != nil {
		d.Trust = rec.Trust.trust()
		l.epochs[d.Workspace] = rec.Trust.NewEpoch
		l.changes[d.Workspace+"/"+rec.Trust.Change] = true
	}
	d.Clock = rec.decision().Clock
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

func copyRecords(in []Record) []Record {
	out := make([]Record, len(in))
	for i, r := range in {
		r.Decision.EndedDelegations = append([]string(nil), r.Decision.EndedDelegations...)
		if r.Decision.Trust != nil {
			t := *r.Decision.Trust
			t.Config.Methods = append([]string{}, t.Config.Methods...)
			t.Config.StepUp = append([]StepRule{}, t.Config.StepUp...)
			r.Decision.Trust = &t
		}
		out[i] = r
	}
	return out
}

func (l *Ledger) Decisions() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	return copyRecords(l.decisions)
}

func (l *Ledger) Verified() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.status.State == StateIsolated {
		return copyRecords(l.verified)
	}
	return copyRecords(l.decisions)
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
