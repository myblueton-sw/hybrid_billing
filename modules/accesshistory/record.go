package accesshistory

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash/crc32"
	"time"
)

const (
	KindSignedIn          = "signed_in"
	KindSignInRefused     = "sign_in_refused"
	KindSignInRefusedMore = "sign_in_refused_more"
	KindPasswordConfirmed = "password_confirmed"
	KindPasswordFailed    = "password_confirmation_failed"
	KindSignedOut         = "signed_out"
	KindSessionEnded      = "session_ended"
	KindPasswordChanged   = "password_changed"

	sysRunStarted      = "run_started"
	sysRunStopped      = "run_stopped"
	sysDropped         = "dropped"
	sysRecordingStop   = "recording_stopped"
	sysRecoveredTail   = "recovered_tail"
	sysEarlyDeletion   = "early_deletion"
	sysPurged          = "purged"
	sysPeriodChanged   = "period_changed"
	sysIncarnation     = "incarnation"
	sysHeld            = "held"
	sysHoldReleased    = "hold_released"
	sysPurgeDeferred   = "purge_deferred"
	maxLine            = 1024
	formatVersion      = 1
	ClassIdentity      = "identity"
	refusedCapPerDay   = 100
	queueSize          = 4096
	batchSize          = 256
	batchWait          = 50 * time.Millisecond
	drainWait          = 2 * time.Second
	DefaultPage        = 50
	MaxPage            = 100
	retentionFile      = "retention.json"
	peopleFile         = "people.log"
	lockFile           = "lock"
	DirName            = "access-history"
	dayLayout          = "2006-01-02"
	daySuffix          = ".log"
	heldDamagedSuffix  = ".held-damaged"
	tempSuffix         = ".tmp"
	auditKeepYears     = 7
	ReasonWrongPass    = "wrong_password"
	ReasonLocked       = "locked"
	ReasonCredentials  = "credentials_changed"
	ReasonAccess       = "access_refused"
	ReasonSessionFail  = "session_unavailable"
	ReasonAttemptLimit = "attempt_limit"
)

const (
	CauseByYou           = "ended_by_you"
	CausePasswordChanged = "password_changed"
	CausePasswordReset   = "password_reset"
	CauseAttemptLimit    = "attempt_limit"
	CauseAdministrator   = "administrator"
	CauseRolePeriod      = "role_period_ended"
	CauseSessionLimit    = "session_limit"
	CauseAccessChanged   = "access_changed"
	CauseTrustChange     = "trust_change"
	CauseBulkRevocation  = "bulk_revocation"
)

var (
	ErrUnsafe     = errors.New("accesshistory: unsafe file or directory")
	ErrLocked     = errors.New("accesshistory: directory is in use by another process")
	ErrConfig     = errors.New("accesshistory: configuration refused")
	ErrDamaged    = errors.New("accesshistory: damaged person-state file")
	ErrClosed     = errors.New("accesshistory: closed")
	ErrStopped    = errors.New("accesshistory: recording stopped")
	errInvalidRec = errors.New("accesshistory: invalid record")
	errInjected   = errors.New("accesshistory: injected fault")
)

var (
	personalKinds = []string{KindSignedIn, KindSignInRefused, KindSignInRefusedMore, KindPasswordConfirmed, KindPasswordFailed, KindSignedOut, KindSessionEnded, KindPasswordChanged}
	systemKinds   = []string{sysRunStarted, sysRunStopped, sysDropped, sysRecordingStop, sysRecoveredTail, sysEarlyDeletion, sysPurged, sysPeriodChanged, sysIncarnation, sysHeld, sysHoldReleased, sysPurgeDeferred}
	reasonsByKind = map[string][]string{
		KindSignInRefused:  {ReasonWrongPass, ReasonLocked, ReasonCredentials, ReasonAccess, ReasonSessionFail},
		KindPasswordFailed: {ReasonWrongPass, ReasonAttemptLimit},
		KindSessionEnded: {CauseByYou, CausePasswordChanged, CausePasswordReset, CauseAttemptLimit, CauseAdministrator, CauseRolePeriod, CauseSessionLimit,
			CauseAccessChanged, CauseTrustChange, CauseBulkRevocation},
	}
	withSession = map[string]bool{KindSignedIn: true, KindPasswordConfirmed: true, KindPasswordFailed: true, KindSignedOut: true, KindSessionEnded: true, KindPasswordChanged: true}
	crcTable    = crc32.MakeTable(crc32.Castagnoli)
)

type Record struct {
	V         int              `json:"v"`
	Seq       uint64           `json:"s"`
	At        int64            `json:"t"`
	User      string           `json:"u,omitempty"`
	Workspace string           `json:"w,omitempty"`
	Kind      string           `json:"k"`
	Reason    string           `json:"r,omitempty"`
	Handle    string           `json:"h,omitempty"`
	Browser   string           `json:"b,omitempty"`
	OS        string           `json:"o,omitempty"`
	Count     int64            `json:"c,omitempty"`
	SourceSeq uint64           `json:"src,omitempty"`
	N         uint64           `json:"n,omitempty"`
	From      int64            `json:"f,omitempty"`
	To        int64            `json:"to,omitempty"`
	Per       map[string]int64 `json:"p,omitempty"`
	Days      int              `json:"d,omitempty"`
	Hold      string           `json:"hold,omitempty"`
	Run       string           `json:"run,omitempty"`
}

func Personal(kind string) bool { return contains(personalKinds, kind) }

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func millis(t time.Time) int64 { return t.UTC().UnixMilli() }

func fromMillis(ms int64) time.Time { return time.UnixMilli(ms).UTC() }

func validHandle(h string) bool {
	if len(h) != 16 {
		return false
	}
	_, err := hex.DecodeString(h)
	return err == nil && h == stringsLower(h)
}

func stringsLower(s string) string { return string(bytes.ToLower([]byte(s))) }

func (r Record) valid() bool {
	switch {
	case r.V != formatVersion || r.Seq == 0 || r.At <= 0:
		return false
	case r.SourceSeq > 0 && (r.Kind != KindSignInRefusedMore || r.SourceSeq > r.Seq):
		return false
	case contains(systemKinds, r.Kind):
		return r.Kind == sysIncarnation || r.User == ""
	case !Personal(r.Kind) || r.User == "" || len(r.User) > 128 || len(r.Workspace) > 128 || r.N == 0:
		return false
	}
	if allowed, ok := reasonsByKind[r.Kind]; ok {
		if !contains(allowed, r.Reason) {
			return false
		}
	} else if r.Reason != "" {
		return false
	}
	if withSession[r.Kind] && (!validHandle(r.Handle) || r.Workspace == "") {
		return false
	}
	if !withSession[r.Kind] && (r.Handle != "" || r.Workspace != "") {
		return false
	}
	if r.Kind == KindSignInRefusedMore && r.Count <= 0 {
		return false
	}
	return r.Browser == "" || contains(Browsers, r.Browser) && contains(Systems, r.OS)
}

func encodeLine(v any) ([]byte, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, len(body)+10)
	out = append(out, []byte(crcHex(body))...)
	out = append(out, ' ')
	out = append(out, body...)
	out = append(out, '\n')
	if len(out) > maxLine {
		return nil, errInvalidRec
	}
	return out, nil
}

func crcHex(body []byte) string {
	sum := crc32.Checksum(body, crcTable)
	const digits = "0123456789abcdef"
	out := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		out[i] = digits[sum&0xf]
		sum >>= 4
	}
	return string(out)
}

func decodeLine(raw []byte, into any) error {
	if len(raw) > maxLine || len(raw) < 11 || raw[8] != ' ' {
		return errInvalidRec
	}
	body := raw[9:]
	if crcHex(body) != string(raw[:8]) {
		return errInvalidRec
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(into); err != nil {
		return errInvalidRec
	}
	return nil
}

func bestEffortUser(raw []byte) string {
	if len(raw) < 10 {
		return ""
	}
	var r struct {
		User string `json:"u"`
	}
	if json.Unmarshal(raw[9:], &r) != nil || len(r.User) > 128 {
		return ""
	}
	return r.User
}

func recordVisible(r Record, boundary uint64) bool {
	if r.Seq <= boundary {
		return false
	}
	if r.Kind == KindSignInRefusedMore {
		return (r.SourceSeq > boundary && r.SourceSeq <= r.Seq) || (r.SourceSeq == 0 && boundary == 0)
	}
	return true
}
