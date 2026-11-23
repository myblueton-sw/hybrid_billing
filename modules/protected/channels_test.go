package protected

import (
	"reflect"
	"testing"
)

func TestChannelsOfDetachedScopedCompleteEvidence(t *testing.T) {
	f := ready(t)
	f.rules(t, "w2", testRules())
	c := Channel{ID: "unadmitted", Subject: "acme", Kind: ChannelPortal, Ref: ref("unadmitted-ref", 'c'), Evidence: "evidence"}
	must(f.r.Register("w1", c, human("onboard")))
	c.ID = "other-subject"
	c.Subject = "other"
	must(f.r.Register("w1", c, human("onboard")))
	c.ID = "other-workspace"
	c.Subject = "acme"
	must(f.r.Register("w2", c, human("onboard")))
	must(f.r.Retire("w1", "ch-phone", human("retirer")))
	listed := must(f.r.ChannelsOf("w1", "acme"))
	if len(listed) != 3 || listed[0].Channel.ID != "ch-phone" || listed[0].RetiredBy != "retirer" || listed[1].Channel.ID != "ch-mail" || listed[2].Channel.ID != "unadmitted" || listed[2].Admitter != "" {
		t.Fatal("lost ordered declared channels")
	}
	original := append([]ChannelView(nil), listed...)
	listed[0].Channel.Ref.ID = "changed"
	if !reflect.DeepEqual(must(f.r.ChannelsOf("w1", "acme")), original) {
		t.Fatal("read aliases retained evidence")
	}
	if got := must(f.r.ChannelsOf("w2", "acme")); len(got) != 1 || got[0].Channel.ID != "other-workspace" {
		t.Fatal("workspace isolation")
	}
	if got := must(f.r.ChannelsOf("w1", "other")); len(got) != 1 || got[0].Channel.ID != "other-subject" {
		t.Fatal("subject isolation")
	}
	if got := must(f.r.ChannelsOf("w1", "missing")); len(got) != 0 {
		t.Fatal("missing subject not empty")
	}
	expect(t, "missing workspace", second(f.r.ChannelsOf("missing", "acme")), "not_found")
	expect(t, "invalid workspace", second(f.r.ChannelsOf("", "acme")), "invalid")
	expect(t, "invalid subject", second(f.r.ChannelsOf("w1", "bad subject")), "invalid")
}
