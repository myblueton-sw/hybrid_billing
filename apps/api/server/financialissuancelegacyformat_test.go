package server

import (
	"testing"
)

func TestFinancialLegacyIssuanceSerialization(t *testing.T) {
	f, _, reviewer, draft, binding := draftApprovalWorld(t)
	review, err := prepareDraftReviewAs(f, reviewer, "legacy-format-review", draft, binding)
	f.must(err)
	values := map[string]any{
		"profile": draft.Profile,
		"channel": f.s.financialIssuance.channels["mature-kind"],
		"change":  f.s.financialIssuance.changes["consume-kind"],
		"term":    binding.Term,
		"binding": binding,
		"draft":   draft,
		"review":  review.Review,
	}
	expected := map[string]string{
		"profile": "b6d5ca8e39f507422c8664a12676d035166a446a30719f86507c4f6432f8355a",
		"channel": "727d04433888c7265a34dd2ce07fd0c7b281fcb3327c176a5ad560cf61adc5c4",
		"change":  "390742bd97f4890e8460bf40fa046c3c2277deb915a98ed28063b70859e7f6b8",
		"term":    "7b6bb620e294592bbca98d06bf6c4fa839ed70cc595455c51ee528e61ee80a01",
		"binding": "751421a3f496a53e8ff73e90c2f2f5149c2e9ca043345933469a999c3f2f3633",
		"draft":   "b5e065279fa94f216619c19dff73b1d033af623951cbf9aa83d008cdee284a6a",
		"review":  "ffc180670e66736089f6b94dbfb9f83c03ac27934ddfb3316c5df4afa81ecc87",
	}
	for name, value := range values {
		digest, err := normalizedDigest(value)
		f.must(err)
		if digest != expected[name] {
			t.Fatalf("legacy %s serialized bytes or nested digest changed: %s", name, digest)
		}
	}
}
