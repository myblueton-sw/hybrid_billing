package migration

import "testing"

func TestKnownAnswers(t *testing.T) {
	d := doc("a", 1)
	k := KeyView{SourceID: "k", Kind: KeyPublic, Fingerprint: sum("pk")}
	b := &batch{spec: BatchSpec{Source: "src-1", ID: "b1", Count: 2}, codes: map[string]string{"a": d.code(), "k": keyCode(k)}}
	if got := batchDigest("t1", "src-1", b); got != "58be7ed5390c29090ff511c132233f5572cfeb464151155ca2f8f7e585a0d04b" {
		t.Errorf("batch %s", got)
	}
	p := plan{classes: []Classified{{SourceID: "a", Class: ClassNew, Incoming: digest([]byte(d.code()))}}, series: map[[2]string]int64{{"ws1", "inv"}: 1},
		printed: map[[2]string]bool{{"ws1", "x"}: true}, keys: map[[2]string]bool{}}
	if got := effectDigest(p); got != "29c8858892ae4f0de53c3593ff7767818ab667d393c86634483bbd69c2181768" {
		t.Errorf("effect %s", got)
	}
	keys := map[string]string{
		fenceKey("src-", "t1", "src-1", "a"): "src-eefca177d25f2845fa42f420a5266f7d",
		fenceKey("ext-", "t1", "erp-1"):      "ext-777c87bbdc3ffd30dc850f7230249d95",
		fenceKey("dsp-", "t1", "INV-1"):      "dsp-cae99e3267ab211d4e2c70ec15aa1c91",
	}
	for got, want := range keys {
		if got != want {
			t.Errorf("fence key %s want %s", got, want)
		}
	}
	other := d
	other.Number = 10
	if other.code() == d.code() {
		t.Error("number not encoded")
	}
}
