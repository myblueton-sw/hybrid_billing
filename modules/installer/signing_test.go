package installer

import (
	"bytes"
	"crypto/ed25519"
	"testing"
	"time"
)

func TestSigningWrappersVerifyThroughProposals(t *testing.T) {
	f := setup(t)
	signer := private("exported")
	key := Change{Kind: ChangeKey, Key: ReleaseKey{Version: "exported", Public: public("exported"), NotBefore: start, NotAfter: start.Add(30 * 24 * time.Hour),
		Proof: ed25519.Sign(signer, KeyChallenge("exported"))}}
	f.approve(key)
	rel := fixture("2.0.0")
	rel.Key = "exported"
	if !bytes.Equal(rel.SigningMessage(), rel.encode()) || !bytes.Equal(KeyChallenge("exported"), challenge("exported")) {
		t.Fatal("wrappers differ from the signed forms")
	}
	if bytes.Equal(KeyChallenge("exported"), KeyChallenge("other")) || bytes.Equal(rel.SigningMessage(), fixture("2.0.1").SigningMessage()) {
		t.Error("wrappers ignore their input")
	}
	done := f.approve(Change{Kind: ChangeRelease, Release: rel, Signature: ed25519.Sign(signer, rel.SigningMessage())})
	if done.State != ProposalApplied {
		t.Fatalf("release %+v", done)
	}
	if got, _, err := f.r.Release("2.0.0"); err != nil || got.Key != "exported" {
		t.Errorf("release %+v %v", got, err)
	}
	other := fixture("2.1.0")
	other.Key = "exported"
	reason(t, "foreign message", f.try(Change{Kind: ChangeRelease, Release: other, Signature: ed25519.Sign(signer, rel.SigningMessage())}), ReasonSignatureInvalid)
	wrong := Change{Kind: ChangeKey, Key: ReleaseKey{Version: "exported-2", Public: public("exported-2"), NotBefore: start, NotAfter: start.Add(time.Hour * 48),
		Proof: ed25519.Sign(private("exported-2"), KeyChallenge("exported"))}}
	reason(t, "proof for another version", f.try(wrong), ReasonKeyMismatch)
}
