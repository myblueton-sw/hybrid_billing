package licensemode

import (
	"crypto/ed25519"
	"encoding/hex"
	"testing"
)

func TestKnownAnswers(t *testing.T) {
	body := Encode(evaluation("g1"))
	priv := seed("k1")
	checks := map[string][2]string{
		"body":  {digest(body), "d8f4e3f5454a2eeb3a5256a5395af2e87ead57976f5b44184979bc936f2774dd"},
		"pub":   {hex.EncodeToString(priv.Public().(ed25519.PublicKey)), "b8776a4880d50f0180ab9ccf69d0d5758584ca2a27a54d2f5e3324308e1693a4"},
		"sig":   {hex.EncodeToString(ed25519.Sign(priv, signedMessage(body))), "6767c63d5f62bf6034fa9f266b5e1a1ca0abedc8a535f81ea76df73f584eef6bb42f2eb240a084ff542afe701e3f44887db76cdc1d57059a2be793c0d9880602"},
		"proof": {hex.EncodeToString(ed25519.Sign(priv, proofMessage("k1", "owner"))), "0e6d6053e88b8a0bc5b0a955bc48aad0c24eff846e575d686d3c430a7fc0ae954a4ff4770af369721209e28b32d731e584123189168edd38d1866d5abb10e00b"},
	}
	for name, pair := range checks {
		if pair[0] != pair[1] {
			t.Errorf("%s %s", name, pair[0])
		}
	}
}
