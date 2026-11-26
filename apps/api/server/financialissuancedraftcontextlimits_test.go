package server

import (
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestFinancialDraftContextUsesActualDraftRepresentability(t *testing.T) {
	f, member, c, digest := applicationWorld(t)
	app, err := applicationAs(f, member, c, "issuance-final", digest)
	f.must(err)
	for _, scale := range []int{4, 5, 7, 8} {
		source := copyIssuanceSource(app.Source)
		source.Basis.Scale = scale
		source.Basis.Net = "15." + strings.Repeat("0", scale)
		source.Basis.Tax = "1." + strings.Repeat("0", scale)
		source.Digest, err = financialEvidenceDigest("issuance_source", source.Basis)
		f.must(err)
		err = draftContextConstructible(c, source)
		if scale == 4 {
			f.must(err)
		} else {
			documentFailure(t, err, commands.PrerequisiteFailed)
		}
	}
	if len(f.s.financialIssuance.drafts) != 0 {
		t.Fatal("context probe retained draft")
	}
}
