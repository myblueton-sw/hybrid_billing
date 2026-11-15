package plugins

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"math/big"
	"strconv"
	"time"
	"unicode/utf8"
)

const (
	maxIDBytes      = 128
	maxTextBytes    = 200
	maxCapabilities = 32
	maxEgress       = 16
	maxEndpointIDs  = 16
	maxScreens      = 16
	maxFields       = 32
	maxRetention    = 3650
	minTimeout      = time.Second
	maxTimeout      = 5 * time.Minute
	minMemory       = 16
	maxMemory       = 4096
	maxConcurrency  = 64
	maxComponent    = 6
	manifestTag     = "hybrid-billing/plugin/v1"
	challengeTag    = "hybrid-billing/plugin-key/v1"
)

var (
	fieldPrime = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))
	smallOrder = []string{
		"0100000000000000000000000000000000000000000000000000000000000000",
		"0100000000000000000000000000000000000000000000000000000000000080",
		"0000000000000000000000000000000000000000000000000000000000000000",
		"0000000000000000000000000000000000000000000000000000000000000080",
		"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f",
		"ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05",
		"26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85",
		"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a",
		"c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa",
	}
)

func validID(id string) bool {
	if id == "" || len(id) > maxIDBytes {
		return false
	}
	for index := 0; index < len(id); index++ {
		c := id[index]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func validText(text string) bool {
	if text == "" || len(text) > maxTextBytes || !utf8.ValidString(text) {
		return false
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 {
			return false
		}
	}
	return true
}

func validHex(text string) bool {
	if len(text) != 64 {
		return false
	}
	for index := 0; index < len(text); index++ {
		c := text[index]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func semver(text string) ([3]int, bool) {
	var out [3]int
	parts := bytes.Split([]byte(text), []byte("."))
	if len(parts) != 3 {
		return out, false
	}
	for index, part := range parts {
		if len(part) == 0 || len(part) > maxComponent || len(part) > 1 && part[0] == '0' {
			return out, false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return out, false
			}
		}
		out[index], _ = strconv.Atoi(string(part))
	}
	return out, true
}

func compare(a, b string) int {
	x, _ := semver(a)
	y, _ := semver(b)
	for index := range x {
		switch {
		case x[index] < y[index]:
			return -1
		case x[index] > y[index]:
			return 1
		}
	}
	return 0
}

func ascending(values []string) bool {
	for index, value := range values {
		if !validID(value) || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func capabilityKind(kind string) bool {
	return kind == CapabilityReadEvents || kind == CapabilityCallCommand || kind == CapabilityServeScreen
}

func less(a, b Capability) bool {
	return a.Kind < b.Kind || a.Kind == b.Kind && a.Target < b.Target
}

func has(capabilities []Capability, c Capability) bool {
	for _, own := range capabilities {
		if own == c {
			return true
		}
	}
	return false
}

func (m Manifest) valid() bool {
	if !validID(m.Plugin) || !validID(m.Publisher) || !validID(m.Key) || !validHex(m.Digest) {
		return false
	}
	if _, ok := semver(m.Version); !ok {
		return false
	}
	if _, ok := semver(m.Core.Min); !ok {
		return false
	}
	if _, ok := semver(m.Core.Max); !ok || compare(m.Core.Min, m.Core.Max) > 0 {
		return false
	}
	if len(m.Capabilities) == 0 || len(m.Capabilities) > maxCapabilities || len(m.Egress) > maxEgress || len(m.Endpoints) > maxEndpointIDs || len(m.Screens) > maxScreens {
		return false
	}
	for index, c := range m.Capabilities {
		if !capabilityKind(c.Kind) || !validID(c.Target) || index > 0 && !less(m.Capabilities[index-1], c) {
			return false
		}
	}
	if !ascending(m.Egress) || !ascending(m.Endpoints) {
		return false
	}
	l := m.Limits
	if l.Timeout < minTimeout || l.Timeout > maxTimeout || l.MemoryMiB < minMemory || l.MemoryMiB > maxMemory || l.Concurrency < 1 || l.Concurrency > maxConcurrency {
		return false
	}
	if !m.Storage.valid() {
		return false
	}
	for index, s := range m.Screens {
		if !validID(s.ID) || index > 0 && m.Screens[index-1].ID >= s.ID || !m.screen(s) {
			return false
		}
	}
	return true
}

func (s Storage) valid() bool {
	if !validID(s.Namespace) || !validID(s.Owner) {
		return false
	}
	if s.Migration != MigrationNone && s.Migration != MigrationAdditive && s.Migration != MigrationCopyForward {
		return false
	}
	switch s.Deletion {
	case DeleteOnRemoval:
		return s.RetentionDays == 0
	case RetainForEvidence:
		return s.RetentionDays >= 1 && s.RetentionDays <= maxRetention
	}
	return false
}

func (m Manifest) screen(s Screen) bool {
	switch s.Component {
	case ComponentTable, ComponentForm, ComponentDetail, ComponentChart:
	default:
		return false
	}
	if !validText(s.Label) || len(s.Fields) == 0 || len(s.Fields) > maxFields || !has(m.Capabilities, Capability{Kind: CapabilityServeScreen, Target: s.ID}) {
		return false
	}
	seen := map[string]bool{}
	for _, f := range s.Fields {
		if !validID(f.Name) || seen[f.Name] || !validText(f.Label) {
			return false
		}
		seen[f.Name] = true
		if f.Source != m.Storage.Namespace && !has(m.Capabilities, Capability{Kind: CapabilityReadEvents, Target: f.Source}) {
			return false
		}
	}
	return true
}

type encoder struct {
	out bytes.Buffer
}

func (e *encoder) text(value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	e.out.Write(size[:])
	e.out.WriteString(value)
}

func (e *encoder) number(value int64) { e.text(strconv.FormatInt(value, 10)) }

func (e *encoder) count(value int) { e.number(int64(value)) }

func (m Manifest) encode() []byte {
	var e encoder
	for _, value := range []string{manifestTag, m.Plugin, m.Version, m.Publisher, m.Key, m.Digest, m.Core.Min, m.Core.Max} {
		e.text(value)
	}
	e.count(len(m.Capabilities))
	for _, c := range m.Capabilities {
		e.text(c.Kind)
		e.text(c.Target)
	}
	for _, list := range [][]string{m.Egress, m.Endpoints} {
		e.count(len(list))
		for _, value := range list {
			e.text(value)
		}
	}
	e.number(int64(m.Limits.Timeout))
	e.count(m.Limits.MemoryMiB)
	e.count(m.Limits.Concurrency)
	e.text(m.Storage.Namespace)
	e.text(m.Storage.Owner)
	e.count(m.Storage.RetentionDays)
	e.text(m.Storage.Deletion)
	e.text(m.Storage.Migration)
	e.count(len(m.Screens))
	for _, s := range m.Screens {
		e.text(s.ID)
		e.text(s.Component)
		e.text(s.Label)
		e.count(len(s.Fields))
		for _, f := range s.Fields {
			e.text(f.Name)
			e.text(f.Label)
			e.text(f.Source)
		}
	}
	return e.out.Bytes()
}

func challenge(publisher, version string) []byte {
	var e encoder
	e.text(challengeTag)
	e.text(publisher)
	e.text(version)
	return e.out.Bytes()
}

func copyManifest(m Manifest) Manifest {
	m.Capabilities = append([]Capability(nil), m.Capabilities...)
	m.Egress = append([]string(nil), m.Egress...)
	m.Endpoints = append([]string(nil), m.Endpoints...)
	screens := make([]Screen, len(m.Screens))
	for index, s := range m.Screens {
		s.Fields = append([]Field(nil), s.Fields...)
		screens[index] = s
	}
	m.Screens = screens
	return m
}

func hexBytes(text string) []byte {
	out := make([]byte, len(text)/2)
	for index := range out {
		out[index] = nibble(text[2*index])<<4 | nibble(text[2*index+1])
	}
	return out
}

func nibble(c byte) byte {
	if c >= 'a' {
		return c - 'a' + 10
	}
	return c - '0'
}

func weak(public []byte) bool {
	if len(public) != ed25519.PublicKeySize {
		return true
	}
	for _, known := range smallOrder {
		if bytes.Equal(public, hexBytes(known)) {
			return true
		}
	}
	reversed := make([]byte, len(public))
	for index, b := range public {
		reversed[len(public)-1-index] = b
	}
	reversed[0] &= 0x7f
	return new(big.Int).SetBytes(reversed).Cmp(fieldPrime) >= 0
}
