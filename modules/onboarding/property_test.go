package onboarding

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"
)

type modelRun struct {
	stage      Stage
	outcome    Outcome
	role       Role
	grant      int
	underSetup bool
	generation int
	sequence   int
	until      time.Time
}

type modelGrant struct {
	live   bool
	number int
	expiry time.Time
}

type model struct {
	connected bool
	stages    []Stage
	sequence  int
	numbers   int
	gen       int
	now       time.Time
	runs      map[string]modelRun
	current   map[Stage]string
	spoiled   map[Stage]Reason
	floor     map[Stage]int
	roles     map[Role]*modelGrant
	ended     map[int]bool
	opened    bool
	closed    Reason
	writes    int
	reserve   int
	longest   time.Duration
}

var permitted = map[Outcome]string{Passed: "auth read data cover focus recon", PassedNoUsage: "data cover", InsufficientPermission: "auth read data cover focus recon/connected",
	AwaitingFirstExport: "data cover/connected", PeriodUnsupported: "data cover recon", PartialFile: "data cover focus", UnsupportedScope: "auth read data", Failed: "auth read data cover focus recon"}

func (m *model) roleLive(role Role) bool {
	held := m.roles[role]
	return held != nil && held.live && (held.expiry.IsZero() || m.now.Before(held.expiry))
}

func (m *model) grantLive(number int) bool {
	for _, held := range m.roles {
		if held != nil && held.number == number {
			return held.live && (held.expiry.IsZero() || m.now.Before(held.expiry))
		}
	}
	return false
}

func (m *model) status(stage Stage) (Status, Reason) {
	name, reached := m.current[stage]
	if !reached {
		return Missing, ""
	}
	evidence := m.runs[name]
	switch {
	case m.spoiled[stage] != "":
		return Stale, m.spoiled[stage]
	case m.connected && (m.ended[evidence.grant] || !m.grantLive(evidence.grant)):
		return Stale, RoleEnded
	case !m.now.Before(evidence.until):
		return Stale, Expired
	}
	return Holding, ""
}

func (m *model) blank(stage Stage) bool {
	name, reached := m.current[stage]
	return reached && m.runs[name].outcome == PassedNoUsage
}

func (m *model) billable() bool {
	for _, stage := range m.stages {
		if stage == BillingOpen {
			continue
		}
		if status, _ := m.status(stage); status != Holding || m.blank(stage) {
			return false
		}
	}
	if m.connected && (m.roleLive(Setup) || m.runs[m.current[OperationalRead]].underSetup) {
		return false
	}
	return m.sequence < m.writes+m.reserve
}

func (m *model) guard(reason Reason) {
	if m.opened && m.closed == "" && !m.billable() {
		m.closed = reason
	}
}

func (m *model) admit(expected int, checked, reserved bool) bool {
	if checked && expected != m.sequence {
		return false
	}
	room := m.writes
	if reserved {
		room += m.reserve
	}
	return m.sequence < room
}

func (m *model) done(after Reason) {
	m.sequence++
	if m.sequence >= m.writes+m.reserve {
		after = CapacityExhausted
	}
	m.guard(after)
}

func (m *model) spoil(from int, reason Reason, floors bool) {
	for _, later := range m.stages[from:] {
		if floors {
			m.floor[later] = m.sequence + 1
		}
		if _, reached := m.current[later]; reached && m.spoiled[later] == "" {
			m.spoiled[later] = reason
		}
	}
	if m.opened && m.closed == "" {
		m.closed = reason
	}
}

func (m *model) record(expected int, name string, given Run) bool {
	rule, connectedOnly := strings.CutSuffix(permitted[given.Outcome], "/connected")
	if !m.admit(expected, true, false) || position(m.stages, given.Stage) < 0 || !strings.Contains(" "+rule+" ", " "+short[given.Stage]+" ") || (connectedOnly && !m.connected) {
		return false
	}
	if given.Validity > m.longest || (m.connected && !m.roleLive(given.Role)) {
		return false
	}
	made := modelRun{stage: given.Stage, outcome: given.Outcome, role: given.Role, generation: m.gen, sequence: m.sequence + 1, until: m.now.Add(given.Validity)}
	if m.connected {
		made.grant, made.underSetup = m.roles[given.Role].number, m.roleLive(Setup)
	}
	m.guard(EvidenceLapsed)
	m.runs[name] = made
	m.done(EvidenceLapsed)
	return true
}

func (m *model) reach(expected int, stage Stage, name string) bool {
	evidence, known := m.runs[name]
	place := position(m.stages, stage)
	if !m.admit(expected, true, false) || !known || place < 0 || evidence.stage != stage || (evidence.outcome != Passed && evidence.outcome != PassedNoUsage) ||
		evidence.generation != m.gen || !m.now.Before(evidence.until) || evidence.sequence <= m.floor[stage] {
		return false
	}
	if m.connected && (evidence.role != CollectorReader || m.ended[evidence.grant] || !m.grantLive(evidence.grant)) {
		return false
	}
	if old, reached := m.current[stage]; reached && m.runs[old].sequence >= evidence.sequence {
		return false
	}
	blank := evidence.outcome == PassedNoUsage
	if stage == FullCoverage {
		first, _ := m.status(FirstData)
		if (first == Holding && m.blank(FirstData) != blank) || (first != Holding && blank) {
			return false
		}
	}
	m.guard(EvidenceLapsed)
	before, _ := m.status(stage)
	renewal := before == Holding && m.blank(stage) == blank
	m.current[stage] = name
	delete(m.spoiled, stage)
	if !renewal {
		m.spoil(place+1, UpstreamRenewed, true)
	}
	m.done(EvidenceLapsed)
	return true
}

func (m *model) change(from Stage) bool {
	place := position(m.stages, from)
	if !m.admit(0, false, true) || place < 0 {
		return false
	}
	m.guard(EvidenceLapsed)
	m.gen++
	m.spoil(place, ConfigChanged, true)
	m.done(EvidenceLapsed)
	return true
}

func (m *model) grant(expected int, role Role, duration time.Duration) bool {
	if !m.admit(expected, true, false) || !m.connected || m.roleLive(role) {
		return false
	}
	m.guard(EvidenceLapsed)
	if old := m.roles[role]; old != nil {
		m.ended[old.number] = true
	}
	m.numbers++
	made := &modelGrant{live: true, number: m.numbers}
	after := EvidenceLapsed
	if role == Setup {
		made.expiry, after = m.now.Add(duration), SetupGranted
	}
	m.roles[role] = made
	m.done(after)
	return true
}

func (m *model) withdraw(role Role) bool {
	if !m.admit(0, false, true) || !m.roleLive(role) {
		return false
	}
	m.guard(EvidenceLapsed)
	m.roles[role].live = false
	m.ended[m.roles[role].number] = true
	m.done(EvidenceLapsed)
	return true
}

func (m *model) open(expected int) bool {
	if !m.admit(expected, true, false) || !m.billable() || (m.opened && m.closed == "") {
		return false
	}
	m.opened, m.closed = true, ""
	m.done(EvidenceLapsed)
	return true
}

func (m *model) shape() string {
	parts := []string{}
	progress, unbroken, alone := 0, true, false
	isOpen := m.opened && m.closed == "" && m.billable()
	for _, stage := range m.stages {
		status, reason := m.status(stage)
		if stage == BillingOpen {
			switch {
			case !m.opened:
				status, reason = Missing, ""
			case isOpen:
				status, reason = Holding, ""
			case m.closed != "":
				status, reason = Stale, m.closed
			default:
				status, reason = Stale, EvidenceLapsed
			}
		}
		text := short[stage] + "=" + string(status)
		if reason != "" {
			text += "/" + string(reason)
		}
		if m.blank(stage) {
			text += "/empty"
		}
		if alone {
			text += "/alone"
		}
		if unbroken && status == Holding {
			progress++
		} else {
			unbroken = false
		}
		if held, _ := m.status(stage); stage != BillingOpen && held != Holding {
			alone = true
		}
		parts = append(parts, text)
	}
	state := "closed"
	if isOpen {
		state = "open"
	}
	return fmt.Sprintf("%s | %d/%d %s", strings.Join(parts, " "), progress, len(m.stages), state)
}

func TestRandomHistoriesAgreeWithAModelOfTheAttempts(t *testing.T) {
	random := rand.New(rand.NewSource(182))
	reached := map[string]int{}
	for round := 0; round < 400; round++ {
		limits := Limits{Projects: 1, Writes: 70, Reserve: 4, MaxValidity: 30 * time.Hour, MaxSetup: 6 * time.Hour}
		if round%4 == 0 {
			limits.Writes, limits.Reserve = 24+random.Intn(20), 1+random.Intn(3)
		}
		path := []Path{ExistingAccount, ParentOperator, FileImport}[random.Intn(3)]
		d := open(t, limits, path)
		m := &model{connected: path != FileImport, stages: Required(path), gen: 1, now: start, runs: map[string]modelRun{}, current: map[Stage]string{}, spoiled: map[Stage]Reason{},
			floor: map[Stage]int{}, roles: map[Role]*modelGrant{}, ended: map[int]bool{}, writes: limits.Writes, reserve: limits.Reserve, longest: limits.MaxValidity}
		names := []string{}
		floors := map[Stage]uint64{}
		wasOpen := false
		lastRun := map[Stage]string{}
		holding, blankBefore := map[Stage]bool{}, map[Stage]bool{}
		highest := start
		for step := 0; step < 110; step++ {
			sequence := d.sequence()
			expected := sequence
			if random.Intn(25) == 0 {
				expected += uint64(1 + random.Intn(2))
			}
			stage := order[random.Intn(7)]
			role := []Role{CollectorReader, CollectorReader, CollectorReader, Setup, ExportWriter}[random.Intn(5)]
			var err error
			accepted, kind := false, ""
			touched := Stage("")
			choice := random.Intn(40)
			steer := Stage("")
			if random.Intn(2) == 0 {
				switch {
				case m.connected && !m.roleLive(CollectorReader):
					choice, role = 24, CollectorReader
				case m.connected && m.roleLive(Setup) && random.Intn(3) == 0:
					choice, role = 27, Setup
				default:
					choice = 31
					for _, wanted := range m.stages {
						if status, _ := m.status(wanted); wanted != BillingOpen && status != Holding {
							steer, choice = wanted, 0
							if name, known := lastRun[wanted]; known && m.runs[name].sequence > m.floor[wanted] && m.runs[name].generation == m.gen && m.now.Before(m.runs[name].until) && m.current[wanted] != name {
								choice = 12
							}
							break
						}
					}
				}
			}
			if steer != "" {
				stage, role = steer, CollectorReader
			}
			switch {
			case choice < 12:
				kind = "run"
				outcome := []Outcome{Passed, Passed, Passed, Passed, Passed, PassedNoUsage, InsufficientPermission, AwaitingFirstExport, PeriodUnsupported, PartialFile}[random.Intn(10)]
				if steer != "" && random.Intn(8) > 0 {
					outcome = Passed
				}
				given := Run{Stage: stage, Outcome: outcome, Validity: time.Duration(1+random.Intn(32)) * time.Hour, Version: "v1", Detail: "d"}
				if path != FileImport {
					given.Role = role
				}
				name := fmt.Sprintf("run-%d", step)
				_, err = d.r.RecordRun("w1", "p1", expected, name, given)
				if stage != BillingOpen {
					accepted = m.record(int(expected), name, given)
				}
				if err == nil {
					names = append(names, name)
					lastRun[stage] = name
				}
			case choice < 24 && len(names) > 0:
				kind = "reach"
				name := names[len(names)-1-random.Intn(min(len(names), 6))]
				if steer != "" {
					name = lastRun[steer]
				} else if random.Intn(3) > 0 {
					for _, stored := range names[max(0, len(names)-8):] {
						if m.runs[stored].stage == stage {
							name = stored
						}
					}
				}
				_, err = d.r.Reach("w1", "p1", expected, d.next(), stage, name)
				if stage != BillingOpen {
					accepted = m.reach(int(expected), stage, name)
				}
				touched = stage
				if err == nil {
					view, _ := d.r.Readiness("w1", "p1")
					for index, shown := range view.Stages {
						if shown.Stage == stage && !(holding[stage] && shown.Empty == blankBefore[stage]) {
							for _, after := range view.Stages[index+1:] {
								floors[after.Stage] = view.Sequence
							}
						}
					}
				}
			case choice < 27:
				kind = "grant"
				duration, condition := time.Duration(0), Condition("")
				if role == Setup {
					duration, condition = time.Duration(1+random.Intn(6))*time.Hour, OnExpiry
				}
				_, err = d.r.GrantRole("w1", "p1", expected, d.next(), role, "holder", duration, condition)
				accepted = m.grant(int(expected), role, duration)
			case choice < 29:
				kind = "withdraw"
				_, err = d.r.WithdrawRole("w1", "p1", d.next(), role)
				accepted = m.withdraw(role)
			case choice < 31:
				kind = "change"
				_, err = d.r.ChangeConfig("w1", "p1", d.next(), fmt.Sprintf("hash-%d", random.Intn(3)), stage)
				accepted = m.change(stage)
				if err == nil {
					for _, later := range m.stages[position(m.stages, stage):] {
						floors[later] = sequence + 1
					}
				}
			case choice < 35:
				kind = "open"
				_, _, err = d.r.OpenBilling("w1", "p1", expected, d.next())
				accepted = m.open(int(expected))
			case choice < 38:
				kind = "clock"
				moved := time.Duration(random.Intn(600)-60) * time.Minute
				d.pass(moved)
				if d.now.After(highest) {
					highest = *d.now
				}
				accepted = true
			default:
				kind = "read"
				accepted = true
			}
			m.now = highest
			if (err == nil) != accepted {
				t.Fatalf("round %d step %d: %s ended with %v; the model says accepted=%v\n%s", round, step, kind, err, accepted, d.shape())
			}
			reached[fmt.Sprintf("%s %v", kind, accepted)]++
			view, _ := d.r.Readiness("w1", "p1")
			again, _ := d.r.Readiness("w1", "p1")
			if again.Sequence != view.Sequence || (kind == "read" || kind == "clock" || err != nil) && view.Sequence != sequence {
				t.Fatalf("round %d step %d: a read or a refused %s changed the sequence", round, step, kind)
			}
			if got, want := d.shape(), m.shape(); got != want {
				t.Fatalf("round %d step %d after %s (%v):\n got %s\nwant %s", round, step, kind, err, got, want)
			}
			if view.BillingOpen && !wasOpen && !(kind == "open" && err == nil) {
				t.Fatalf("round %d step %d: billing opened by %s", round, step, kind)
			}
			if kind == "open" && err == nil && !view.BillingOpen {
				t.Fatalf("round %d step %d: billing does not read open right after it was opened", round, step)
			}
			runs, _ := d.r.Runs("w1", "p1")
			byName := map[string]RunView{}
			for _, stored := range runs {
				byName[stored.ID] = stored
			}
			setupLive := false
			live := map[uint64]bool{}
			for _, held := range view.Roles {
				live[held.Grant] = held.Active
				if held.Role == Setup && held.Active {
					setupLive = true
				}
			}
			for _, shown := range view.Stages {
				if shown.Stage == BillingOpen {
					continue
				}
				now := shown.Status == Holding
				if now && !holding[shown.Stage] && !(touched == shown.Stage && err == nil) {
					t.Fatalf("round %d step %d: %s holds again without being reached", round, step, shown.Stage)
				}
				holding[shown.Stage] = now
				blankBefore[shown.Stage] = shown.Empty
				evidence := byName[shown.Run]
				if now && (evidence.Sequence <= floors[shown.Stage] || !view.AsOf.Before(evidence.Until)) {
					t.Fatalf("round %d step %d: %s holds on the run %s of sequence %d with the floor %d", round, step, shown.Stage, shown.Run, evidence.Sequence, floors[shown.Stage])
				}
				if view.BillingOpen && path != FileImport && !live[evidence.Grant] {
					t.Fatalf("round %d step %d: billing is open while %s rests on the grant %d, which is not active", round, step, shown.Stage, evidence.Grant)
				}
				if view.BillingOpen && (!now || shown.Empty || (shown.Stage == OperationalRead && evidence.UnderSetup)) {
					t.Fatalf("round %d step %d: billing is open while %s is %s (empty %v, under setup %v)", round, step, shown.Stage, shown.Status, shown.Empty, evidence.UnderSetup)
				}
			}
			if view.BillingOpen && (setupLive || view.Sequence >= uint64(limits.Writes+limits.Reserve)) {
				t.Fatalf("round %d step %d: billing is open with setup active or the reserve used up", round, step)
			}
			if view.BillingOpen {
				reached["open reads"]++
			}
			if view.Sequence >= uint64(limits.Writes+limits.Reserve) {
				reached["reserve used up"]++
			}
			if last := view.Stages[len(view.Stages)-1]; last.Reason == CapacityExhausted && wasOpen {
				reached["closed by the reserve"]++
			}
			if wasOpen && !view.BillingOpen {
				reached["opened then closed"]++
			}
			wasOpen = view.BillingOpen
		}
	}
	for _, name := range []string{"run true", "run false", "reach true", "reach false", "grant true", "grant false", "withdraw true", "withdraw false", "change true", "open true", "open false", "open reads", "opened then closed", "reserve used up"} {
		if reached[name] < 20 {
			t.Errorf("%s was reached %d times", name, reached[name])
		}
	}
	t.Logf("%v", reached)
}
