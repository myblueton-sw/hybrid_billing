package decisionledger

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (p Prior) Floor() (uint64, bool) {
	floor, known := uint64(0), false
	raise := func(g uint64) {
		floor, known = max(floor, g), true
	}
	if p.MarkerUsable {
		raise(p.MarkerGeneration)
	}
	if p.LedgerID != "" {
		raise(p.Generation)
	}
	for _, r := range p.Verified {
		if r.Decision.Trust != nil {
			raise(r.Decision.Trust.NewEpoch >> 32)
		}
	}
	if p.UnverifiedRead {
		raise(p.Unverified)
	}
	for _, r := range p.Retained {
		if r.GenesisRead {
			raise(r.Generation + 1)
		}
		if r.HighestEpoch > 0 {
			raise(r.HighestEpoch>>32 + 1)
		}
	}
	return floor, known
}

func (p Prior) Next() (uint64, bool) {
	switch {
	case p.MarkerUsable:
		return p.MarkerGeneration + 1, true
	case p.LedgerID != "":
		return p.Generation + 1, true
	}
	return 0, false
}

func (l *Ledger) prior() Prior {
	p := Prior{Isolation: l.status.Isolation, ProblemLine: l.status.ProblemLine, MarkerUsable: l.markerUsable(), MarkerID: l.mark.LedgerID,
		MarkerGeneration: l.mark.G, LedgerID: l.oldID, Generation: l.oldG, Verified: copyRecords(l.verified), Retained: l.retained()}
	p.Unverified, p.UnverifiedRead = unverifiedHighest(l.path, l.o.MaxBytes)
	return p
}

func unverifiedHighest(path string, limit int64) (uint64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	reader := bufio.NewReaderSize(io.LimitReader(f, limit), LineLimit+1)
	highest, found := uint64(0), false
	for {
		data, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			for errors.Is(err, bufio.ErrBufferFull) {
				_, err = reader.ReadSlice('\n')
			}
			if err != nil {
				return highest, found
			}
			continue
		}
		if l, ok := decodeLine(bytes.TrimSuffix(data, []byte("\n"))); ok && l.shaped() {
			if l.Kind == KindGenesis {
				highest, found = max(highest, l.G), true
			} else if l.Trust != nil {
				highest, found = max(highest, l.Trust.NewEpoch>>32), true
			}
		}
		if err != nil {
			return highest, found
		}
	}
}

func (l *Ledger) retained() []Retained {
	entries, _ := os.ReadDir(l.dir)
	out := []Retained{}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), FileName+ReplacedTag) || !e.Type().IsRegular() {
			continue
		}
		r := Retained{File: e.Name()}
		if f, err := os.Open(filepath.Join(l.dir, e.Name())); err == nil {
			if info, err := f.Stat(); err == nil {
				s := verify(f, info.Size())
				r.Generation, r.GenesisRead, r.HighestEpoch = s.g, s.ledgerID != "", s.highest
			}
			f.Close()
		}
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a].File < out[b].File })
	return out
}

func (l *Ledger) successor() (Successor, error) {
	p := l.prior()
	floor, known := p.Floor()
	if l.o.Successor == nil {
		next, _ := p.Next()
		next = max(next, floor+1)
		if next >= MaxGeneration {
			return Successor{}, ErrGenerations
		}
		return Successor{Generation: next}, nil
	}
	next, err := l.o.Successor(p)
	switch {
	case err != nil:
		return Successor{}, err
	case next.Generation >= MaxGeneration:
		return Successor{}, ErrGenerations
	case known && next.Generation <= floor:
		return Successor{}, ErrInvalid
	}
	return next, nil
}
