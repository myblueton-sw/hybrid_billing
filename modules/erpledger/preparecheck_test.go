package erpledger

import "testing"

func TestPreparationCheckDoesNotReserveAndDoesNotReplay(t *testing.T) {
	f := newFixture(t)
	sub := invoice("checked", "api", line("l", "s", "10"))
	for i := 0; i < 3; i++ {
		check := must(f.r.CheckPreparation("w1", sub, human("biller")))
		if check.Total.Net != "10" || len(check.Mappings) != 3 {
			t.Fatal(check)
		}
		check.Mappings[0] = "tampered"
	}
	if len(f.r.workspaces["w1"].submissions) != 0 || len(f.r.workspaces["w1"].bySource) != 0 {
		t.Fatal("preflight reserved state")
	}
	prepared := must(f.r.Prepare("w1", sub, human("biller")))
	expect(t, "fresh check cannot replay", second(f.r.CheckPreparation("w1", sub, human("biller"))), "conflict")
	if again := must(f.r.Prepare("w1", sub, human("biller"))); again.Digest != prepared.Digest {
		t.Fatal("historical replay changed")
	}
	sub.Key = "different"
	expect(t, "source already prepared", second(f.r.CheckPreparation("w1", sub, human("biller"))), ReasonAlreadySubmitted)
}

func TestPreparationCheckFreshEvidenceScaleAndCapacity(t *testing.T) {
	for _, mode := range []string{"mapping", "survey", "scale", "capacity"} {
		t.Run(mode, func(t *testing.T) {
			f := newFixture(t)
			sub := invoice("new", "api", line("l", "s", "10"))
			want := ReasonMappingMissing
			switch mode {
			case "mapping":
				sub.Lines[0].Product = "missing"
			case "survey":
				sub.Period = "2026-11"
				want = ReasonNotSurveyed
			case "scale":
				sub.Lines[0].Tax = "0.1"
				want = "invalid"
			case "capacity":
				f.r.limits.Submissions = 1
				must(f.r.Prepare("w1", invoice("other", "api", line("other", "other-source", "2")), human("biller")))
				want = "limit"
			}
			expect(t, mode, second(f.r.CheckPreparation("w1", sub, human("biller"))), want)
			expect(t, "no effect", second(f.r.SubmissionOf("w1", "new")), "not_found")
		})
	}
}
