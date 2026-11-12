package identity

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
	"time"
)

type mUser struct {
	service bool
	state   string
	epoch   uint64
	block   bool
}

type mMember struct {
	admin   bool
	state   string
	version uint64
}

type mSession struct {
	verifier Verifier
	user     string
	epoch    uint64
	subject  string
	opened   time.Time
	idle     time.Duration
	idleEnd  time.Time
	end      time.Time
	gone     bool
}

type mCredential struct {
	id       string
	user     string
	from, to time.Time
	revoked  bool
}

type mInvitation struct {
	id             string
	inviter        string
	inviterEpoch   uint64
	inviterVersion uint64
	invitee        string
	admin          bool
	to             time.Time
	state          string
	attempts       int
}

type mAnswer struct {
	answer     Answer
	user       string
	epoch      uint64
	admin      bool
	version    uint64
	subject    string
	session    *mSession
	credential *mCredential
}

type model struct {
	users       map[string]*mUser
	members     map[string]*mMember
	subjects    map[string]string
	linked      map[string]int
	sessions    []*mSession
	credentials []*mCredential
	invitations []*mInvitation
	answers     []mAnswer
	order       int
}

func (m *model) human(id string) bool {
	found := m.users[id]
	return found != nil && !found.service && found.state == "active"
}

func (m *model) current(s *mSession, now time.Time) bool {
	return !s.gone && !now.Before(s.opened) && now.Before(s.idleEnd) && now.Before(s.end) && m.linked[s.subject] > 0 &&
		m.subjects[s.subject] == s.user && m.human(s.user) && m.users[s.user].epoch == s.epoch
}

func (m *model) admin(id string) bool {
	found := m.members[id]
	return found != nil && found.admin && found.state == "active" && m.human(id)
}

func (m *model) admins() int {
	count := 0
	for id := range m.members {
		if m.admin(id) {
			count++
		}
	}
	return count
}

func (m *model) records() bool {
	for _, found := range m.members {
		if found.state != "removed" {
			return true
		}
	}
	for _, found := range m.users {
		if found.service && found.state != "departed" {
			return true
		}
	}
	return false
}

func (m *model) clone() *model {
	next := &model{users: map[string]*mUser{}, members: map[string]*mMember{}, subjects: m.subjects, linked: m.linked}
	for id, found := range m.users {
		copied := *found
		next.users[id] = &copied
	}
	for id, found := range m.members {
		copied := *found
		next.members[id] = &copied
	}
	return next
}

func (m *model) strands(change func(*model)) bool {
	after := m.clone()
	change(after)
	return m.admins() >= 1 && after.admins() == 0 && after.records()
}

func (m *model) seats(except string) int {
	count := 0
	for id, found := range m.members {
		if id != except && found.admin && found.state == "active" && m.users[id].state != "departed" {
			count++
		}
	}
	return count
}

func (m *model) forget(user string) {
	for _, held := range m.sessions {
		if held.user == user {
			held.gone = true
		}
	}
}

func (m *model) inviterStands(found *mInvitation) bool {
	own := m.members[found.inviter]
	return m.admin(found.inviter) && m.users[found.inviter].epoch == found.inviterEpoch && own.version == found.inviterVersion
}

func TestOperationsAgreeWithAReferenceModel(t *testing.T) {
	limits := testLimits
	limits.PendingInvitations, limits.CredentialsPerServiceUser, limits.PageSize = 100000, 100000, 100
	r := rand.New(rand.NewSource(172))
	reached := map[string]int{}
	for world := 0; world < 120; world++ {
		f := setupWith(t, limits)
		m := &model{users: map[string]*mUser{"root": {state: "active", epoch: 1}}, members: map[string]*mMember{"root": {admin: true, state: "active", version: 1}},
			subjects: map[string]string{"root": "root"}, linked: map[string]int{"root": 1}, order: 1}
		humans, services := []string{"root"}, []string{}
		orders := map[string]int{"root": 1}
		pick := func(list []string) string {
			if len(list) == 0 {
				return "nobody"
			}
			return list[r.Intn(len(list))]
		}
		session := func() *mSession {
			if len(m.sessions) == 0 {
				return &mSession{verifier: v("no-session")}
			}
			if r.Intn(10) < 7 {
				for attempt := 0; attempt < 12; attempt++ {
					candidate := m.sessions[len(m.sessions)-1-r.Intn(min(len(m.sessions), 12))]
					if m.current(candidate, *f.now) && m.admin(candidate.user) {
						return candidate
					}
				}
			}
			return m.sessions[r.Intn(len(m.sessions))]
		}
		administrators := func() []string {
			list := []string{}
			for _, id := range humans {
				if own := m.members[id]; own != nil && own.admin && own.state != "removed" {
					list = append(list, id)
				}
			}
			return list
		}
		outcome := func(kind string, err error, want bool, stranded bool) {
			t.Helper()
			if (err == nil) != want {
				t.Fatalf("world %d %s: error %v; model expects success %v", world, kind, err, want)
			}
			if errors.Is(err, ErrLastAdministrator) != stranded {
				t.Fatalf("world %d %s: error %v; model expects the last-administrator refusal %v", world, kind, err, stranded)
			}
			switch {
			case err == nil:
				reached[kind+" ok"]++
			case stranded:
				reached[kind+" last administrator"]++
			default:
				reached[kind+" refused"]++
			}
		}
		growth := []int{0, 1, 1, 2, 6, 7, 8, 9, 10, 16, 17, 20}
		for step := 0; step < 160; step++ {
			now := *f.now
			op := r.Intn(22)
			if step < 40 || (op >= 4 && op <= 5 && r.Intn(2) == 0) {
				op = growth[r.Intn(len(growth))]
			}
			switch op {
			case 0:
				id := f.next("u")
				outcome("create user", f.in.CreateUser(operator, id, triple(id), ""), true, false)
				m.users[id] = &mUser{state: "active", epoch: 1}
				m.order++
				m.subjects[id], m.linked[id], orders[id] = id, 1, m.order
				humans = append(humans, id)
			case 1, 2:
				subject := pick(humans)
				if r.Intn(10) < 4 {
					subject = pick(administrators())
				}
				if r.Intn(6) == 0 {
					subject = fmt.Sprintf("%s-x%d", pick(humans), r.Intn(3))
				}
				owner, known := m.subjects[subject]
				want := known && m.linked[subject] > 0 && m.human(owner)
				verifier := v(f.next("p"))
				idle := time.Duration(1+r.Intn(30)) * time.Minute
				length := idle + time.Duration(r.Intn(120))*time.Minute
				outcome("login", f.b.OpenSession(triple(subject), verifier, idle, length), want, false)
				if want {
					for _, earlier := range m.sessions {
						if earlier.user == owner && !now.Before(earlier.end) {
							earlier.gone = true
						}
					}
					m.sessions = append(m.sessions, &mSession{verifier: verifier, user: owner, epoch: m.users[owner].epoch, subject: subject, opened: now, idle: idle, idleEnd: now.Add(idle), end: now.Add(length)})
				}
			case 3:
				s := session()
				found := false
				for _, known := range m.sessions {
					found = found || known == s
				}
				want := found && !s.gone
				outcome("end session", f.b.EndSession(s.verifier), want, false)
				if want {
					s.gone = true
				}
			case 4, 5:
				id := pick(append(append([]string{}, humans...), services...))
				found := m.users[id]
				if found == nil || (op == 5 && r.Intn(3) != 0) {
					continue
				}
				choice := r.Intn(4)
				if found.state == "suspended" && r.Intn(3) != 0 {
					choice = 2
				}
				if choice < 2 && m.admin(id) && m.admins() == 1 && r.Intn(5) != 0 {
					continue
				}
				if choice == 3 && r.Intn(2) == 0 && len(administrators()) > 0 {
					id = pick(administrators())
					found = m.users[id]
				}
				switch choice {
				case 0, 1:
					want := found.state == "active" || (found.service && found.state == "suspended" && !found.block)
					outcome("suspend user", f.in.SuspendUser(operator, id, found.epoch), want, false)
					if want {
						found.state, found.block = "suspended", found.service
						found.epoch++
						m.forget(id)
					}
				case 2:
					want := found.state == "suspended"
					outcome("reinstate user", f.in.ReinstateUser(operator, id, found.epoch), want, false)
					if want {
						found.state, found.block = "active", false
						found.epoch++
					}
				case 3:
					if r.Intn(2) != 0 {
						continue
					}
					depart := func(after *model) {
						after.users[id].state = "departed"
						if own := after.members[id]; own != nil {
							own.state = "removed"
						}
					}
					own := m.members[id]
					stranded := found.state != "departed" && own != nil && own.state != "removed" &&
						(m.strands(depart) || (own.admin && m.seats(id) == 0 && func() bool { after := m.clone(); depart(after); return after.records() }()))
					want := found.state != "departed" && !stranded
					outcome("depart user", f.in.DepartUser(operator, id, found.epoch), want, stranded)
					held := own != nil && own.state != "removed"
					if want {
						depart(m)
						found.epoch++
						m.forget(id)
						if held {
							own.version++
						}
						for subject, owner := range m.subjects {
							if owner == id {
								m.linked[subject] = 0
							}
						}
					}
				}
			case 6, 7:
				s := session()
				invitee := pick(append(append([]string{}, humans...), services...))
				for attempt := 0; attempt < 8 && len(m.sessions) > 0; attempt++ {
					candidate := m.sessions[len(m.sessions)-1-r.Intn(min(len(m.sessions), 12))]
					if own := m.members[candidate.user]; m.current(candidate, now) && (own == nil || own.state == "removed") {
						invitee = candidate.user
						break
					}
				}
				id := f.next("i")
				admin := r.Intn(3) == 0
				role := Member
				if admin {
					role = Administrator
				}
				to := now.Add(time.Duration(1+r.Intn(300)) * time.Minute)
				want := m.current(s, now) && m.admin(s.user)
				outcome("invite", f.w.Invite(s.verifier, "w1", id, role, v(id), Validity{NotBefore: now, NotAfter: to}, invitee), want, false)
				if want {
					m.invitations = append(m.invitations, &mInvitation{id: id, inviter: s.user, inviterEpoch: m.users[s.user].epoch, inviterVersion: m.members[s.user].version, invitee: invitee, admin: admin, to: to, state: "pending"})
				}
			case 8, 9, 10:
				if len(m.invitations) == 0 {
					continue
				}
				found := m.invitations[len(m.invitations)-1-r.Intn(min(len(m.invitations), 4))]
				s := session()
				if r.Intn(8) != 0 {
					for _, candidate := range m.sessions {
						if candidate.user == found.invitee && m.current(candidate, now) {
							s = candidate
						}
					}
				}
				right := r.Intn(5) != 0
				verifier := v(found.id)
				if !right {
					verifier = v("guess")
				}
				own := m.members[s.user]
				reaches := m.current(s, now) && found.state == "pending" && now.Before(found.to) && found.invitee == s.user && m.inviterStands(found) && (own == nil || own.state == "removed")
				want := reaches && right
				outcome("accept", f.w.Accept("w1", found.id, verifier, s.verifier), want, false)
				if reaches && !right {
					found.attempts++
					if found.attempts >= limits.Attempts {
						found.state = "locked"
					}
				}
				if want {
					found.state = "accepted"
					version := uint64(1)
					if own != nil {
						version = own.version + 1
					}
					m.members[s.user] = &mMember{admin: found.admin, state: "active", version: version}
				}
			case 11, 12, 13, 14:
				s := session()
				target := pick(humans)
				kind := []string{"change role", "suspend membership", "reinstate membership", "remove membership"}[op-11]
				if r.Intn(10) < 8 {
					wanted := []string{}
					for _, id := range humans {
						if own := m.members[id]; own != nil && own.state != "removed" && (kind != "reinstate membership" || own.state == "suspended") {
							wanted = append(wanted, id)
						}
					}
					if len(wanted) > 0 {
						target = pick(wanted)
					}
				}
				own := m.members[target]
				if own == nil {
					own = &mMember{state: "removed"}
				}
				var change func(*model)
				allowed := own.state != "removed"
				switch kind {
				case "change role":
					change = func(after *model) { after.members[target].admin = !own.admin }
				case "suspend membership":
					allowed = allowed && own.state == "active"
					change = func(after *model) { after.members[target].state = "suspended" }
				case "reinstate membership":
					allowed = allowed && own.state == "suspended"
					change = func(after *model) { after.members[target].state = "active" }
				default:
					change = func(after *model) { after.members[target].state = "removed" }
				}
				actor := m.current(s, now) && (m.admin(s.user) || (kind == "remove membership" && s.user == target && own.state == "active"))
				stranded := actor && allowed && m.strands(change)
				want := actor && allowed && !stranded
				var err error
				switch kind {
				case "change role":
					role := Administrator
					if own.admin {
						role = Member
					}
					err = f.w.ChangeRole(s.verifier, "w1", target, role, own.version)
				case "suspend membership":
					err = f.w.SuspendMembership(s.verifier, "w1", target, own.version)
				case "reinstate membership":
					err = f.w.ReinstateMembership(s.verifier, "w1", target, own.version)
				default:
					err = f.w.RemoveMembership(s.verifier, "w1", target, own.version)
				}
				outcome(kind, err, want, stranded)
				if want {
					change(m)
					m.members[target].version++
				}
			case 15:
				s := session()
				recent := m.current(s, now) && now.Sub(s.opened) <= limits.Reauthentication
				if r.Intn(2) == 0 {
					subject := f.next("extra")
					outcome("link", f.b.LinkIdentity(s.verifier, triple(subject), "", "proof"), recent, false)
					if recent {
						m.order++
						m.subjects[subject], m.linked[subject], orders[subject] = s.user, 1, m.order
						m.users[s.user].epoch++
						m.forget(s.user)
					}
					continue
				}
				known := make([]string, 0, len(m.subjects))
				for subject := range m.subjects {
					known = append(known, subject)
				}
				sort.Strings(known)
				candidates := []string{}
				for _, subject := range known {
					if m.subjects[subject] == s.user || r.Intn(8) == 0 {
						candidates = append(candidates, subject)
					}
				}
				if len(candidates) == 0 {
					continue
				}
				subject := candidates[r.Intn(len(candidates))]
				count := 0
				for other, owner := range m.subjects {
					if owner == s.user && m.linked[other] > 0 {
						count++
					}
				}
				want := recent && m.subjects[subject] == s.user && m.linked[subject] > 0 && count > 1 && orders[s.subject] <= orders[subject]
				outcome("unlink", f.b.UnlinkIdentity(s.verifier, triple(subject)), want, false)
				if want {
					m.linked[subject] = 0
					m.users[s.user].epoch++
					m.forget(s.user)
				}
			case 16:
				s := session()
				owner := pick(humans)
				id := f.next("svc")
				own := m.members[owner]
				want := m.current(s, now) && m.admin(s.user) && own != nil && own.state == "active" && m.human(owner)
				outcome("create service user", f.w.CreateServiceUser(s.verifier, "w1", id, owner), want, false)
				if want {
					m.users[id] = &mUser{service: true, state: "active", epoch: 1}
					services = append(services, id)
				}
			case 17:
				s := session()
				id := pick(services)
				found := m.users[id]
				credential := f.next("c")
				from := now.Add(time.Duration(r.Intn(3)-1) * 10 * time.Minute)
				to := from.Add(time.Duration(11+r.Intn(200)) * time.Minute)
				want := m.current(s, now) && m.admin(s.user) && found != nil && found.state != "departed"
				outcome("add credential", f.w.AddCredential(s.verifier, "w1", id, credential, v(credential), Validity{NotBefore: from, NotAfter: to}), want, false)
				if want {
					m.credentials = append(m.credentials, &mCredential{id: credential, user: id, from: from, to: to})
				}
			case 18:
				if len(m.credentials) == 0 {
					continue
				}
				s := session()
				found := m.credentials[r.Intn(len(m.credentials))]
				want := m.current(s, now) && m.admin(s.user) && !found.revoked
				outcome("revoke credential", f.w.RevokeCredential(s.verifier, "w1", found.id), want, false)
				if want {
					found.revoked = true
				}
			case 19:
				s := session()
				id := pick(services)
				found := m.users[id]
				if found == nil {
					continue
				}
				actor := m.current(s, now) && m.admin(s.user)
				choice := r.Intn(4)
				if found.state == "suspended" && r.Intn(3) != 0 {
					choice = 2
				}
				switch choice {
				case 0, 1:
					want := actor && found.state == "active"
					outcome("suspend service user", f.w.SuspendServiceUser(s.verifier, "w1", id, found.epoch), want, false)
					if want {
						found.state, found.block = "suspended", false
						found.epoch++
					}
				case 2:
					want := actor && found.state == "suspended" && !found.block
					outcome("reinstate service user", f.w.ReinstateServiceUser(s.verifier, "w1", id, found.epoch), want, false)
					if want {
						found.state = "active"
						found.epoch++
					}
				default:
					if r.Intn(2) != 0 {
						continue
					}
					want := actor && found.state != "departed"
					outcome("retire service user", f.w.RetireServiceUser(s.verifier, "w1", id, found.epoch), want, false)
					if want {
						found.state, found.block = "departed", false
						found.epoch++
					}
				}
			default:
				advance := time.Duration(1+r.Intn(90)) * time.Second
				if op != 20 {
					continue
				}
				if len(m.sessions) > 0 && r.Intn(5) == 0 {
					s := session()
					for _, boundary := range []time.Time{s.idleEnd, s.end, s.opened.Add(limits.Reauthentication)} {
						if boundary.After(now) && r.Intn(2) == 0 {
							advance = boundary.Sub(now) - time.Duration(r.Intn(2))*time.Nanosecond
						}
					}
				}
				if len(m.credentials) > 0 && r.Intn(8) == 0 {
					found := m.credentials[r.Intn(len(m.credentials))]
					if found.to.After(now) {
						advance = found.to.Sub(now) - time.Duration(r.Intn(2))*time.Nanosecond
					}
				}
				if advance > 0 {
					*f.now = now.Add(advance)
					reached["clock ok"]++
				}
			}
			now = *f.now
			for _, s := range m.sessions {
				own := m.members[s.user]
				want := m.current(s, now) && own != nil && own.state == "active"
				answer, err := f.b.Access(s.verifier, "w1")
				if (err == nil) != want {
					t.Fatalf("world %d step %d: access of a session of %s: %v; model %v", world, step, s.user, err, want)
				}
				if want {
					reached["access ok"]++
					if renewed := now.Add(s.idle); renewed.After(s.idleEnd) {
						s.idleEnd = renewed
					}
					if s.idleEnd.After(s.end) {
						s.idleEnd = s.end
					}
					if r.Intn(6) == 0 {
						m.answers = append(m.answers, mAnswer{answer: answer, user: s.user, epoch: s.epoch, admin: own.admin, version: own.version, subject: s.subject, session: s})
					}
				} else {
					reached["access refused"]++
				}
			}
			for _, found := range m.credentials {
				owner := m.users[found.user]
				want := !found.revoked && !now.Before(found.from) && now.Before(found.to) && owner.state == "active"
				answer, err := f.b.AccessByCredential(found.id, v(found.id))
				if (err == nil) != want {
					t.Fatalf("world %d step %d: access by a credential of %s: %v; model %v", world, step, found.user, err, want)
				}
				if want {
					reached["credential access ok"]++
					if r.Intn(6) == 0 {
						m.answers = append(m.answers, mAnswer{answer: answer, user: found.user, epoch: owner.epoch, credential: found})
					}
				} else {
					reached["credential access refused"]++
				}
			}
			if len(m.answers) > 150 {
				m.answers = m.answers[len(m.answers)-150:]
			}
			for _, held := range m.answers {
				owner := m.users[held.user]
				want := owner.state == "active" && owner.epoch == held.epoch
				if held.credential != nil {
					want = want && !held.credential.revoked && !now.Before(held.credential.from) && now.Before(held.credential.to)
				} else {
					own := m.members[held.user]
					want = want && own.state == "active" && own.admin == held.admin && own.version == held.version && m.linked[held.subject] > 0 &&
						!held.session.gone && !now.Before(held.session.opened) && now.Before(held.session.end)
				}
				if err := f.b.Recheck(held.answer); (err == nil) != want {
					t.Fatalf("world %d step %d: recheck of an answer of %s: %v; model %v", world, step, held.user, err, want)
				}
				if want {
					reached["recheck ok"]++
				} else {
					reached["recheck refused"]++
				}
			}
			reported, err := f.in.WithoutAdministrator()
			want := []string{}
			if m.admins() == 0 && m.records() {
				want = []string{"w1"}
				blocked := false
				for id, own := range m.members {
					blocked = blocked || (own.admin && own.state == "active" && m.users[id].state == "suspended")
				}
				if !blocked {
					t.Fatalf("world %d step %d: the workspace lost its administrators without a security suspension", world, step)
				}
				reached["without administrator"]++
			}
			if err != nil || !reflect.DeepEqual(reported, want) {
				t.Fatalf("world %d step %d: without administrator %v %v; model %v", world, step, reported, err, want)
			}
		}
	}
	for _, category := range []string{
		"create user ok", "login ok", "login refused", "end session ok", "suspend user ok", "reinstate user ok", "depart user ok", "depart user last administrator",
		"invite ok", "invite refused", "accept ok", "accept refused", "change role ok", "change role refused", "change role last administrator",
		"suspend membership ok", "suspend membership last administrator", "reinstate membership ok", "remove membership ok", "remove membership last administrator",
		"link ok", "link refused", "unlink ok", "unlink refused", "create service user ok", "add credential ok", "revoke credential ok",
		"suspend service user ok", "reinstate service user ok", "reinstate service user refused", "retire service user ok", "clock ok",
		"end session refused", "suspend user refused", "reinstate user refused", "suspend membership refused", "reinstate membership refused", "remove membership refused",
		"create service user refused", "add credential refused", "revoke credential refused", "suspend service user refused", "retire service user refused",
		"access ok", "access refused", "credential access ok", "credential access refused", "recheck ok", "recheck refused", "without administrator",
	} {
		if reached[category] < 5 {
			t.Errorf("the generator reached %q only %d times", category, reached[category])
		}
	}
	t.Logf("%v", reached)
}
