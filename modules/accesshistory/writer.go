package accesshistory

import (
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"time"
)

func (h *History) Offer(r Record) {
	h.offerMu.Lock()
	defer h.offerMu.Unlock()
	at := h.now()
	if h.closed {
		h.afterClose++
		if h.afterClose == 1 {
			h.o.Log("access_history_offer_after_close", "count", h.afterClose)
		}
		return
	}
	r.V, r.At = formatVersion, millis(at)
	h.seq++
	r.Seq = h.seq
	if h.stopped {
		h.countDrop(r)
		return
	}
	select {
	case h.queue <- item{rec: r}:
	default:
		h.countDrop(r)
	}
}

func (h *History) countDrop(r Record) {
	if !Personal(r.Kind) {
		return
	}
	h.dropped.add(r, 1)
	h.droppedAll++
	if h.dropFrom == 0 || r.At < h.dropFrom {
		h.dropFrom = r.At
	}
	h.dropTo = max(h.dropTo, r.At)
}

func (h *History) takeDrops() (Record, bool) {
	h.offerMu.Lock()
	defer h.offerMu.Unlock()
	if h.droppedAll == 0 {
		return Record{}, false
	}
	r := Record{Kind: sysDropped, Count: h.droppedAll, From: h.dropFrom, To: h.dropTo, Per: h.dropped.encoded()}
	h.dropped, h.droppedAll, h.dropFrom, h.dropTo = lossCounts{}, 0, 0, 0
	return r, true
}

func (h *History) nextSeq() uint64 {
	h.offerMu.Lock()
	defer h.offerMu.Unlock()
	h.seq++
	return h.seq
}

func (h *History) control(f func()) {
	done := make(chan struct{})
	h.offerMu.Lock()
	closed := h.closed
	h.offerMu.Unlock()
	if closed {
		return
	}
	select {
	case h.queue <- item{control: f, done: done}:
		<-done
	case <-h.quit:
	}
}

func (h *History) Flush() {
	h.control(func() {
		if !h.stopped {
			if r, ok := h.takeDrops(); ok {
				h.writeSystem(r)
			}
		}
	})
}

func (h *History) Tick() { h.control(func() { h.tick(h.now()) }) }

func (h *History) writer() {
	defer h.wg.Done()
	timer := time.NewTicker(time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-h.quit:
			return
		case <-timer.C:
			if !h.o.Manual {
				h.tick(h.now())
			}
		case it := <-h.queue:
			h.process(it)
		}
	}
}

func (h *History) process(first item) {
	batch := []Record{}
	handle := func(it item) bool {
		if it.control != nil {
			h.writeBatch(batch)
			batch = batch[:0]
			it.control()
			close(it.done)
			return false
		}
		batch = append(batch, it.rec)
		return len(batch) < batchSize
	}
	if handle(first) {
		deadline := time.NewTimer(batchWait)
	gather:
		for {
			select {
			case it := <-h.queue:
				if !handle(it) {
					break gather
				}
			case <-deadline.C:
				break gather
			default:
				if len(h.queue) == 0 {
					break gather
				}
			}
		}
		deadline.Stop()
	}
	h.writeBatch(batch)
	if len(h.queue) == 0 && !h.stopped {
		if r, ok := h.takeDrops(); ok {
			h.writeSystem(r)
		}
		if h.afterDrain != nil {
			h.afterDrain()
		}
	}
}

func (h *History) tick(now time.Time) {
	day := now.Format(dayLayout)
	if day != h.refusedDay && h.refusedDay != "" {
		h.flushMore()
	}
	if day != h.lastPurge || ((h.overdue > 0 || h.deferred != "") && millis(now)-h.lastRetry >= time.Hour.Milliseconds()) {
		h.purgeLocked(now)
	}
}

func (h *History) flushMore() {
	h.offerMu.Lock()
	more := h.more
	source := h.moreSeq
	h.more = map[string]int64{}
	h.moreSeq = map[string]uint64{}
	h.offerMu.Unlock()
	users := make([]string, 0, len(more))
	for u := range more {
		users = append(users, u)
	}
	sort.Strings(users)
	recs := []Record{}
	for _, u := range users {
		recs = append(recs, Record{V: formatVersion, Seq: h.nextSeq(), At: millis(h.now()), User: u, Kind: KindSignInRefusedMore, Count: more[u], SourceSeq: source[u]})
	}
	h.writeBatch(recs)
}

func (h *History) writeSystem(r Record) {
	r.V, r.Seq, r.At = formatVersion, h.nextSeq(), millis(h.now())
	h.writeBatch([]Record{r})
}

func (h *History) appendDirect(r Record, now time.Time) error {
	r.V, r.Seq, r.At = formatVersion, h.nextSeq(), millis(now)
	return h.writeRecords([]Record{r})
}

func (h *History) flushPending() {
	recs := h.pendingRecs
	h.pendingRecs = nil
	for _, r := range recs {
		if err := h.appendDirect(r, h.now()); err != nil {
			h.stop(millis(h.now()), "write_failed", err)
			return
		}
	}
}

func (h *History) writeBatch(batch []Record) {
	if len(batch) == 0 {
		return
	}
	for _, r := range batch {
		h.writtenSeq = max(h.writtenSeq, r.Seq)
	}
	if h.stopped {
		h.offerMu.Lock()
		for _, r := range batch {
			if Personal(r.Kind) {
				h.countDrop(r)
			}
		}
		h.offerMu.Unlock()
		return
	}
	kept := batch[:0:0]
	for _, r := range batch {
		if r.Kind == KindSignInRefused {
			day := fromMillis(r.At).Format(dayLayout)
			if day != h.refusedDay {
				if h.refusedDay != "" {
					h.offerMu.Lock()
					pending := len(h.more) > 0
					h.offerMu.Unlock()
					if pending {
						h.flushMore()
					}
				}
				h.offerMu.Lock()
				h.refusedDay, h.refused = day, map[string]int{}
				h.offerMu.Unlock()
			}
			h.mu.RLock()
			previous := r.Seq <= h.floors[r.User]
			skip := false
			if !previous {
				h.offerMu.Lock()
				h.refused[r.User]++
				if h.refused[r.User] > refusedCapPerDay {
					h.more[r.User]++
					if h.moreSeq[r.User] == 0 || r.Seq < h.moreSeq[r.User] {
						h.moreSeq[r.User] = r.Seq
					}
					skip = true
				}
				h.offerMu.Unlock()
			}
			h.mu.RUnlock()
			if skip {
				continue
			}
		}
		kept = append(kept, r)
	}
	if err := h.writeRecords(kept); err != nil {
		h.stop(millis(h.now()), h.stopReasonOf(err), err)
		h.mu.Lock()
		for _, r := range kept {
			if r.Kind == sysDropped {
				h.addLossPer("dropped", r.From, r.To, r.Per, r.Count)
			}
		}
		h.mu.Unlock()
		h.offerMu.Lock()
		for _, r := range kept {
			if Personal(r.Kind) {
				h.countDrop(r)
			}
		}
		h.offerMu.Unlock()
	}
}

var errFull = errors.New("accesshistory: size limit reached")
var errFullHeld = errors.New("accesshistory: size limit reached and the oldest days are held")

func (h *History) stopReasonOf(err error) string {
	switch {
	case errors.Is(err, errFull):
		return "full"
	case errors.Is(err, errFullHeld):
		return "full_held"
	}
	return "write_failed"
}

func (h *History) stop(at int64, reason string, err error) {
	h.offerMu.Lock()
	h.stopped, h.stopSince, h.stopReason = true, at, reason
	h.offerMu.Unlock()
	h.o.Log("access_history_stopped", "reason", reason, "error", errClass(err))
	if reason != "write_failed" {
		_ = h.appendRaw(Record{V: formatVersion, Seq: h.nextSeq(), At: at, Kind: sysRecordingStop, Run: reason})
	}
}

func errClass(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, errFull), errors.Is(err, errFullHeld):
		return "bound"
	case errors.Is(err, fs.ErrPermission):
		return "permission"
	}
	return "io"
}

func (h *History) writeRecords(recs []Record) error {
	recs = h.splitLossRecords(recs)
	for _, r := range recs {
		h.writtenSeq = max(h.writtenSeq, r.Seq)
	}
	if len(recs) == 0 {
		return nil
	}
	h.mu.Lock()
	for i := range recs {
		r := &recs[i]
		if Personal(r.Kind) {
			if !recordVisible(*r, h.floors[r.User]) {
				r.N = max(r.N, 1)
				continue
			}
			s := h.personOf(r.User).seqOf(sequenceKey(*r))
			s.next++
			r.N = s.next
		}
	}
	h.mu.Unlock()
	byDay := map[string][]byte{}
	var size int64
	for _, r := range recs {
		line, err := encodeLine(r)
		if err != nil {
			return err
		}
		name := fromMillis(r.At).Format(dayLayout) + daySuffix
		byDay[name] = append(byDay[name], line...)
		size += int64(len(line))
	}
	if err := h.makeRoom(size, byDay); err != nil {
		return err
	}
	names := make([]string, 0, len(byDay))
	for name := range byDay {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := h.appendDay(name, byDay[name]); err != nil {
			return err
		}
	}
	h.closeOthers(names[len(names)-1])
	h.mu.Lock()
	cut := h.cutoff(h.now())
	for _, r := range recs {
		h.lastAt = max(h.lastAt, r.At)
		switch r.Kind {
		case sysDropped:
			h.addLossPer("dropped", r.From, r.To, r.Per, r.Count)
		case sysRecoveredTail:
			h.addLoss("recovered_tail", r.From, r.To, "", 0)
		}
		if !Personal(r.Kind) || !recordVisible(r, h.floors[r.User]) || r.At < cut {
			continue
		}
		s := h.personOf(r.User).seqOf(sequenceKey(r))
		s.entries = append(s.entries, h.entryOf(r))
	}
	h.mu.Unlock()
	if len(h.pendingRecs) > 0 {
		h.flushPending()
	}
	return nil
}

func (h *History) appendRaw(r Record) error {
	line, err := encodeLine(r)
	if err != nil {
		return err
	}
	name := fromMillis(r.At).Format(dayLayout) + daySuffix
	return h.appendDay(name, line)
}

func (h *History) appendDay(name string, data []byte) error {
	f := h.files[name]
	if f == nil {
		path := filepath.Join(h.dir, name)
		var err error
		f, err = h.sys.openFile(path, false)
		if errors.Is(err, fs.ErrNotExist) {
			f, err = h.sys.openFile(path, true)
			if err == nil {
				err = h.sys.syncDir(h.dir)
			}
		}
		if err != nil {
			if f != nil {
				f.Close()
			}
			return err
		}
		h.files[name] = f
	}
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	h.sizes[name] += int64(len(data))
	h.total += int64(len(data))
	return nil
}

func (h *History) closeOthers(keep string) {
	for name, f := range h.files {
		if name != keep {
			f.Close()
			delete(h.files, name)
		}
	}
}

func (h *History) closeFile(name string) {
	if f := h.files[name]; f != nil {
		f.Close()
		delete(h.files, name)
	}
}

func (h *History) Close() error {
	h.offerMu.Lock()
	if h.closed {
		h.offerMu.Unlock()
		return nil
	}
	h.offerMu.Unlock()
	done := make(chan struct{})
	go func() {
		h.control(func() {})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(drainWait):
	}
	h.offerMu.Lock()
	h.closed = true
	h.offerMu.Unlock()
	close(h.quit)
	h.wg.Wait()
	for {
		select {
		case it := <-h.queue:
			if it.control != nil {
				close(it.done)
				continue
			}
			h.offerMu.Lock()
			h.countDrop(it.rec)
			h.offerMu.Unlock()
			continue
		default:
		}
		break
	}
	if !h.stopped {
		if r, ok := h.takeDrops(); ok {
			_ = h.appendDirect(r, h.now())
		}
		h.flushMore()
		_ = h.appendDirect(Record{Kind: sysRunStopped, Run: h.run}, h.now())
	}
	h.stateMu.Lock()
	err := h.saveState()
	h.stateMu.Unlock()
	h.release()
	return err
}
