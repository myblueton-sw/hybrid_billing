package evidence

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"
)

func errName(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "notfound"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "unknown"
}

type mInstance struct {
	custody Custody
	lost    bool
}

type mArtifact struct {
	spec      ArtifactSpec
	deleted   bool
	reason    DeleteReason
	instances []*mInstance
}

type mBundle struct {
	spec   BundleSpec
	retain time.Time
	purged bool
	reason PurgeReason
	holds  map[string]bool
}

type mHold struct {
	spec     HoldSpec
	released bool
	reason   ReleaseReason
}

type model struct {
	limits    Limits
	now       time.Time
	artifacts map[string]*mArtifact
	bundles   map[string]*mBundle
	holds     map[string]*mHold
	places    map[string]string
}

func sameCustody(a, b Custody) bool {
	return a.Kind == b.Kind && a.Location == b.Location && a.Receipt == b.Receipt && a.Evidence == b.Evidence && a.VerifiedUntil.Equal(b.VerifiedUntil)
}

func (m *model) held(a *mArtifact, until time.Time) bool {
	if a.deleted {
		return false
	}
	if until.Before(m.now) {
		until = m.now
	}
	for _, instance := range a.instances {
		if instance.lost {
			continue
		}
		if instance.custody.Kind == Product || (!instance.custody.VerifiedUntil.IsZero() && !instance.custody.VerifiedUntil.Before(until)) {
			return true
		}
	}
	return false
}

func (m *model) register(spec ArtifactSpec) string {
	if stored := m.artifacts[spec.ID]; stored != nil {
		same := stored.spec
		if same.Kind != spec.Kind || same.Version != spec.Version || same.Digest != spec.Digest || same.Size != spec.Size || !sameCustody(same.Custody, spec.Custody) {
			return "conflict"
		}
		return "ok"
	}
	if spec.Custody.Kind == Product {
		if user, used := m.places[spec.Custody.Location]; used && user != spec.ID {
			return "conflict"
		}
	}
	if len(m.artifacts) >= m.limits.Artifacts {
		return "limit"
	}
	if spec.Custody.Kind == Product {
		m.places[spec.Custody.Location] = spec.ID
	}
	m.artifacts[spec.ID] = &mArtifact{spec: spec, instances: []*mInstance{{custody: spec.Custody}}}
	return "ok"
}

func (m *model) add(id string, custody Custody) string {
	a := m.artifacts[id]
	if a == nil {
		return "notfound"
	}
	if a.deleted {
		return "rejected"
	}
	for _, instance := range a.instances {
		if instance.custody.Kind != custody.Kind || instance.custody.Location != custody.Location {
			continue
		}
		if instance.lost {
			return "rejected"
		}
		if custody.Kind == Product && instance.custody.Receipt != custody.Receipt {
			return "conflict"
		}
		if custody.VerifiedUntil.After(instance.custody.VerifiedUntil) {
			instance.custody = custody
		}
		return "ok"
	}
	if custody.Kind == Product {
		if user, used := m.places[custody.Location]; used && user != id {
			return "conflict"
		}
	}
	if len(a.instances) >= m.limits.Instances {
		return "limit"
	}
	if custody.Kind == Product {
		m.places[custody.Location] = id
	}
	a.instances = append(a.instances, &mInstance{custody: custody})
	return "ok"
}

func (m *model) lose(id string, kind CustodyKind, location string, reason LossReason, reasons map[*mInstance]LossReason) string {
	a := m.artifacts[id]
	if a == nil {
		return "notfound"
	}
	if a.deleted {
		return "rejected"
	}
	for _, instance := range a.instances {
		if instance.custody.Kind != kind || instance.custody.Location != location {
			continue
		}
		if instance.lost {
			if reasons[instance] != reason {
				return "conflict"
			}
			return "ok"
		}
		if reason == VerificationWithdrawn && kind == Product {
			return "rejected"
		}
		instance.lost = true
		reasons[instance] = reason
		return "ok"
	}
	return "notfound"
}

func (m *model) naming(id string) int {
	count := 0
	for _, b := range m.bundles {
		if b.purged {
			continue
		}
		for _, item := range b.spec.Items {
			if item.Artifact == id {
				count++
				break
			}
		}
	}
	return count
}

func (m *model) remove(id string, reason DeleteReason) string {
	a := m.artifacts[id]
	if a == nil {
		return "notfound"
	}
	if a.deleted {
		if a.reason != reason {
			return "conflict"
		}
		return "ok"
	}
	if count := m.naming(id); count > 0 {
		return fmt.Sprintf("rejected:%d", count)
	}
	a.deleted, a.reason = true, reason
	return "ok"
}

func sameItems(a, b []Item) bool {
	text := func(items []Item) string {
		var parts []string
		for _, item := range items {
			parts = append(parts, string(item.Role)+"/"+item.Artifact)
		}
		sort.Strings(parts)
		return strings.Join(parts, " ")
	}
	return text(a) == text(b)
}

func (m *model) confirm(spec BundleSpec) string {
	if len(spec.Items) > m.limits.Items {
		return "limit"
	}
	if stored := m.bundles[spec.ID]; stored != nil {
		if stored.spec.Customer != spec.Customer || stored.spec.Period != spec.Period || !stored.spec.RetainUntil.Equal(spec.RetainUntil) || stored.spec.FX != spec.FX || stored.spec.Modules != spec.Modules || !sameItems(stored.spec.Items, spec.Items) {
			return "conflict"
		}
		return "ok"
	}
	var found []string
	count := map[Kind]int{}
	named := map[string]int{}
	for _, item := range spec.Items {
		count[item.Role]++
		named[item.Artifact]++
		a := m.artifacts[item.Artifact]
		switch {
		case a == nil:
			found = append(found, "0"+UnknownArtifact)
		case a.deleted:
			found = append(found, "1"+DeletedArtifact)
		case a.spec.Kind != item.Role:
			found = append(found, "2"+WrongKind)
		case !m.held(a, spec.RetainUntil):
			found = append(found, "8"+NotHeld)
		}
	}
	for _, times := range named {
		if times > 1 {
			found = append(found, "3"+NamedTwice)
		}
	}
	for _, role := range []Kind{SourceOriginal, InputManifest, Mapping, Attribution, Policy, Engine, Result} {
		if count[role] == 0 {
			found = append(found, "4"+RoleMissing)
		}
	}
	if spec.FX != (count[FX] > 0) {
		found = append(found, "5"+FXMismatch)
	}
	if spec.Modules != (count[Module] > 0) {
		found = append(found, "6"+ModuleMismatch)
	}
	if !spec.RetainUntil.After(m.now) {
		found = append(found, "7"+RetentionNotAhead)
	}
	if len(found) > 0 {
		sort.Strings(found)
		for index := range found {
			found[index] = found[index][1:]
		}
		return "rejected:" + strings.Join(found, ",")
	}
	if len(m.bundles) >= m.limits.Bundles {
		return "limit"
	}
	m.bundles[spec.ID] = &mBundle{spec: spec, retain: spec.RetainUntil, holds: map[string]bool{}}
	return "ok"
}

func (m *model) protected(b *mBundle) bool {
	return !b.purged && (m.now.Before(b.retain) || len(b.holds) > 0)
}

func (m *model) verdict(b *mBundle) string {
	broken, risk := 0, 0
	for _, item := range b.spec.Items {
		a := m.artifacts[item.Artifact]
		switch {
		case !m.held(a, m.now):
			broken++
		case !m.protected(b):
		case len(b.holds) > 0:
			copied := false
			for _, instance := range a.instances {
				copied = copied || !instance.lost && instance.custody.Kind == Product
			}
			if !copied {
				risk++
			}
		case !m.held(a, b.retain):
			risk++
		}
	}
	verdict := Held
	if broken > 0 {
		verdict = Broken
	}
	return fmt.Sprintf("%s %d %d %v", verdict, broken, risk, m.protected(b))
}

func (m *model) extend(id string, until time.Time) string {
	b := m.bundles[id]
	if b == nil {
		return "notfound"
	}
	if b.purged {
		return "rejected"
	}
	if until.After(b.retain) {
		b.retain = until
	}
	return "ok:" + b.retain.UTC().Format(time.RFC3339)
}

func (m *model) purge(id string, reason PurgeReason) string {
	b := m.bundles[id]
	if b == nil {
		return "notfound"
	}
	if b.purged {
		if b.reason != reason {
			return "conflict"
		}
		return "ok"
	}
	if m.protected(b) {
		return "rejected"
	}
	b.purged, b.reason = true, reason
	return "ok"
}

func (m *model) hold(spec HoldSpec) string {
	if stored := m.holds[spec.ID]; stored != nil {
		a, b := append([]string(nil), stored.spec.Bundles...), append([]string(nil), spec.Bundles...)
		sort.Strings(a)
		sort.Strings(b)
		if stored.spec.Reason != spec.Reason || strings.Join(a, " ") != strings.Join(b, " ") {
			return "conflict"
		}
		return "ok"
	}
	for _, name := range spec.Bundles {
		if m.bundles[name] == nil {
			return "notfound"
		}
	}
	for _, name := range spec.Bundles {
		if m.bundles[name].purged {
			return "rejected"
		}
	}
	if len(m.holds) >= m.limits.Holds {
		return "limit"
	}
	m.holds[spec.ID] = &mHold{spec: spec}
	for _, name := range spec.Bundles {
		m.bundles[name].holds[spec.ID] = true
	}
	return "ok"
}

func (m *model) release(id string, reason ReleaseReason) string {
	stored := m.holds[id]
	if stored == nil {
		return "notfound"
	}
	if stored.released {
		if stored.reason != reason {
			return "conflict"
		}
		return "ok"
	}
	stored.released, stored.reason = true, reason
	for _, name := range stored.spec.Bundles {
		delete(m.bundles[name].holds, id)
	}
	return "ok"
}

func TestRandomHistoriesAgainstAModelAndInvariants(t *testing.T) {
	random := rand.New(rand.NewSource(19000))
	limits := Limits{Artifacts: 11, Bundles: 4, Holds: 3, Gaps: 5, Instances: 3, Items: 12, HoldBundles: 2, Restricted: 5, Page: 20, Listed: 100}
	kindOf := []Kind{SourceOriginal, InputManifest, Mapping, Attribution, Policy, Engine, Result, Document, SourceOriginal, FX, Module, Policy}
	reached := map[string]int{}
	for history := 0; history < 300; history++ {
		clock := &testClock{at: start}
		r, err := NewRegistry(limits, clock.read)
		if err != nil {
			t.Fatal(err)
		}
		h := &harness{t: t, r: r, clock: clock}
		m := &model{limits: limits, now: start, artifacts: map[string]*mArtifact{}, bundles: map[string]*mBundle{}, holds: map[string]*mHold{}, places: map[string]string{}}
		lossReasons := map[*mInstance]LossReason{}
		store := &simulatedStore{objects: map[string]time.Time{}, clock: clock}
		downloads := map[string]string{}
		custody := func(id string) Custody {
			if random.Intn(5) < 3 {
				name := fmt.Sprintf("p-%s-%d", id, random.Intn(3))
				if random.Intn(12) == 0 {
					name = fmt.Sprintf("p%d", random.Intn(4))
				}
				return Custody{Kind: Product, Location: name, Receipt: "rc-" + name}
			}
			made := Custody{Kind: Remote, Location: fmt.Sprintf("r%d", random.Intn(4))}
			if roll := random.Intn(8); roll > 0 {
				made.VerifiedUntil = clock.at.Add(time.Duration([]int{-3, 5, 30, 90, 400, 400, 400, 400}[roll]) * day).Truncate(time.Second)
				made.Evidence = "ev"
			}
			return made
		}
		pickArtifact := func() string { return fmt.Sprintf("a%d", 1+random.Intn(len(kindOf))) }
		if history%5 != 0 {
			for index := 1; index <= 9; index++ {
				id := fmt.Sprintf("a%d", index)
				spec := ArtifactSpec{WorkspaceID: "ws1", ID: id, Kind: kindOf[index-1], Version: "v1", Digest: digestOf(id), Size: 5, Custody: custody(id)}
				if _, err := r.Register(spec); errName(err) != m.register(spec) {
					t.Fatal("setup")
				} else if err == nil {
					store.put(id, spec.Custody)
				}
			}
		}
		step := 0
		check := func(what, got, want string) {
			t.Helper()
			if got != want {
				t.Fatalf("history %d step %d %s:\n got %s\nwant %s", history, step, what, got, want)
			}
			reached[what+" "+strings.SplitN(want, ":", 2)[0]]++
		}
		for step = 0; step < 120; step++ {
			switch roll := random.Intn(100); {
			case roll < 12:
				id := pickArtifact()
				index := 0
				fmt.Sscanf(id, "a%d", &index)
				spec := ArtifactSpec{WorkspaceID: "ws1", ID: id, Kind: kindOf[index-1], Version: "v1", Digest: digestOf(id), Size: 5, Custody: custody(id)}
				if stored := m.artifacts[id]; stored != nil && random.Intn(3) > 0 {
					spec = stored.spec
				}
				_, existed := m.artifacts[id]
				_, err := r.Register(spec)
				want := m.register(spec)
				if want == "ok" && err == nil && !existed {
					store.put(spec.ID, spec.Custody)
				}
				check("register", errName(err), want)
			case roll < 22:
				id := pickArtifact()
				made := custody(id)
				_, err := r.AddCustody("ws1", id, made)
				want := m.add(id, made)
				if want == "ok" && err == nil {
					store.put(id, made)
				}
				check("custody", errName(err), want)
			case roll < 30:
				id := pickArtifact()
				kind, location := Product, fmt.Sprintf("p%d", random.Intn(4))
				if a := m.artifacts[id]; a != nil && random.Intn(4) > 0 {
					chosen := a.instances[random.Intn(len(a.instances))]
					kind, location = chosen.custody.Kind, chosen.custody.Location
				}
				reason := []LossReason{Expired, LostDeleted, AccessRevoked, Corrupted, VerificationWithdrawn}[random.Intn(5)]
				_, err := r.ReportLoss("ws1", id, kind, location, reason)
				want := m.lose(id, kind, location, reason, lossReasons)
				if want == "ok" && err == nil {
					store.remove(id, Custody{Kind: kind, Location: location})
				}
				check("loss", errName(err), want)
			case roll < 56:
				spec := BundleSpec{WorkspaceID: "ws1", ID: fmt.Sprintf("b%d", 1+random.Intn(5)), Customer: "c1", Period: "p1", RetainUntil: clock.at.Add(time.Duration([]int{10, 30, 90, 30, 10, 0}[random.Intn(6)]) * day).Truncate(time.Second)}
				for index := 1; index <= 7; index++ {
					if random.Intn(30) > 0 {
						spec.Items = append(spec.Items, Item{kindOf[index-1], fmt.Sprintf("a%d", index)})
					}
				}
				for _, index := range []int{8, 9, 12} {
					if random.Intn(3) == 0 {
						spec.Items = append(spec.Items, Item{kindOf[index-1], fmt.Sprintf("a%d", index)})
					}
				}
				spec.FX, spec.Modules = random.Intn(4) == 0, random.Intn(5) == 0
				if spec.FX && random.Intn(9) > 0 {
					spec.Items = append(spec.Items, Item{FX, "a10"})
				}
				if spec.Modules && random.Intn(9) > 0 {
					spec.Items = append(spec.Items, Item{Module, "a11"})
				}
				if random.Intn(40) == 0 && len(spec.Items) > 0 {
					spec.Items = append(spec.Items, spec.Items[0])
				}
				if random.Intn(40) == 0 {
					spec.Items = append(spec.Items, Item{Document, "a1"})
				}
				if stored := m.bundles[spec.ID]; stored != nil && random.Intn(2) == 0 {
					spec = stored.spec
				}
				if len(spec.Items) == 0 {
					continue
				}
				out, err := r.Confirm(spec)
				got := errName(err)
				if err == ErrRejected {
					var found []string
					for _, reason := range out.Reasons {
						found = append(found, reason.Code)
					}
					got += ":" + strings.Join(found, ",")
				}
				check("confirm", got, m.confirm(spec))
			case roll < 64:
				id := fmt.Sprintf("b%d", 1+random.Intn(5))
				until := clock.at.Add(time.Duration(random.Intn(200)-20) * day).Truncate(time.Second)
				out, err := r.Extend("ws1", id, until)
				got := errName(err)
				if err == nil {
					got += ":" + out.Until.Format(time.RFC3339)
				}
				check("extend", got, m.extend(id, until))
			case roll < 72:
				spec := HoldSpec{WorkspaceID: "ws1", ID: fmt.Sprintf("h%d", 1+random.Intn(4)), Reason: "case", Bundles: []string{fmt.Sprintf("b%d", 1+random.Intn(5))}}
				if random.Intn(3) == 0 {
					if other := fmt.Sprintf("b%d", 1+random.Intn(5)); other != spec.Bundles[0] {
						spec.Bundles = append(spec.Bundles, other)
					}
				}
				_, err := r.PlaceHold(spec)
				check("hold", errName(err), m.hold(spec))
			case roll < 77:
				id, reason := fmt.Sprintf("h%d", 1+random.Intn(4)), []ReleaseReason{MatterClosed, MatterClosed, PlacedInError}[random.Intn(3)]
				check("release", errName(r.ReleaseHold("ws1", id, reason)), m.release(id, reason))
			case roll < 85:
				id, reason := fmt.Sprintf("b%d", 1+random.Intn(5)), []PurgeReason{PurgeRetentionEnded, PurgeRetentionEnded, PurgedBy}[random.Intn(3)]
				check("purge", errName(r.Purge("ws1", id, reason)), m.purge(id, reason))
			case roll < 89:
				id, reason := pickArtifact(), []DeleteReason{DeleteRetentionEnded, DeleteRetentionEnded, DeletedBy}[random.Intn(3)]
				names, err := r.DeleteArtifact("ws1", id, reason)
				got := errName(err)
				if err == ErrRejected {
					got += fmt.Sprintf(":%d", names.Total)
				}
				want := m.remove(id, reason)
				if want == "ok" && err == nil {
					for _, instance := range m.artifacts[id].instances {
						if instance.custody.Kind == Product {
							store.remove(id, instance.custody)
						}
					}
				}
				check("delete", got, want)
			default:
				clock.advance(time.Duration(random.Intn(45)) * day)
				m.now = clock.at
				r.Exposed("ws1")
			}
			var exposed []string
			for id, b := range m.bundles {
				out, err := r.Inspect("ws1", id)
				if b.purged {
					if err != ErrRejected {
						t.Fatalf("history %d step %d: a purged bundle was inspected: %v", history, step, err)
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				want := m.verdict(b)
				if got := fmt.Sprintf("%s %d %d %v", out.Verdict, out.BrokenTotal, out.AtRiskTotal, out.Protected); got != want {
					t.Fatalf("history %d step %d bundle %s:\n got %s\nwant %s", history, step, id, got, want)
				}
				reached["verdict "+out.Verdict]++
				if out.AtRiskTotal > 0 {
					reached["at risk"]++
				}
				if out.BrokenTotal > 0 || out.AtRiskTotal > 0 {
					exposed = append(exposed, id)
				}
				for _, broken := range out.Broken {
					reached["cause "+broken.Cause]++
				}
				h.backed(store, id)
				for _, item := range b.spec.Items {
					got, err := r.Download("ws1", id, item.Role, item.Artifact)
					if err != nil {
						continue
					}
					line := fmt.Sprintf("%s %s %s %d", got.ID, got.Version, got.Digest, got.Size)
					if earlier, known := downloads[id+"/"+item.Artifact]; known && earlier != line {
						t.Fatalf("history %d step %d: a download changed", history, step)
					}
					downloads[id+"/"+item.Artifact] = line
				}
			}
			sort.Strings(exposed)
			if listed, _ := r.Exposed("ws1"); fmt.Sprint(listed.Items) != fmt.Sprint(exposed) && len(exposed)+len(listed.Items) > 0 {
				t.Fatalf("history %d step %d exposed %v, model %v", history, step, listed.Items, exposed)
			}
			ws := r.workspaces["ws1"]
			if ws == nil {
				continue
			}
			for id, held := range ws.bundles {
				if held.state == Purged && (len(held.holds) != 0 || len(held.entries) != 0 || held.spec.Items != nil) {
					t.Fatalf("history %d step %d: purged bundle %s is protected or keeps items", history, step, id)
				}
				for _, item := range held.entries {
					if item.artifact.state == Deleted {
						t.Fatalf("history %d step %d: bundle %s names an artifact deleted through the registry", history, step, id)
					}
				}
			}
			for id, held := range ws.artifacts {
				far, near := r.now.Add(60*day), r.now.Add(7*day)
				if held.heldUntil(r.now, far) && !held.heldUntil(r.now, near) {
					t.Fatalf("history %d step %d: %s is held until a later time and not until an earlier one", history, step, id)
				}
				if (held.state == Deleted) != m.artifacts[id].deleted {
					t.Fatalf("history %d step %d: state of %s", history, step, id)
				}
			}
		}
	}
	for _, name := range []string{"register ok", "register conflict", "register limit", "custody ok", "custody notfound", "custody rejected", "custody conflict", "custody limit",
		"loss ok", "loss notfound", "loss conflict", "loss rejected", "confirm ok", "confirm conflict", "confirm rejected", "confirm limit", "extend ok", "extend notfound", "extend rejected",
		"hold ok", "hold notfound", "hold rejected", "hold conflict", "hold limit", "release ok", "release notfound", "release conflict", "purge ok", "purge notfound", "purge rejected", "purge conflict",
		"delete ok", "delete notfound", "delete rejected", "delete conflict", "verdict held", "verdict broken", "at risk", "cause lost", "cause verification_ended"} {
		if reached[name] < 3 {
			t.Errorf("the histories reached %q %d times", name, reached[name])
		}
	}
	if t.Failed() {
		t.Log(reached)
	}
}
