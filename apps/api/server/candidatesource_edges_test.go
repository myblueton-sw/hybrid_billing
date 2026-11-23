package server

import (
	"errors"
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/intake"
	"github.com/myblueton-sw/hybrid_billing/modules/normalize"
)

func nextSourcePacket(t *testing.T, packet sourcePacket, id string, revision uint64) sourcePacket {
	t.Helper()
	out := sourcePacketCopy(t, packet)
	out.Run.ID = id
	for i := range out.Artifacts {
		out.Artifacts[i].ID += "-" + id
		out.Artifacts[i].Revision = revision
	}
	return out
}

func TestSourceAssemblyUnpublishedChangesAndHistory(t *testing.T) {
	for _, change := range []string{intake.DiffersContent, intake.DiffersAttribution} {
		t.Run(change, func(t *testing.T) {
			p, b, packet := normalizedFixtureSetup(t, false, nil)
			firstOrigin, err := p.assembleSource(packet)
			if err != nil {
				t.Fatal(err)
			}
			first, err := p.capture("workspace", "normalized", "old", b)
			if err != nil {
				t.Fatal(err)
			}
			next := nextSourcePacket(t, packet, "next-run", 2)
			if change == intake.DiffersContent {
				next.Artifacts[0].Rows[0].Cells["note"] = normalize.Cell{Text: "changed source"}
			}
			p.domain.Lock()
			prepared, err := p.prepareSource(next)
			if err != nil {
				t.Fatal(err)
			}
			held, err := p.reserveSource(prepared)
			if err != nil {
				t.Fatal(err)
			}
			for i := range prepared.specs {
				if err := p.stageSourceArtifact(prepared, held, i); err != nil {
					t.Fatal(err)
				}
			}
			status, err := p.intake.Status(prepared.subject)
			p.domain.Unlock()
			if err != nil || status.Latest != 1 || status.Differs != change || status.Staging != 0 || status.Behind.Total != 0 || status.SameBytes.Total != 0 || status.BlockerTotal != 0 {
				t.Fatal("missing isolated unpublished condition", status, err)
			}
			_, err = p.capture("workspace", "normalized", "reusable", b)
			var refused sourceCondition
			if !errors.As(err, &refused) || refused.Reason != "unpublished_"+change || refused.Observed.Difference != change {
				t.Fatal("unpublished change accepted", err)
			}
			old, err := p.candidates.read("workspace", "old")
			if err != nil || !reflect.DeepEqual(first, old) {
				t.Fatal("history changed", err)
			}
			secondOrigin, err := p.assembleSource(next)
			if err != nil || secondOrigin.Manifest != 2 || secondOrigin.Attribution == firstOrigin.Attribution {
				t.Fatal("new source publication", secondOrigin, err)
			}
			if (secondOrigin.Content == firstOrigin.Content) != (change == intake.DiffersAttribution) {
				t.Fatal("content and attribution conflated")
			}
			made, err := p.capture("workspace", "next-run", "reusable", b)
			if err != nil || made.Envelope.Result.Total != "5" || made.Digest == first.Digest {
				t.Fatal("new capture or reusable ID failed", err)
			}
			if _, err := p.assembleSource(packet); err == nil {
				t.Fatal("retry silently adopted later publication")
			}
			if _, err := p.capture("workspace", "normalized", "old", b); err == nil {
				t.Fatal("producer replay bypassed current source admission")
			}
			old, err = p.candidates.read("workspace", "old")
			if err != nil || !reflect.DeepEqual(first, old) {
				t.Fatal("old bytes no longer readable", err)
			}
		})
	}
}

func TestSourceAssemblyChecksCurrentAnnotations(t *testing.T) {
	for _, condition := range []string{"staged_artifacts", "behind_revision"} {
		t.Run(condition, func(t *testing.T) {
			p, b, packet := normalizedFixtureSetup(t, false, nil)
			if _, err := p.assembleSource(packet); err != nil {
				t.Fatal(err)
			}
			p.domain.Lock()
			next := nextSourcePacket(t, packet, "pending", 2)
			prepared, err := p.prepareSource(next)
			if err != nil {
				t.Fatal(err)
			}
			spec := prepared.specs[0]
			if _, _, err := p.intake.Register(spec); err != nil {
				t.Fatal(err)
			}
			if condition == "behind_revision" {
				if err := p.intake.Abandon("workspace", spec.ID, intake.Superseded); err != nil {
					t.Fatal(err)
				}
			}
			published, err := p.intake.Publish(prepared.subject, 1)
			manifest, readErr := p.intake.Manifest(prepared.subject, 1)
			p.domain.Unlock()
			if err != nil || readErr != nil || published.Number != 1 || published.Written || manifest.Staging != 0 || manifest.Behind.Total != 0 {
				t.Fatal("unchanged publication fixture", published, manifest, err, readErr)
			}
			_, err = p.capture("workspace", "normalized", "candidate", b)
			var refused sourceCondition
			if !errors.As(err, &refused) || refused.Reason != condition {
				t.Fatal("historical annotations substituted for current status", err)
			}
		})
	}
}

func TestSourceAssemblyFullManifestCoverage(t *testing.T) {
	p, b, packet := normalizedFixtureSetup(t, false, nil)
	if _, err := p.assembleSource(packet); err != nil {
		t.Fatal(err)
	}
	partial := nextSourcePacket(t, packet, "partial", 2)
	partial.Artifacts = partial.Artifacts[:1]
	if _, err := p.assembleSource(partial); err == nil {
		t.Fatal("manifest subset accepted")
	}
	if _, err := p.capture("workspace", "partial", "reusable", b); err == nil || len(p.candidates.records) != 0 {
		t.Fatal("partial manifest captured")
	}
	complete := nextSourcePacket(t, packet, "complete", 3)
	if _, err := p.assembleSource(complete); err != nil {
		t.Fatal(err)
	}
	if _, err := p.capture("workspace", "complete", "reusable", b); err != nil {
		t.Fatal("failed subset reserved candidate", err)
	}
}

func TestSourceAssemblyDuplicateBytesAndEconomicIdentity(t *testing.T) {
	p, _, packet := normalizedFixtureSetup(t, false, nil)
	copy := sourcePacketCopy(t, packet)
	copy.Artifacts[1].Rows[0].Cells = copy.Artifacts[0].Rows[0].Cells
	copy.Artifacts[1].Sums["USD"] = "2"
	_, err := p.assembleSource(copy)
	var refused sourceCondition
	if !errors.As(err, &refused) || refused.Reason != "duplicate_byte_signal" || refused.Observed.SameBytes != 1 {
		t.Fatal("known byte duplication not retained", err)
	}
	q, _, duplicate := normalizedFixtureSetup(t, false, nil)
	duplicate.Artifacts[1].Document = duplicate.Artifacts[0].Document
	duplicate.Artifacts[1].Rows[0].Key.Document = duplicate.Artifacts[0].Document
	if _, err := q.assembleSource(duplicate); err == nil || len(q.sources) != 0 {
		t.Fatal("duplicate economic identity reserved input")
	}
}

func TestSourceAssemblyNullAndAbsentRawCells(t *testing.T) {
	seen := make(map[string]bool)
	for _, mode := range []string{"absent", "null", "empty"} {
		p, _, packet := normalizedFixtureSetup(t, false, nil)
		cells := packet.Artifacts[0].Rows[0].Cells
		switch mode {
		case "absent":
			delete(cells, "note")
		case "null":
			cells["note"] = normalize.Cell{Null: true}
		default:
			cells["note"] = normalize.Cell{}
		}
		origin, err := p.assembleSource(packet)
		if err != nil {
			t.Fatal(mode, err)
		}
		if seen[origin.Rows[0].Digest] {
			t.Fatal("raw cell states collapsed", mode)
		}
		seen[origin.Rows[0].Digest] = true
	}
}

func TestSourceAssemblyCannotBindUnownedClosedRun(t *testing.T) {
	p, b, packet := normalizedFixtureSetup(t, false, nil)
	p.domain.Lock()
	prepared, err := p.prepareSource(packet)
	if err != nil {
		t.Fatal(err)
	}
	run, err := p.sourceRun(prepared, &sourceAssembly{Manifest: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.normal.Open(run); err != nil {
		t.Fatal(err)
	}
	if _, err := p.normal.Check("workspace", "normalized", 1, prepared.rows); err != nil {
		t.Fatal(err)
	}
	if _, _, err := p.normal.Close("workspace", "normalized"); err != nil {
		t.Fatal(err)
	}
	p.domain.Unlock()
	if _, err := p.capture("workspace", "normalized", "candidate", b); err == nil {
		t.Fatal("caller-supplied manifest number trusted")
	}
	if _, err := p.capture("foreign", "normalized", "candidate", b); err == nil {
		t.Fatal("foreign source bound")
	}
	if len(p.candidates.records) != 0 {
		t.Fatal("unbound input published")
	}
}
