package contracts

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrInvalid  = errors.New("request or record is incomplete or malformed")
	ErrNotFound = errors.New("contract or knowledge point does not exist in this workspace")
	ErrConflict = errors.New("write conflicts with the stored state")
	ErrRejected = errors.New("write is refused by a lifecycle or settlement rule")
	ErrState    = errors.New("settlement state does not allow this step")
	ErrLimit    = errors.New("a configured limit is exceeded")
)

const maxIDBytes = 128

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

const secondsPerDay = 86400

var (
	epoch   = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC).Unix() / secondsPerDay
	lastDay = time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC).Unix()/secondsPerDay - epoch + 1
)

type Day struct{ n int32 }

func NewDay(year, month, day int) (Day, error) {
	if year < 1 || year > 9999 || month < 1 || month > 12 || day < 1 || day > 31 {
		return Day{}, ErrInvalid
	}
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return Day{}, ErrInvalid
	}
	return Day{n: int32(date.Unix()/secondsPerDay-epoch) + 1}, nil
}

func (d Day) Valid() bool { return d.n > 0 }

func (d Day) Before(other Day) bool { return d.n < other.n }

func (d Day) date() time.Time {
	return time.Unix((int64(d.n)-1+epoch)*secondsPerDay, 0).UTC()
}

func (d Day) String() string {
	if !d.Valid() {
		return ""
	}
	return d.date().Format("2006-01-02")
}

func (d Day) Add(days int) (Day, error) {
	moved := int64(d.n) + int64(days)
	if !d.Valid() || moved < 1 || moved > lastDay {
		return Day{}, ErrInvalid
	}
	return Day{n: int32(moved)}, nil
}

type Interval struct {
	start Day
	end   Day
}

func Between(start, end Day) (Interval, error) {
	if !start.Valid() || !end.Valid() || !start.Before(end) {
		return Interval{}, ErrInvalid
	}
	return Interval{start: start, end: end}, nil
}

func (i Interval) Valid() bool { return i.start.Valid() && i.start.Before(i.end) }

func (i Interval) Start() Day { return i.start }

func (i Interval) End() Day { return i.end }

func (i Interval) Days() int { return int(i.end.n - i.start.n) }

func (i Interval) String() string { return fmt.Sprintf("[%s,%s)", i.start, i.end) }

type Kind string

const (
	Start   Kind = "start"
	Change  Kind = "change"
	Renew   Kind = "renew"
	Suspend Kind = "suspend"
	Resume  Kind = "resume"
	End     Kind = "end"
)

type Event struct {
	ID       string
	Kind     Kind
	From     Day
	Revision uint64
}

func (e Event) valid() bool {
	if !validID(e.ID) || !e.From.Valid() {
		return false
	}
	switch e.Kind {
	case Start, Change:
		return e.Revision > 0
	case Renew, Suspend, Resume, End:
		return e.Revision == 0
	}
	return false
}

type State string

const (
	NotStarted State = "not_started"
	Active     State = "active"
	Suspended  State = "suspended"
	Ended      State = "ended"
)

type Segment struct {
	Interval Interval
	State    State
	Revision uint64
}

type SettlementState string

const (
	NoSettlement SettlementState = "none"
	PendingFinal SettlementState = "pending_final"
	Settled      SettlementState = "settled"
)

type CancelReason string

const (
	RecordedInError   CancelReason = "recorded_in_error"
	ContractContinues CancelReason = "contract_continues"
)

type ReopenReason string

const (
	LateSource ReopenReason = "late_source"
	LateCredit ReopenReason = "late_credit"
	Correcting ReopenReason = "correction"
)

type EvidenceKind string

const (
	Secured EvidenceKind = "secured"
	Missing EvidenceKind = "missing"
)

type GapReason string

const (
	SourceUnavailable        GapReason = "source_unavailable"
	AccessWithdrawnEmergency GapReason = "access_withdrawn_emergency"
	ProviderRefused          GapReason = "provider_refused"
	NotAttempted             GapReason = "not_attempted"
	LateSourceReceived       GapReason = "late_source_received"
)

type Evidence struct {
	Kind         EvidenceKind
	Reference    string
	CoveredUntil Day
	Resolves     []string
	Reason       GapReason
	OwnerID      string
	FollowUp     string
}

func (e Evidence) valid() bool {
	switch e.Kind {
	case Secured:
		seen := map[string]bool{}
		for _, id := range e.Resolves {
			if !validID(id) || seen[id] {
				return false
			}
			seen[id] = true
		}
		return validID(e.Reference) && e.CoveredUntil.Valid() && e.Reason == "" && e.OwnerID == "" && e.FollowUp == ""
	case Missing:
		switch e.Reason {
		case SourceUnavailable, AccessWithdrawnEmergency, ProviderRefused, NotAttempted, LateSourceReceived:
		default:
			return false
		}
		return validID(e.OwnerID) && validID(e.FollowUp) && e.Reference == "" && !e.CoveredUntil.Valid() && len(e.Resolves) == 0
	}
	return false
}

type EvidenceRecord struct {
	Sequence uint64
	Write    string
	Evidence Evidence
	Resolved bool
}

type WriteKind string

const (
	WriteBatch    WriteKind = "batch"
	WriteEvidence WriteKind = "evidence"
	WriteSettle   WriteKind = "settle"
	WriteReopen   WriteKind = "reopen"
)

type Record struct {
	Sequence    uint64
	At          time.Time
	Write       string
	Kind        WriteKind
	Withdrawals []string
	Events      []Event
	Cancel      CancelReason
	Evidence    Evidence
	AsOf        Day
	Reopen      ReopenReason
	State       SettlementState
}

type Outcome struct {
	Sequence uint64
	Replay   bool
}

type Settlement struct {
	State      SettlementState
	Evidence   []EvidenceRecord
	Reopenings int
}

type Position struct {
	State    State
	Revision uint64
}

type Limits struct {
	Contracts int
	Writes    int
	Reserve   int
	Batch     int
}

func (l Limits) valid() bool {
	return l.Contracts > 0 && l.Writes > 0 && l.Reserve > 0 && l.Batch > 0
}
