package accesshistory

import (
	"errors"
	"strings"
	"time"
)

var errDeletionDeferred = errors.New("accesshistory: deletion awaits clock acceptance")

func (h *History) deletionDeferred() bool {
	reason := ""
	if h.state.Deferral != nil {
		reason = "clock_jump"
	} else if h.o.ClockUnverified() {
		reason = "clock_unverified"
	}
	h.mu.Lock()
	previous := h.deferred
	h.deferred = reason
	h.mu.Unlock()
	if reason != "" && previous != reason {
		h.pendingRecs = append(h.pendingRecs, Record{Kind: sysPurgeDeferred, Run: reason})
		h.o.Log("access_history_purge_deferred", "reason", reason)
	}
	return reason != ""
}

func (h *History) persistHolds(live []Hold, now time.Time) error {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	h.observeHolds(live, now)
	return h.saveState()
}

func (h *History) persistSplitMarkers(name string, sp split) error {
	h.stateMu.Lock()
	defer h.stateMu.Unlock()
	for i := range h.state.Markers {
		m := &h.state.Markers[i]
		day := strings.TrimSuffix(name, daySuffix)
		if sp.days[m.ID] && !contains(m.Days, day) {
			m.Days = append(m.Days, day)
		}
		if len(sp.damaged) > 0 {
			dname := day + heldDamagedSuffix
			if !contains(m.HeldDamaged, dname) {
				m.HeldDamaged = append(m.HeldDamaged, dname)
			}
		}
	}
	return h.saveState()
}

func (h *History) publishMaintenance(failed bool, now time.Time) {
	effective := h.coverage()
	h.mu.Lock()
	if failed {
		if h.overdue == 0 {
			h.overdue = millis(now)
		}
	} else {
		h.overdue = 0
	}
	h.effective = effective
	h.mu.Unlock()
}
