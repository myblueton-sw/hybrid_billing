package intake

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var modelAmount = regexp.MustCompile(`^-?[0-9]{1,12}(\.[0-9]{1,4})?$`)
var modelID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
var modelDigest = regexp.MustCompile(`^[0-9a-f]{8,128}$`)
var modelCurrency = regexp.MustCompile(`^[A-Z]{3}$`)

type mRow struct {
	amount   int64
	currency string
	digest   string
}

type mArtifact struct {
	spec     Spec
	norm     string
	state    State
	chunks   map[int]string
	rows     map[string]mRow
	dropped  bool
	count    int
	verified string
	reason   string
	sequence int
}

type mDocument struct {
	id        string
	subject   Subject
	kind      KeyKind
	highest   uint64
	artifacts []*mArtifact
}

type mManifest struct {
	content     []string
	attribution []string
}

type model struct {
	limits    Limits
	artifacts map[string]*mArtifact
	documents map[string]*mDocument
	bySubject map[Subject][]*mDocument
	manifests map[Subject][]mManifest
	records   map[string]int
	count     map[string]int
	sequence  int
}

func newModel(limits Limits) *model {
	return &model{limits: limits, artifacts: map[string]*mArtifact{}, documents: map[string]*mDocument{}, bySubject: map[Subject][]*mDocument{},
		manifests: map[Subject][]mManifest{}, records: map[string]int{}, count: map[string]int{}}
}

func mScaled(text string) int64 {
	negative := strings.HasPrefix(text, "-")
	whole, fraction, _ := strings.Cut(strings.TrimPrefix(text, "-"), ".")
	for len(fraction) < 4 {
		fraction += "0"
	}
	var value int64
	for _, c := range whole + fraction {
		value = value*10 + int64(c-'0')
	}
	if negative {
		return -value
	}
	return value
}

func mSubjectOK(s Subject) bool {
	return modelID.MatchString(s.WorkspaceID) && modelID.MatchString(s.Provider) && modelID.MatchString(s.Scope) && modelID.MatchString(s.Period)
}

func mCheck(c Check) bool { return c == Passed || c == Failed || c == NotRun }

func (m *model) shape(s Spec) string {
	wellFormed := mSubjectOK(s.Subject) && modelID.MatchString(s.ID) && modelID.MatchString(s.Document) && modelDigest.MatchString(s.Bytes) && modelID.MatchString(s.Location) &&
		(s.Kind == EventKeys || s.Kind == LocatorKeys) && s.Revision >= 1 && s.Size > 0 && s.Expanded >= s.Size && s.Chunks >= 0 && s.Rows >= 0 &&
		(s.Collection == "" || modelID.MatchString(s.Collection)) && mCheck(s.Scan) && mCheck(s.Syntax) && mCheck(s.Bodies)
	if !wellFormed {
		return "invalid"
	}
	withRows := false
	switch {
	case s.Role == Original && s.Original == "" && s.Method == "" && (s.Holds == Rows || s.Holds == Empty || s.Holds == AwaitingExtraction):
		withRows = s.Holds == Rows
	case s.Role == Derived && s.Holds == "" && modelID.MatchString(s.Original) && modelID.MatchString(s.Method):
		withRows = true
	default:
		return "invalid"
	}
	for currency, sum := range s.Sums {
		if !modelCurrency.MatchString(currency) || !modelAmount.MatchString(sum) {
			return "invalid"
		}
	}
	if !withRows {
		if s.Chunks != 0 || s.Rows != 0 || len(s.Sums) != 0 || s.Totals != "" {
			return "invalid"
		}
		return ""
	}
	if s.Totals != SourceStated && s.Totals != CallerComputed {
		return "invalid"
	}
	if s.Chunks < 1 || s.Rows < s.Chunks || len(s.Sums) < 1 || len(s.Sums) > s.Rows {
		return "invalid"
	}
	if s.Chunks > m.limits.Chunks {
		return "limit"
	}
	if s.Rows > s.Chunks*m.limits.ChunkRecords {
		return "invalid"
	}
	return ""
}

func mNorm(s Spec) string {
	var sums []string
	for currency, sum := range s.Sums {
		sums = append(sums, fmt.Sprintf("%s=%d", currency, mScaled(sum)))
	}
	sort.Strings(sums)
	s.Sums = nil
	return fmt.Sprintf("%+v %v", s, sums)
}

func (m *model) register(s Spec) string {
	if refusal := m.shape(s); refusal != "" {
		return refusal
	}
	ws := s.Subject.WorkspaceID
	norm := mNorm(s)
	if held := m.artifacts[ws+"|"+s.ID]; held != nil {
		if held.norm != norm {
			return "conflict"
		}
		return "ok:" + string(held.state)
	}
	doc := m.documents[ws+"|"+s.Document]
	if doc != nil && (doc.subject != s.Subject || doc.kind != s.Kind) {
		return "conflict"
	}
	if s.Role == Derived {
		from := m.artifacts[ws+"|"+s.Original]
		if from == nil {
			return "notfound"
		}
		if from.spec.Role != Original || from.spec.Holds != AwaitingExtraction || from.spec.Document != s.Document || from.spec.Revision != s.Revision || (from.state != Staging && from.state != Sealed) {
			return "rejected"
		}
	}
	if m.count[ws] >= m.limits.Artifacts {
		return "limit"
	}
	m.count[ws]++
	if doc == nil {
		doc = &mDocument{id: s.Document, subject: s.Subject, kind: s.Kind}
		m.documents[ws+"|"+s.Document] = doc
		m.bySubject[s.Subject] = append(m.bySubject[s.Subject], doc)
	}
	if s.Revision > doc.highest {
		doc.highest = s.Revision
	}
	made := &mArtifact{spec: s, norm: norm, state: Staging, chunks: map[int]string{}, rows: map[string]mRow{}}
	if s.Size > m.limits.MaxSize || s.Expanded > m.limits.MaxRatio*s.Size || s.Scan != Passed || s.Syntax != Passed || s.Bodies != Passed {
		made.state = Rejected
		made.dropped = true
	}
	m.artifacts[ws+"|"+s.ID] = made
	doc.artifacts = append(doc.artifacts, made)
	return "ok:" + string(made.state)
}

func (m *model) drop(ws string, a *mArtifact) {
	if !a.dropped {
		m.records[ws] -= len(a.rows)
		a.dropped = true
		a.rows = nil
	}
}

func (m *model) stage(ws, id string, number int, records []Record) string {
	if !modelID.MatchString(ws) || !modelID.MatchString(id) || number < 1 || len(records) < 1 || len(records) > m.limits.ChunkRecords {
		return "invalid"
	}
	var print []string
	for _, record := range records {
		if !modelID.MatchString(record.Key) || !modelDigest.MatchString(record.Digest) || !modelCurrency.MatchString(record.Currency) || !modelAmount.MatchString(record.Amount) {
			return "invalid"
		}
		print = append(print, fmt.Sprintf("%s/%d/%s/%s", record.Key, mScaled(record.Amount), record.Currency, record.Digest))
	}
	a := m.artifacts[ws+"|"+id]
	if a == nil {
		return "notfound"
	}
	if number > a.spec.Chunks {
		return "invalid"
	}
	sent := strings.Join(print, " ")
	staged, known := a.chunks[number]
	if known && staged == sent {
		return "ok:" + string(a.state)
	}
	if a.state != Staging {
		return "rejected"
	}
	contradiction := known
	seen := map[string]bool{}
	for _, record := range records {
		if _, held := a.rows[record.Key]; held || seen[record.Key] {
			contradiction = true
		}
		seen[record.Key] = true
	}
	if contradiction {
		a.state = Inconsistent
		m.drop(ws, a)
		return "ok:" + string(a.state)
	}
	if m.records[ws]+len(records) > m.limits.Records {
		return "limit"
	}
	for _, record := range records {
		a.rows[record.Key] = mRow{mScaled(record.Amount), record.Currency, record.Digest}
	}
	a.chunks[number] = sent
	a.count += len(records)
	m.records[ws] += len(records)
	return "ok:" + string(a.state)
}

func mAmount(value int64) string {
	sign := ""
	if value < 0 {
		sign, value = "-", -value
	}
	text := strings.TrimRight(fmt.Sprintf("%d.%04d", value/10000, value%10000), "0")
	return sign + strings.TrimSuffix(text, ".")
}

func mDigest(rows map[string]mRow) string {
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var encoded []byte
	for _, key := range keys {
		encoded = append(encoded, secondEncoding(key, mAmount(rows[key].amount), rows[key].currency, rows[key].digest)...)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (m *model) verify(ws, id, digest, verifier, evidence string) string {
	if !modelID.MatchString(ws) || !modelID.MatchString(id) || !modelDigest.MatchString(digest) || !modelID.MatchString(verifier) || !modelID.MatchString(evidence) {
		return "invalid"
	}
	a := m.artifacts[ws+"|"+id]
	if a == nil {
		return "notfound"
	}
	if a.spec.Role != Derived {
		return "rejected"
	}
	stated := digest + " " + verifier + " " + evidence
	if a.verified != "" {
		if a.verified != stated {
			return "conflict"
		}
		return "ok"
	}
	if a.state != Staging || len(a.chunks) != a.spec.Chunks {
		return "rejected"
	}
	if mDigest(a.rows) != digest {
		return "conflict"
	}
	a.verified = stated
	return "ok"
}

func (m *model) seal(ws, id string) string {
	if !modelID.MatchString(ws) || !modelID.MatchString(id) {
		return "invalid"
	}
	a := m.artifacts[ws+"|"+id]
	if a == nil {
		return "notfound"
	}
	if a.state == Sealed || a.state == Retired {
		return "ok:" + string(a.state)
	}
	if a.state != Staging {
		return "rejected"
	}
	var gaps []string
	if len(a.chunks) != a.spec.Chunks {
		gaps = append(gaps, GapChunks)
	}
	if a.count != a.spec.Rows {
		gaps = append(gaps, GapRows)
	}
	staged := map[string]int64{}
	present := map[string]bool{}
	for _, row := range a.rows {
		staged[row.currency] += row.amount
		present[row.currency] = true
	}
	var currencies []string
	for currency := range present {
		currencies = append(currencies, currency)
	}
	for currency := range a.spec.Sums {
		if !present[currency] {
			currencies = append(currencies, currency)
		}
	}
	sort.Strings(currencies)
	for _, currency := range currencies {
		declared, stated := a.spec.Sums[currency]
		switch {
		case !stated:
			gaps = append(gaps, GapCurrency+":"+currency)
		case !present[currency]:
			gaps = append(gaps, GapNoRows+":"+currency)
		case mScaled(declared) != staged[currency]:
			gaps = append(gaps, GapSum+":"+currency)
		}
	}
	if a.spec.Role == Derived {
		if m.artifacts[ws+"|"+a.spec.Original].state != Sealed {
			gaps = append(gaps, GapOriginal)
		}
		if a.verified == "" {
			gaps = append(gaps, GapVerified)
		}
	}
	if len(gaps) > 0 {
		return "rejected:" + strings.Join(gaps, ",")
	}
	m.sequence++
	a.sequence = m.sequence
	a.state = Sealed
	return "ok:" + string(a.state)
}

func (m *model) abandon(ws, id string, reason AbandonReason) string {
	if !modelID.MatchString(ws) || !modelID.MatchString(id) || (reason != TransferFailed && reason != Superseded && reason != Unsealable && reason != AbandonedBy) {
		return "invalid"
	}
	a := m.artifacts[ws+"|"+id]
	switch {
	case a == nil:
		return "notfound"
	case a.state == Abandoned && a.reason == string(reason):
		return "ok"
	case a.state == Abandoned:
		return "conflict"
	case a.state != Staging:
		return "rejected"
	}
	a.state = Abandoned
	a.reason = string(reason)
	m.drop(ws, a)
	return "ok"
}

func (m *model) retire(ws, id string, reason RetireReason) string {
	if !modelID.MatchString(ws) || !modelID.MatchString(id) || (reason != WrongSource && reason != Reissued && reason != VerificationWithdrawn && reason != RetiredBy) {
		return "invalid"
	}
	a := m.artifacts[ws+"|"+id]
	switch {
	case a == nil:
		return "notfound"
	case a.state == Retired && a.reason == string(reason):
		return "ok"
	case a.state == Retired:
		return "conflict"
	case a.state != Sealed:
		return "rejected"
	}
	a.state = Retired
	a.reason = string(reason)
	m.drop(ws, a)
	return "ok"
}

type mPicture struct {
	blockers    []string
	behind      []string
	same        []string
	staging     int
	content     []string
	attribution []string
	sums        map[string]int64
	live        map[string]bool
}

func (m *model) live(a *mArtifact) bool {
	if a.state != Sealed {
		return false
	}
	if a.spec.Role == Derived {
		return m.artifacts[a.spec.Subject.WorkspaceID+"|"+a.spec.Original].state == Sealed
	}
	return true
}

func (m *model) picture(at Subject) mPicture {
	pic := mPicture{sums: map[string]int64{}, live: map[string]bool{}}
	type claim struct {
		row      mRow
		artifact string
		sequence int
		document string
	}
	claims := map[string][]claim{}
	bytes := map[string]map[string]bool{}
	for _, doc := range m.bySubject[at] {
		var counting uint64
		for _, a := range doc.artifacts {
			if a.state == Staging {
				pic.staging++
			}
			if m.live(a) {
				pic.live[a.spec.ID] = true
				if a.spec.Revision > counting {
					counting = a.spec.Revision
				}
				if bytes[a.spec.Bytes] == nil {
					bytes[a.spec.Bytes] = map[string]bool{}
				}
				bytes[a.spec.Bytes][doc.id] = true
			}
		}
		if counting < doc.highest {
			pic.behind = append(pic.behind, doc.id)
		}
		if counting == 0 {
			continue
		}
		var carriers []*mArtifact
		for _, a := range doc.artifacts {
			if m.live(a) && a.spec.Revision == counting && (a.spec.Role == Derived || a.spec.Holds == Rows || a.spec.Holds == Empty) {
				carriers = append(carriers, a)
			}
		}
		if len(carriers) == 0 {
			pic.blockers = append(pic.blockers, Unreadable+":"+doc.id)
			continue
		}
		sort.Slice(carriers, func(i, j int) bool { return carriers[i].sequence < carriers[j].sequence })
		conflict := false
		for _, other := range carriers[1:] {
			if mDigest(other.rows) != mDigest(carriers[0].rows) {
				conflict = true
			}
		}
		if conflict {
			pic.blockers = append(pic.blockers, InConflict+":"+doc.id)
			continue
		}
		for key, row := range carriers[0].rows {
			full := "event//" + key
			if doc.kind == LocatorKeys {
				full = "locator/" + doc.id + "/" + key
			}
			claims[full] = append(claims[full], claim{row, carriers[0].spec.ID, carriers[0].sequence, doc.id})
		}
	}
	for full, made := range claims {
		sort.Slice(made, func(i, j int) bool { return made[i].sequence < made[j].sequence })
		conflict := false
		for _, other := range made[1:] {
			if other.row != made[0].row {
				conflict = true
			}
		}
		if conflict {
			pic.blockers = append(pic.blockers, KeyConflict+":"+strings.TrimPrefix(full, "event//"))
			continue
		}
		line := fmt.Sprintf("%s=%s%s#%s", full, mAmount(made[0].row.amount), made[0].row.currency, made[0].row.digest)
		pic.content = append(pic.content, line)
		pic.attribution = append(pic.attribution, line+"@"+made[0].artifact)
		pic.sums[made[0].row.currency] += made[0].row.amount
	}
	for _, documents := range bytes {
		var ids []string
		for id := range documents {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for i := range ids {
			for j := i + 1; j < len(ids); j++ {
				pic.same = append(pic.same, ids[i]+"+"+ids[j])
			}
		}
	}
	sort.Strings(pic.same)
	var unique []string
	for index, pair := range pic.same {
		if index == 0 || pair != pic.same[index-1] {
			unique = append(unique, pair)
		}
	}
	pic.same = unique
	sort.Strings(pic.behind)
	sort.Strings(pic.blockers)
	sort.Strings(pic.content)
	sort.Strings(pic.attribution)
	return pic
}

func (m *model) publish(at Subject, expected uint64) (string, mPicture) {
	if !mSubjectOK(at) {
		return "invalid", mPicture{}
	}
	if len(m.bySubject[at]) == 0 {
		return "notfound", mPicture{}
	}
	pic := m.picture(at)
	if len(pic.blockers) > 0 {
		return "blocked:" + strings.Join(pic.blockers, ","), pic
	}
	held := m.manifests[at]
	if len(held) > 0 {
		latest := held[len(held)-1]
		if strings.Join(latest.attribution, " ") == strings.Join(pic.attribution, " ") {
			return fmt.Sprintf("same:%d", len(held)), pic
		}
	}
	if expected != uint64(len(held)) {
		return "conflict", pic
	}
	if len(pic.content) == 0 && (len(held) == 0 || len(held[len(held)-1].content) == 0) {
		return "rejected", pic
	}
	if len(held) >= m.limits.Manifests {
		return "limit", pic
	}
	m.manifests[at] = append(held, mManifest{content: pic.content, attribution: pic.attribution})
	return fmt.Sprintf("written:%d", len(held)+1), pic
}

func (m *model) differs(at Subject, pic mPicture) string {
	held := m.manifests[at]
	switch {
	case len(pic.blockers) > 0:
		return Blocked
	case len(held) == 0 && len(pic.content) > 0:
		return DiffersContent
	case len(held) == 0:
		return Same
	case strings.Join(held[len(held)-1].content, " ") != strings.Join(pic.content, " "):
		return DiffersContent
	case strings.Join(held[len(held)-1].attribution, " ") != strings.Join(pic.attribution, " "):
		return DiffersAttribution
	}
	return Same
}
