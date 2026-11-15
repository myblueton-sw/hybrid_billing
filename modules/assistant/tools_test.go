package assistant

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestNewRegistry(t *testing.T) {
	clock := func() time.Time { return start }
	good := Config{Commands: &fake{}, Client: "asst", Protected: []string{"account"}}
	if _, err := NewRegistry(good, limits, clock); err != nil {
		t.Fatal(err)
	}
	expect(t, "no clock", second(NewRegistry(good, limits, nil)), ErrInvalid)
	for name, config := range map[string]Config{
		"no commands":           {Client: "asst", Protected: []string{"account"}},
		"no client":             {Commands: &fake{}, Protected: []string{"account"}},
		"a client that is text": {Commands: &fake{}, Client: "a b", Protected: []string{"account"}},
		"no protected kinds":    {Commands: &fake{}, Client: "asst"},
		"a protected free text": {Commands: &fake{}, Client: "asst", Protected: []string{"account", "a b"}},
	} {
		expect(t, name, second(NewRegistry(config, limits, clock)), ErrInvalid)
	}
	counts := map[string]struct {
		field func(*Limits) *int
		max   int
	}{
		"workspaces":    {func(l *Limits) *int { return &l.Workspaces }, maxWorkspaces},
		"tools":         {func(l *Limits) *int { return &l.Tools }, maxTools},
		"jobs":          {func(l *Limits) *int { return &l.Jobs }, maxJobs},
		"conversations": {func(l *Limits) *int { return &l.Conversations }, maxConversations},
		"cards":         {func(l *Limits) *int { return &l.Cards }, maxCards},
		"calls":         {func(l *Limits) *int { return &l.Calls }, maxCalls},
		"targets":       {func(l *Limits) *int { return &l.Targets }, maxTargets},
		"fields":        {func(l *Limits) *int { return &l.Fields }, maxFields},
		"bytes":         {func(l *Limits) *int { return &l.Bytes }, maxBytes},
	}
	for name, c := range counts {
		for value, ok := range map[int]bool{0: false, 1: true, c.max: true, c.max + 1: false} {
			l := limits
			*c.field(&l) = value
			_, err := NewRegistry(good, l, clock)
			if ok != (err == nil) || !ok && err != ErrInvalid {
				t.Errorf("%s %d: %v", name, value, err)
			}
		}
	}
	durations := map[string]struct {
		field    func(*Limits) *time.Duration
		min, max time.Duration
	}{
		"duration":  {func(l *Limits) *time.Duration { return &l.Duration }, minDuration, maxDuration},
		"retention": {func(l *Limits) *time.Duration { return &l.Retention }, minRetention, maxRetention},
	}
	for name, c := range durations {
		for value, ok := range map[time.Duration]bool{c.min - 1: false, c.min: true, c.max: true, c.max + 1: false} {
			l := limits
			*c.field(&l) = value
			_, err := NewRegistry(good, l, clock)
			if ok != (err == nil) {
				t.Errorf("%s %v: %v", name, value, err)
			}
		}
	}
}

func TestToolForms(t *testing.T) {
	f := setup(t)
	good := Tool{Command: "statement.read", Stage: StageRead, Results: []string{"total"}}
	many := make([]string, maxFields+1)
	for index := range many {
		many[index] = fmt.Sprintf("f%d", index)
	}
	for name, c := range map[string]struct {
		workspace string
		tool      Tool
		actor     Actor
	}{
		"no workspace":          {"", good, ann},
		"an empty actor":        {"w1", good, Actor{Kind: KindHuman}},
		"an unknown actor kind": {"w1", good, Actor{ID: "ann", Kind: "robot"}},
		"a command as text":     {"w1", Tool{Command: "a b", Stage: StageRead}, ann},
		"an unknown stage":      {"w1", Tool{Command: "statement.read", Stage: "write"}, ann},
		"a repeated result":     {"w1", Tool{Command: "statement.read", Stage: StageRead, Results: []string{"a", "a"}}, ann},
		"a result as text":      {"w1", Tool{Command: "statement.read", Stage: StageRead, Results: []string{"a b"}}, ann},
		"too many results":      {"w1", Tool{Command: "statement.read", Stage: StageRead, Results: many}, ann},
	} {
		expect(t, name, second(f.r.ProposeTool(c.workspace, c.tool, c.actor)), ErrInvalid)
	}
	if _, err := f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead, Results: many[:maxFields]}, ann); err != nil {
		t.Errorf("the maximum number of results: %v", err)
	}
	reason(t, "an assistant proposing", second(f.r.ProposeTool("w1", good, bot)), ReasonAINotAllowed)
	reason(t, "a system proposing", second(f.r.ProposeTool("w1", good, sweeper)), ReasonNotHuman)
	expect(t, "an uncatalogued command", second(f.r.ProposeTool("w1", Tool{Command: "statement.nothing", Stage: StageRead}, ann)), ErrNotFound)
	for _, id := range []string{"x", strings.Repeat("a", maxIDBytes), "A.z_0:-9", "Z", "z", "9"} {
		if !validID(id) {
			t.Errorf("%q is refused", id)
		}
	}
	for _, id := range []string{"", strings.Repeat("a", maxIDBytes+1), "a/b", "a b", "é", "a@", "[", "`", "{"} {
		if validID(id) {
			t.Errorf("%q is accepted", id)
		}
	}
}

func TestStages(t *testing.T) {
	f := setup(t)
	allowed := map[string][]string{
		StageRead:     {"statement.read"},
		StageDraft:    {"statement.draft", "statement.simulate"},
		StageSimulate: {"statement.draft", "statement.simulate"},
		StageRequest:  {"statement.read", "statement.cost", "statement.draft", "statement.issue", "statement.approve", "statement.download"},
	}
	refused := map[string][]string{
		StageRead:     {"statement.cost", "statement.draft", "statement.simulate", "statement.issue", "statement.approve", "statement.download"},
		StageDraft:    {"statement.read", "statement.cost", "statement.issue", "statement.approve", "statement.download"},
		StageSimulate: {"statement.read", "statement.cost", "statement.issue", "statement.approve", "statement.download"},
	}
	for stage, list := range allowed {
		for _, command := range list {
			view, err := f.r.ProposeTool("w1", Tool{Command: command, Stage: stage}, ann)
			if err != nil || view.State != StateProposed {
				t.Errorf("%s at %s: %v", command, stage, err)
			}
			f.must(second(f.r.WithdrawTool("w1", command, ann)))
		}
	}
	for stage, list := range refused {
		for _, command := range list {
			reason(t, command+" at "+stage, second(f.r.ProposeTool("w1", Tool{Command: command, Stage: stage}, ann)), ReasonStage)
		}
	}
	for _, kind := range protected {
		for _, stage := range []string{StageRead, StageDraft, StageSimulate, StageRequest} {
			reason(t, kind+" at "+stage, second(f.r.ProposeTool("w1", Tool{Command: kind + ".change", Stage: stage}, ann)), ReasonProtected)
		}
	}
	reason(t, "a command without a handler", second(f.r.ProposeTool("w1", Tool{Command: "statement.planned", Stage: StageRequest}, ann)), ReasonUnsupported)
}

func TestToolLifecycle(t *testing.T) {
	f := setup(t)
	tool := Tool{Command: "statement.read", Stage: StageRead, Results: []string{"total", "name"}}
	view, err := f.r.ProposeTool("w1", tool, ann)
	f.must(err)
	if view.Revision != 1 || view.State != StateProposed || view.Proposer != "ann" || view.Approver != "" || strings.Join(view.Tool.Results, ",") != "name,total" {
		t.Errorf("proposal %+v", view)
	}
	view.Tool.Results[0] = "changed"
	tool.Results[0] = "changed-too"
	same, err := f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead, Results: []string{"name", "total"}}, ann)
	if err != nil || strings.Join(same.Tool.Results, ",") != "name,total" || same.Revision != 1 {
		t.Errorf("the repeated proposal %+v %v", same, err)
	}
	expect(t, "another proposer while pending", second(f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead, Results: []string{"name", "total"}}, bob)), ErrConflict)
	expect(t, "another tool while pending", second(f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead}, ann)), ErrConflict)
	expect(t, "another stage while pending", second(f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRequest, Results: []string{"name", "total"}}, ann)), ErrConflict)
	for name, c := range map[string]struct{ workspace, command string }{"no workspace": {"", "statement.read"}, "a command as text": {"w1", "a b"}} {
		expect(t, "approve "+name, second(f.r.ApproveTool(c.workspace, c.command, bob)), ErrInvalid)
		expect(t, "withdraw "+name, second(f.r.WithdrawTool(c.workspace, c.command, bob)), ErrInvalid)
	}
	expect(t, "approve an invalid actor", second(f.r.ApproveTool("w1", "statement.read", Actor{ID: "bob"})), ErrInvalid)
	expect(t, "withdraw an invalid actor", second(f.r.WithdrawTool("w1", "statement.read", Actor{ID: "bob"})), ErrInvalid)
	reason(t, "an assistant approving", second(f.r.ApproveTool("w1", "statement.read", bot)), ReasonAINotAllowed)
	reason(t, "a system approving", second(f.r.ApproveTool("w1", "statement.read", sweeper)), ReasonNotHuman)
	reason(t, "an assistant withdrawing", second(f.r.WithdrawTool("w1", "statement.read", bot)), ReasonAINotAllowed)
	expect(t, "approve in an unknown workspace", second(f.r.ApproveTool("w9", "statement.read", bob)), ErrNotFound)
	expect(t, "approve an unknown tool", second(f.r.ApproveTool("w1", "statement.issue", bob)), ErrNotFound)
	expect(t, "withdraw in an unknown workspace", second(f.r.WithdrawTool("w9", "statement.read", bob)), ErrNotFound)
	expect(t, "withdraw an unknown tool", second(f.r.WithdrawTool("w1", "statement.issue", bob)), ErrNotFound)
	reason(t, "the proposer approving", second(f.r.ApproveTool("w1", "statement.read", ann)), ReasonSelfApproval)
	active, err := f.r.ApproveTool("w1", "statement.read", bob)
	if err != nil || active.State != StateActive || active.Approver != "bob" || active.Revision != 1 {
		t.Errorf("approval %+v %v", active, err)
	}
	if again, err := f.r.ApproveTool("w1", "statement.read", bob); err != nil || again.State != StateActive {
		t.Errorf("the repeated approval %+v %v", again, err)
	}
	expect(t, "another approver of an active tool", second(f.r.ApproveTool("w1", "statement.read", cat)), ErrConflict)
	gone, err := f.r.WithdrawTool("w1", "statement.read", cat)
	if err != nil || gone.State != StateWithdrawn {
		t.Errorf("withdrawal %+v %v", gone, err)
	}
	if again, err := f.r.WithdrawTool("w1", "statement.read", ann); err != nil || again.State != StateWithdrawn {
		t.Errorf("the repeated withdrawal %+v %v", again, err)
	}
	reason(t, "approving a withdrawn tool", second(f.r.ApproveTool("w1", "statement.read", bob)), ReasonNotPending)
	again, err := f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead}, bob)
	if err != nil || again.Revision != 2 || again.Proposer != "bob" || again.Approver != "" {
		t.Errorf("a new exposure %+v %v", again, err)
	}
	f.must(second(f.r.ApproveTool("w1", "statement.read", ann)))
	if same, err := f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead}, cat); err != nil || same.Revision != 2 || same.State != StateActive {
		t.Errorf("an identical proposal of an active tool %+v %v", same, err)
	}
	changed, err := f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead, Results: []string{"total"}}, ann)
	if err != nil || changed.Revision != 3 || changed.State != StateProposed {
		t.Errorf("a change of an active tool %+v %v", changed, err)
	}
}

func TestToolRoom(t *testing.T) {
	l := limits
	l.Tools, l.Workspaces = 1, 1
	f := setupWith(t, l)
	f.must(second(f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead}, ann)))
	expect(t, "a second tool", second(f.r.ProposeTool("w1", Tool{Command: "statement.issue", Stage: StageRequest}, ann)), ErrLimit)
	f.must(second(f.r.WithdrawTool("w1", "statement.read", ann)))
	f.must(second(f.r.ProposeTool("w1", Tool{Command: "statement.read", Stage: StageRead}, ann)))
	expect(t, "a second workspace", second(f.r.ProposeTool("w2", Tool{Command: "statement.read", Stage: StageRead}, ann)), ErrLimit)
	expect(t, "a stage refusal comes before room", second(f.r.ProposeTool("w2", Tool{Command: "statement.issue", Stage: StageRead}, ann)), ErrRejected)
}

func TestCatalogueChanges(t *testing.T) {
	read := commands.Entry{ID: "x.read", Permission: commands.Permission{Action: "view", Kind: "statement"}, Supported: true}
	c := &fake{entries: []commands.Entry{read}}
	now := start
	r, err := NewRegistry(Config{Commands: c, Client: "asst", Protected: protected}, limits, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.ProposeTool("w1", Tool{Command: "x.read", Stage: StageRead}, ann); err != nil {
		t.Fatal(err)
	}
	c.entries = nil
	expect(t, "approving an uncatalogued command", second(r.ApproveTool("w1", "x.read", bob)), ErrNotFound)
	changed := read
	changed.Mutating = true
	c.entries = []commands.Entry{changed}
	reason(t, "approving a command whose definition changed", second(r.ApproveTool("w1", "x.read", bob)), ReasonStage)
	c.entries = []commands.Entry{read}
	if _, err := r.ApproveTool("w1", "x.read", bob); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateJob("w1", Job{ID: "j1"}, ann); err != nil {
		t.Fatal(err)
	}
	if _, err := r.OpenConversation("w1", Conversation{ID: "c1", Owner: "ann", Expires: now.Add(time.Hour)}, ann); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Attach("w1", "j1", "c1", ann); err != nil {
		t.Fatal(err)
	}
	f := setup(t)
	answer := f.answer("ann")
	c.entries = nil
	reason(t, "invoking an uncatalogued command", second(r.Invoke("w1", "j1", Invocation{Conversation: "c1", Tool: "x.read", Target: "t1", Answer: answer}, bot)), ReasonNotExposed)
	c.entries = []commands.Entry{changed}
	reason(t, "invoking a command whose definition changed", second(r.Invoke("w1", "j1", Invocation{Conversation: "c1", Tool: "x.read", Target: "t1", Answer: answer}, bot)), ReasonStage)
	if len(c.calls) != 0 {
		t.Errorf("a refused tool reached the executor")
	}
}
