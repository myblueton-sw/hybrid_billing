package server

import (
	"math/rand/v2"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
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
