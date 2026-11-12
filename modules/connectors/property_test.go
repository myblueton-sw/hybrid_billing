package connectors

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

type modelPage struct {
	cursor, next, sum string
	rows              int64
}

type modelCollection struct {
	spec       Spec
	state      State
	capability uint64
	declared   bool
	origin     Origin
	manifest   map[string]Part
	verified   map[string]bool
	pages      []modelPage
	total      *[2]int64
	losses     []LossReason
	resumed    int
	abandon    AbandonReason
	retire     RetireReason
	content    string
	rows       int64
}

type modelSubject struct {
	known     uint64
	effective string
	pointer   uint64
	completed int
	members   []string
}

type modelCapability struct {
	stated   Capability
	revision uint64
}

type model struct {
	capabilities map[string]*modelCapability
	collections  map[string]*modelCollection
	subjects     map[Subject]*modelSubject
	bindings     map[string]string
	limit        int
	pages        int
}

func capName(provider, version string, operation Operation, kind Kind, condition Condition) string {
	return fmt.Sprint(provider, "|", version, "|", operation, "|", kind, "|", condition)
}

func (m *model) allowed(c *modelCollection) bool {
	held := m.capabilities[capName(c.spec.Provider, c.spec.Version, c.spec.Subject.Operation, c.spec.Subject.Kind, c.spec.Condition)]
	return held != nil && held.stated.State == Verified && held.revision == c.capability
}

func (m *model) declare(stated Capability) {
	name := capName(stated.Provider, stated.Version, stated.Operation, stated.Kind, stated.Condition)
	held := m.capabilities[name]
	switch {
	case held == nil:
		m.capabilities[name] = &modelCapability{stated, 1}
	case held.stated != stated:
		held.stated = stated
		held.revision++
	}
}

func (m *model) learn(about Subject, revision uint64) {
	topic := m.subjects[about]
	if revision <= topic.known {
		return
	}
	topic.known = revision
	for _, id := range topic.members {
		if member := m.collections[id]; (member.state == Collecting || member.state == Interrupted) && member.spec.Revision < revision {
			member.state = Superseded
		}
	}
}

func (m *model) start(spec Spec) bool {
	if existing := m.collections[spec.ID]; existing != nil {
		return existing.spec == spec
	}
	held := m.capabilities[capName(spec.Provider, spec.Version, spec.Subject.Operation, spec.Subject.Kind, spec.Condition)]
	if held == nil || held.stated.State != Verified {
		return false
	}
	if bound, known := m.bindings[spec.Subject.ConnectionID]; known && bound != spec.Provider {
		return false
	}
	if topic := m.subjects[spec.Subject]; topic != nil && spec.Revision < topic.known {
		return false
	}
	if len(m.collections) >= m.limit {
		return false
	}
	if m.subjects[spec.Subject] == nil {
		m.subjects[spec.Subject] = &modelSubject{}
	}
	m.bindings[spec.Subject.ConnectionID] = spec.Provider
	m.collections[spec.ID] = &modelCollection{spec: spec, state: Collecting, capability: held.revision, manifest: map[string]Part{}, verified: map[string]bool{}}
	m.subjects[spec.Subject].members = append(m.subjects[spec.Subject].members, spec.ID)
	m.learn(spec.Subject, spec.Revision)
	return true
}

func (m *model) manifest(id string, origin Origin, list []Part) bool {
	c := m.collections[id]
	if c == nil || c.spec.Shape != Files {
		return false
	}
	if c.declared {
		same := origin == c.origin && len(list) == len(c.manifest)
		for _, given := range list {
			if c.manifest[given.ID] != given {
				same = false
			}
		}
		if same {
			return true
		}
		if c.state != Collecting && c.state != Interrupted {
			return false
		}
		c.state = Inconsistent
		return true
	}
	if c.state != Collecting {
		return false
	}
	c.declared, c.origin = true, origin
	for _, given := range list {
		c.manifest[given.ID] = given
	}
	return true
}

func (m *model) part(id string, given Part) bool {
	c := m.collections[id]
	if c == nil || c.spec.Shape != Files || !c.declared {
		return false
	}
	stated, listed := c.manifest[given.ID]
	if !listed {
		return false
	}
	if stated == given && c.verified[given.ID] {
		return true
	}
	if c.verified[given.ID] && (c.state == Collecting || c.state == Interrupted) {
		c.state = Inconsistent
		return true
	}
	if c.state != Collecting {
		return false
	}
	if stated == given {
		c.verified[given.ID] = true
	}
	return true
}

func (c *modelCollection) settle() {
	if c.total == nil {
		return
	}
	sum := int64(0)
	for _, received := range c.pages {
		sum += received.rows
	}
	ended := len(c.pages) > 0 && c.pages[len(c.pages)-1].next == ""
	if int64(len(c.pages)) > c.total[0] || (ended && (int64(len(c.pages)) != c.total[0] || sum != c.total[1])) {
		c.state = Inconsistent
	}
}

func (m *model) total(id string, pages, rows int64) bool {
	c := m.collections[id]
	if c == nil || c.spec.Shape != Pages {
		return false
	}
	if c.total != nil && *c.total == [2]int64{pages, rows} {
		return true
	}
	if c.total != nil {
		if c.state != Collecting && c.state != Interrupted {
			return false
		}
		c.state = Inconsistent
		return true
	}
	if c.state != Collecting {
		return false
	}
	c.total = &[2]int64{pages, rows}
	c.settle()
	return true
}

func (m *model) page(id string, given modelPage) bool {
	c := m.collections[id]
	if c == nil || c.spec.Shape != Pages {
		return false
	}
	waiting, ended := c.spec.Cursor, false
	for _, received := range c.pages {
		if received.cursor == given.cursor {
			if received == given {
				return true
			}
			if c.state != Collecting && c.state != Interrupted {
				return false
			}
			c.state = Inconsistent
			return true
		}
		waiting, ended = received.next, received.next == ""
	}
	if c.state != Collecting || ended || given.cursor != waiting || len(c.pages) >= m.pages {
		return false
	}
	loops := given.next == given.cursor || given.next == c.spec.Cursor
	for _, received := range c.pages {
		if received.cursor == given.next || (given.rows > 0 && received.rows > 0 && received.sum == given.sum) {
			loops = true
		}
	}
	if loops {
		c.state = Inconsistent
		return true
	}
	c.pages = append(c.pages, given)
	c.settle()
	return true
}

func (m *model) lost(id string, number int, reason LossReason) bool {
	c := m.collections[id]
	if c == nil {
		return false
	}
	if number <= len(c.losses) {
		return c.losses[number-1] == reason
	}
	if number != len(c.losses)+1 || c.state != Collecting {
		return false
	}
	c.losses, c.state = append(c.losses, reason), Interrupted
	return true
}

func (m *model) resume(id string, number int) bool {
	c := m.collections[id]
	if c == nil || number > len(c.losses) {
		return false
	}
	if number <= c.resumed {
		return true
	}
	if c.state != Interrupted || !m.allowed(c) {
		return false
	}
	c.state, c.resumed = Collecting, number
	return true
}

func (m *model) changed(id string, revision uint64) bool {
	c := m.collections[id]
	if c == nil || revision <= c.spec.Revision {
		return false
	}
	m.learn(c.spec.Subject, revision)
	return true
}

func (m *model) abandon(id string, reason AbandonReason) bool {
	c := m.collections[id]
	if c == nil {
		return false
	}
	if c.state == Abandoned {
		return c.abandon == reason
	}
	if c.state != Collecting && c.state != Interrupted {
		return false
	}
	c.state, c.abandon = Abandoned, reason
	return true
}

func (m *model) complete(id string) bool {
	c := m.collections[id]
	if c == nil {
		return false
	}
	if c.state == Completed || c.state == Retired {
		return true
	}
	if c.state != Collecting || !m.allowed(c) {
		return false
	}
	texts := []string{}
	if c.spec.Shape == Files {
		if !c.declared || len(c.verified) != len(c.manifest) {
			return false
		}
		for name, stated := range c.manifest {
			texts = append(texts, fmt.Sprint(name, stated.Bytes, stated.Rows, stated.Checksum))
			c.rows += stated.Rows
		}
		sort.Strings(texts)
	} else {
		if len(c.pages) == 0 || c.pages[len(c.pages)-1].next != "" {
			return false
		}
		for _, received := range c.pages {
			texts = append(texts, fmt.Sprint(received.rows, received.sum))
			c.rows += received.rows
		}
	}
	c.state, c.content = Completed, strings.Join(texts, ";")
	topic := m.subjects[c.spec.Subject]
	if topic.completed == 0 {
		topic.effective = id
		topic.pointer++
	}
	topic.completed++
	return true
}

func (m *model) adopt(id string, pointer uint64) bool {
	c := m.collections[id]
	if c == nil || c.state != Completed {
		return false
	}
	topic := m.subjects[c.spec.Subject]
	if pointer != topic.pointer {
		return false
	}
	if topic.effective == id {
		return true
	}
	if topic.effective != "" && c.spec.Revision < m.collections[topic.effective].spec.Revision {
		return false
	}
	topic.effective = id
	topic.pointer++
	return true
}

func (m *model) retireIt(id string, reason RetireReason) bool {
	c := m.collections[id]
	if c == nil {
		return false
	}
	if c.state == Retired {
		return c.retire == reason
	}
	if c.state != Completed {
		return false
	}
	c.state, c.retire = Retired, reason
	if topic := m.subjects[c.spec.Subject]; topic.effective == id {
		topic.effective = ""
		topic.pointer++
	}
	return true
}

func (m *model) standings(about Subject) string {
	topic := m.subjects[about]
	if topic == nil {
		return ErrNotFound.Error()
	}
	text := "none"
	if topic.effective != "" {
		current := m.collections[topic.effective]
		text = topic.effective
		if topic.known > current.spec.Revision {
			text += "/stale"
		}
		if current.rows == 0 {
			text += "/empty"
		}
	}
	for _, id := range topic.members {
		member := m.collections[id]
		if id == topic.effective || (member.state != Completed && member.state != Retired) {
			continue
		}
		standing := Candidate
		switch {
		case member.state == Retired:
			standing = IsRetired
		case topic.effective == "":
		case member.spec.Revision > m.collections[topic.effective].spec.Revision:
			standing = Newer
		case member.spec.Revision < m.collections[topic.effective].spec.Revision:
			standing = Older
		case member.content == m.collections[topic.effective].content:
			standing = Duplicate
		default:
			standing = Conflicting
		}
		text += fmt.Sprintf(" %s=%s", id, standing)
	}
	return fmt.Sprintf("%s @%d known %d", text, topic.pointer, topic.known)
}

func TestRandomHistoriesAgreeWithAModelOfTheAttempts(t *testing.T) {
	random := rand.New(rand.NewSource(184))
	limits := Limits{Capabilities: 12, Collections: 14, Parts: 6, Pages: 5, MaxBytes: 1 << 40, MaxRows: 1 << 30, MaxWait: time.Hour}
	reached := map[string]int{}
	final := map[State]bool{Completed: true, Superseded: true, Inconsistent: true, Abandoned: true, Retired: true}
	for round := 0; round < 400; round++ {
		b := open(t, limits)
		m := &model{capabilities: map[string]*modelCapability{}, collections: map[string]*modelCollection{}, subjects: map[Subject]*modelSubject{}, bindings: map[string]string{}, limit: limits.Collections, pages: limits.Pages}
		for _, operation := range []Operation{FetchCosts, FetchInvoices} {
			for _, kind := range []Kind{ProviderOriginal, ProForma} {
				m.declare(verified(operation, kind))
			}
		}
		ids := []string{}
		subjects := []Subject{}
		previous := map[string]State{}
		pointers := map[Subject]string{}
		for step := 0; step < 140; step++ {
			id := fmt.Sprintf("c%d", random.Intn(len(ids)+1))
			if len(ids) > 0 && random.Intn(5) > 0 {
				id = ids[random.Intn(len(ids))]
			}
			running := []string{}
			for _, known := range ids {
				if m.collections[known].state == Collecting {
					running = append(running, known)
				}
			}
			var err error
			accepted, kind := false, ""
			choice := random.Intn(40)
			if len(running) > 0 && random.Intn(2) == 0 {
				id = running[random.Intn(len(running))]
				held := m.collections[id]
				switch {
				case held.spec.Shape == Files && !held.declared:
					choice = 5
				case held.spec.Shape == Files && len(held.verified) < len(held.manifest):
					choice = 8
				case held.spec.Shape == Pages && (len(held.pages) == 0 || held.pages[len(held.pages)-1].next != ""):
					choice = 13
				default:
					choice = 30
				}
			}
			switch {
			case choice < 5:
				kind = "start"
				spec := files(id, uint64(1+random.Intn(4)))
				spec.Subject.ScopeID = []string{"payer-1", "payer-2"}[random.Intn(2)]
				if random.Intn(2) == 0 {
					spec.Shape, spec.Cursor = Pages, "c0"
				}
				if random.Intn(12) == 0 {
					spec.Subject.Kind = ProForma
				}
				err = b.r.Start(spec)
				accepted = m.start(spec)
				if err == nil && m.collections[id] != nil && previous[id] == "" {
					ids = append(ids, id)
					seen := false
					for _, known := range subjects {
						seen = seen || known == spec.Subject
					}
					if !seen {
						subjects = append(subjects, spec.Subject)
					}
				}
			case choice < 8:
				kind = "manifest"
				list := parts(1 + random.Intn(3))
				origin := []Origin{SourceListed, SourceListed, AdapterDerived}[random.Intn(3)]
				if held := m.collections[id]; held != nil && held.declared && random.Intn(4) > 0 {
					list, origin = nil, held.origin
					for _, stated := range held.manifest {
						list = append(list, stated)
					}
				}
				_, err = b.r.DeclareManifest("w1", id, origin, list)
				accepted = m.manifest(id, origin, list)
			case choice < 13:
				kind = "part"
				given := parts(4)[random.Intn(4)]
				if held := m.collections[id]; held != nil && choice == 8 {
					for _, stated := range held.manifest {
						if !held.verified[stated.ID] {
							given = stated
						}
					}
				}
				if random.Intn(9) == 0 {
					given.Rows++
				}
				_, err = b.r.ReceivePart("w1", id, given)
				accepted = m.part(id, given)
			case choice < 20:
				kind = "page"
				given := modelPage{cursor: fmt.Sprintf("c%d", random.Intn(5)), next: fmt.Sprintf("c%d", random.Intn(6)), rows: int64(random.Intn(3)), sum: sum(random.Intn(6))}
				if held := m.collections[id]; held != nil && choice == 13 {
					given.cursor = held.spec.Cursor
					if len(held.pages) > 0 {
						given.cursor = held.pages[len(held.pages)-1].next
					}
					given.next, given.sum, given.rows = fmt.Sprintf("c%d", len(held.pages)+1), sum(100+len(held.pages)), int64(1+random.Intn(3))
					if random.Intn(3) == 0 || len(held.pages) >= 3 {
						given.next = ""
					}
				}
				if random.Intn(8) == 0 {
					given.next = ""
				}
				_, err = b.r.ReceivePage("w1", id, given.cursor, given.next, given.rows, given.sum)
				accepted = m.page(id, given)
			case choice < 22:
				kind = "total"
				pages, rows := int64(1+random.Intn(3)), int64(random.Intn(6))
				_, err = b.r.DeclareTotal("w1", id, int(pages), rows)
				accepted = m.total(id, pages, rows)
			case choice < 24:
				kind = "lost"
				number := 1 + random.Intn(2)
				if held := m.collections[id]; held != nil && random.Intn(2) == 0 {
					number = len(held.losses) + 1
				}
				reason := []LossReason{PermissionWithdrawn, CredentialExpired}[random.Intn(2)]
				err = b.r.AccessLost("w1", id, uint64(number), reason)
				accepted = m.lost(id, number, reason)
			case choice < 26:
				kind = "resume"
				number := 1 + random.Intn(2)
				if held := m.collections[id]; held != nil && len(held.losses) > 0 && random.Intn(3) > 0 {
					number = len(held.losses)
				}
				err = b.r.Resume("w1", id, uint64(number))
				accepted = m.resume(id, number)
			case choice < 27:
				kind = "changed"
				revision := uint64(1 + random.Intn(6))
				err = b.r.SourceChanged("w1", id, revision)
				accepted = m.changed(id, revision)
			case choice < 28:
				kind = "abandon"
				reason := []AbandonReason{ByOperator, CapabilityChanged}[random.Intn(2)]
				err = b.r.Abandon("w1", id, reason)
				accepted = m.abandon(id, reason)
			case choice < 34:
				kind = "complete"
				_, err = b.r.Complete("w1", id)
				accepted = m.complete(id)
			case choice < 37:
				kind = "adopt"
				pointer := uint64(random.Intn(4))
				if held := m.collections[id]; held != nil && random.Intn(4) > 0 {
					pointer = m.subjects[held.spec.Subject].pointer
				}
				err = b.r.Adopt("w1", id, pointer)
				accepted = m.adopt(id, pointer)
			case choice < 38:
				kind = "retire"
				reason := []RetireReason{RetireConflict, WrongSource}[random.Intn(2)]
				err = b.r.Retire("w1", id, reason)
				accepted = m.retireIt(id, reason)
			case choice < 39:
				kind = "declare"
				stated := verified(FetchCosts, ProviderOriginal)
				switch random.Intn(3) {
				case 0:
					stated.Evidence = fmt.Sprintf("evidence-%d", random.Intn(3))
				case 1:
					stated.State, stated.Evidence, stated.Reason = Unsupported, "", "parser-wrong"
				}
				_, err = b.r.Declare(stated)
				m.declare(stated)
				accepted = true
			default:
				kind = "throttled"
				err = b.r.Throttled("w1", id, time.Second)
				accepted = m.collections[id] != nil && m.collections[id].state == Collecting
			}
			if (err == nil) != accepted {
				t.Fatalf("round %d step %d: %s on %s ended with %v; the model says accepted=%v", round, step, kind, id, err, accepted)
			}
			reached[fmt.Sprintf("%s %v", kind, accepted)]++
			for _, known := range ids {
				view := b.view(known)
				held := m.collections[known]
				if view.State != held.state {
					t.Fatalf("round %d step %d after %s on %s: %s is %s; the model says %s", round, step, kind, id, known, view.State, held.state)
				}
				if before := previous[known]; final[before] && view.State != before && !(before == Completed && view.State == Retired) {
					t.Fatalf("round %d step %d: %s left the final state %s for %s", round, step, known, before, view.State)
				}
				if view.State == Completed && previous[known] != Completed {
					reached["completed "+string(view.Spec.Shape)]++
					if view.Totals.Rows != held.rows {
						t.Fatalf("round %d step %d: %s completed with %d rows; the model counts %d", round, step, known, view.Totals.Rows, held.rows)
					}
				}
				if view.State == Inconsistent && previous[known] == Interrupted {
					reached["inconsistent while interrupted"]++
				}
				if view.State != previous[known] && final[view.State] {
					reached["became "+string(view.State)]++
				}
				previous[known] = view.State
			}
			for _, about := range subjects {
				got, want := b.standings(about), m.standings(about)
				if got != want || got != b.standings(about) {
					t.Fatalf("round %d step %d after %s on %s:\n got %s\nwant %s", round, step, kind, id, got, want)
				}
				view, _ := b.r.Effective(about)
				now := "none"
				if view.Collection != nil {
					now = view.Collection.Spec.ID
				}
				if before, seen := pointers[about]; seen && before != now && !(err == nil && (kind == "complete" || kind == "adopt" || kind == "retire")) {
					t.Fatalf("round %d step %d: what counts for %v moved from %s to %s by %s", round, step, about, before, now, kind)
				}
				if before, seen := pointers[about]; seen && before != now {
					reached["pointer moved by "+kind]++
				}
				pointers[about] = now
				for _, known := range ids {
					if member := b.view(known); member.Spec.Subject == about && member.State.running() && member.Spec.Revision < view.Known {
						t.Fatalf("round %d step %d: %s runs at revision %d below the known %d", round, step, known, member.Spec.Revision, view.Known)
					}
				}
			}
		}
	}
	for _, name := range []string{"start true", "start false", "manifest true", "part true", "part false", "page true", "page false", "total true", "lost true", "resume true", "changed true", "abandon true",
		"complete true", "complete false", "adopt true", "adopt false", "retire true", "completed files", "completed pages", "became superseded", "became inconsistent", "became abandoned", "became retired",
		"pointer moved by complete", "pointer moved by adopt", "pointer moved by retire",
		"manifest false", "total false", "lost false", "resume false", "changed false", "abandon false", "retire false"} {
		if reached[name] < 20 {
			t.Errorf("%s was reached %d times", name, reached[name])
		}
	}
	if reached["inconsistent while interrupted"] < 5 {
		t.Errorf("a contradiction while interrupted was reached %d times", reached["inconsistent while interrupted"])
	}
	t.Logf("%v", reached)
}
