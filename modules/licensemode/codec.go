package licensemode

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"strconv"
)

const (
	claimsHeader  = "hybrid-billing/license-claims/v1"
	signedDomain  = "hybrid-billing/license/v1"
	proofDomain   = "hybrid-billing/license-key/v1"
	maxBody       = 4096
	maxScope      = 32
	maxTime       = 253402300799
	maxTimeDigits = 12
)

func timeText(t int64) string {
	if t == 0 {
		return ""
	}
	return strconv.FormatInt(t, 10)
}

func boolText(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func Encode(c Claims) []byte {
	var e encoder
	for _, value := range []string{claimsHeader, c.Kind, c.ID, c.Issuer, c.KeyID, timeText(c.IssuedAt), c.SubjectID, c.Subject, c.Agreement, c.Supersedes,
		c.Product, c.Installation} {
		e.text(value)
	}
	e.text(strconv.Itoa(len(c.Scope)))
	for _, value := range c.Scope {
		e.text(value)
	}
	for _, value := range []string{c.Channel, c.Metric, timeText(c.Start), timeText(c.End), boolText(c.Perpetual), c.Terms, timeText(c.ActivateBy)} {
		e.text(value)
	}
	return e.out.Bytes()
}

func proofMessage(id, issuer string) []byte {
	var e encoder
	for _, value := range []string{proofDomain, id, issuer} {
		e.text(value)
	}
	return e.out.Bytes()
}

func signedMessage(body []byte) []byte {
	return append([]byte(signedDomain), body...)
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type decoder struct {
	in  []byte
	bad bool
}

func (d *decoder) text() string {
	if d.bad || len(d.in) < 8 {
		d.bad = true
		return ""
	}
	size := binary.BigEndian.Uint64(d.in[:8])
	if size > uint64(len(d.in)-8) {
		d.bad = true
		return ""
	}
	out := string(d.in[8 : 8+size])
	d.in = d.in[8+size:]
	return out
}

func (d *decoder) time(optional bool) int64 {
	text := d.text()
	if text == "" && optional {
		return 0
	}
	if !digitsOnly(text, maxTimeDigits) {
		d.bad = true
		return 0
	}
	value, _ := strconv.ParseInt(text, 10, 64)
	if value < 1 || value > maxTime {
		d.bad = true
	}
	return value
}

func digitsOnly(text string, limit int) bool {
	if text == "" || len(text) > limit || len(text) > 1 && text[0] == '0' {
		return false
	}
	for index := 0; index < len(text); index++ {
		if text[index] < '0' || text[index] > '9' {
			return false
		}
	}
	return true
}

func oneOf(value string, set ...string) bool {
	for _, item := range set {
		if item == value {
			return true
		}
	}
	return false
}

func optionalID(value string) bool { return value == "" || validID(value) }

func Decode(body []byte) (Claims, bool) {
	if len(body) > maxBody {
		return Claims{}, false
	}
	d := &decoder{in: body}
	var c Claims
	if d.text() != claimsHeader {
		return Claims{}, false
	}
	c.Kind, c.ID, c.Issuer, c.KeyID = d.text(), d.text(), d.text(), d.text()
	c.IssuedAt = d.time(false)
	c.SubjectID, c.Subject, c.Agreement, c.Supersedes, c.Product, c.Installation = d.text(), d.text(), d.text(), d.text(), d.text(), d.text()
	count := d.text()
	if !digitsOnly(count, 2) {
		return Claims{}, false
	}
	n, _ := strconv.Atoi(count)
	if n < 1 || n > maxScope {
		return Claims{}, false
	}
	for index := 0; index < n; index++ {
		c.Scope = append(c.Scope, d.text())
	}
	c.Channel, c.Metric = d.text(), d.text()
	c.Start, c.End = d.time(true), d.time(true)
	switch d.text() {
	case "1":
		c.Perpetual = true
	case "0":
	default:
		d.bad = true
	}
	c.Terms = d.text()
	c.ActivateBy = d.time(true)
	if d.bad || len(d.in) != 0 || !c.valid() || !bytes.Equal(Encode(c), body) {
		return Claims{}, false
	}
	return c, true
}

func (c Claims) valid() bool {
	if !validID(c.ID) || !validID(c.Issuer) || !validID(c.KeyID) || !validID(c.SubjectID) || !validText(c.Subject) || !validID(c.Product) || !validID(c.Installation) {
		return false
	}
	for index, value := range c.Scope {
		if !validID(value) || index > 0 && c.Scope[index-1] >= value {
			return false
		}
	}
	switch c.Kind {
	case DocEvaluation:
		return validText(c.Terms) && c.Agreement == "" && c.Supersedes == "" && c.Channel == "" && c.Metric == "" && c.Start == 0 && c.End == 0 && !c.Perpetual
	case DocCommercial:
		term := c.Perpetual && c.End == 0 || !c.Perpetual && c.End > c.Start
		return validID(c.Agreement) && optionalID(c.Supersedes) && oneOf(c.Channel, ChannelDirect, ChannelMarketplace) && oneOf(c.Metric, Metrics...) &&
			c.Start > 0 && term && c.Terms == "" && c.ActivateBy == 0
	}
	return false
}
