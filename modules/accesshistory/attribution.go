package accesshistory

import (
	"encoding/json"
	"math"
)

type lossScope struct{ user, workspace string }
type lossRange struct {
	count       int64
	first, last uint64
}
type lossCounts map[lossScope]lossRange

func (c lossCounts) add(r Record, count int64) {
	if r.User == "" || r.Seq == 0 || count <= 0 {
		return
	}
	first := r.Seq
	if r.Kind == KindSignInRefusedMore {
		if r.SourceSeq == 0 || r.SourceSeq > r.Seq {
			return
		}
		first = r.SourceSeq
	}
	key := lossScope{r.User, r.Workspace}
	v := c[key]
	if v.first == 0 || first < v.first {
		v.first = first
	}
	v.last = max(v.last, r.Seq)
	v.count += count
	c[key] = v
}

func (c lossCounts) encoded() map[string]int64 {
	out := map[string]int64{}
	for scope, v := range c {
		raw, _ := json.Marshal([]any{scope.user, scope.workspace, v.first, v.last})
		out[string(raw)] = v.count
	}
	return out
}

func decodeAttribution(key string) (lossScope, lossRange, bool) {
	var tuple []json.RawMessage
	if json.Unmarshal([]byte(key), &tuple) != nil || len(tuple) != 4 || len(tuple[0]) == 0 || tuple[0][0] != '"' || len(tuple[1]) == 0 || tuple[1][0] != '"' {
		return lossScope{}, lossRange{}, false
	}
	var scope lossScope
	var v lossRange
	if json.Unmarshal(tuple[0], &scope.user) != nil || json.Unmarshal(tuple[1], &scope.workspace) != nil || json.Unmarshal(tuple[2], &v.first) != nil || json.Unmarshal(tuple[3], &v.last) != nil {
		return scope, v, false
	}
	return scope, v, scope.user != "" && len(scope.user) <= 128 && len(scope.workspace) <= 128 && v.first > 0 && v.first <= v.last
}

func attributionCount(per map[string]int64, user, workspace string, boundary uint64) (int64, bool) {
	var exact int64
	unknown := false
	for key, count := range per {
		if count <= 0 {
			continue
		}
		scope, span, ok := decodeAttribution(key)
		if !ok {
			unknown = true
			continue
		}
		if scope.user == user && span.last <= boundary {
			continue
		}
		if scope.user != user || (scope.workspace != "" && scope.workspace != workspace) {
			unknown = true
			continue
		}
		if span.first <= boundary || count > math.MaxInt64-exact {
			unknown = true
			continue
		}
		exact += count
	}
	return exact, unknown
}

func bestEffortRecord(raw []byte) (Record, bool) {
	if len(raw) < 10 {
		return Record{}, false
	}
	var r Record
	if json.Unmarshal(raw[9:], &r) != nil || r.User == "" || len(r.User) > 128 || len(r.Workspace) > 128 || r.Seq == 0 {
		return Record{}, false
	}
	return r, true
}

func (h *History) splitLossRecords(recs []Record) []Record {
	out := make([]Record, 0, len(recs))
	for _, r := range recs {
		if (r.Kind != sysDropped && r.Kind != sysEarlyDeletion) || len(r.Per) == 0 {
			out = append(out, r)
			continue
		}
		var attributed int64
		part := 0
		for key, count := range r.Per {
			if count <= 0 {
				continue
			}
			one := r
			if part > 0 {
				one.Seq = h.nextSeq()
			}
			part++
			one.Per = map[string]int64{key: count}
			one.Count = count
			out = append(out, one)
			attributed += count
		}
		if r.Count > attributed {
			r.Per = nil
			if part > 0 {
				r.Seq = h.nextSeq()
			}
			r.Count -= attributed
			out = append(out, r)
		}
	}
	return out
}
