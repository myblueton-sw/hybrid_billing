package receivables

import (
	"testing"
	"time"
)

func TestNetDaysFromTheIssueDateInTheContractZone(t *testing.T) {
	r, clock := openWith(t, testLimits())
	setup(t, r, "k30", "Asia/Seoul", 30)
	setup(t, r, "k45", "America/New_York", 45)
	setup(t, r, "k0", "Europe/Berlin", 0)
	clock.set(time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC))
	seoul, _ := time.LoadLocation("Asia/Seoul")
	york, _ := time.LoadLocation("America/New_York")
	cases := []struct {
		contract string
		at       time.Time
		issue    string
		due      string
	}{
		{"k30", time.Date(2026, 10, 1, 0, 0, 0, 0, seoul), "2026-10-01", "2026-10-31"},
		{"k30", time.Date(2026, 9, 30, 23, 59, 59, 999999999, seoul), "2026-09-30", "2026-10-30"},
		{"k30", time.Date(2026, 9, 30, 15, 0, 0, 0, time.UTC), "2026-10-01", "2026-10-31"},
		{"k30", time.Date(2026, 9, 30, 14, 59, 59, 0, time.UTC), "2026-09-30", "2026-10-30"},
		{"k45", time.Date(2026, 10, 1, 12, 0, 0, 0, york), "2026-10-01", "2026-11-15"},
		{"k45", time.Date(2026, 10, 2, 3, 30, 0, 0, time.UTC), "2026-10-01", "2026-11-15"},
		{"k45", time.Date(2026, 11, 1, 1, 30, 0, 0, york), "2026-11-01", "2026-12-16"},
		{"k45", time.Date(2026, 3, 8, 2, 30, 0, 0, york), "2026-03-08", "2026-04-22"},
		{"k30", time.Date(2026, 1, 31, 9, 0, 0, 0, seoul), "2026-01-31", "2026-03-02"},
		{"k0", time.Date(2026, 3, 29, 1, 59, 0, 0, time.UTC), "2026-03-29", "2026-03-29"},
		{"k0", time.Date(2026, 10, 24, 22, 30, 0, 0, time.UTC), "2026-10-25", "2026-10-25"},
	}
	for index, c := range cases {
		id := "d" + itoa(index)
		v := must(r.Issue("ws", invoice(id, c.contract, "10", c.at), "clerk"))
		if v.Snapshot.IssueDate != c.issue || v.Snapshot.OriginalDue != c.due || v.Snapshot.EffectiveDue != c.due {
			t.Errorf("%s %v: %+v", c.contract, c.at, v.Snapshot)
		}
	}
}

func TestLeapYearAndMonthEnds(t *testing.T) {
	for date, want := range map[[2]string]string{
		{"2028-02-28", "1"}:  "2028-02-29",
		{"2028-02-29", "1"}:  "2028-03-01",
		{"2027-02-28", "1"}:  "2027-03-01",
		{"2028-01-30", "30"}: "2028-02-29",
		{"2026-12-31", "1"}:  "2027-01-01",
		{"2026-10-01", "0"}:  "2026-10-01",
		{"2026-10-01", "45"}: "2026-11-15",
	} {
		days := 0
		for _, c := range date[1] {
			days = days*10 + int(c-'0')
		}
		if got := addDays(date[0], days); got != want {
			t.Errorf("%v: %s", date, got)
		}
		if back := daysBetween(date[0], want); back != days {
			t.Errorf("%v back: %d", date, back)
		}
	}
	if daysBetween("2026-03-29", "2026-03-30") != 1 || daysBetween("2026-11-02", "2026-11-01") != -1 || daysBetween("2024-01-01", "2025-01-01") != 366 {
		t.Error("calendar days")
	}
}

func TestDueTodayUntilTheLocalDateEnds(t *testing.T) {
	r, clock := openWith(t, testLimits())
	setup(t, r, "k", "America/New_York", 30)
	york, _ := time.LoadLocation("America/New_York")
	clock.set(time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC))
	must(r.Issue("ws", invoice("d", "k", "100", time.Date(2026, 10, 1, 9, 0, 0, 0, york)), "clerk"))
	must(r.Confirm("ws", "en", "USD", time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), "sync"))
	for _, c := range []struct {
		at     time.Time
		status string
		days   int
	}{
		{time.Date(2026, 10, 30, 23, 59, 59, 0, york), StatusNotDue, -1},
		{time.Date(2026, 10, 31, 0, 0, 0, 0, york), StatusDueToday, 0},
		{time.Date(2026, 10, 31, 23, 59, 59, 999999999, york), StatusDueToday, 0},
		{time.Date(2026, 11, 1, 0, 0, 0, 0, york), StatusUnknown, 1},
	} {
		clock.set(c.at)
		v := must(r.Status("ws", "d"))
		if v.Status != c.status || v.Days != c.days {
			t.Errorf("%v: %s %d", c.at, v.Status, v.Days)
		}
		if c.status != StatusUnknown && !v.Stale {
			t.Errorf("%v: confirmation of 2026-10-01 should be stale", c.at)
		}
	}
	must(r.Confirm("ws", "en", "USD", clock.read().Add(-time.Hour), "sync"))
	clock.set(time.Date(2026, 11, 1, 0, 0, 0, 0, york))
	if v := must(r.Status("ws", "d")); v.Status != StatusOverdue || v.Days != 1 || v.Stale || v.Outstanding != "100" {
		t.Errorf("overdue: %+v", v)
	}
	clock.set(time.Date(2026, 11, 2, 1, 30, 0, 0, york))
	must(r.Confirm("ws", "en", "USD", clock.read(), "sync"))
	if v := must(r.Status("ws", "d")); v.Days != 2 {
		t.Errorf("across the DST change: %d", v.Days)
	}
}
