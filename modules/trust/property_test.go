package trust

import (
	"math/rand"
	"net/netip"
	"testing"
	"time"
)

type span struct {
	bytes []byte
	bits  int
	class string
}

func v4(a, b, c, d byte, bits int, class string) span {
	return span{bytes: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, a, b, c, d}, bits: 96 + bits, class: class}
}

func v6(prefix []byte, bits int, class string) span {
	full := make([]byte, 16)
	copy(full, prefix)
	return span{bytes: full, bits: bits, class: class}
}

const (
	oracleMetadata  = "metadata"
	oracleForbidden = "forbidden"
	oraclePrivate   = "private"
	oracleGlobal    = "global"
)

var oracleSpans = []span{
	v4(169, 254, 0, 0, 16, oracleMetadata), v6([]byte{0xfe, 0x80}, 10, oracleMetadata),
	v4(169, 254, 169, 254, 32, oracleMetadata), v4(192, 0, 0, 192, 32, oracleMetadata), v4(100, 100, 100, 200, 32, oracleMetadata),
	v6([]byte{0xfd, 0, 0x0e, 0xc2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x02, 0x54}, 128, oracleMetadata),
	v6([]byte{0xfd, 0x20, 0, 0xce, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x02, 0x54}, 128, oracleMetadata),
	v4(0, 0, 0, 0, 8, oracleForbidden), v4(127, 0, 0, 0, 8, oracleForbidden), v4(192, 0, 0, 0, 24, oracleForbidden), v4(192, 0, 2, 0, 24, oracleForbidden),
	v4(192, 88, 99, 0, 24, oracleForbidden), v4(198, 18, 0, 0, 15, oracleForbidden), v4(198, 51, 100, 0, 24, oracleForbidden), v4(203, 0, 113, 0, 24, oracleForbidden),
	v4(224, 0, 0, 0, 4, oracleForbidden), v4(240, 0, 0, 0, 4, oracleForbidden),
	v6(nil, 96, oracleForbidden), v6([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0, 0}, 96, oracleForbidden), v6([]byte{0, 0x64, 0xff, 0x9b}, 96, oracleForbidden),
	v6([]byte{0, 0x64, 0xff, 0x9b, 0, 1}, 48, oracleForbidden), v6([]byte{1, 0}, 64, oracleForbidden), v6([]byte{0x20, 0x01}, 23, oracleForbidden),
	v6([]byte{0x3f, 0xff}, 20, oracleForbidden), v6([]byte{0x20, 0x02}, 16, oracleForbidden), v6([]byte{0x5f, 0}, 16, oracleForbidden),
	v6([]byte{0xfe, 0xc0}, 10, oracleForbidden), v6([]byte{0xff}, 8, oracleForbidden),
	v4(10, 0, 0, 0, 8, oraclePrivate), v4(172, 16, 0, 0, 12, oraclePrivate), v4(192, 168, 0, 0, 16, oraclePrivate), v4(100, 64, 0, 0, 10, oraclePrivate),
	v6([]byte{0xfc}, 7, oraclePrivate),
	v4(8, 8, 8, 8, 8, oracleGlobal), v6([]byte{0x20, 0x01, 0x48, 0x60}, 32, oracleGlobal), v6([]byte{0x26, 0x00}, 16, oracleGlobal), v6([]byte{0x40}, 4, oracleGlobal),
	v6([]byte{0x10}, 4, oracleGlobal),
}

func matches(address [16]byte, s span) bool {
	for bit := 0; bit < s.bits; bit++ {
		mask := byte(0x80 >> (bit % 8))
		if address[bit/8]&mask != s.bytes[bit/8]&mask {
			return false
		}
	}
	return true
}

func oracle(address [16]byte, allowed [][2]span) string {
	for _, class := range []string{oracleMetadata, oracleForbidden} {
		for _, s := range oracleSpans {
			if s.class == class && matches(address, s) {
				return class
			}
		}
	}
	for _, s := range oracleSpans {
		if s.class == oraclePrivate && matches(address, s) {
			for _, a := range allowed {
				if matches(address, a[0]) {
					return oracleGlobal
				}
			}
			return oraclePrivate
		}
	}
	mapped := address[10] == 0xff && address[11] == 0xff
	for index := 0; index < 10; index++ {
		mapped = mapped && address[index] == 0
	}
	if !mapped && address[0]&0xe0 != 0x20 {
		return oracleForbidden
	}
	return oracleGlobal
}

func TestAddressProperty(t *testing.T) {
	random := rand.New(rand.NewSource(7))
	names := map[string]string{"": oracleGlobal, ReasonMetadataAddress: oracleMetadata, ReasonForbiddenAddress: oracleForbidden, ReasonPrivateAddress: oraclePrivate}
	seen, rounds := map[string]int{}, 200000
	if testing.Short() {
		rounds = 20000
	}
	for round := 0; round < rounds; round++ {
		seed := oracleSpans[random.Intn(len(oracleSpans))]
		var address [16]byte
		for index := range address {
			address[index] = byte(random.Intn(256))
		}
		for bit := 0; bit < seed.bits; bit++ {
			mask := byte(0x80 >> (bit % 8))
			address[bit/8] = address[bit/8]&^mask | seed.bytes[bit/8]&mask
		}
		var allowed []netip.Prefix
		var model [][2]span
		if random.Intn(2) == 0 {
			target := address
			if random.Intn(3) == 0 {
				target[15] ^= 1
			}
			host, bits := netip.AddrFrom16(target), 96+random.Intn(33)
			if host.Is4In6() {
				host, bits = host.Unmap(), bits-96
			} else if random.Intn(2) == 0 {
				bits = 7 + random.Intn(122)
			}
			prefix := netip.PrefixFrom(host, bits)
			if random.Intn(2) == 0 {
				prefix = prefix.Masked()
			}
			if allowable(prefix) != oracleAllowable(prefix) {
				t.Fatalf("allowlist shape %v: %v", prefix, allowable(prefix))
			}
			if oracleAllowable(prefix) {
				allowed = append(allowed, prefix)
				form := prefix.Addr().As16()
				extra := 0
				if prefix.Addr().Is4() {
					extra = 96
				}
				model = append(model, [2]span{{bytes: form[:], bits: prefix.Bits() + extra}})
			}
		}
		input := netip.AddrFrom16(address)
		if random.Intn(2) == 0 {
			input = input.Unmap()
		}
		got := names[classify(input, allowed)]
		want := oracle(address, model)
		if got != want {
			t.Fatalf("%v under %v: %s; want %s", netip.AddrFrom16(address), allowed, got, want)
		}
		seen[want]++
		if want == oracleGlobal && len(allowed) > 0 && within(netip.AddrFrom16(address).Unmap(), private) {
			seen["allowed"]++
		}
	}
	for _, class := range []string{oracleMetadata, oracleForbidden, oraclePrivate, oracleGlobal, "allowed"} {
		if seen[class] < 100 {
			t.Errorf("class %s seen %d times", class, seen[class])
		}
	}
}

type modelSeen struct {
	digest  string
	expires time.Time
}

func TestReplayProperty(t *testing.T) {
	for seed := int64(1); seed <= 100; seed++ {
		random := rand.New(rand.NewSource(seed))
		f := setup(t)
		a := f.connect("w1", "c1", print("a"))
		f.key("w1", "c1", "v1", "one", start.Add(-time.Hour), start.Add(48*time.Hour))
		model := map[string]modelSeen{}
		quarantined := 0
		for step := 0; step < 120; step++ {
			f.advance(time.Duration(random.Intn(20)) * time.Second)
			now := *f.now
			signed := now.Add(time.Duration(random.Intn(int(4*limits.Skew/time.Second))-int(2*limits.Skew/time.Second)) * time.Second)
			body := []string{"x", "y"}[random.Intn(2)]
			count := 1 + random.Intn(limits.Items)
			pool := []string{"e1", "e2", "e3", "e4", "e5", "e6"}
			random.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
			ids := pool[:count]
			err := f.r.Verify(a, sign(a, "one", "v1", signed, body, ids...))
			want := ""
			live := 0
			for _, entry := range model {
				if !now.After(entry.expires) {
					live++
				}
			}
			same, overlap := 0, 0
			for _, id := range ids {
				if entry, ok := model[id]; ok && !now.After(entry.expires) {
					overlap++
					if entry.digest == body {
						same++
					}
				}
			}
			switch {
			case now.Sub(signed) > limits.Skew || signed.Sub(now) > limits.Skew:
				want = ReasonStale
			case same == len(ids):
				want = ReasonAlreadyAccepted
			case overlap > 0:
				want = ReasonDuplicate
			case live-overlap+len(ids) > limits.Events:
				want = ReasonReplayCapacity
			}
			if want == "" {
				if err != nil {
					t.Fatalf("seed %d step %d: %v", seed, step, err)
				}
				for _, id := range ids {
					model[id] = modelSeen{digest: body, expires: signed.Add(limits.Skew)}
				}
				continue
			}
			reason(t, "replay model", err, want)
			if want != ReasonAlreadyAccepted {
				quarantined++
			}
		}
		entries, dropped, _ := f.r.Quarantine("w1", 0, 1000, ann)
		if len(entries)+dropped != quarantined {
			t.Fatalf("seed %d: quarantine %d dropped %d; model %d", seed, len(entries), dropped, quarantined)
		}
	}
}

func oracleAllowable(prefix netip.Prefix) bool {
	if !prefix.IsValid() {
		return false
	}
	form, bits := prefix.Addr().As16(), prefix.Bits()
	if prefix.Addr().Is4() {
		bits += 96
	}
	for bit := bits; bit < 128; bit++ {
		if form[bit/8]&byte(0x80>>(bit%8)) != 0 {
			return false
		}
	}
	for _, s := range oracleSpans {
		if s.class == oraclePrivate && s.bits <= bits && matches(form, s) {
			return true
		}
	}
	return false
}
