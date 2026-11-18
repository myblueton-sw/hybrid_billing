package decisionledger

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type Stamp struct {
	Line     int
	At       time.Time
	Decision string
	Clock    *ClockBase
}

type rawLine struct {
	At          string `json:"at"`
	Kind        string `json:"kind"`
	Decision    string `json:"decision"`
	LedgerTime  string `json:"ledger_time"`
	WallTime    string `json:"wall_time"`
	Choice      string `json:"choice"`
	OffsetNanos int64  `json:"offset_ns"`
}

func Scan(dir string) ([]Stamp, error) {
	f, err := os.Open(filepath.Join(dir, DirName, FileName))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	out := []Stamp{}
	reader := bufio.NewReaderSize(f, LineLimit+1)
	for number := 1; ; number++ {
		data, err := reader.ReadSlice('\n')
		if errors.Is(err, bufio.ErrBufferFull) {
			for errors.Is(err, bufio.ErrBufferFull) {
				_, err = reader.ReadSlice('\n')
			}
			continue
		}
		var raw rawLine
		if json.Unmarshal(data, &raw) == nil {
			if at, perr := time.Parse(time.RFC3339Nano, raw.At); perr == nil {
				stamp := Stamp{Line: number, At: at.UTC(), Decision: raw.Decision}
				if raw.Kind == KindDecision && raw.Decision == ClockBaseChoice {
					ledgerAt, _ := time.Parse(time.RFC3339Nano, raw.LedgerTime)
					wallAt, _ := time.Parse(time.RFC3339Nano, raw.WallTime)
					stamp.Clock = &ClockBase{LedgerTime: ledgerAt.UTC(), WallTime: wallAt.UTC(), Choice: raw.Choice, Offset: time.Duration(raw.OffsetNanos)}
				}
				out = append(out, stamp)
			}
		}
		if errors.Is(err, io.EOF) {
			return out, nil
		}
		if err != nil {
			return out, err
		}
	}
}
