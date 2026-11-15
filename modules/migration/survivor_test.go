package migration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"testing"
	"time"
)

func TestReimportAtHighWaterAddsNoSeries(t *testing.T) {
	f := standard(t)
	a := doc("a", 7)
	f.accept("src-1", "t1", docs(a))
	_, v := f.batch("src-1", docs(a), "t1")
	if v.State != BatchVerified || len(v.Series) != 0 || f.r.series != 1 {
		t.Errorf("series %+v %d", v.Series, f.r.series)
	}
	f.approve(Change{Kind: ChangeAccept, Accept: Accept{Source: "src-1", Batch: "b2", Effect: v.Effect}})
	f.accept("src-1", "t1", docs(doc("b", 9)))
	if f.r.series != 1 {
		t.Errorf("series counter %d", f.r.series)
	}
	if n, err := f.r.HighWater("t1", "ws1", "inv"); n != 9 || err != nil {
		t.Errorf("high water %d %v", n, err)
	}
	if n, err := f.r.HighWater("t1", "ws1", ""); n != 0 || err == nil {
		t.Errorf("invalid series %d %v", n, err)
	}
}

func TestCertificateAtSizeCap(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	build := func(pad int) []byte {
		tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "legacy"}, NotBefore: start, NotAfter: start.Add(time.Hour),
			ExtraExtensions: []pkix.Extension{{Id: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 99999, 1}, Value: make([]byte, pad)}}}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
		if err != nil {
			t.Fatal(err)
		}
		return der
	}
	pad := maxDER - 600
	der := build(pad)
	for tries := 0; len(der) != maxDER && tries < 200; tries++ {
		pad += maxDER - len(der)
		der = build(pad)
	}
	if len(der) != maxDER {
		t.Skipf("no exact size, got %d", len(der))
	}
	if _, err := keyView(KeyReference{SourceID: "k1", Kind: KeyCertificate, DER: der}); err != nil {
		t.Errorf("exact cap refused %v", err)
	}
}
