package trustchange

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"sort"
)

type encoder struct{ out bytes.Buffer }

func (e *encoder) text(value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	e.out.Write(size[:])
	e.out.WriteString(value)
}

func (e *encoder) number(value uint64) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], value)
	e.out.Write(size[:])
}

func (e *encoder) sum() string {
	s := sha256.Sum256(e.out.Bytes())
	return hex.EncodeToString(s[:])
}

func tagged(tag string) *encoder {
	e := &encoder{}
	e.text(tag)
	return e
}

func ConfigDigest(c Candidate) string {
	e := tagged("trust_config_v1")
	e.number(uint64(len(c.Methods)))
	for _, m := range c.Methods {
		e.text(m)
	}
	e.text(c.MFA)
	e.number(uint64(c.MaxAge))
	e.number(uint64(len(c.StepUp)))
	for _, r := range c.StepUp {
		e.text(r.Action)
		e.number(uint64(r.MaxAge))
		if r.MFA {
			e.number(1)
		} else {
			e.number(0)
		}
	}
	return e.sum()
}

func EmptyDigest() string { return tagged("trust_config_v1").sum() }

func changeDigest(v View) string {
	e := tagged("trust_change_v1")
	for _, part := range []string{v.Workspace, v.Route, v.ID} {
		e.text(part)
	}
	e.number(v.Revision)
	e.number(v.OldEpoch)
	e.number(v.OldPolicy)
	e.text(v.OldDigest)
	e.text(v.NewDigest)
	e.text(v.Proposer)
	return e.sum()
}

func EvidenceDigest(change string, facts []string) string {
	sorted := append([]string(nil), facts...)
	sort.Strings(sorted)
	e := tagged("trust_evidence_v1")
	e.text(change)
	e.number(uint64(len(sorted)))
	for _, f := range sorted {
		e.text(f)
	}
	return e.sum()
}
