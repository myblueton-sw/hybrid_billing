package server

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestSessionSelectionEntropyCollisionPreservesLedgerInIsolatedProcess(t *testing.T) {
	const childFlag = "HB304_SELECTION_COLLISION_CHILD"
	const success = "HB304_ENTROPY_COLLISION_PASS"
	const assertion = "HB304_ENTROPY_COLLISION_ASSERTION"
	if os.Getenv(childFlag) != "1" {
		cmd := exec.Command(os.Args[0], "-test.run=^TestSessionSelectionEntropyCollisionPreservesLedgerInIsolatedProcess$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), childFlag+"=1")
		if output := os.Getenv("HB304_COLLISION_COVERAGE"); testing.CoverMode() != "" && output != "" {
			if _, err := os.Stat(output); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("isolated coverage output is not a fresh owned path")
			}
			cmd.Args = append(cmd.Args, "-test.coverprofile="+output)
		}
		output, err := cmd.CombinedOutput()
		if err != nil {
			if bytes.Contains(output, []byte(assertion)) {
				t.Fatal(assertion)
			}
			t.Fatal("isolated collision process or instrumentation failed", err)
		}
		if !bytes.Contains(output, []byte(success)) {
			t.Fatal("isolated collision process did not execute its success assertion")
		}
		return
	}
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	caller := f.signedIn(MemberLogin, pw)
	found := capturedAdmissionSession(f, caller)
	at, revision := f.s.clock(), f.s.Revision()
	index := f.s.sessions.indexState(found.owner, at, 0)
	unrelated := f.s.newSessionRecord(found, at, at.Add(time.Minute), index.Version, revision)
	if unrelated == nil {
		t.Fatal("ordinary unrelated record premise")
	}
	if err := f.s.Close(); err != nil {
		t.Fatal("owned fixture did not quiesce before entropy override")
	}
	block := bytes.Repeat([]byte{71}, 16)
	if unrelated.ref == "session-selection-"+hex.EncodeToString(block) {
		block[0]++
	}
	original := rand.Reader
	func() {
		defer func() { rand.Reader = original }()
		rand.Reader = bytes.NewReader(bytes.Repeat(block, 2))
		first := f.s.newSessionRecord(found, at, at.Add(time.Minute), index.Version, revision)
		units := f.s.sessionRecordUnits
		second := f.s.newSessionRecord(found, at, at.Add(time.Minute), index.Version, revision)
		rand.Reader = original
		if first == nil || second != nil || f.s.sessionRecords[first.ref] != first || f.s.sessionRecords[unrelated.ref] != unrelated || first.ref == unrelated.ref || f.s.sessionRecordUnits != units || units != 2 || len(f.s.sessionRecords) != 2 {
			t.Fatal(assertion)
		}
		f.s.forgetSessionRecord(first)
	}()
	if f.s.sessionRecordUnits != 1 || len(f.s.sessionRecords) != 1 || f.s.sessionRecords[unrelated.ref] != unrelated {
		t.Fatal(assertion)
	}
	f.s.forgetSessionRecord(unrelated)
	if f.s.sessionRecordUnits != 0 || len(f.s.sessionRecords) != 0 {
		t.Fatal(assertion)
	}
	t.Log(success)
}
