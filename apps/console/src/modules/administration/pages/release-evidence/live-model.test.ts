import { currentScope } from "./live-model";
import { refusalDiagnostics } from "./live-model";
import type { RowDiagnostic } from "../../../../service/command-diagnostics";
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { createElement, type ComponentType } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import type { AdminClient } from "../../../../service/admin";
import { evidenceOf, proposalOf, requirementOf, requirementsAnswerOf, requirementsPath, type EvidencePage, type RequirementRow } from "../../../../service/release-evidence";
import { AdminError, adminClient } from "../../../../service/admin";
import { sample } from "../../../../service/release-evidence.sample";
import {
  approveUsable, blockerPolicy, buildsOf, cancelled, deniedOutcome, differences, draftOf, duplicateIn, fieldReaction, forget, frozenTargetOk, goneKind, hadEarlier, keepDraft, keepsAttempt,
  keptKey, modeOf, pairOf, pendingFor, pendingNow, pendingRetire, profileRetire, profileTarget, proposalTarget, pruneRefusals, react, readDenied, readSucceeded, refusedNow, renew,
  replacementOptions, replacementProblem, requirementRetire, requirementTarget, retireValues, revokedNow, selectionOf, setScope, settle as settleRecorded, snapshotOf, splitWidth, stateSentence, textProblem,
  utf8Bytes, vanished, rowId, closedProposal, verdictLine, viewOf, withdrawUsable,
  filtering, missingProfiles, missingRequirements, noFilter, profileById, profileTargetsInUse, withHeldProfiles, requirementByLabel, requirementByTarget, requirementByTest, requirementTargetsInUse, stateAtCandidate, stillAllowed, withHeld,
} from "./live-model";
import { blockerText, f, k } from "./live-copy";
import { allWordTables, category, counted, failingVerdicts, plurals, verdictWords } from "./live-codes";
import { Free, href } from "./live-format";
import type { Shared } from "./LiveCommand";
import { MatrixView } from "./LiveMatrix";
import { RequirementsView, RetireImpact } from "./LiveRequirements";
import { LoadView, ProfileImpact } from "./LiveLoad";
import { DecisionsView } from "./LiveDecisions";
import { ArtifactsView } from "./LiveArtifacts";
import { retireFields } from "./LiveRetireForm";

const locales = ["en", "ko", "ar"] as const;
type Raw = Record<string, any>;

function page(mut: (s: Raw) => void = () => {}): EvidencePage {
  const s = JSON.parse(JSON.stringify(sample)) as Raw;
  mut(s);
  return evidenceOf(s);
}

function shared(p: EvidencePage, over: Partial<Shared> = {}): Shared {
  return {
    l: "en", page: p, api: {} as AdminClient, me: "sample-admin", canCommand: p.revision !== "", mode: "split",
    selection: { path: "", requirement: "", from: "", to: "", build: "", profile: "", proposal: "" },
    onDone: () => {}, onNotice: () => {}, onReload: async () => null, onSignedOut: () => {}, ...over,
  };
}

function html(view: ComponentType<Shared>, p: EvidencePage, over: Partial<Shared> = {}): string {
  return renderToStaticMarkup(createElement(view, shared(p, over))).replace(/<!-- -->/g, "").replace(/<\/span><wbr\/><span class="p05-id-seg">/g, "").replace(/[\u2066-\u2069]/g, "");
}

const text = (h: string) => h.replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&").replace(/\s+/g, " ").replace(/\( /g, "(").replace(/ \)/g, ")").replace(/ ([.,:])/g, "$1");
const unexpected = "Unexpected value from the server";
const xlsx = (s: Raw) => s.matrix.paths.rows.find((x: Raw) => x.id === "sample-path-xlsx");
const req = (s: Raw, id: string) => s.requirements.requirements.rows.find((x: Raw) => x.id === id);
const prof = (s: Raw, id: string) => s.load.profiles.rows.find((x: Raw) => x.id === id);

test("views fall back to the matrix", () => {
  assert.equal(viewOf("load"), "load");
  assert.equal(viewOf("nope"), "matrix");
});

test("the split mode follows the measured width, with the same number in JS and CSS", () => {
  assert.equal(modeOf(939.99), "step");
  assert.equal(modeOf(940), "split");
  assert.equal(modeOf(940.4), "split");
  assert.equal(modeOf(0), "step");
  assert.equal(modeOf(Number.NaN), "step");
  assert.equal(modeOf(1120), "split");
  assert.equal(modeOf(358), "step");
  const css = readFileSync(new URL("./live.css", import.meta.url), "utf8");
  const m = /@container \(min-width: (\d+)px\) \{\s*\.p05-split\[data-open="true"\]/.exec(css);
  assert.equal(Number(m?.[1]), splitWidth);
});

test("rule 1: the gate and readiness never read ready", () => {
  const a = text(html(RequirementsView, page((s) => { s.requirements.gate.ready = true; })));
  assert.ok(a.includes(unexpected + " (gate.ready)"));
  assert.ok(!/not ready, 5 problems/.test(a));
  const b = text(html(LoadView, page((s) => { s.load.readiness.ready = true; })));
  assert.ok(b.includes(unexpected + " (readiness.ready)"));
  assert.ok(!b.includes("not ready"));
});

test("rule 1: an active path or a path build fails closed", () => {
  for (const mut of [(s: Raw) => { xlsx(s).state = "active"; }, (s: Raw) => { xlsx(s).build = "0.0.2"; }]) {
    const t = text(html(MatrixView, page(mut)));
    assert.ok(t.includes(unexpected + " (path.state)") || t.includes(unexpected + " (path.build)"));
    assert.equal((t.match(/Designed, not active/g) ?? []).length, 1);
  }
});

test("rule 1: customer proven, passed verdicts and statuses, approved budgets", () => {
  const a = text(html(RequirementsView, page((s) => { req(s, "sample-approval").states["0.0.2"] = "customer_proven"; })));
  assert.ok(a.includes(unexpected + " (requirement.state)"));
  const run = text(html(LoadView, page((s) => { prof(s, "sample-e2e").runs.rows[0].verdict = "passed"; })));
  assert.ok(run.includes(unexpected + " (run.verdict)"));
  const status = text(html(LoadView, page((s) => { s.load.readiness.gates[0].status = "passed"; prof(s, "sample-e2e").status = "passed"; })));
  assert.ok(status.includes(unexpected + " (gate.status)") && status.includes(unexpected + " (run.verdict)"));
  assert.ok(!/\bPassed\b/.test(status));
  const approved = text(html(LoadView, page((s) => { prof(s, "sample-e2e").active_budget.state = "approved"; })));
  assert.ok(approved.includes(unexpected + " (budget.state)"));
  for (const key of ["approval_by", "approved_by", "approved_at"]) {
    const t = text(html(LoadView, page((s) => { prof(s, "sample-e2e").active_budget[key] = key === "approved_at" ? "2026-11-17T00:00:00Z" : { id: "a", login: "a@example.invalid", directory: true }; })));
    assert.ok(t.includes(unexpected + " (budget.approval)"), key);
  }
});

test("rule 1: customer proof above 0, templates, empty blockers and path_build", () => {
  const proof = text(html(MatrixView, page((s) => { xlsx(s).columns.customer_proof.count = 1; })));
  assert.ok(proof.includes(unexpected + " (customer_proof.count)"));
  assert.ok(!proof.includes("1 of 2 Intake"));
  const tpl = text(html(RequirementsView, page((s) => { s.requirements.gate.templates = ["sample-template-xlsx"]; })));
  assert.ok(tpl.includes(unexpected + " (gate.templates)"));
  assert.ok(!tpl.includes("None. A template"));
  const empty = text(html(MatrixView, page((s) => { xlsx(s).blockers = []; })));
  assert.ok(empty.includes(unexpected + " (path.blockers)"));
  const reduced = text(html(MatrixView, page((s) => { xlsx(s).blockers = []; }), { selection: { path: "sample-path-api", requirement: "", from: "", to: "", build: "", profile: "", proposal: "" } }));
  assert.ok(reduced.includes(unexpected + " (path.blockers)"));
  assert.ok(!/\b0 problems\b/.test(reduced));
  assert.ok(reduced.includes("4 problems"));
  const nullBlockers = text(html(MatrixView, page((s) => { xlsx(s).blockers = null; })));
  assert.ok(nullBlockers.includes("Unknown") && !nullBlockers.includes(unexpected));
  const build = text(html(MatrixView, page((s) => { xlsx(s).blockers.push({ problem: "path_build", subject: "sample-path-xlsx" }); })));
  assert.ok(build.includes(unexpected + " (path.blockers)"));
  const gate = text(html(RequirementsView, page((s) => { s.requirements.gate.problems.rows.push({ problem: "path_build", subject: "x" }); })));
  assert.ok(gate.includes(unexpected + " (path.blockers)"));
});

test("rule 1: applied activation, customer proof and budget approval proposals", () => {
  const p = page((s) => {
    s.decisions.proposals.rows.push(
      { module: "release_trace", number: 30, target: "trace-proposal-30", kind: "activate", state: "applied", summary: { path: "sample-path-xlsx", build: "0.0.2" }, closing_rationale: "yes" },
      { module: "release_trace", number: 31, target: "trace-proposal-31", kind: "proof", state: "applied", summary: { label: "sample/sample-approval", build: "0.0.2", period: "2026-10" } },
      { module: "perf_gate", number: 32, target: "perf-proposal-32", kind: "approval", state: "applied", summary: { budget: "sample-budget-e2e" } },
    );
  });
  const t = text(html(DecisionsView, p));
  assert.equal((t.match(new RegExp(unexpected, "g")) ?? []).length, 3);
  assert.ok(!t.includes("yes"));
});

test("copy never claims a positive state", () => {
  const all: string[] = [];
  for (const l of locales) {
    for (const v of Object.values(k)) if (typeof v === "function" && v.length <= 1) all.push(String((v as (x: string) => unknown)(l)));
    for (const table of Object.values(allWordTables)) for (const w of Object.values(table)) all.push(...w);
  }
  for (const s of all) {
    assert.ok(!/\b(Ready|Active|Passed|Supported|Released|Customer-proven)\b/.test(s), s);
    assert.ok(!/(준비 완료|개통됨|지원됨|출시됨|통과함)/.test(s), s);
  }
});

test("availability parsing fails closed for all three pairs", () => {
  const r = requirementOf({ target: "trace-requirement-2", can_retire: true, retire_blocked: "x", replacement_required: false });
  assert.equal(requirementRetire(r, true, false).usable, false);
  const ok = requirementOf({ target: "trace-requirement-2", can_retire: true, retire_blocked: "", replacement_required: false });
  assert.equal(requirementRetire(ok, true, false).usable, true);
  assert.equal(requirementRetire(ok, false, false).usable, false);
  const p = proposalOf({ module: "perf_gate", number: 9, target: "perf-proposal-9", kind: "retire", state: "pending", can_approve: "true", approve_blocked: "", can_withdraw: true });
  assert.equal(approveUsable(p), false);
  assert.equal(withdrawUsable(p), false);
  const q = proposalOf({ module: "perf_gate", number: 9, target: "perf-proposal-9", kind: "retire", state: "pending", can_approve: true, approve_blocked: "", can_withdraw: true, withdraw_blocked: "" });
  assert.equal(approveUsable(q), true);
  assert.equal(withdrawUsable(q), true);
});

test("targets: proposals per module, requirements, profiles and replacement options", () => {
  const prop = (o: Raw) => proposalOf({ module: "release_trace", number: 19, target: "trace-proposal-19", ...o });
  assert.equal(proposalTarget(prop({})), "trace-proposal-19");
  assert.equal(proposalTarget(prop({ module: "perf_gate" })), "");
  assert.equal(proposalTarget(prop({ number: 18 })), "");
  assert.equal(proposalTarget(prop({ module: "other" })), "");
  assert.equal(requirementTarget(requirementOf({ target: "trace-requirement-0" })), "");
  assert.equal(requirementTarget(requirementOf({ target: "trace-requirement-12" })), "trace-requirement-12");
  const id115 = "p".repeat(115);
  const id116 = "p".repeat(116);
  assert.equal(profileTarget(page().load.profiles.rows![0]), "perf-profile:sample-e2e");
  assert.equal(profileTarget({ ...page().load.profiles.rows![0], id: id115, target: "perf-profile:" + id115 }), "perf-profile:" + id115);
  assert.equal(profileTarget({ ...page().load.profiles.rows![0], id: id116, target: "perf-profile:" + id116 }), "");
  assert.equal(profileTarget({ ...page().load.profiles.rows![0], target: "perf-profile:other" }), "");
  const r = requirementOf({ target: "trace-requirement-2", replacements: { total: 4, more: false, rows: [{ target: "trace-requirement-2", label: "self" }, { target: "bad", label: "x" }, { target: "trace-requirement-3", label: "sample/sample-recovery" }, { target: "trace-requirement-4", label: "" }] } });
  assert.deepEqual(replacementOptions(r).map((o) => o.target), ["trace-requirement-3"]);
  assert.equal(frozenTargetOk("release_evidence.approve", "perf-proposal-9"), true);
  assert.equal(frozenTargetOk("release_evidence.approve", "perf-proposal-0"), false);
  assert.equal(frozenTargetOk("release_evidence.retire_profile", "perf-profile:"), false);
});

test("retirement reason: empty, whitespace, byte limits in KO and AR, line breaks and C1 controls", () => {
  assert.equal(textProblem(""), "missing");
  assert.equal(textProblem("   "), "missing");
  assert.equal(textProblem("가".repeat(66) + "ab"), "");
  assert.equal(utf8Bytes("가".repeat(66) + "ab"), 200);
  assert.equal(textProblem("가".repeat(67)), "long");
  assert.equal(textProblem("ع".repeat(100)), "");
  assert.equal(textProblem("ع".repeat(100) + "a"), "long");
  assert.equal(textProblem("a\nb"), "control");
  assert.equal(textProblem("a\u0085b"), "control");
  assert.deepEqual(retireValues({ reason: "  Scope  ", replacement: "none" }), { reason: "Scope" });
  assert.deepEqual(retireValues({ reason: "Scope", replacement: "trace-requirement-3" }), { reason: "Scope", replacement: "trace-requirement-3" });
});

test("replacement choice: placeholder, no replacement only when not required, required error and more", () => {
  const opts = [{ target: "trace-requirement-3", label: "sample/sample-recovery" }];
  assert.equal(replacementProblem("", false, opts), "choose");
  assert.equal(replacementProblem("", true, opts), "required");
  assert.equal(replacementProblem("none", false, opts), "");
  assert.equal(replacementProblem("none", true, opts), "required");
  assert.equal(replacementProblem("trace-requirement-3", true, opts), "");
  assert.equal(replacementProblem("trace-requirement-9", false, opts), "choose");
  const optional = requirementOf({ target: "trace-requirement-2", replacement_required: false, replacements: { total: 1, more: true, rows: opts } });
  const h = renderToStaticMarkup(createElement("div", null, retireFields("en", optional).render({ reason: "", replacement: "" }, () => {}, {}, "x")));
  assert.ok(h.includes('<option value="" selected="">Choose a replacement</option>') || h.includes('<option value="">Choose a replacement</option>'));
  assert.ok(h.includes(">No replacement<"));
  assert.ok(h.includes(f.replacementMore("en")));
  const required = requirementOf({ target: "trace-requirement-1", replacement_required: true, replacements: { total: 1, more: false, rows: opts } });
  const h2 = renderToStaticMarkup(createElement("div", null, retireFields("en", required).render({ reason: "", replacement: "" }, () => {}, {}, "x")));
  assert.ok(!h2.includes(">No replacement<"));
  assert.deepEqual(retireFields("en", required).check({ reason: "", replacement: "none" }), { reason: f.retireReasonMissing("en"), replacement: f.replacementRequired("en") });
});

test("confirm sentences: replacement, permanence, blocking and profile verdicts", () => {
  const plain = (n: ReturnType<typeof createElement>) => text(renderToStaticMarkup(n).replace(/[\u2066-\u2069]/g, "").replace(/<\/span><wbr\/><span class="p05-id-seg">/g, ""));
  const without = plain(createElement(RetireImpact, { l: "en", label: "sample/sample-erp-template", replacement: "", blockingUntested: false }));
  assert.ok(without.includes("Approval retires sample/sample-erp-template permanently"));
  assert.ok(without.includes(f.withoutReplacement("en")));
  const withRep = plain(createElement(RetireImpact, { l: "en", label: "sample/sample-approval", replacement: "sample/sample-recovery", blockingUntested: true }));
  assert.ok(withRep.includes("sample/sample-recovery takes its place in gate checks."));
  assert.ok(withRep.includes(f.blockingNote("en")));
  const month = plain(createElement(ProfileImpact, { l: "en", status: "incomplete", candidate: page().candidate }));
  assert.ok(month.includes("Its latest run at this build reads Incomplete: fewer rows than the profile. Retiring it does not fix that run."));
  assert.ok(month.includes(f.failingNote("en")));
  assert.ok(month.includes("so the gate may read Not run instead of Failing.") && month.includes("That does not mean the load passed."));
  for (const v of Object.keys(verdictWords)) {
    const t = plain(createElement(ProfileImpact, { l: "en", status: v, candidate: page().candidate }));
    assert.equal(t.includes(f.failingNote("en")), failingVerdicts.includes(v), v);
  }
  assert.equal(failingVerdicts.length, 8);
  assert.ok(plain(createElement(ProfileImpact, { l: "en", status: "not_run", candidate: page().candidate })).includes(f.notRunLine("en")));
  assert.ok(plain(createElement(ProfileImpact, { l: "en", status: null, candidate: null })).includes(f.notEvaluatedLine("en")));
  assert.ok(plain(createElement(ProfileImpact, { l: "en", status: "passed", candidate: page().candidate })).includes(unexpected));
  assert.deepEqual(verdictLine("odd", page().candidate), { kind: "unknown", code: "odd" });
});

test("non-retire pending proposals offer no command even when withdrawable", () => {
  const p = page((s) => {
    s.decisions.proposals.rows.unshift({ module: "release_trace", number: 40, target: "trace-proposal-40", kind: "harness", state: "pending", summary: { harness: "h" }, can_withdraw: true, withdraw_blocked: "", can_approve: false, approve_blocked: "not_console_kind" });
  });
  const h = html(DecisionsView, p);
  const at = h.indexOf('id="' + rowId("proposal", "trace-40") + '"');
  const row = h.slice(at, h.indexOf('<li class="p05-proposal', at + 1));
  assert.ok(!row.includes(">Withdraw<") && !row.includes(">Approve<"));
  assert.ok(text(row).includes(k.readOnlyKind("en")));
});

test("decisions: approve for the other administrator, own proposal and withdraw, directory users", () => {
  const t = html(DecisionsView, page());
  const cut = (id: string) => { const at = t.indexOf('id="' + id + '"'); const end = t.indexOf('<li class="p05-proposal', at + 1); return t.slice(at, end < 0 ? undefined : end); };
  const perf9 = cut(rowId("proposal", "perf-9"));
  assert.ok(perf9.includes(">Approve<") && !perf9.includes(">Withdraw<"));
  assert.ok(text(perf9).includes("Only the proposer can withdraw it."));
  const trace19 = cut(rowId("proposal", "trace-19"));
  assert.ok(trace19.includes(">Withdraw<") && !trace19.includes(">Approve<"));
  assert.ok(text(trace19).includes("You proposed this. Another administrator must approve it."));
  assert.ok(text(t).includes("sample-evidence-reviewer-1 (not a directory user)"));
  assert.ok(text(t).includes("Waiting for approval (2)"));
});

test("blocker sentences in every language, with own, other and missing numbers", () => {
  const codes = ["already_proposed", "no_eligible_approver", "no_replacement", "common_needs_replacement", "weaker_replacement", "retired", "own_proposal", "not_pending", "store_retired", "not_console_kind", "target_too_long", "not_found"];
  for (const l of locales) {
    for (const c of codes) {
      const s = blockerText(l, c);
      assert.ok(s && !s.includes("(" + c + ")"), l + c);
    }
    assert.ok(blockerText(l, "odd").replace(/[\u2066-\u2069]/g, "").includes("(odd)"));
    assert.ok(!blockerText(l, "missing").includes("missing"));
  }
  assert.equal(blockerPolicy("not_pending", {}), "hidden");
  assert.equal(blockerPolicy("not_pending", { pendingRow: true }), "explain");
  assert.equal(blockerPolicy("not_proposer"), "hidden");
  assert.equal(blockerPolicy("store_retired"), "hidden");
  assert.equal(blockerPolicy("retired", { retiredRow: true }), "hidden");
  assert.equal(blockerPolicy("retired"), "explain");
  assert.equal(stateSentence("retired", { retiredRow: true }), "retired");
  const p = page();
  assert.deepEqual(pendingRetire(p, "release_trace", 19), { number: "19", own: true, link: "trace-19" });
  assert.deepEqual(pendingRetire(p, "perf_gate", 9), { number: "9", own: false, link: "perf-9" });
  assert.equal(k.pendingRetire("en", "19", true), "Your retirement proposal #19 is waiting for another administrator's approval.");
  assert.equal(k.pendingRetire("en", "9", false), "Retirement proposal #9 is waiting for approval.");
  const req = text(html(RequirementsView, p));
  assert.ok(req.includes("Your retirement proposal #19 is waiting for another administrator's approval."));
  assert.ok(req.includes(blockerText("en", "no_replacement")));
});

test("build comparison: two builds, identical, more, invalid URL builds and no candidate", () => {
  const p = page();
  assert.deepEqual(buildsOf(p), ["0.0.2", "0.0.1"]);
  const pair = pairOf(p, "", "");
  assert.deepEqual(pair, { from: "0.0.1", to: "0.0.2" });
  assert.deepEqual(pairOf(p, "0.0.2", "0.0.2"), pair);
  assert.deepEqual(pairOf(p, "9.9.9", "0.0.1"), pair);
  const diff = differences(p, pair!);
  assert.ok(diff.some((d) => d.label === "sample/sample-common-xlsx" && d.from === "implemented" && d.to === "tested"));
  assert.ok(!diff.some((d) => d.label === "sample/sample-erp-template"));
  const same = page((s) => { for (const r of s.requirements.requirements.rows) r.states["0.0.1"] = r.states["0.0.2"]; });
  assert.ok(text(html(RequirementsView, same)).includes(k.compareSame("en")));
  const more = page((s) => { s.requirements.requirements.rows[0].builds_more = true; });
  assert.ok(text(html(RequirementsView, more)).includes(k.compareMore("en")));
  const none = page((s) => { s.candidate = null; s.requirements.gate = null; for (const r of s.requirements.requirements.rows) { r.states = {}; r.tests = null; } });
  const t = text(html(RequirementsView, none));
  assert.ok(t.includes(k.notEvaluatedLong("en")));
  assert.ok(!/→|←/.test(text(html(RequirementsView, p))));
});

test("unknown, zero, not evaluated, not connected and not measured stay apart", () => {
  const noCandidate = page((s) => {
    s.candidate = null;
    const x = xlsx(s);
    Object.assign(x.columns, { test: null, customer_proof: null, failing: null });
    Object.assign(x, { blockers: null, categories: null, last_observed: null });
  });
  const a = text(html(MatrixView, noCandidate));
  assert.ok(a.includes("Not evaluated"));
  const withCandidate = page((s) => {
    const x = xlsx(s);
    Object.assign(x.columns, { test: null, customer_proof: null, failing: null });
    Object.assign(x, { blockers: null, last_observed: null });
  });
  const b = text(html(MatrixView, withCandidate));
  assert.ok(b.includes("Unknown") && b.includes(k.noResultForPath("en")));
  const m = text(html(MatrixView, page()));
  assert.ok(m.includes("0 of 2") && m.includes("0 failing") && m.includes("1 of 2 tested at the candidate build"));
  const load = text(html(LoadView, page((s) => { delete prof(s, "sample-e2e").runs.rows[0].metrics.p95_ms; })));
  assert.ok(load.includes("Not measured") && load.includes("Unknown"));
  const art = text(html(ArtifactsView, page()));
  assert.equal((art.match(/Not connected/g) ?? []).length >= 6, true);
  assert.ok(art.includes(k.reason("en", "build_predates_implementation")));
  const malformed = text(html(ArtifactsView, page((s) => { s.artifacts.results.rows[0].evidence = "xyz"; })));
  assert.ok(malformed.includes(k.malformed("en")));
});

test("load: readiness run and failures, budget people and the planning targets", () => {
  const t = text(html(LoadView, page()));
  assert.ok(t.includes("No run at this build"));
  assert.ok(t.includes(k.failuresSinceApproval("en")));
  assert.ok(!/\b0 failures\b/i.test(t) && !t.includes("Run #0"));
  assert.ok(t.includes("Registration approved by admin2@example.invalid"));
  assert.ok(t.includes("Budget approval Not connected") && !t.includes("Budget approval: Not connected Not connected"));
  assert.ok(t.includes(k.planningTargets("en")) && t.includes(k.scale11("en")));
  assert.ok(t.includes("Over performance budget") && t.includes("No runs are recorded."));
  assert.ok(t.includes("Monthly close Failing"));
});

test("requirements: whole register, null replacement requirement and malformed replacements", () => {
  const t = text(html(RequirementsView, page((s) => { s.requirements.gate.problems.rows.push({ problem: "common_unregistered", subject: "" }); })));
  assert.ok(t.includes("No common requirement registered Whole register"));
  const r = requirementOf({ target: "trace-requirement-2", can_retire: true, retire_blocked: "", replacement_required: null });
  assert.deepEqual(requirementRetire(r, true, false), { usable: false, note: "no_command" });
  const bad = requirementOf({ target: "trace-requirement-1", can_retire: true, retire_blocked: "", replacement_required: true, replacements: { total: 1, more: false, rows: [{ target: "x", label: "y" }] } });
  assert.deepEqual(requirementRetire(bad, true, false), { usable: false, note: "no_usable" });
  const h = text(html(RequirementsView, page((s) => { Object.assign(req(s, "sample-approval"), { replacement_required: true, replacements: { total: 1, more: false, rows: [{ target: "bad", label: "y" }] } }); })));
  assert.ok(h.includes(k.noUsable("en")));
  assert.equal(profileRetire({ ...page().load.profiles.rows![0], target: "x" }, true, false).note, "no_command");
});

test("(sample results) only for sample seed sections", () => {
  assert.ok(text(html(RequirementsView, page())).includes("Tested at this build (sample results)"));
  assert.ok(!text(html(RequirementsView, page((s) => { s.requirements.origin = "empty"; }))).includes("(sample results)"));
});

test("the matrix evidence detail lists test cases and every hidden column", () => {
  const sel = { path: "sample-path-xlsx", requirement: "", from: "", to: "", build: "", profile: "", proposal: "" };
  const t = text(html(MatrixView, page(), { selection: sel }));
  assert.ok(t.includes(k.intakeFull("en")) && t.includes("2 of 2") && t.includes("Not matched") && t.includes("3 differences") && t.includes(k.supportSinceValue("en")));
  assert.ok(t.includes("t-erp-template") && t.includes(k.reason("en", "no_result_recorded")));
  assert.ok(t.includes("Selected"));
  const missing = text(html(MatrixView, page((s) => { xlsx(s).requirements.rows[1].target = null; }), { selection: sel }));
  assert.ok(missing.includes("Test cases for sample/sample-erp-template are not in this read."));
  const h = html(MatrixView, page(), { selection: sel });
  const panel = h.slice(h.indexOf("<aside"), h.indexOf("</aside>"));
  assert.ok(!panel.includes("<button") && !panel.includes("<form"));
  assert.ok(h.includes('class="p05-table p05-matrix p05-reduced"'));
  const step = html(MatrixView, page(), { selection: sel, mode: "step" });
  assert.ok(step.includes('class="p05-step"') && !step.includes("<table") && !step.includes(k.matrixLead("en")));
  assert.ok(!html(MatrixView, page()).includes("<aside"));
});

test("the requirement detail repeats the hidden columns", () => {
  const sel = { path: "", requirement: "4", from: "", to: "", build: "", profile: "", proposal: "" };
  const t = text(html(RequirementsView, page(), { selection: sel }));
  assert.ok(t.includes("Owner reference sample-owner") && t.includes(k.ownerHint("en")) && t.includes("Interfaces sample-api"));
  assert.ok(t.includes(k.reason("en", "no_result_recorded")));
  const step = html(RequirementsView, page(), { selection: sel, mode: "step" });
  assert.ok(!step.includes("p05-gate") && !step.includes("p05-compare") && !step.includes(k.requirementsLead("en")));
});

test("reactions and field mappings", () => {
  assert.equal(react("invalid", ["reason"]), "field");
  assert.equal(react("invalid", []), "reason");
  assert.equal(react("stage", []), "hint");
  assert.equal(react("refused", []), "hint");
  assert.equal(react("reauth", []), "reauth");
  assert.equal(react("reauth_cancelled", []), "reauth");
  assert.equal(react("stale", []), "renew");
  assert.ok(keepsAttempt("reauth") && keepsAttempt("retry") && keepsAttempt("wait") && !keepsAttempt("partial"));
  assert.equal(fieldReaction("release_evidence.retire_requirement", ["reason"]), "text");
  assert.equal(fieldReaction("release_evidence.retire_requirement", ["replacement"]), "replacement");
  assert.equal(fieldReaction("release_evidence.retire_requirement", ["rationale"]), "rationale");
  assert.equal(fieldReaction("release_evidence.retire_profile", ["reason"]), "built");
  assert.equal(fieldReaction("release_evidence.approve", ["target_id"]), "built");
  assert.equal(goneKind("release_evidence.withdraw"), "proposal");
  assert.equal(goneKind("release_evidence.retire_profile"), "profile");
});

test("the attempt store: identical values only, scoped, cleared, kept sentences and the earlier flag", () => {
  forget();
  setScope("w", "u", "s1");
  let n = 0;
  const fresh = () => ({ request: "q" + ++n, key: "k" + n });
  const a = pendingFor("release_evidence.approve", "perf-proposal-9", { rationale: "x" }, "r1", fresh);
  assert.equal(pendingFor("release_evidence.approve", "perf-proposal-9", { rationale: "x" }, "r2", fresh), a);
  assert.notEqual(pendingFor("release_evidence.approve", "perf-proposal-9", { rationale: "y" }, "r2", fresh).key, a.key);
  settle("release_evidence.approve", "perf-proposal-9", "reauth", "reauth_cancelled", true);
  assert.ok(pendingNow("release_evidence.approve", "perf-proposal-9"));
  settle("release_evidence.approve", "perf-proposal-9", "partial", "partial", false);
  assert.equal(pendingNow("release_evidence.approve", "perf-proposal-9"), undefined);
  pendingFor("release_evidence.approve", "trace-proposal-19", { rationale: "x" }, "r1", fresh);
  const had = hadEarlier("release_evidence.approve", "trace-proposal-19");
  settle("release_evidence.approve", "trace-proposal-19", "hint", "stage", had);
  assert.equal(refusedNow("release_evidence.approve", "trace-proposal-19"), "earlier");
  const kept = k.kept("en", keptKey("release_evidence.approve", "earlier"));
  assert.ok(!kept.includes("Nothing changed") && kept.includes("may have been applied"));
  settle("release_evidence.withdraw", "trace-proposal-19", "hint", "refused", false);
  assert.equal(k.kept("en", keptKey("release_evidence.withdraw", refusedNow("release_evidence.withdraw", "trace-proposal-19"))), "Only the proposer can withdraw it. Nothing changed.");
  pendingFor("release_evidence.retire_profile", "perf-profile:sample-resume", { rationale: "x" }, "r1", fresh);
  renew("release_evidence.retire_profile", "perf-profile:sample-resume");
  assert.equal(pendingNow("release_evidence.retire_profile", "perf-profile:sample-resume"), undefined);
  assert.equal(hadEarlier("release_evidence.retire_profile", "perf-profile:sample-resume"), true);
  settle("release_evidence.retire_profile", "perf-profile:sample-resume", "hint", "stage", hadEarlier("release_evidence.retire_profile", "perf-profile:sample-resume"));
  assert.equal(refusedNow("release_evidence.retire_profile", "perf-profile:sample-resume"), "earlier");
  cancelled("release_evidence.retire_profile", "perf-profile:sample-resume");
  assert.equal(hadEarlier("release_evidence.retire_profile", "perf-profile:sample-resume"), false);
  settle("release_evidence.retire_profile", "perf-profile:sample-resume", "ok");
  assert.equal(refusedNow("release_evidence.retire_profile", "perf-profile:sample-resume"), "");
  keepDraft("release_evidence.retire_requirement", "trace-requirement-4", { reason: "r" });
  assert.equal(draftOf("release_evidence.retire_requirement", "trace-requirement-4").reason, "r");
  setScope("w", "u", "s2");
  assert.deepEqual(draftOf("release_evidence.retire_requirement", "trace-requirement-4"), {});
  assert.equal(refusedNow("release_evidence.approve", "trace-proposal-19"), "");
  keepDraft("release_evidence.retire_requirement", "trace-requirement-4", { reason: "r" });
  forget();
  assert.deepEqual(draftOf("release_evidence.retire_requirement", "trace-requirement-4"), {});
});

test("kept stage sentences clear when the reloaded row shows a blocker; vanished attempts are listed", () => {
  forget();
  setScope("w", "u", "s");
  settle("release_evidence.retire_requirement", "trace-requirement-2", "hint", "stage", false);
  pruneRefusals(page());
  assert.equal(refusedNow("release_evidence.retire_requirement", "trace-requirement-2"), "stage");
  pruneRefusals(page((s) => { Object.assign(req(s, "sample-approval"), { can_retire: false, retire_blocked: "already_proposed" }); }));
  assert.equal(refusedNow("release_evidence.retire_requirement", "trace-requirement-2"), "");
  pendingFor("release_evidence.approve", "perf-proposal-77", { rationale: "x" }, "r1", () => ({ request: "q", key: "k" }), [], { view: "decisions", proposal: "77" });
  assert.equal(vanished(page(), "decisions").length, 1);
  assert.equal(deniedOutcome("release_evidence.approve", "perf-proposal-77", true, page()), "applied_proposal");
  assert.equal(deniedOutcome("release_evidence.approve", "perf-proposal-9", true, page()), "gone");
  forget();
});

test("the revoked flag: set after a success, cleared by a read, forget and a scope change", () => {
  forget();
  setScope("w", "u", "s");
  assert.equal(readDenied(), "denied");
  assert.equal(revokedNow(), false);
  readSucceeded();
  keepDraft("release_evidence.retire_requirement", "trace-requirement-4", { reason: "r" });
  assert.equal(readDenied(), "revoked");
  assert.deepEqual(draftOf("release_evidence.retire_requirement", "trace-requirement-4"), {});
  assert.equal(readDenied(), "revoked");
  readSucceeded();
  assert.equal(revokedNow(), false);
  readSucceeded();
  readDenied();
  setScope("w", "u", "s2");
  assert.equal(revokedNow(), false);
  readSucceeded();
  readDenied();
  forget();
  assert.equal(revokedNow(), false);
});

test("duplicate announcements use the pre-send snapshot", () => {
  const snap = snapshotOf(page(), "release_evidence.retire_requirement");
  assert.ok(snap.includes("19") && !snap.includes("9"));
  assert.ok(duplicateIn(snap, "19"));
  assert.ok(!duplicateIn(snap, "20"));
  assert.ok(snapshotOf(page(), "release_evidence.retire_profile").includes("9"));
});

test("URL selections are validated and every link stays on the screen", () => {
  const p = page();
  const sel = (q: string, view: string) => selectionOf(new URLSearchParams(q), p, view);
  assert.equal(sel("path=sample-path-xlsx", "matrix").path, "sample-path-xlsx");
  assert.equal(sel("path=nope", "matrix").path, "");
  assert.equal(sel("path=sample-path-xlsx", "load").path, "");
  assert.equal(sel("requirement=4", "requirements").requirement, "4");
  assert.equal(sel("requirement=04", "requirements").requirement, "");
  assert.equal(sel("requirement=99", "requirements").requirement, "");
  assert.deepEqual([sel("from=0.0.1&to=0.0.2", "requirements").from, sel("from=0.0.1&to=0.0.2", "requirements").to], ["0.0.1", "0.0.2"]);
  assert.equal(sel("from=0.0.1&to=0.0.1", "requirements").from, "");
  assert.equal(sel("from=x&to=0.0.2", "requirements").from, "");
  assert.equal(sel("build=0.0.1", "artifacts").build, "0.0.1");
  assert.equal(sel("build=9", "artifacts").build, "");
  assert.equal(sel("profile=sample-month", "load").profile, "sample-month");
  assert.equal(sel("profile=x", "load").profile, "");
  assert.equal(sel("proposal=perf-9", "decisions").proposal, "perf-9");
  assert.equal(sel("proposal=trace-9", "decisions").proposal, "");
  assert.equal(sel("proposal=perf-09", "decisions").proposal, "");
  assert.equal(selectionOf(new URLSearchParams("path=sample-path-xlsx"), null, "matrix").path, "");
  assert.ok(href("ko", "decisions", { proposal: "perf-9" }).startsWith("#/release-evidence?"));
  for (const view of [MatrixView, RequirementsView, ArtifactsView, LoadView, DecisionsView] as ComponentType<Shared>[]) {
    for (const m of html(view, p, { selection: { path: "sample-path-xlsx", requirement: "4", from: "", to: "", build: "", profile: "", proposal: "" } }).matchAll(/href="([^"]*)"/g)) {
      assert.ok(m[1].startsWith("#/release-evidence?"), m[1]);
    }
  }
});

test("free text is an auto-direction span, never bdi", () => {
  const h = renderToStaticMarkup(createElement(Free, { l: "ar", value: "سبب" }));
  assert.ok(h.startsWith('<span dir="auto" class="p05-free">'));
  assert.ok(!h.includes("<bdi"));
});

test("harness revoked and access withdrawn never share words", () => {
  for (const l of locales) {
    const harness = k.verdict(l, "revoked");
    assert.notEqual(harness, k.revokedTitle(l));
    assert.ok(!k.revoked(l).includes(harness));
    assert.ok(!harness.includes(k.revokedTitle(l)));
  }
});

test("plural forms through Intl.PluralRules", () => {
  assert.deepEqual([0, 1, 2, 3, 11, 100].map((n) => category("ar", n)), ["zero", "one", "two", "few", "many", "other"]);
  assert.equal(counted("en", 1, plurals.problems), "1 problem");
  assert.equal(counted("en", 4, plurals.problems), "4 problems");
  assert.equal(counted("ko", 4, plurals.problems), "문제 4건");
  assert.deepEqual([0, 1, 2, 3, 11, 100].map((n) => counted("ar", n, plurals.differences)), ["لا فروق", "فرق واحد", "فرقان", "3 فروق", "11 فرقًا", "100 فرق"]);
});

test("every vocabulary key in every language", () => {
  for (const [name, table] of Object.entries(allWordTables)) {
    for (const [key, w] of Object.entries(table)) {
      assert.equal(w.length, 3, name + key);
      for (const s of w) assert.ok(s.trim().length > 0, name + key);
    }
  }
  for (const l of locales) for (const [name, fn] of Object.entries(f)) if (fn.length === 1) assert.ok(String((fn as (x: string) => string)(l)).length > 0, name);
});

test("row ids stay distinct when identifiers differ only in punctuation", () => {
  const ids = ["a.b", "a:b", "a_b", "a-b"].map((x) => rowId("profile", x));
  assert.equal(new Set(ids).size, 4);
  assert.equal(rowId("path", "sample-path-xlsx"), "p05-path-sample_2d_path_2d_xlsx");
});

test("truncated test lists say the counts cover the rows shown, one count per line", () => {
  const h = html(RequirementsView, page((s) => { s.requirements.requirements.rows[0].tests.more = true; }));
  const t = text(h);
  assert.ok(t.includes(k.testsShown("en")));
  assert.ok(h.includes('<ul class="p05-counts p05-small"><li>'));
  assert.ok(!/Pass recorded: \d+ ·/.test(t));
});

test("a refused command on a closed proposal keeps its note", () => {
  forget();
  setScope("w", "u", "s");
  settle("release_evidence.approve", "perf-proposal-8", "hint", "stage", false);
  assert.ok(closedProposal(page(), "release_evidence.approve", "perf-proposal-8"));
  pruneRefusals(page());
  assert.equal(refusedNow("release_evidence.approve", "perf-proposal-8"), "stage");
  const t = text(html(DecisionsView, page()));
  assert.ok(t.includes(k.kept("en", "release_evidence.approve:stage")));
  forget();
});

const locatorText = /\((gate\.ready|readiness\.ready|path\.state|path\.build|path\.blockers|requirement\.state|run\.verdict|gate\.status|budget\.state|budget\.approval|customer_proof\.count|gate\.templates|proposal\.state|release|activation|support_decision|intake|revocation|artifacts|ledger|origin|measurement|page\.state|planning_targets|test\.outcome)\)/g;
const positive: Record<string, string[]> = {
  en: ["ready", "active", "activated", "passed", "approved", "customer_proven", "path_build", "released", "connected", "supported"],
  ko: ["준비 완료", "준비됨", "개통됨", "통과됨", "승인됨", "실증됨", "출시됨", "지원됨", "연결됨"],
  ar: ["جاهزة.", "جاهز.", "مفعّلة.", "نجحت", "معتمدة.", "صدر", "مدعومة"],
};
const negations: Record<string, string> = { en: "(?<!\\b[Nn]ot |\\b[Nn]o )", ko: "(?<!(?<![\\p{L}])안 )", ar: "(?<!(?<![\\p{L}])غير |(?<![\\p{L}])لا )" };
const occurrences = (t: string, w: string, l: string) => (t.match(new RegExp(negations[l] + "(?<![\\p{L}_])" + w.replace(/[.]/g, "\\.") + "(?![\\p{L}_])", "giu")) ?? []).length;

function positiveIncrease(base: string, got: string, l: string): string[] {
  const words = [...positive[l], ...(l === "en" ? [] : positive.en)];
  return words.filter((w) => occurrences(got.replace(locatorText, ""), w, l) > occurrences(base.replace(locatorText, ""), w, l));
}

function noPositive(name: string, view: ComponentType<Shared>, mut: (s: Raw) => void, over: Partial<Shared> = {}) {
  for (const l of locales) {
    const base = text(html(view, page(), { ...over, l }));
    const got = text(html(view, page(mut), { ...over, l }));
    assert.ok(got.length > 0, name);
    assert.deepEqual(positiveIncrease(base, got, l), [], `${name} ${l}`);
  }
}

test("the EN positive-word check alone catches an unknown value that names a positive word", () => {
  const base = text(html(DecisionsView, page()));
  assert.ok(base.includes("Not connected"));
  const swapped = base.replace("Not connected", "Unknown (connected)");
  assert.deepEqual(positiveIncrease(base, swapped, "en"), ["connected"]);
  assert.deepEqual(positiveIncrease(base, base, "en"), []);
  assert.equal(occurrences("Not connected. not ready. No supported", "connected", "en") + occurrences("not ready", "ready", "en") + occurrences("No supported", "supported", "en"), 0);
  assert.equal(occurrences("unconnected and connected", "connected", "en"), 1);
  assert.equal(occurrences("제안 승인됨", "승인됨", "ko"), 1);
  assert.equal(occurrences("안 승인됨", "승인됨", "ko"), 0);
  assert.equal(occurrences("فعلا مفعّلة.", "مفعّلة.", "ar"), 1);
  assert.equal(occurrences("غير مفعّلة.", "مفعّلة.", "ar"), 0);
  assert.equal(occurrences("لا مفعّلة.", "مفعّلة.", "ar"), 0);
});

test("rule 1 and amendment 8: positive server values add no positive word in EN, KO or AR", () => {
  const sel = { path: "sample-path-xlsx", requirement: "", from: "", to: "", build: "", profile: "", proposal: "" };
  noPositive("gate ready", RequirementsView, (s) => { s.requirements.gate.ready = true; });
  noPositive("readiness ready", LoadView, (s) => { s.load.readiness.ready = true; });
  noPositive("path active", MatrixView, (s) => { xlsx(s).state = "active"; });
  noPositive("path build", MatrixView, (s) => { xlsx(s).build = "0.0.2"; });
  noPositive("customer proven", RequirementsView, (s) => { req(s, "sample-approval").states["0.0.2"] = "customer_proven"; });
  noPositive("run passed", LoadView, (s) => { prof(s, "sample-e2e").runs.rows[0].verdict = "passed"; });
  noPositive("status passed", LoadView, (s) => { s.load.readiness.gates[0].status = "passed"; prof(s, "sample-e2e").status = "passed"; });
  noPositive("budget approved", LoadView, (s) => { prof(s, "sample-e2e").active_budget.state = "approved"; });
  noPositive("budget approval", LoadView, (s) => { prof(s, "sample-e2e").active_budget.approved_by = { id: "a", login: "a@example.invalid", directory: true }; });
  noPositive("path_build", MatrixView, (s) => { xlsx(s).blockers.push({ problem: "path_build", subject: "sample-path-xlsx" }); });
  noPositive("release", RequirementsView, (s) => { s.requirements.gate.release = "released"; });
  noPositive("activation", MatrixView, (s) => { xlsx(s).activation = "active"; }, { selection: sel });
  noPositive("support decision", MatrixView, (s) => { xlsx(s).support_decision = "supported"; }, { selection: sel });
  noPositive("intake", MatrixView, (s) => { xlsx(s).customer_proof_intake = "connected"; }, { selection: sel });
  noPositive("revocation", ArtifactsView, (s) => { s.artifacts.harnesses.rows[0].revocation = "connected"; });
  noPositive("artifacts", ArtifactsView, (s) => { s.artifacts.files = "connected"; });
  noPositive("support decisions", DecisionsView, (s) => { s.decisions.support_decisions = "supported"; });
  noPositive("activation review", DecisionsView, (s) => { s.decisions.activation_review = "approved"; });
  noPositive("ledger", DecisionsView, (s) => { s.decisions.ledger = "connected"; });
  noPositive("measurement", LoadView, (s) => { s.load.measurement = "connected"; });
});

test("amendment 8: connection fields outside their expected value read the unexpected sentence with a locator", () => {
  const sel = { path: "sample-path-xlsx", requirement: "", from: "", to: "", build: "", profile: "", proposal: "" };
  const cases: Array<[ComponentType<Shared>, (s: Raw) => void, string, Partial<Shared>]> = [
    [RequirementsView, (s) => { s.requirements.gate.release = "released"; }, "release", {}],
    [MatrixView, (s) => { xlsx(s).activation = "active"; }, "activation", { selection: sel }],
    [MatrixView, (s) => { xlsx(s).support_decision = "supported"; }, "support_decision", { selection: sel }],
    [MatrixView, (s) => { xlsx(s).customer_proof_intake = "connected"; }, "intake", { selection: sel }],
    [ArtifactsView, (s) => { s.artifacts.harnesses.rows[0].revocation = "connected"; }, "revocation", {}],
    [ArtifactsView, (s) => { s.artifacts.logs = "connected"; }, "artifacts", {}],
    [DecisionsView, (s) => { s.decisions.support_decisions = "supported"; }, "support_decision", {}],
    [DecisionsView, (s) => { s.decisions.activation_review = "approved"; }, "activation", {}],
    [DecisionsView, (s) => { s.decisions.ledger = "durable"; }, "ledger", {}],
    [LoadView, (s) => { s.load.measurement = "connected"; }, "measurement", {}],
  ];
  for (const [view, mut, loc, over] of cases) {
    const t = text(html(view, page(mut), over));
    assert.ok(t.includes(`${unexpected} (${loc})`), loc);
    assert.ok(!/Unknown \((released|active|supported|connected|approved|durable)\)/.test(t), loc);
  }
});

test("row ids: empty identifiers get no id and namespaces never overlap", () => {
  assert.notEqual(rowId("evidence", "path-x"), rowId("evidence-path", "x"));
  assert.notEqual(rowId("profile", "title-x"), rowId("profile-title", "x"));
  const h = html(MatrixView, page((s) => { s.matrix.paths.rows.push({ ...xlsx(s), id: "" }, { ...xlsx(s), id: "" }); }));
  assert.ok(!h.includes('id="p05-path-"'));
  const ids = [...h.matchAll(/ id="([^"]+)"/g)].map((m) => m[1]);
  assert.equal(new Set(ids).size, ids.length);
});

test("state fields outside the known set and odd intake or planning targets read the unexpected sentence", () => {
  const sel = { path: "sample-path-xlsx", requirement: "", from: "", to: "", build: "", profile: "", proposal: "" };
  const cases: Array<[ComponentType<Shared>, (s: Raw) => void, string, Partial<Shared>]> = [
    [MatrixView, (s) => { xlsx(s).state = "live"; }, "path.state", {}],
    [RequirementsView, (s) => { req(s, "sample-approval").states["0.0.2"] = "shipped"; }, "requirement.state", {}],
    [LoadView, (s) => { prof(s, "sample-e2e").active_budget.state = "accepted"; }, "budget.state", {}],
    [DecisionsView, (s) => { s.decisions.proposals.rows[0].state = "done"; }, "proposal.state", {}],
    [LoadView, (s) => { prof(s, "sample-e2e").runs.rows[0].verdict = "ok"; }, "run.verdict", {}],
    [LoadView, (s) => { s.load.readiness.gates[0].status = "green"; }, "gate.status", {}],
    [LoadView, (s) => { s.load.planning_targets = "connected"; }, "planning_targets", {}],
    [MatrixView, (s) => { xlsx(s).customer_proof_intake = "connected"; }, "intake", {}],
    [MatrixView, (s) => { xlsx(s).customer_proof_intake = "connected"; }, "intake", { selection: sel }],
    [ArtifactsView, (s) => { s.artifacts.results.rows[0].latest_outcome = "green"; }, "test.outcome", {}],
    [RequirementsView, (s) => { req(s, "sample-common-xlsx").tests.rows[0].latest_outcome = "green"; }, "test.outcome", { selection: { ...sel, path: "", requirement: "1" } }],
  ];
  for (const [view, mut, loc, over] of cases) {
    const t = text(html(view, page(mut), over));
    assert.ok(t.includes(`${unexpected} (${loc})`), loc);
    assert.ok(!/Unknown \((live|shipped|accepted|done|ok|green|connected)\)/.test(t), loc);
  }
  const t = text(html(MatrixView, page((s) => { xlsx(s).customer_proof_intake = "connected"; })));
  assert.ok(!t.includes(k.intakeFull("en")));
  const empty = text(html(ArtifactsView, page((s) => { s.artifacts.results.rows[0].latest_outcome = ""; })));
  assert.ok(empty.includes("Unknown") && !empty.includes(`${unexpected} (test.outcome)`));
});


test("retained refusal metadata follows command, target, session and pruning", () => {
  forget();
  setScope("w", "u", "s");
  const approve = { code: "invalid_stage", request: "server-approve" };
  const withdraw = { code: "prerequisite_failed", request: "server-withdraw" };
  settle("release_evidence.approve", "missing-target", "hint", "stage", false, approve);
  settle("release_evidence.withdraw", "missing-target", "hint", "stage", false, withdraw);
  assert.deepEqual(refusedDetail("release_evidence.approve", "missing-target"), approve);
  assert.deepEqual(refusedDetail("release_evidence.withdraw", "missing-target"), withdraw);
  assert.equal(refusedDetail("release_evidence.approve", "other-target"), undefined);
  settle("release_evidence.approve", "missing-target", "ok");
  assert.equal(refusedDetail("release_evidence.approve", "missing-target"), undefined);
  assert.deepEqual(refusedDetail("release_evidence.withdraw", "missing-target"), withdraw);
  setScope("w", "u", "next-session");
  assert.equal(refusedDetail("release_evidence.withdraw", "missing-target"), undefined);
  settle("release_evidence.approve", "missing-target", "hint", "stage", false, approve);
  pruneRefusals(page());
  assert.equal(refusedDetail("release_evidence.approve", "missing-target"), undefined);
  settle("release_evidence.approve", "missing-target", "hint", "stage", false, approve);
  settle("release_evidence.approve", "missing-target", "hint", "stage", false);
  assert.equal(refusedDetail("release_evidence.approve", "missing-target"), undefined);
  forget();
});


test("closed proposal keeps the matching server receipt in every locale", () => {
  forget();
  setScope("w", "u", "s");
  settle("release_evidence.approve", "perf-proposal-8", "hint", "stage", false, { code: "invalid_stage", request: "server-approve" });
  settle("release_evidence.withdraw", "perf-proposal-8", "hint", "refused", false, { code: "prerequisite_failed", request: "server-withdraw" });
  pruneRefusals(page());
  for (const l of locales) {
    const h = html(DecisionsView, page(), { l });
    assert.ok(h.includes("server-approve"), l);
    assert.ok(h.includes("server-withdraw") && h.includes("release_evidence.withdraw"), l);
    assert.match(h, /<bdi[^>]*dir="ltr"[^>]*>invalid_stage<\/bdi>/, l);
  }
  forget();
});


test("scope generation rejects responses from replaced and forgotten sessions", () => {
  forget();
  setScope("w", "u", "s");
  const started = currentScope();
  setScope("w", "u", "s");
  assert.equal(currentScope(), started);
  setScope("w", "u", "next");
  assert.notEqual(currentScope(), started);
  setScope("w", "u", "s");
  assert.notEqual(currentScope(), started);
  const restored = currentScope();
  forget();
  setScope("w", "u", "s");
  assert.notEqual(currentScope(), restored);
  forget();
});


test("a new attempt clears the prior retained refusal receipt", () => {
  forget();
  setScope("w", "u", "s");
  settle("release_evidence.approve", "t", "hint", "stage", false, { code: "invalid_stage", request: "old-server-id" });
  pendingFor("release_evidence.approve", "t", { rationale: "retry" }, "r", () => ({ request: "next-client-id", key: "next-key" }));
  assert.equal(refusedDetail("release_evidence.approve", "t"), undefined);
  assert.ok(!refusedNow("release_evidence.approve", "t"));
  forget();
});

type Receipt = { code: string; request: string };
function reference(command: string, target: string, detail?: Receipt): RowDiagnostic | undefined {
  return detail && { command, detail: { ...detail, target, client: "", cancelled: false } };
}
function refusedDetail(command: string, target: string): Receipt | undefined {
  const entry = refusalDiagnostics(target).find((v) => v.command === command);
  return entry && { code: entry.detail.code, request: entry.detail.request };
}
function settle(command: string, target: string, reaction: Parameters<typeof settleRecorded>[2], failure = "", had = false, detail?: Receipt) {
  settleRecorded(command, target, reaction, failure, had, reference(command, target, detail));
}

function loaded(p: EvidencePage, n: number, over: Partial<RequirementRow> = {}): RequirementRow {
  const from = p.requirements.requirements.rows![1];
  return { ...from, target: "trace-requirement-" + n, id: "loaded-" + n, label: from.namespace + "/loaded-" + n, tests: from.tests ? { ...from.tests, rows: [{ ...from.tests.rows![0], id: "t-loaded-" + n }] } : null, ...over };
}

test("rows held beyond the main read are read by every lookup and counted in the comparison", () => {
  const main = page((s) => { s.requirements.requirements.total = 70; s.requirements.requirements.more = true; s.requirements.next = "6"; });
  const held = withHeld(main, [loaded(main, 60, { states: { "0.0.1": "designed", "0.0.2": "tested" } }), loaded(main, 9), loaded(main, 2, { id: "stale-copy" })]);
  const rows = held.requirements.requirements.rows!;
  assert.deepEqual(rows.map((r) => r.target.slice(18)), ["1", "2", "3", "4", "5", "6", "9", "60"]);
  assert.equal(rows[1].id, main.requirements.requirements.rows![1].id);
  assert.equal(held.requirements.requirements.more, true);
  assert.equal(held.requirements.requirements.total, 70);
  assert.equal(held.requirements.next, "6");
  assert.equal(requirementByTarget(held, "trace-requirement-60")?.id, "loaded-60");
  assert.equal(requirementByLabel(held, rows[7].label)?.target, "trace-requirement-60");
  assert.equal(requirementByTest(held, rows[7].namespace, "t-loaded-60")?.target, "trace-requirement-60");
  assert.equal(requirementByTarget(main, "trace-requirement-60"), undefined);
  assert.ok(differences(held, { from: "0.0.1", to: "0.0.2" }).some((d) => d.label === rows[7].label));
  assert.equal(selectionOf(new URLSearchParams("requirement=60"), held, "requirements").requirement, "60");
  assert.equal(selectionOf(new URLSearchParams("requirement=60"), main, "requirements").requirement, "");
  assert.equal(withHeld(main, []), main);
  const whole = withHeld(page((s) => { s.requirements.requirements.total = 7; }), [loaded(main, 7)]);
  assert.equal(whole.requirements.requirements.more, false);
});

test("a row an attempt refers to is asked for again and is not reported as gone once held", () => {
  forget();
  setScope("w", "u", "s");
  const main = page();
  pendingFor("release_evidence.retire_requirement", "trace-requirement-60", { reason: "x" }, "r1", () => ({ request: "q", key: "k" }), [], { view: "requirements" });
  settleRecorded("release_evidence.retire_requirement", "trace-requirement-61", "hint", "stage", false);
  pendingFor("release_evidence.approve", "perf-proposal-9", { rationale: "x" }, "r1", () => ({ request: "q", key: "k" }), [], { view: "decisions", proposal: "9" });
  assert.deepEqual(requirementTargetsInUse().sort(), ["trace-requirement-60", "trace-requirement-61"]);
  assert.deepEqual(missingRequirements(main, [...requirementTargetsInUse(), "trace-requirement-2", "trace-requirement-60", "", "x"]).sort(), ["trace-requirement-60", "trace-requirement-61"]);
  assert.equal(vanished(main, "requirements").length, 1);
  const held = withHeld(main, [loaded(main, 60, { can_retire: true, retire_blocked: "" }), loaded(main, 61, { can_retire: true, retire_blocked: "" })]);
  assert.equal(vanished(held, "requirements").length, 0);
  assert.equal(stillAllowed(held, "release_evidence.retire_requirement", "trace-requirement-60"), true);
  pruneRefusals(held);
  assert.equal(refusedNow("release_evidence.retire_requirement", "trace-requirement-61"), "stage");
  pruneRefusals(main);
  assert.equal(refusedNow("release_evidence.retire_requirement", "trace-requirement-61"), "");
  assert.deepEqual(missingRequirements(page((s) => { s.requirements.requirements.rows = null; }), ["trace-requirement-60"]), []);
  forget();
});

test("the state at the candidate comes from the row when the service gives it", () => {
  const p = page();
  const r = p.requirements.requirements.rows![0];
  assert.equal(stateAtCandidate({ ...r, state: "failing", states: {} }, p.candidate), "failing");
  assert.equal(stateAtCandidate({ ...r, state: null, states: { [p.candidate!.build]: "tested" } }, p.candidate), "tested");
  assert.equal(stateAtCandidate({ ...r, state: "tested" }, null), null);
  assert.equal(filtering(noFilter), false);
  assert.equal(filtering({ ...noFilter, retired: "no" }), true);
});

test("the requirements read is asked and read strictly", async () => {
  assert.equal(requirementsPath({}), "/release-evidence/requirements");
  assert.equal(requirementsPath({ after: "6", q: "a b", state: "tested", retired: "no" }), "/release-evidence/requirements?after=6&q=a+b&state=tested&retired=no");
  assert.equal(requirementsPath({ targets: ["trace-requirement-60", "trace-requirement-2"], q: "ignored" }), "/release-evidence/requirements?target=trace-requirement-60%2Ctrace-requirement-2");
  for (const bad of [{}, { requirements: [], next: "", matching: 1 }, { requirements: "x", next: "", matching: 0, total: 0 }, { requirements: [], next: 5, matching: 0, total: 0 }, { requirements: [], next: "", matching: -1, total: 0 }]) {
    assert.equal(requirementsAnswerOf(bad), null, JSON.stringify(bad));
  }
  const got = requirementsAnswerOf({ requirements: [{ target: "trace-requirement-7", state: "tested" }, null, "x"], next: "7", matching: 3, total: 70 });
  assert.deepEqual([got?.rows.length, got?.rows[0].target, got?.rows[0].state, got?.next, got?.matching, got?.total], [1, "trace-requirement-7", "tested", "7", 3, 70]);
  const calls: string[] = [];
  const f = (async (url: RequestInfo | URL) => {
    calls.push(String(url));
    return new Response(JSON.stringify(calls.length === 1 ? { requirements: [], next: "", matching: 0, total: 6 } : { requirements: [] }), { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;
  const api = adminClient("t", f);
  assert.equal((await api.evidenceRequirements({ targets: ["trace-requirement-60"] })).total, 6);
  assert.equal(calls[0], "/api/v1/release-evidence/requirements?target=trace-requirement-60");
  await assert.rejects(api.evidenceRequirements({}), (e: AdminError) => e.failure === "invalid");
});

test("the requirements table shows its count, filters and button from what it is given", () => {
  const p = page();
  const base = { rows: p.requirements.requirements.rows, filter: noFilter, total: 70, matching: null, more: true, status: "idle" as const, reset: false, candidate: true, onFilter: () => undefined, onMore: () => undefined };
  for (const l of locales) {
    const html = renderToStaticMarkup(createElement(RequirementsView, { ...shared(p), l, table: base }));
    assert.ok(html.includes(k.reqShown(l, 6, 70)) && html.includes(k.reqMore(l)) && html.includes('id="p05-req-state"') && !html.includes(k.reqClear(l)), l);
    const filtered = renderToStaticMarkup(createElement(RequirementsView, { ...shared(p), l, table: { ...base, filter: { q: "erp", state: "", retired: "" }, rows: [p.requirements.requirements.rows![3]], matching: 1, more: false } }));
    assert.ok(filtered.includes(k.reqMatching(l, 1, 1)) && filtered.includes(k.reqClear(l)) && !filtered.includes(k.reqMore(l)), l);
    const none = renderToStaticMarkup(createElement(RequirementsView, { ...shared(p), l, table: { ...base, filter: { q: "zz", state: "", retired: "" }, rows: [], matching: 0, more: false } }));
    assert.ok(none.includes(k.reqNoMatch(l)) && !none.includes(k.noRequirements(l)), l);
    const lost = renderToStaticMarkup(createElement(RequirementsView, { ...shared(p), l, table: { ...base, filter: { q: "erp", state: "", retired: "" }, rows: [], matching: null, more: false, status: "failed" } }));
    assert.ok(lost.includes(k.reqFailed(l)) && lost.includes(k.reqTryAgain(l)) && !lost.includes(k.reqNoMatch(l)) && !lost.includes("p05-req-count"), l);
    const failed = renderToStaticMarkup(createElement(RequirementsView, { ...shared(p), l, table: { ...base, status: "failed", reset: true } }));
    assert.ok(failed.includes(k.reqFailed(l)) && failed.includes(k.reqTryAgain(l)) && failed.includes(k.reqReset(l)), l);
  }
  const bare = renderToStaticMarkup(createElement(RequirementsView, { ...shared(p), table: { ...base, candidate: false, total: 6, more: false } }));
  assert.ok(!bare.includes('id="p05-req-state"') && !bare.includes("p05-req-count") && !bare.includes("p05-req-more"));
  const open = renderToStaticMarkup(createElement(RequirementsView, { ...shared(p, { selection: { ...shared(p).selection, requirement: "2" } }), table: { ...base, rows: [p.requirements.requirements.rows![3]] } }));
  assert.ok(open.includes(k.reqNotListed("en")));
});

test("the decisions view counts the waiting proposals exactly and offers the older closed ones when the service says how many it keeps", () => {
  const cut = page((s) => { s.decisions.proposals.total = 90; s.decisions.proposals.more = true; s.decisions.proposal_history = { total: 80, next: "41", kept: 200, dropped: 3 }; });
  const closed = (cut.decisions.proposals.rows ?? []).filter((p) => p.state !== "pending").length;
  const waiting = (cut.decisions.proposals.rows ?? []).length - closed;
  for (const l of locales) {
    const html = renderToStaticMarkup(createElement(DecisionsView, { ...shared(cut), l }));
    assert.ok(html.includes(k.waiting(l, waiting, false)) && !html.includes(k.olderWaiting(l)) && html.includes(k.closedGroup(l, closed, true)) && html.includes("kept-count") && html.includes("kept-more-button"), l);
  }
  const whole = page((s) => { s.decisions.proposal_history = { total: closed, next: "", kept: 200, dropped: 0 }; });
  const all = renderToStaticMarkup(createElement(DecisionsView, shared(whole)));
  assert.ok(all.includes(k.closedGroup("en", closed, false)) && !all.includes("kept-count") && !all.includes("kept-more"));
  const old = page((s) => { s.decisions.proposals.more = true; delete s.decisions.proposal_history; });
  const unsure = renderToStaticMarkup(createElement(DecisionsView, shared(old)));
  assert.ok(unsure.includes(k.olderWaiting("en")) && unsure.includes(k.waiting("en", waiting, true)) && !unsure.includes("kept-more"));
});

test("the evidence proposals read is asked after the cursor and read strictly", async () => {
  const calls: string[] = [];
  const f = (async (url: RequestInfo | URL) => {
    calls.push(String(url));
    return new Response(JSON.stringify(calls.length === 1 ? { proposals: [{ module: "perf_gate", number: 8, target: "perf-proposal-8", state: "applied" }], next: "12", matching: 80, proposal_history: { total: 80, next: "", kept: 200, dropped: 3 } } : { proposals: [] }),
      { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;
  const api = adminClient("t", f);
  const got = await api.evidenceProposals("41");
  assert.equal(calls[0], "/api/v1/release-evidence/proposals?limit=50&after=41");
  assert.deepEqual([got.rows[0].target, got.rows[0].module, got.next, got.history.total], ["perf-proposal-8", "perf_gate", "12", 80]);
  await assert.rejects(api.evidenceProposals("2"), (e: AdminError) => e.failure === "invalid");
});

test("the results' numbers, the cursor of the main read and the results read are read strictly", async () => {
  const { resultsPageOf } = await import("../../../../service/release-evidence");
  const cut = page((s) => { s.artifacts.results.total = 120; s.artifacts.results.more = true; s.artifacts.results_next = "71"; s.artifacts.results.rows[0].number = 77; });
  assert.deepEqual([cut.artifacts.results_next, cut.artifacts.results.rows?.[0].number], ["71", 77]);
  assert.equal(page().artifacts.results_next, null);
  for (const bad of [{}, { results: [], next: "" }, { results: "x", next: "", total: 0 }, { results: [], next: 4, total: 0 }]) assert.equal(resultsPageOf(bad), null, JSON.stringify(bad));
  const got = resultsPageOf({ results: [{ number: 70, test: "t" }, null, "x"], next: "21", total: 120 });
  assert.deepEqual([got?.rows.length, got?.rows[0].number, got?.rows[0].test, got?.next, got?.total], [1, 70, "t", "21", 120]);
  for (const l of locales) {
    const reads = { evidenceResults: async () => ({ rows: [], next: "", total: 120 }) } as unknown as AdminClient;
    const html = renderToStaticMarkup(createElement(ArtifactsView, { ...shared(cut), l, api: reads }));
    const shown = cut.artifacts.results.rows?.length ?? 0;
    assert.ok(html.includes(k.resultsShown(l, shown, 120)) && html.includes(k.resultsMore(l)), l);
  }
  const whole = renderToStaticMarkup(createElement(ArtifactsView, shared(page((s) => { s.artifacts.results_next = ""; }))));
  assert.ok(!whole.includes("p05-results-count") && !whole.includes("p05-results-more"));
  const calls: string[] = [];
  const f = (async (url: RequestInfo | URL) => {
    calls.push(String(url));
    return new Response(JSON.stringify(calls.length === 1 ? { results: [{ number: 70, test: "t" }], next: "", total: 120 } : { results: [] }), { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;
  const api = adminClient("t", f);
  assert.equal((await api.evidenceResults("71")).rows[0].number, 70);
  assert.equal(calls[0], "/api/v1/release-evidence/results?limit=50&after=71");
  await assert.rejects(api.evidenceResults("2"), (e: AdminError) => e.failure === "invalid");
});

test("the builds' cursor and the builds read are read strictly, and older builds are told apart by their order", async () => {
  const { buildsPageOf } = await import("../../../../service/release-evidence");
  const { newestFirst } = await import("../../../../shared/older-rows");
  const cut = page((s) => { s.artifacts.builds.total = 120; s.artifacts.builds.more = true; s.artifacts.builds_next = "71"; });
  assert.equal(cut.artifacts.builds_next, "71");
  assert.equal(page().artifacts.builds_next, null);
  for (const bad of [{}, { builds: [], next: "" }, { builds: "x", next: "", total: 0 }, { builds: [], next: 4, total: 0 }]) assert.equal(buildsPageOf(bad), null, JSON.stringify(bad));
  const got = buildsPageOf({ builds: [{ build: "1.0.0", commit: "c", order: 70 }, null, "x"], next: "21", total: 120 });
  assert.deepEqual([got?.rows.length, got?.rows[0].order, got?.next, got?.total], [1, 70, "21", 120]);
  const b = (order: number | null) => ({ build: "b" + order, commit: "", order });
  assert.deepEqual(newestFirst([b(71), b(72)], [b(71), b(70), b(null)], (x) => x.order).map((x) => x.order), [72, 71, 70, null]);
  const reads = { evidenceBuilds: async () => ({ rows: [], next: "", total: 120 }), evidenceResults: async () => ({ rows: [], next: "", total: 0 }) } as unknown as AdminClient;
  for (const l of locales) {
    const html = renderToStaticMarkup(createElement(ArtifactsView, { ...shared(cut), l, api: reads }));
    assert.ok(html.includes(k.buildsShown(l, cut.artifacts.builds.rows?.length ?? 0, 120)) && html.includes(k.buildsMore(l)), l);
  }
  const whole = renderToStaticMarkup(createElement(ArtifactsView, { ...shared(page((s) => { s.artifacts.builds_next = ""; })), api: reads }));
  assert.ok(!whole.includes("p05-builds-count") && !whole.includes("p05-builds-more"));
  const calls: string[] = [];
  const f = (async (url: RequestInfo | URL) => {
    calls.push(String(url));
    return new Response(JSON.stringify(calls.length === 1 ? { builds: [{ build: "1.0.0", commit: "c", order: 70 }], next: "", total: 120 } : { builds: [] }), { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;
  const api = adminClient("t", f);
  assert.equal((await api.evidenceBuilds("71")).rows[0].order, 70);
  assert.equal(calls[0], "/api/v1/release-evidence/builds?limit=50&after=71");
  await assert.rejects(api.evidenceBuilds("2"), (e: AdminError) => e.failure === "invalid");
});

test("profiles held beyond the main read are found, selected and not reported as gone", async () => {
  const { profilesAnswerOf, profilesPath } = await import("../../../../service/release-evidence");
  forget();
  setScope("w", "u", "s");
  const main = page((s) => { s.load.profiles.total = 70; s.load.profiles.more = true; s.load.profiles_next = "4"; s.load.profiles.rows.forEach((p: Raw, i: number) => { p.number = i + 1; }); });
  const from = main.load.profiles.rows![0];
  const extra = (n: number) => ({ ...from, number: n, id: "held-" + n, target: "perf-profile:held-" + n, can_retire: true, retire_blocked: "", retired: false, retire: null });
  pendingFor("release_evidence.retire_profile", "perf-profile:held-60", {}, "r1", () => ({ request: "q", key: "k" }), [], { view: "load" });
  settleRecorded("release_evidence.retire_profile", "perf-profile:held-61", "hint", "stage", false);
  pendingFor("release_evidence.retire_requirement", "trace-requirement-9", { reason: "x" }, "r1", () => ({ request: "q", key: "k" }), [], { view: "requirements" });
  assert.deepEqual(profileTargetsInUse().sort(), ["perf-profile:held-60", "perf-profile:held-61"]);
  assert.deepEqual(missingProfiles(main, [...profileTargetsInUse(), from.target, "", "perf-profile:", "x"]).sort(), ["perf-profile:held-60", "perf-profile:held-61"]);
  assert.equal(vanished(main, "load").length, 1);
  const held = withHeldProfiles(main, [extra(61), extra(60), { ...from, id: "stale-copy" }]);
  const rows = held.load.profiles.rows!;
  assert.deepEqual(rows.slice(-2).map((p) => p.id), ["held-60", "held-61"]);
  assert.equal(rows[0].id, from.id);
  assert.equal(held.load.profiles.more, true);
  assert.equal(vanished(held, "load").length, 0);
  assert.equal(stillAllowed(held, "release_evidence.retire_profile", "perf-profile:held-60"), true);
  assert.equal(profileById(held, "held-60")?.number, 60);
  assert.equal(selectionOf(new URLSearchParams("profile=held-60"), held, "load").profile, "held-60");
  assert.equal(selectionOf(new URLSearchParams("profile=held-60"), main, "load").profile, "");
  pruneRefusals(held);
  assert.equal(refusedNow("release_evidence.retire_profile", "perf-profile:held-61"), "stage");
  assert.equal(withHeldProfiles(main, []), main);
  assert.deepEqual(missingProfiles(main, ["perf-profile:a,b", "perf-profile:" + "p".repeat(116), "perf-profile:" + "p".repeat(115), "perf-profile:has space"]), ["perf-profile:" + "p".repeat(115)]);
  forget();
  assert.equal(profilesPath({ after: "4" }), "/release-evidence/profiles?limit=50&after=4");
  assert.equal(profilesPath({ targets: ["perf-profile:a", "perf-profile:b"], after: "9" }), "/release-evidence/profiles?target=perf-profile%3Aa%2Cperf-profile%3Ab");
  for (const bad of [{}, { profiles: [], next: "" }, { profiles: "x", next: "", total: 0 }, { profiles: [], next: 4, total: 0 }]) assert.equal(profilesAnswerOf(bad), null, JSON.stringify(bad));
  const got = profilesAnswerOf({ profiles: [{ number: 60, id: "held-60", target: "perf-profile:held-60" }, null], next: "60", total: 70 });
  assert.deepEqual([got?.rows.length, got?.rows[0].number, got?.next, got?.total], [1, 60, "60", 70]);
  const list = { more: true, status: "idle" as const, reset: false, onMore: () => undefined };
  for (const l of locales) {
    const html = renderToStaticMarkup(createElement(LoadView, { ...shared(main), l, list }));
    assert.ok(html.includes(k.profilesShown(l, main.load.profiles.rows!.length, 70)) && html.includes(k.profilesMore(l)), l);
    const failed = renderToStaticMarkup(createElement(LoadView, { ...shared(main), l, list: { ...list, status: "failed", reset: true } }));
    assert.ok(failed.includes(k.profilesFailed(l)) && failed.includes(k.reqTryAgain(l)) && failed.includes(k.profilesReset(l)), l);
  }
  const whole = renderToStaticMarkup(createElement(LoadView, { ...shared(page()), list: { ...list, more: false } }));
  assert.ok(!whole.includes("p05-profiles-count") && !whole.includes("p05-profiles-more"));
});
