package onboarding

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("request is incomplete or malformed")
	ErrNotFound = errors.New("project or run does not exist in this workspace")
	ErrConflict = errors.New("request conflicts with what is stored")
	ErrRejected = errors.New("the state of the project does not allow this step")
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

type Path string

const (
	ExistingAccount Path = "existing_account"
	ParentOperator  Path = "parent_operator"
	FileImport      Path = "file_import"
	NoAlternative   Path = "none"
)

func (p Path) known() bool { return p == ExistingAccount || p == ParentOperator || p == FileImport }

func (p Path) connected() bool { return p == ExistingAccount || p == ParentOperator }

type Stage string

const (
	Authenticated    Stage = "authenticated"
	OperationalRead  Stage = "operational_read"
	FirstData        Stage = "first_data"
	FullCoverage     Stage = "full_coverage"
	FocusConformance Stage = "focus_conformance"
	Reconciled       Stage = "reconciled"
	BillingOpen      Stage = "billing_open"
)

var order = []Stage{Authenticated, OperationalRead, FirstData, FullCoverage, FocusConformance, Reconciled, BillingOpen}

func Required(path Path) []Stage {
	switch path {
	case ExistingAccount, ParentOperator:
		return append([]Stage(nil), order...)
	case FileImport:
		return append([]Stage(nil), order[2:]...)
	}
	return nil
}

func position(stages []Stage, stage Stage) int {
	for index, candidate := range stages {
		if candidate == stage {
			return index
		}
	}
	return -1
}

type Outcome string

const (
	Passed                 Outcome = "passed"
	PassedNoUsage          Outcome = "passed_no_usage"
	InsufficientPermission Outcome = "insufficient_permission"
	AwaitingFirstExport    Outcome = "awaiting_first_export"
	PeriodUnsupported      Outcome = "period_unsupported"
	PartialFile            Outcome = "partial_file"
	UnsupportedScope       Outcome = "unsupported_scope"
	Failed                 Outcome = "failed"
)

func (o Outcome) reaches() bool { return o == Passed || o == PassedNoUsage }

func allowed(outcome Outcome, stage Stage, path Path) bool {
	switch outcome {
	case Passed, Failed:
		return true
	case PassedNoUsage:
		return stage == FirstData || stage == FullCoverage
	case InsufficientPermission:
		return path.connected()
	case AwaitingFirstExport:
		return path.connected() && (stage == FirstData || stage == FullCoverage)
	case PeriodUnsupported:
		return stage == FirstData || stage == FullCoverage || stage == Reconciled
	case PartialFile:
		return stage == FirstData || stage == FullCoverage || stage == FocusConformance
	case UnsupportedScope:
		return stage == Authenticated || stage == OperationalRead || stage == FirstData
	}
	return false
}

type Role string

const (
	Setup           Role = "setup"
	ExportWriter    Role = "export_writer"
	CollectorReader Role = "collector_reader"
)

func (r Role) known() bool { return r == Setup || r == ExportWriter || r == CollectorReader }

type Condition string

const (
	AfterSetupComplete Condition = "after_setup_complete"
	OnExpiry           Condition = "on_expiry"
	OnRequest          Condition = "on_request"
)

func (c Condition) known() bool { return c == AfterSetupComplete || c == OnExpiry || c == OnRequest }

type Run struct {
	Stage       Stage
	Outcome     Outcome
	Role        Role
	Validity    time.Duration
	Version     string
	Detail      string
	Alternative Path
	Code        string
}

type RunView struct {
	ID         string
	Run        Run
	Sequence   uint64
	Generation uint64
	Grant      uint64
	UnderSetup bool
	At         time.Time
	Until      time.Time
}

type Status string

const (
	Missing Status = "missing"
	Holding Status = "holding"
	Stale   Status = "stale"
)

type Reason string

const (
	ConfigChanged     Reason = "config_changed"
	UpstreamRenewed   Reason = "upstream_renewed"
	RoleEnded         Reason = "role_ended"
	Expired           Reason = "expired"
	SetupGranted      Reason = "setup_granted"
	CapacityExhausted Reason = "capacity_exhausted"
	EvidenceLapsed    Reason = "evidence_lapsed"
	Empty             Reason = "empty"
	NotReached        Reason = "missing"
	SetupActive       Reason = "setup_active"
	ReadUnderSetup    Reason = "operational_read_under_setup"
)

type StageView struct {
	Stage         Stage
	Status        Status
	Reason        Reason
	Empty         bool
	Prerequisites bool
	Run           string
}

type RoleView struct {
	Role      Role
	Grant     uint64
	Holder    string
	Active    bool
	At        time.Time
	Expiry    time.Time
	Condition Condition
}

type Blocker struct {
	Stage  Stage
	Reason Reason
}

type Readiness struct {
	WorkspaceID string
	ProjectID   string
	Path        Path
	Source      string
	Hash        string
	Generation  uint64
	Sequence    uint64
	Stages      []StageView
	Progress    int
	Required    int
	BillingOpen bool
	Blockers    []Blocker
	Roles       []RoleView
	AsOf        time.Time
}

type Result struct {
	Sequence uint64
	Replay   bool
}

type Limits struct {
	Projects    int
	Writes      int
	Reserve     int
	MaxValidity time.Duration
	MaxSetup    time.Duration
}

func (l Limits) valid() bool {
	return l.Projects > 0 && l.Writes > 0 && l.Reserve > 0 && l.MaxValidity > 0 && l.MaxSetup > 0
}
