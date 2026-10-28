package customerusage

import (
	"context"
	"encoding/json"
	"os"
	"sync/atomic"
	"testing"
	"time"

	ca "github.com/myblueton-sw/hybrid_billing/modules/customeraccess"
)

type exportIdentityKey struct{}

func exportFixture(t *testing.T, mode string) (*Exports, *ca.Store, ca.Subject, *atomic.Int64, string) {
	t.Helper()
	g, store, subject, _, marker := testGateway(t, mode, 5*time.Second)
	subject.Grants[0].Action = ca.UsageExport
	if _, err := store.PutSubject(subject, 1); err != nil {
		t.Fatal(err)
	}
	g.verify = func(ctx context.Context) (ca.Principal, error) {
		id := "subject"
		if value, ok := ctx.Value(exportIdentityKey{}).(string); ok {
			id = value
		}
		return ca.Principal{Installation: g.profile.Installation, SubjectID: id, Epoch: 1, ValidUntil: time.Now().Add(time.Minute)}, nil
	}
	clock := &atomic.Int64{}
	clock.Store(time.Now().UnixNano())
	e, err := NewExports(g, ExportLimits{MaxJobs: 2, MaxRows: 200, MaxBytes: 65536, TTL: time.Minute}, func() time.Time { return time.Unix(0, clock.Load()) })
	if err != nil {
		t.Fatal(err)
	}
	return e, store, subject, clock, marker
}

func createJob(t *testing.T, e *Exports) string {
	t.Helper()
	id, err := e.Create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestExportGenerateDownloadAndCopiedArtifact(t *testing.T) {
	e, _, _, _, _ := exportFixture(t, "formula")
	id := createJob(t, e)
	if download, err := e.Download(context.Background(), id); download != nil || err == nil {
		t.Fatal("pending job downloaded")
	}
	if err := e.Run(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	download, err := e.Download(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if download.Filename != "usage.json" || download.ContentType != "application/json" || hashBytes(download.Bytes) != download.SHA256 {
		t.Fatal("unsafe or corrupt download metadata")
	}
	var content ExportPackage
	if err := json.Unmarshal(download.Bytes, &content); err != nil {
		t.Fatal(err)
	}
	if content.Manifest.RowCount != 1 || content.Manifest.Completion != "complete-supported-selection" || content.Manifest.AmountBasis != "usage-only" || content.Manifest.Currency != "not-applicable" || content.Manifest.ProfileRevision != e.gateway.revision {
		t.Fatal("incorrect export manifest")
	}
	payload, _ := json.Marshal(content.Rows)
	if content.Manifest.PayloadSHA256 != hashBytes(payload) || content.Rows[0].Values["quantity"] != "0.000000000000000000" || content.Rows[0].Values["service"] != "=HYPERLINK(\"https://example.invalid\")" {
		t.Fatal("exact inert JSON payload changed")
	}
	download.Bytes[0] = 'x'
	again, err := e.Download(context.Background(), id)
	if err != nil || again.Bytes[0] != '{' {
		t.Fatal("download aliases stored artifact")
	}
	if err := e.Run(context.Background(), id); err == nil {
		t.Fatal("ready job executed twice")
	}
}

func TestReadGrantCannotQueueExport(t *testing.T) {
	g, _, _, _, _ := testGateway(t, "ok", time.Second)
	e, err := NewExports(g, ExportLimits{MaxJobs: 1, MaxRows: 10, MaxBytes: 4096, TTL: time.Minute}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if id, err := e.Create(context.Background()); id != "" || err != ca.ErrDenied {
		t.Fatal("read-only export allowed")
	}
}

func TestOtherRecipientCannotDiscoverOrInvalidateJob(t *testing.T) {
	e, store, subject, _, _ := exportFixture(t, "ok")
	other := subject
	other.ID = "other"
	if _, err := store.PutSubject(other, 0); err != nil {
		t.Fatal(err)
	}
	id := createJob(t, e)
	ctx := context.WithValue(context.Background(), exportIdentityKey{}, "other")
	if err := e.Run(ctx, id); err != ca.ErrDenied {
		t.Fatal("foreign worker allowed")
	}
	if d, err := e.Download(ctx, id); d != nil || err != ca.ErrDenied {
		t.Fatal("foreign download allowed")
	}
	if d, err := e.Download(ctx, "missing"); d != nil || err != ca.ErrDenied {
		t.Fatal("job existence exposed")
	}
	if err := e.Run(context.Background(), id); err != nil {
		t.Fatal("foreign attempt poisoned job", err)
	}
	if d, err := e.Download(context.Background(), id); err != nil || d == nil {
		t.Fatal("owner download failed", err)
	}
}

func TestRegrantInvalidatesPendingAndReadyJobs(t *testing.T) {
	for _, ready := range []bool{false, true} {
		t.Run(map[bool]string{false: "pending", true: "ready"}[ready], func(t *testing.T) {
			e, store, subject, _, _ := exportFixture(t, "ok")
			id := createJob(t, e)
			if ready {
				if err := e.Run(context.Background(), id); err != nil {
					t.Fatal(err)
				}
			}
			removed := subject
			removed.Grants = nil
			v, err := store.PutSubject(removed, 2)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.PutSubject(subject, v); err != nil {
				t.Fatal(err)
			}
			if ready {
				if d, err := e.Download(context.Background(), id); d != nil || err != ca.ErrDenied {
					t.Fatal("regrant revived ready job")
				}
			} else if err := e.Run(context.Background(), id); err != ca.ErrDenied {
				t.Fatal("regrant revived pending job")
			}
			e.mu.Lock()
			defer e.mu.Unlock()
			if e.jobs[id].state != "invalid" || e.jobs[id].data != nil {
				t.Fatal("invalid bytes retained")
			}
		})
	}
}

func TestExpiryPurgeAndAdmissionLimits(t *testing.T) {
	e, _, _, clock, _ := exportFixture(t, "ok")
	id := createJob(t, e)
	createJob(t, e)
	if _, err := e.Create(context.Background()); err != ca.ErrUnavailable {
		t.Fatal("job quota exceeded")
	}
	if err := e.Run(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	clock.Add(int64(time.Minute))
	if d, err := e.Download(context.Background(), id); d != nil || err != ca.ErrDenied {
		t.Fatal("expired artifact downloaded")
	}
	e.PurgeExpired()
	if len(e.jobs) != 0 {
		t.Fatal("expired jobs retained")
	}
	createJob(t, e)
}

func TestBadSourceAndPackageBudgetNeverPublish(t *testing.T) {
	for _, mode := range []string{"page-marker", "wrong-count", "dead", "hidden", "oversized", "byte-budget"} {
		t.Run(mode, func(t *testing.T) {
			helper := mode
			if mode == "byte-budget" {
				helper = "ok"
			}
			e, _, _, _, _ := exportFixture(t, helper)
			if mode == "byte-budget" {
				e.limits.MaxBytes = 128
			}
			id := createJob(t, e)
			if err := e.Run(context.Background(), id); err == nil {
				t.Fatal("failed source published")
			}
			if d, err := e.Download(context.Background(), id); d != nil || err == nil {
				t.Fatal("failed job downloaded")
			}
			if e.jobs[id].state != "invalid" || e.jobs[id].data != nil {
				t.Fatal("partial bytes retained")
			}
		})
	}
}

func TestDownloadRejectsCorruptionAndChangedSource(t *testing.T) {
	for _, mode := range []string{"corrupt", "changed"} {
		t.Run(mode, func(t *testing.T) {
			e, _, _, _, _ := exportFixture(t, "ok")
			id := createJob(t, e)
			if err := e.Run(context.Background(), id); err != nil {
				t.Fatal(err)
			}
			if mode == "corrupt" {
				e.jobs[id].data[0] = 'x'
			} else {
				e.gateway.command[len(e.gateway.command)-2] = "changed"
			}
			if d, err := e.Download(context.Background(), id); d != nil || err != ca.ErrUnavailable {
				t.Fatal("changed artifact escaped")
			}
			if e.jobs[id].data != nil {
				t.Fatal("invalid artifact retained")
			}
		})
	}
}

func TestPausedExportConcurrencyRevocationAndExpiry(t *testing.T) {
	for _, mode := range []string{"duplicate", "revoked", "expired"} {
		t.Run(mode, func(t *testing.T) {
			e, store, subject, clock, marker := exportFixture(t, "paused")
			id := createJob(t, e)
			done := make(chan error, 1)
			go func() { done <- e.Run(context.Background(), id) }()
			deadline := time.Now().Add(3 * time.Second)
			for {
				if _, err := os.Stat(marker); err == nil {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("worker did not start")
				}
				time.Sleep(time.Millisecond)
			}
			switch mode {
			case "duplicate":
				if err := e.Run(context.Background(), id); err != ca.ErrUnavailable {
					t.Fatal("duplicate worker accepted")
				}
			case "revoked":
				subject.Grants = nil
				if _, err := store.PutSubject(subject, 2); err != nil {
					t.Fatal(err)
				}
			case "expired":
				clock.Add(int64(time.Minute))
				e.PurgeExpired()
			}
			if err := os.WriteFile(marker+"-release", []byte("release"), 0600); err != nil {
				t.Fatal(err)
			}
			err := <-done
			if mode == "duplicate" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatal("invalidated export published")
			}
		})
	}
}

func TestExpiryDuringPausedDownloadDiscardsBytes(t *testing.T) {
	e, _, _, clock, marker := exportFixture(t, "ok")
	id := createJob(t, e)
	if err := e.Run(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	e.gateway.command[len(e.gateway.command)-2] = "paused"
	type outcome struct {
		download *Download
		err      error
	}
	done := make(chan outcome, 1)
	go func() { d, err := e.Download(context.Background(), id); done <- outcome{d, err} }()
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("download did not start")
		}
		time.Sleep(time.Millisecond)
	}
	clock.Add(int64(time.Minute))
	if err := os.WriteFile(marker+"-release", []byte("release"), 0600); err != nil {
		t.Fatal(err)
	}
	result := <-done
	if result.download != nil || result.err == nil {
		t.Fatal("expired download released")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if job := e.jobs[id]; job != nil && job.data != nil {
		t.Fatal("expired artifact bytes retained after final check")
	}
}
