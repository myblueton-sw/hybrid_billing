package server

import (
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/jobs"
)

const (
	cleanupKind   = "bulk_revocation_cleanup"
	cleanupWorker = "system:bulk-cleanup"
	cleanupChunk  = 100
)

var cleanupPolicy = jobs.Policy{Lease: 30 * time.Second, Life: time.Hour, Attempts: 3, BackoffBase: time.Second, BackoffCap: 10 * time.Second}

func cleanupLimits() jobs.Limits {
	return jobs.Limits{Jobs: 100000, Tasks: 100, Attempts: 3, PerType: 1, PerWorkspace: 1, Batch: 1}
}

func bulkCleanupItems(restarted bool) []bulkrevoke.Cleanup {
	reason := "sessions_ended"
	if restarted {
		reason = "server_restarted"
	}
	return []bulkrevoke.Cleanup{
		{Item: "B1", State: accesschange.ItemDone, Reason: "deny_applied"},
		{Item: "B2", State: accesschange.ItemDone, Reason: reason},
		{Item: "B3", State: accesschange.ItemDone, Reason: reason},
		{Item: "B4", State: accesschange.ItemDone, Reason: reason},
		{Item: "B5", State: accesschange.ItemNotApplicable, Reason: "links_give_no_access_while_suspended"},
		{Item: "B6", State: accesschange.ItemNotApplicable, Reason: "people_hold_no_api_keys"},
		{Item: "B7", State: accesschange.ItemOutside, Reason: "service_users_not_changed"},
		{Item: "B8", State: accesschange.ItemOutside, Reason: "in_flight_until_restart"},
		{Item: "B9", State: accesschange.ItemOutside, Reason: "no_person_jobs"},
		{Item: "B10", State: accesschange.ItemOutside, Reason: "cannot_recall"},
	}
}

func (s *Server) startCleanup(v bulkrevoke.View) {
	tasks := []jobs.TaskSpec{}
	first, n := -1, 0
	flush := func(last int) {
		tasks = append(tasks, jobs.TaskSpec{ID: "chunk-" + strconv.Itoa(len(tasks)+1), Input: strconv.Itoa(first) + ":" + strconv.Itoa(last), Repeatable: true})
		first, n = -1, 0
	}
	current, _ := s.bulk.Get(v.Workspace, v.Owner, v.ID)
	last := -1
	for _, item := range current.Items {
		if item.State != bulkrevoke.DenyConfirmed {
			continue
		}
		if first < 0 {
			first = item.Index
		}
		last, n = item.Index, n+1
		if n == cleanupChunk {
			flush(last)
		}
	}
	if first >= 0 {
		flush(last)
	}
	if len(tasks) == 0 {
		return
	}
	_, err := s.jobs.Create(v.Workspace, v.ID, cleanupKind, cleanupPolicy, tasks)
	if s.faulty("cleanup-create") || err != nil {
		s.log.Error("bulk_cleanup_not_started", "bulk", v.ID)
		_ = s.bulk.Cleanup(v.Workspace, v.ID, false)
		return
	}
	_ = s.bulk.Cleanup(v.Workspace, v.ID, true)
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Server) drainCleanup() int {
	done := 0
	for {
		s.domain.Lock()
		leases, err := s.jobs.Lease(cleanupWorker, []string{cleanupKind}, 1)
		if err != nil || len(leases) == 0 {
			s.domain.Unlock()
			return done
		}
		s.cleanTask(leases[0])
		s.domain.Unlock()
		done++
	}
}

func taskSpan(input string) (int, int) {
	a, b, _ := strings.Cut(input, ":")
	from, _ := strconv.Atoi(a)
	to, _ := strconv.Atoi(b)
	return from, to
}

func (s *Server) cleanTask(lease jobs.Lease) {
	ref := lease.Ref()
	if _, err := s.jobs.Extend(ref); err != nil {
		return
	}
	v, found := s.bulk.Get(lease.WorkspaceID, "", lease.JobID)
	if !found {
		_, _ = s.jobs.Fail(ref, "job_missing", true)
		return
	}
	from, to := taskSpan(lease.Input)
	if lease.Checkpoint != "" {
		done, _ := strconv.Atoi(lease.Checkpoint)
		from = done + 1
	}
	checkpoint := lease.Checkpoint
	for index := from; index <= to && index < len(v.Items); index++ {
		item := v.Items[index]
		if item.State != bulkrevoke.DenyConfirmed && item.State != bulkrevoke.CleanupRunning {
			continue
		}
		_ = s.bulk.Cleaning(v.Workspace, v.ID, index)
		if s.faulty("cleanup") {
			report, _ := s.jobs.Fail(ref, "cleanup_failed", false)
			if report.State == jobs.TaskFailed {
				for i := index; i <= to && i < len(v.Items); i++ {
					_ = s.bulk.Failed(v.Workspace, v.ID, i, "retry_budget_spent")
				}
			}
			return
		}
		s.cleanPerson(item.Target.User, v.DenyAt)
		_ = s.bulk.Cleaned(v.Workspace, v.ID, index, bulkCleanupItems(false))
		checkpoint = strconv.Itoa(index)
		_ = s.jobs.Checkpoint(ref, checkpoint)
	}
	_, _ = s.jobs.Complete(ref, checkpoint)
}

func (s *Server) cleanPerson(user string, denyAt time.Time) {
	for _, v := range s.sessionsOf(user) {
		found, ok := s.sessions.get(v)
		if !ok || found.created.After(denyAt) {
			continue
		}
		_ = s.identity.Boundary().EndSession(v)
		s.sessions.drop(v)
		s.dir.dropPickups(func(p pickup) bool { return p.session == v })
	}
	s.dir.dropPickups(func(p pickup) bool { return p.target == user })
}

func (s *Server) cleaning(v bulkrevoke.View) bulkrevoke.Cleaning {
	job, err := s.jobs.Job(v.Workspace, v.ID)
	if err != nil {
		return bulkrevoke.Cleaning{}
	}
	return bulkrevoke.Cleaning{State: string(job.State), Reason: string(job.Reason)}
}

func (s *Server) cleanupProgress(v bulkrevoke.View) (int, int) {
	done, total := 0, 0
	for _, item := range v.Items {
		if item.Outcome == bulkrevoke.OutcomeExcluded || item.Outcome == bulkrevoke.OutcomeHeld {
			continue
		}
		total++
		if item.Outcome == bulkrevoke.OutcomeSucceeded || item.Outcome == bulkrevoke.OutcomeFailed {
			done++
		}
	}
	return done, total
}

func (s *Server) runWorker() {
	if s.manual {
		return
	}
	go every(cleanupEvery, s.stop, s.wake, func() { s.drainCleanup() })
	go every(sweepEvery, s.stop, nil, func() {
		s.domain.Lock()
		s.sweepRoles()
		s.domain.Unlock()
	})
}
