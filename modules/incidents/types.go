package incidents

import (
	"errors"
	"time"
)

var (
	ErrInvalid  = errors.New("a request or an input is incomplete or malformed")
	ErrNotFound = errors.New("what the request names does not exist in this workspace")
	ErrConflict = errors.New("another fact is already recorded under this identifier")
	ErrRejected = errors.New("a recorded state or rule refuses this request")
	ErrLimit    = errors.New("a configured limit or bound is exceeded")
)

const (
	maxIDBytes       = 128
	maxWorkspaces    = 100000
	maxIncidents     = 1000000
	maxSignals       = 10000000
	maxRecipients    = 100000
	maxNotes         = 10000000
	maxSubscriptions = 1000000
	maxSnooze        = 30 * 24 * time.Hour
	minRenotify      = 5 * time.Minute
	maxRenotify      = 90 * 24 * time.Hour
	minutesInDay     = 24 * 60
	hour             = time.Hour
	day              = 24 * time.Hour
	defaultZoneName  = "UTC"

	KindHuman     = "human"
	KindAssistant = "assistant"
	KindSystem    = "system"

	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"

	ReasonFixed      = "fixed"
	ReasonFalseAlarm = "false_alarm"
	ReasonSuperseded = "superseded"
	ReasonOther      = "other"

	DigestImmediate = "immediate"
	DigestHourly    = "hourly"
	DigestDaily     = "daily"

	WithheldRefused     = "refused"
	WithheldUnavailable = "unavailable"
	WithheldNoTarget    = "no_target"
	WithheldClosed      = "closed"
	WithheldSnoozed     = "snoozed"

	RefusalAINotAllowed      = "ai_not_allowed"
	RefusalNotHuman          = "not_human"
	RefusalNotSystem         = "not_system"
	RefusalFutureOccurrence  = "future_occurrence"
	RefusalNotPermitted      = "not_permitted"
	RefusalPermissionUnavail = "permission_unavailable"
	RefusalPermissionRevoked = "permission_revoked"
	RefusalResolved          = "resolved"
	RefusalNotSelf           = "not_self"
	RefusalNotSubscribed     = "not_subscribed"
	RefusalSnoozeOutOfRange  = "snooze_out_of_range"
	RefusalNotTarget         = "not_target"
	RefusalZoneUnavailable   = "zone_unavailable"
	RefusalAlternateIsSelf   = "alternate_is_self"
)

type Refusal struct {
	Reason string
}

func (e Refusal) Error() string { return ErrRejected.Error() + ": " + e.Reason }

func (e Refusal) Unwrap() error { return ErrRejected }

type Actor struct {
	ID   string
	Kind string
}

type Permissions interface {
	Allowed(workspace, person, subject string) (bool, error)
}

type Config struct {
	Permissions Permissions
	Zones       func(string) (*time.Location, error)
}

type Limits struct {
	Workspaces    int
	Incidents     int
	Signals       int
	Recipients    int
	Subscriptions int
	Notifications int
}

type Signal struct {
	ID        string
	Condition string
	Subject   string
	Severity  string
	Blocking  bool
	Occurred  time.Time
}

type Resolution struct {
	Reason   string
	Evidence string
}

type Incident struct {
	ID         string
	Condition  string
	Subject    string
	Severity   string
	Blocking   bool
	First      time.Time
	Last       time.Time
	Count      int
	Late       int
	Recurs     string
	Resolved   bool
	Resolution Resolution
	Resolver   string
	ResolvedAt time.Time
	Opened     time.Time
}

type Subscription struct {
	Recipient string
	Condition string
	Minimum   string
}

type Quiet struct {
	Start int
	End   int
}

type Settings struct {
	Recipient string
	Zone      string
	Quiet     *Quiet
	Digest    string
	Renotify  map[string]time.Duration
	Alternate string
	Away      bool
}

type Notification struct {
	ID        string
	Target    string
	Incidents []string
	Digest    bool
	Planned   time.Time
	HandedOff bool
	Handoff   time.Time
	Delivered []string
}

type Withholding struct {
	Target   string
	Incident string
	Reason   string
	At       time.Time
}
