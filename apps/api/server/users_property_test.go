package server

import (
	"fmt"
	"maps"
	"math/rand/v2"
	"net/url"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

type modelUser struct {
	state   string
	role    string
	member  string
	known   bool
	pending bool
}

func TestUserAdministrationMatchesAModel(t *testing.T) {
	for seed := uint64(1); seed <= 4; seed++ {
		f := setup(t, SeedSample).ready()
		root := f.as(AdminLogin)
		f.setPassword(Admin2Login, pw)
		f.setPassword(MemberLogin, pw)
		model := map[string]*modelUser{
			SampleAdmin:  {state: "active", role: "administrator", member: "active", known: true},
			SampleAdmin2: {state: "active", role: "administrator", member: "active", known: true},
			SampleMember: {state: "active", role: "member", member: "active", known: true},
		}
		for _, login := range []string{"c1@example.invalid", "c2@example.invalid"} {
			user, tok := f.created(root, login, "member")
			f.redeem(tok, pw)
			model[user] = &modelUser{state: "active", role: "member", member: "active", known: true}
		}
		targets := []string{}
		for user := range model {
			targets = append(targets, user)
		}
		rng := rand.New(rand.NewPCG(seed, 263))
		ops := []string{"identity.suspend_user", "identity.reinstate_user", "identity.depart_user", "identity.change_role", "identity.suspend_membership",
			"identity.reinstate_membership", "identity.remove_membership", "identity.invite"}
		reached := map[string]int{}
		for step := 0; step < 120; step++ {
			op := ops[rng.IntN(len(ops))]
			target := targets[rng.IntN(len(targets))]
			fields := map[string]string{}
			role := []string{"administrator", "member"}[rng.IntN(2)]
			if op == "identity.change_role" || op == "identity.invite" {
				fields["role"] = role
			}
			m := model[target]
			want := ""
			self := target == SampleAdmin
			switch op {
			case "identity.suspend_user":
				if self {
					want = string(commands.PrerequisiteFailed)
				} else if m.state == "active" {
					m.state = "suspended"
				} else {
					want = string(commands.InvalidStage)
				}
			case "identity.reinstate_user":
				if m.state == "suspended" {
					m.state = "active"
				} else {
					want = string(commands.InvalidStage)
				}
			case "identity.depart_user":
				if self {
					want = string(commands.PrerequisiteFailed)
				} else if m.state != "departed" {
					m.state, m.member, m.known, m.pending = "departed", "", false, false
				} else {
					want = string(commands.InvalidStage)
				}
			case "identity.change_role":
				switch {
				case m.member == "":
					want = string(commands.InvalidStage)
				case self:
					want = string(commands.PrerequisiteFailed)
				case m.role == role:
					want = string(commands.InvalidStage)
				default:
					m.role = role
				}
			case "identity.suspend_membership", "identity.remove_membership":
				switch {
				case m.member == "":
					want = string(commands.InvalidStage)
				case self:
					want = string(commands.PrerequisiteFailed)
				case op == "identity.remove_membership":
					m.member = ""
				case m.member == "active":
					m.member = "suspended"
				default:
					want = string(commands.InvalidStage)
				}
			case "identity.reinstate_membership":
				if m.member == "suspended" {
					m.member = "active"
				} else {
					want = string(commands.InvalidStage)
				}
			case "identity.invite":
				if m.member != "" || !m.known || m.state != "active" || m.pending {
					want = string(commands.InvalidStage)
				} else {
					m.pending, m.role = true, role
				}
			}
			f.advance(time.Second)
			got := f.run(root, op, target, fields)
			if want == "" && got.status != 200 || want != "" && got.body["code"] != want {
				t.Fatalf("seed %d step %d %s on %s: got %d %v want %q", seed, step, op, target, got.status, got.body["code"], want)
			}
			reached[want]++
			for user, expected := range model {
				row := f.row(root, user)
				membership, _ := row["membership"].(map[string]any)
				gotMember, gotRole := "", ""
				if membership != nil {
					gotMember, gotRole = membership["state"].(string), membership["role"].(string)
				}
				pending := row["invitation_expires"] != ""
				if row["state"] != expected.state || gotMember != expected.member || (gotMember != "" && gotRole != expected.role) || pending != expected.pending ||
					(row["login"] != "") != expected.known {
					t.Fatalf("seed %d step %d %s: %s row %+v model %+v", seed, step, op, user, row, *expected)
				}
			}
			if row := f.row(root, SampleAdmin); row["installation_administrator"] != true || row["state"] != "active" {
				t.Fatalf("grant holder lost %+v", row)
			}
		}
		for _, code := range []string{"", string(commands.InvalidStage), string(commands.PrerequisiteFailed)} {
			if reached[code] < 5 {
				t.Errorf("seed %d reached %q only %d times", seed, code, reached[code])
			}
		}
	}
}

type modelPerson struct {
	id, login, email         string
	status, membership, role string
	until                    time.Time
	signin                   []string
	invited, holder          bool
}

func (m modelPerson) period(now time.Time) string {
	switch {
	case m.role == "":
		return ""
	case m.until.IsZero():
		return "open"
	case !now.Before(m.until):
		return "ended"
	case m.until.Sub(now) <= 14*24*time.Hour:
		return "ends_soon"
	}
	return "bounded"
}

func (m modelPerson) member(now time.Time) string {
	if m.membership == "active" && !m.until.IsZero() && !now.Before(m.until) {
		return membershipEnded
	}
	return m.membership
}

func (m modelPerson) holds(key, value string, now time.Time) bool {
	switch key {
	case "status":
		return m.status == value
	case "membership":
		return m.member(now) == value
	case "role":
		return m.role == value
	case "period":
		return m.period(now) == value
	case "signin":
		return slices.Contains(m.signin, value)
	case "invitation":
		return m.invited || slices.Contains(m.signin, signinSetLink)
	case "holder":
		return m.holder
	}
	return containsFold(m.login, value) || containsFold(m.email, value) || containsFold(m.id, value)
}

func containsFold(text, part string) bool {
	runes, want := []rune(text), []rune(part)
	for i := 0; i+len(want) <= len(runes); i++ {
		same := true
		for j := range want {
			same = same && strings.EqualFold(string(runes[i+j]), string(want[j]))
		}
		if same {
			return true
		}
	}
	return false
}

var loginParts = []string{"ann", "Bob", "kim", "김", "سارة", "Zoë", "ΣΟΦ", "x", "X", "lee", "ⅷ"}

func generatePeople(t *testing.T, p *population, rng *rand.Rand, n int) map[string]*modelPerson {
	now := p.f.s.clock()
	people := map[string]*modelPerson{}
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("g-%05d", rng.IntN(1000000))
		if people[id] != nil {
			continue
		}
		login := fmt.Sprintf("%s%d@example.invalid", loginParts[rng.IntN(len(loginParts))], i)
		m := &modelPerson{id: id, login: login, email: "contact-" + login, status: "active", membership: membershipNone}
		people[id] = m
		p.human(id, login)
		if rng.IntN(4) > 0 {
			m.role = []string{"administrator", "member"}[rng.IntN(2)]
			p.join(id, login, identity.Role(m.role))
			m.membership = "active"
			switch rng.IntN(4) {
			case 1:
				m.until = now.Add(time.Duration(1+rng.IntN(180)) * time.Minute)
				p.until(id, m.until)
			case 2:
				m.until = now.Add(time.Duration(1+rng.IntN(40)) * 24 * time.Hour)
				p.until(id, m.until)
			}
			if rng.IntN(6) == 0 {
				p.suspendMembership(id)
				m.membership = "suspended"
			}
			if rng.IntN(8) == 0 {
				p.removeMembership(id)
				m.membership, m.role, m.until = membershipNone, "", time.Time{}
			}
		} else if rng.IntN(2) == 0 {
			p.invite(id, login)
			m.invited = true
		}
		switch rng.IntN(4) {
		case 1:
			p.password(login)
			m.signin = []string{signinPassword}
		case 2:
			p.link(login, passwords.PurposeSet)
			m.signin = []string{signinSetLink}
		case 3:
			p.password(login)
			p.link(login, passwords.PurposeReset)
			m.signin = []string{signinPassword, signinResetLink}
		default:
			m.signin = []string{signinNotSet}
		}
		if rng.IntN(5) == 0 {
			p.holder(id)
			m.holder = true
		}
		switch rng.IntN(10) {
		case 0:
			p.suspendUser(id)
			m.status = "suspended"
		case 1:
			p.depart(id)
			*m = modelPerson{id: id, status: "departed", membership: membershipNone, signin: []string{signinNotSet}}
		}
	}
	return people
}

func randomFilter(rng *rand.Rand, people map[string]*modelPerson) url.Values {
	q := url.Values{}
	for _, key := range []string{"status", "membership", "role", "period", "signin", "invitation", "holder"} {
		if rng.IntN(5) == 0 {
			values := userFilterValues[key]
			q.Set(key, values[rng.IntN(len(values))])
		}
	}
	if rng.IntN(3) == 0 {
		ids := slices.Sorted(maps.Keys(people))
		m := people[ids[rng.IntN(len(ids))]]
		text := []string{m.login, m.email, m.id}[rng.IntN(3)]
		if text == "" {
			text = m.id
		}
		runes := []rune(text)
		from := rng.IntN(len(runes))
		part := string(runes[from : from+1+rng.IntN(min(4, len(runes)-from))])
		if strings.TrimSpace(part) != "" {
			q.Set("q", strings.ToUpper(part))
		}
	}
	return q
}

func (f world) expected(p *population, filter url.Values) []string {
	f.t.Helper()
	now := f.s.clock()
	query, code, _ := parseUsersQuery(filter)
	if code != "" {
		f.t.Fatalf("filter %v %s", filter, code)
	}
	matcher := newMatcher(query.filter)
	type entry struct{ id, login string }
	listed, departed := []entry{}, []string{}
	for after := ""; ; {
		page, err := f.s.identity.Installation().Users(after, 100)
		p.must(err)
		for _, u := range page {
			if u.Kind != identity.Human {
				continue
			}
			found, err := f.s.identity.Workspace().People(p.admin, SampleWorkspace, []string{u.ID})
			p.must(err)
			row := f.s.row(found[0].User, found[0].Member, now)
			if !matcher.matches(row, found[0].Member, now) {
				continue
			}
			if u.State == identity.Departed {
				departed = append(departed, u.ID)
			} else {
				listed = append(listed, entry{u.ID, row.Login})
			}
		}
		if len(page) < 100 {
			break
		}
		after = page[len(page)-1].ID
	}
	sort.Slice(listed, func(a, b int) bool {
		return listed[a].login < listed[b].login || listed[a].login == listed[b].login && listed[a].id < listed[b].id
	})
	out := []string{}
	for _, e := range listed {
		out = append(out, e.id)
	}
	return append(out, departed...)
}

func modelOrder(people map[string]*modelPerson, filter url.Values, now time.Time) []string {
	listed, departed := []*modelPerson{}, []string{}
	for _, m := range people {
		ok := true
		for key := range filter {
			if key != "limit" {
				ok = ok && m.holds(key, filter.Get(key), now)
			}
		}
		switch {
		case !ok:
		case m.status == "departed":
			departed = append(departed, m.id)
		default:
			listed = append(listed, m)
		}
	}
	sort.Slice(listed, func(a, b int) bool {
		return listed[a].login < listed[b].login || listed[a].login == listed[b].login && listed[a].id < listed[b].id
	})
	sort.Strings(departed)
	out := []string{}
	for _, m := range listed {
		out = append(out, m.id)
	}
	return append(out, departed...)
}

func generatedOnly(ids []string, people map[string]*modelPerson) []string {
	out := []string{}
	for _, id := range ids {
		if people[id] != nil {
			out = append(out, id)
		}
	}
	return out
}

func (f world) cursorsIncrease(cursors []string, logins map[string]string) bool {
	previous := userCursor{}
	for i, raw := range cursors {
		c, ok := parseCursor(raw)
		if !ok {
			return false
		}
		if i > 0 {
			switch {
			case previous.departed && !c.departed:
				return false
			case previous.departed && c.user <= previous.user:
				return false
			case !previous.departed && !c.departed && (logins[c.user] < logins[previous.user] || logins[c.user] == logins[previous.user] && c.user <= previous.user):
				return false
			}
		}
		previous = *c
	}
	return true
}

func (m userMatcher) withField(key, value string) userMatcher {
	f := m.filter
	*f.field(key) = value
	return newMatcher(f)
}

func TestUsersPagingMatchesABruteForceFilter(t *testing.T) {
	rounds := []int{20, 60, 120, 300, 300, 2000}
	for round, size := range rounds {
		f := setupWith(t, func(c *Config) { c.Passwords.Links = 10000 })
		p := newPopulation(f)
		rng := rand.New(rand.NewPCG(uint64(round)+1, 294))
		people := generatePeople(t, p, rng, size)
		f.advance(time.Duration(rng.IntN(240)) * time.Minute)
		p.refresh()
		root := f.as(AdminLogin)
		now := f.s.clock()
		logins := map[string]string{}
		for user, entry := range f.s.dir.people {
			logins[user] = entry.login
		}
		for id, m := range people {
			found, err := f.s.identity.Workspace().People(p.admin, SampleWorkspace, []string{id})
			p.must(err)
			row := f.s.row(found[0].User, found[0].Member, now)
			for key, values := range userFilterValues {
				for _, value := range values {
					if got := newMatcher(userFilter{}).withField(key, value).matches(row, found[0].Member, now); got != m.holds(key, value, now) {
						t.Fatalf("round %d %s %s=%s: %v; model %+v", round, id, key, value, got, *m)
					}
				}
				for _, text := range []string{m.login, m.email, m.id, "nobody"} {
					if text == "" {
						continue
					}
					runes := []rune(text)
					part := strings.ToUpper(string(runes[min(1, len(runes)-1):min(4, len(runes))]))
					if got := newMatcher(userFilter{Q: part}).matches(row, found[0].Member, now); got != m.holds("q", part, now) {
						t.Fatalf("round %d %s q=%s: %v", round, id, part, got)
					}
				}
				hits := 0
				for _, value := range userFilterValues["membership"] {
					if newMatcher(userFilter{Membership: value}).matches(row, found[0].Member, now) {
						hits++
					}
				}
				if hits != 1 {
					t.Fatalf("round %d %s matched %d membership values", round, id, hits)
				}
			}
		}
		trials := 25
		if size > 1000 {
			trials = 4
		}
		for trial := 0; trial < trials; trial++ {
			filter := randomFilter(rng, people)
			f.s.cfg.Access.UserFilterScan = 1 + rng.IntN(30)
			if size > 1000 {
				f.s.cfg.Access.UserFilterScan = 100 + rng.IntN(400)
			}
			limit := 1 + rng.IntN(10)
			filter.Set("limit", fmt.Sprint(limit))
			want := f.expected(p, filter)
			got, cursors := f.walk(root, filter.Encode())
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("round %d trial %d filter %v budget %d: %v; want %v", round, trial, filter, f.s.cfg.Access.UserFilterScan, got, want)
			}
			if model, walked := modelOrder(people, filter, now), generatedOnly(got, people); strings.Join(walked, ",") != strings.Join(model, ",") {
				t.Fatalf("round %d trial %d filter %v: generated rows %v; model %v", round, trial, filter, walked, model)
			}
			if len(slices.Compact(slices.Sorted(slices.Values(got)))) != len(got) {
				t.Fatalf("round %d trial %d duplicates %v", round, trial, got)
			}
			if !f.cursorsIncrease(cursors, logins) {
				t.Fatalf("round %d trial %d cursors %v", round, trial, cursors)
			}
			if q := filter.Get("q"); q != "" {
				for _, id := range got {
					if m := people[id]; m != nil && !m.holds("q", q, now) {
						t.Fatalf("round %d trial %d %s does not hold %q", round, trial, id, q)
					}
				}
			}
		}
	}
}
