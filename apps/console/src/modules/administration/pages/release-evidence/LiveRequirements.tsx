import { useEffect, useState } from "react";
import type { Locale } from "../../../../shared/types";
import type { Candidate, EvidencePage, Problem, RequirementRow, TestRow } from "../../../../service/release-evidence";
import { blockerText, f, k } from "./live-copy";
import { categoryWords, counted, has, outcomeWords, plurals, problemWords, reasonWords, requirementStateWords, words } from "./live-codes";
import {
  blockerPolicy, buildsMore, buildsOf, differences, duplicateIn, frozenValues, keptKey, pairOf, pendingNow, pendingRetire, problemUnexpected, reasonLine, refusedNow, replacementOptions,
  filtering, noFilter, requirementByTarget, requirementRetire, utf8Bytes, requirementState, requirementTarget, rowId, sortBuilds, stateAtCandidate, stateSentence, testCounts, type AlreadyProposed, type RequirementFilter,
} from "./live-model";
import { Absent, Code, Commit, DetailShell, Fixed, Known, State, Digest, EvidenceLink, Gap, Label, ListOr, MoreLine, Num, Template, Unexpected, gapOf, href, ltr } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";
import type { AdminClient } from "../../../../service/admin";
import { retireFields, useReplacementDetail, useReplacementSearch } from "./LiveRetireForm";

const command = "release_evidence.retire_requirement";
const searchBytes = 64;

export function StateText({ l, state, candidate, sample }: { l: Locale; state: string | null; candidate: Candidate; sample: boolean }) {
  const s = requirementState(state);
  if (!s) return <Gap l={l} why={gapOf(candidate)} />;
  if (s.kind === "unexpected") return <Unexpected l={l} code={s.code} />;
  if (!has(requirementStateWords, s.code)) return <State l={l} value={s.code} table={requirementStateWords} say={k.requirementState} locator="requirement.state" />;
  return <>{k.requirementState(l, s.code)}{s.code === "tested" && sample ? k.sampleResults(l) : ""}</>;
}

export function Tags({ l, r }: { l: Locale; r: RequirementRow }) {
  const tags: string[] = [];
  if (r.common === true) tags.push(k.common(l));
  if (r.blocking === true) tags.push(k.blocking(l));
  if (r.retired === true) tags.push(k.retiredTag(l));
  if (tags.length === 0) return null;
  return <span className="p05-tags">{tags.map((x) => <span key={x} className="p05-tag">{x}</span>)}</span>;
}

export function RetiredLine({ l, r }: { l: Locale; r: RequirementRow }) {
  if (r.retired === null) return <Absent>{k.unknown(l)}</Absent>;
  if (!r.retired) return <>{k.notRetired(l)}</>;
  return <>{k.retiredTag(l)} · {r.replacement ? <Template text={k.replacedBy(l)} slots={{ label: <Label value={r.replacement} /> }} /> : k.noReplacement(l)}</>;
}

export function Outcome({ l, x }: { l: Locale; x: TestRow }) {
  if (x.latest_outcome === null) return <>{k.noResult(l)}</>;
  return <State l={l} value={x.latest_outcome} table={outcomeWords} say={k.outcome} locator="test.outcome" />;
}

export function CountedText({ l, v }: { l: Locale; v: boolean | null }) {
  return v === null ? <Absent>{k.unknown(l)}</Absent> : <>{v ? k.counted(l) : k.notCounted(l)}</>;
}

export function ReasonText({ l, x }: { l: Locale; x: TestRow }) {
  const r = reasonLine(x);
  if (r.kind === "none") return null;
  if (r.kind === "unknown") return <p className="p05-reason">{k.reasonUnknown(l)}</p>;
  return <p className="p05-reason"><Code l={l} value={r.code} table={reasonWords} say={k.reason} /></p>;
}

export function TestBlocks({ l, rows }: { l: Locale; rows: TestRow[] }) {
  return (
    <div className="p05-tests">
      {rows.map((x, i) => (
        <div key={x.id || i} className="p05-test">
          <p className="p05-test-id">{x.id ? ltr(x.id) : <Absent>{k.unknown(l)}</Absent>}</p>
          <dl className="p05-test-facts">
            <div><dt>{k.colCategory(l)}</dt><dd>{x.category === null ? k.noCategory(l) : <Code l={l} value={x.category} table={categoryWords} say={k.category} />}</dd></div>
            <div><dt>{k.colOutcome(l)}</dt><dd><Outcome l={l} x={x} /></dd></div>
            <div><dt>{k.colCounted(l)}</dt><dd><CountedText l={l} v={x.counted} /></dd></div>
            <div><dt>{k.passes(l)}</dt><dd><Num l={l} n={x.passes} /></dd></div>
            <div><dt>{k.failures(l)}</dt><dd><Num l={l} n={x.failures} /></dd></div>
          </dl>
          <ReasonText l={l} x={x} />
        </div>
      ))}
    </div>
  );
}

export function TestsSummary({ l, r, candidate }: { l: Locale; r: RequirementRow; candidate: Candidate }) {
  if (!r.tests) return <Gap l={l} why={gapOf(candidate)} />;
  const c = testCounts(r);
  if (!c) return <Absent>{k.unknown(l)}</Absent>;
  return (
    <>
      <span className="p05-line">{c.total === null ? k.unknown(l) : counted(l, c.total, plurals.tests)}</span>
      <ul className="p05-counts p05-small">
        <li><Template text={k.testsPass(l)} slots={{ n: <Num l={l} n={c.pass} /> }} /></li>
        <li><Template text={k.testsFail(l)} slots={{ n: <Num l={l} n={c.fail} /> }} /></li>
        <li><Template text={k.testsNone(l)} slots={{ n: <Num l={l} n={c.none} /> }} /></li>
      </ul>
      {r.tests.more !== false && <span className="p05-line p05-small p05-muted">{k.testsShown(l)}</span>}
    </>
  );
}

export function ProblemWords({ l, p }: { l: Locale; p: Problem }) {
  if (problemUnexpected(p.problem)) return <Unexpected l={l} code="path.blockers" />;
  return <strong><Code l={l} value={p.problem} table={problemWords} say={k.problem} /></strong>;
}

export function Subject({ l, p }: { l: Locale; p: Problem }) {
  if (problemUnexpected(p.problem)) return null;
  if (!p.subject) return p.problem === "common_unregistered" ? <>{k.wholeRegister(l)}</> : <Absent>{k.unknown(l)}</Absent>;
  return <Label value={p.subject} />;
}

export function PendingLink({ l, already }: { l: Locale; already: AlreadyProposed }) {
  if (!already) return null;
  return <a className="p05-link" href={href(l, "decisions", { proposal: already.link })}>{k.pendingRetire(l, already.number, already.own)}</a>;
}

function GateBlock({ l, page }: { l: Locale; page: EvidencePage }) {
  const g = page.requirements.gate;
  const c = page.candidate;
  if (!g) return <section className="p05-gate"><h3>{k.gateTitle(l)}</h3><p>{c === null ? k.notEvaluatedLong(l) : k.unknown(l)}</p></section>;
  const rows = g.problems.rows;
  return (
    <section className="p05-gate" aria-labelledby="p05-gate-title">
      <h3 id="p05-gate-title">
        {g.ready !== false || g.problems.total === null || !c ? k.gateTitle(l) : counted(l, g.problems.total, plurals.gateNotReady, { version: "\u2066" + c.version + "\u2069", build: "\u2066" + c.build + "\u2069" })}
      </h3>
      {g.ready === true && <Unexpected l={l} code="gate.ready" />}
      {g.ready === null && <p><Absent>{k.unknown(l)}</Absent></p>}
      <ListOr l={l} items={rows} empty={k.none(l)}>
        {(list) => (
          <div className="live-table p05-reg">
            <table className="p05-table p05-small-reg p05-problems-table">
              <caption className="visually-hidden">{k.problemsCaption(l)}</caption>
              <thead><tr><th scope="col" className="p05-w55">{k.colProblem(l)}</th><th scope="col" className="p05-w45">{k.colSubject(l)}</th></tr></thead>
              <tbody>
                {list.map((p, i) => (
                  <tr key={i}>
                    <td data-label={k.colProblem(l)}><ProblemWords l={l} p={p} /></td>
                    <td data-label={k.colSubject(l)}><Subject l={l} p={p} /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
      <MoreLine l={l} more={g.problems.more} total={g.problems.total} forms={plurals.showing100} />
      <dl className="p05-sum">
        <div><dt>{k.retiredBlocking(l)}</dt><dd><ListOr l={l} items={g.retired_blocking} empty={k.none(l)}>{(list) => <ul className="p05-plain">{list.map((x) => <li key={x}><Label value={x} /></li>)}</ul>}</ListOr></dd></div>
        <div><dt>{k.templates(l)}</dt><dd>{g.templates === null ? <Absent>{k.unknown(l)}</Absent> : g.templates.length === 0 ? k.templatesNone(l) : <Unexpected l={l} code="gate.templates" />}</dd></div>
        <div><dt>{k.release(l)}</dt><dd><Fixed l={l} value={g.release} locator="release" /></dd></div>
      </dl>
    </section>
  );
}

function CompareBlock({ l, page, from, to, requirement }: { l: Locale; page: EvidencePage; from: string; to: string; requirement: string }) {
  const sample = page.requirements.origin === "sample_seed";
  const head = (
    <>
      <h3 id="p05-compare-title">{k.compareTitle(l)}</h3>
      <p className="p05-small p05-muted">{k.compareHint(l)}</p>
    </>
  );
  if (page.candidate === null || buildsOf(page).length === 0) return <section className="p05-compare" aria-labelledby="p05-compare-title">{head}<p>{k.notEvaluatedLong(l)}</p></section>;
  const pair = pairOf(page, from, to);
  if (!pair) return <section className="p05-compare" aria-labelledby="p05-compare-title">{head}<p>{k.compareFew(l)}</p></section>;
  const builds = buildsOf(page);
  const go = (a: string, b: string) => {
    location.hash = href(l, "requirements", { from: a, to: b, ...(requirement ? { requirement } : {}) }).slice(1);
  };
  const diff = differences(page, pair);
  const state = (s: string | null) => (s === null ? <Absent>{k.noState(l)}</Absent> : <StateText l={l} state={s} candidate={page.candidate} sample={sample} />);
  return (
    <section className="p05-compare" aria-labelledby="p05-compare-title">
      {head}
      <div className="p05-pair">
        <div className="service-field">
          <label htmlFor="p05-compare-from">{k.compareFrom(l)}</label>
          <select id="p05-compare-from" value={pair.from} onChange={(e) => go(e.target.value, pair.to === e.target.value ? pair.from : pair.to)}>
            {builds.map((b) => <option key={b} value={b}>{b}</option>)}
          </select>
        </div>
        <div className="service-field">
          <label htmlFor="p05-compare-to">{k.compareWith(l)}</label>
          <select id="p05-compare-to" value={pair.to} onChange={(e) => go(pair.from === e.target.value ? pair.to : pair.from, e.target.value)}>
            {builds.map((b) => <option key={b} value={b}>{b}</option>)}
          </select>
        </div>
      </div>
      {buildsMore(page) && <p className="p05-small p05-muted">{k.compareMore(l)}</p>}
      {page.requirements.requirements.more !== false && <p className="p05-small p05-muted">{k.compareLimited(l)}</p>}
      {diff.length === 0 ? (
        <p>{k.compareSame(l)}</p>
      ) : (
        <div className="live-table p05-reg">
          <table className="p05-table p05-small-reg p05-compare-table">
            <caption className="visually-hidden">{k.compareCaption(l)}</caption>
            <thead>
              <tr>
                <th scope="col" className="p05-w40">{k.colRequirement(l)}</th>
                <th scope="col" className="p05-w30"><Template text={k.stateAt(l)} slots={{ build: ltr(pair.from) }} /></th>
                <th scope="col" className="p05-w30"><Template text={k.stateAt(l)} slots={{ build: ltr(pair.to) }} /></th>
              </tr>
            </thead>
            <tbody>
              {diff.map((d) => (
                <tr key={d.label}>
                  <td data-label={k.colRequirement(l)}><Label value={d.label} /></td>
                  <td data-label={k.stateAt(l).replace("{build}", "\u2066" + pair.from + "\u2069")}>{state(d.from)}</td>
                  <td data-label={k.stateAt(l).replace("{build}", "\u2066" + pair.to + "\u2069")}>{state(d.to)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function testsText(l: Locale, r: RequirementRow): string {
  const c = testCounts(r);
  if (!c) return "";
  const n = (s: string, v: number) => s.replace("{n}", String(v));
  const lines = [c.total === null ? k.unknown(l) : counted(l, c.total, plurals.tests), n(k.testsPass(l), c.pass), n(k.testsFail(l), c.fail), n(k.testsNone(l), c.none)];
  if (r.tests?.more !== false) lines.push(k.testsShown(l));
  return lines.join("\n");
}

function RetireFacts({ l, page, api, ctx, draft }: { l: Locale; page: EvidencePage; api: AdminClient; ctx: Record<string, string>; draft: Record<string, string> }) {
  const sample = page.requirements.origin === "sample_seed";
  const rep = useReplacementDetail(api, page, ctx.replacement ?? "");
  return (
    <>
      <div className="p05-span">
        <dt>{f.factRequirement(l)}</dt>
        <dd>
          <Label value={ctx.label ?? ""} />
          {(ctx.common === "1" || ctx.blocking === "1") && (
            <span className="p05-tags"> {ctx.common === "1" && <span className="p05-tag">{k.common(l)}</span>} {ctx.blocking === "1" && <span className="p05-tag">{k.blocking(l)}</span>}</span>
          )}
        </dd>
      </div>
      <div><dt>{f.factState(l)}</dt><dd><StateText l={l} state={ctx.state || null} candidate={page.candidate} sample={sample} /></dd></div>
      <div><dt>{f.factTests(l)}</dt><dd className="p05-pre">{ctx.tests || <Gap l={l} why={gapOf(page.candidate)} />}</dd></div>
      <div className="p05-span">
        <dt>{f.factReplacement(l)}</dt>
        <dd>
          {ctx.replacement ? (
            <>
              <Label value={ctx.replacement_label || ctx.replacement} />
              {rep === "failed" ? <p className="p05-small p05-muted p05-replacement-detail">{f.replacementDetailFailed(l)}</p>
                : rep === "loading" || rep === null ? null : <> · <StateText l={l} state={stateAtCandidate(rep, page.candidate)} candidate={page.candidate} sample={sample} /></>}
            </>
          ) : k.noReplacement(l)}
        </dd>
      </div>
      <div className="p05-span"><dt>{f.factRetireReason(l)}</dt><dd><span dir="auto" className="p05-free">{(draft.reason ?? "").trim()}</span></dd></div>
    </>
  );
}

export function RetireImpact({ l, label, replacement, blockingUntested, approve = false, unknown = false }: { l: Locale; label: string; replacement: string; blockingUntested: boolean; approve?: boolean; unknown?: boolean }) {
  return (
    <>
      <p className="p05-weight"><Template text={approve ? f.approvePermanent(l) : f.permanent(l)} slots={{ label: <Label value={label} /> }} /></p>
      {replacement ? <p><Template text={f.withReplacement(l)} slots={{ replacement: <Label value={replacement} /> }} /></p> : <p>{f.withoutReplacement(l)}</p>}
      {unknown ? <p className="p05-weight">{f.blockingUnknown(l)}</p> : blockingUntested && <p className="p05-weight">{f.blockingNote(l)}</p>}
    </>
  );
}

export function blockingUntested(r: RequirementRow | undefined, candidate: Candidate): boolean {
  if (!r || r.blocking !== true) return false;
  return stateAtCandidate(r, candidate) !== "tested";
}

function RequirementBody({ r, reduced, ...shared }: Shared & { r: RequirementRow; reduced: boolean }) {
  const { l, page, selection, canCommand, mode } = shared;
  const { open, toggle, close, focusNote, mark, diagnostics } = useOpen<"retire">(requirementTarget(r));
  const find = useReplacementSearch(shared.api, requirementTarget(r));
  const target = requirementTarget(r);
  const n = target ? target.slice("trace-requirement-".length) : "";
  const pending = target ? pendingNow(command, target) : undefined;
  const gate = requirementRetire(r, canCommand, !!pending);
  const already = r.retire !== null ? pendingRetire(page, "release_trace", r.retire) : null;
  const refused = target ? refusedNow(command, target) : "";
  const kept = refused ? k.kept(l, keptKey(command, refused)) : "";
  const retiredRow = r.retired === true;
  const blockers: Blocker[] = [];
  let state = "";
  if (refused !== "earlier") {
    if (gate.note === "no_command") blockers.push(line("", k.noCommand(l)));
    else if (gate.note === "no_usable") blockers.push(line("", k.noUsable(l)));
    else if (!gate.usable && !already && !r.can_retire) {
      if (blockerPolicy(r.retire_blocked, { retiredRow }) === "explain") blockers.push(line("", blockerText(l, r.retire_blocked)));
      else state = stateSentence(r.retire_blocked, { retiredRow }) ? k.stateSentence(l, stateSentence(r.retire_blocked, { retiredRow })) : "";
    }
  }
  const current = selection.requirement !== "" && selection.requirement === n;
  const sample = page.requirements.origin === "sample_seed";
  const candState = stateAtCandidate(r, page.candidate);
  const evidenceId = rowId("evidence-requirement", n);
  const frozen = frozenValues(pending);
  const fields = retireFields(l, r, frozen?.replacement ?? "", find);
  const cols = reduced ? 3 : 5;
  const head = (
    <>
      <span className="p05-row-title"><Label value={r.label} /></span>
      {current && <span className="p05-tag p05-selected-tag" aria-hidden="true">{k.selected(l)}</span>}
      <Tags l={l} r={r} />
      {r.retired === true && <span className="p05-line p05-small"><RetiredLine l={l} r={r} /></span>}
      {n && <span className="p05-line"><EvidenceLink l={l} label={r.label} id={evidenceId} to={href(l, "requirements", { requirement: n })} controls={mode === "split" && selection.requirement ? "p05-detail" : undefined} /></span>}
    </>
  );
  return (
    <tbody className={"p05-row" + (current ? " live-selected" : "")} id={n ? rowId("requirement", n) : undefined} data-note={target || undefined} aria-current={current || undefined}>
      <tr>
        <td className="p05-first">{head}</td>
        {!reduced && (
          <td data-label={k.colReferences(l)}>
            <dl className="p05-refs">
              <div><dt>{k.source(l)}</dt><dd><Known l={l} value={r.source} /></dd></div>
              <div><dt>{k.ownerRef(l)}</dt><dd><Known l={l} value={r.owner} /></dd></div>
              <div><dt>{k.interfaces(l)}</dt><dd><ListOr l={l} items={r.interfaces} empty={k.none(l)}>{(list) => <>{list.map((x, i) => <span key={x}>{i > 0 && ", "}{ltr(x)}</span>)}</>}</ListOr></dd></div>
            </dl>
          </td>
        )}
        <td data-label={k.colStateAtCandidate(l)}><StateText l={l} state={candState} candidate={page.candidate} sample={sample} /></td>
        {!reduced && <td data-label={k.colImplementation(l)}><Commit l={l} value={r.commit} short /></td>}
        <td data-label={k.colTests(l)}><TestsSummary l={l} r={r} candidate={page.candidate} /></td>
      </tr>
      <tr className="p05-follow">
        <td colSpan={cols}>
          <RowNote l={l} diagnostics={diagnostics(target)} kept={kept} blockers={blockers} focus={focusNote} state={state} pending={pending && !open ? f.pendingAttempt(l) : ""} />
          {already && !open && <p className="p05-act-line"><PendingLink l={l} already={already} /></p>}
          {gate.usable && !open && !already && (
            <button type="button" className="live-secondary p05-opener" aria-expanded={false} data-opener={mark("retire")} onClick={() => toggle("retire")}>{k.proposeRetirement(l)}</button>
          )}
          {gate.usable && open === "retire" && (
            <CommandForm
              {...shared} command={command} target={target} fields={fields}
              heading={<Template text={f.retireRequirementHeading(l)} slots={{ label: <Label value={r.label} /> }} />}
              title={f.confirmRetire(l)}
              context={(d) => {
                const choice = d.replacement && d.replacement !== "none" ? d.replacement : "";
                const opt = replacementOptions(r).find((o) => o.target === choice) ?? (find.seen[choice] ? { target: choice, label: find.seen[choice] } : undefined);
                return {
                  view: "requirements", label: r.label, common: r.common ? "1" : "0", blocking: r.blocking ? "1" : "0", state: candState ?? "", tests: testsText(l, r),
                  replacement: choice, replacement_label: opt?.label ?? "", untested: blockingUntested(r, page.candidate) ? "1" : "0",
                };
              }}
              facts={(ctx, d) => <RetireFacts l={l} page={page} api={shared.api} ctx={ctx} draft={d} />}
              impact={(ctx) => <RetireImpact l={l} label={ctx.label ?? r.label} replacement={ctx.replacement ? ctx.replacement_label || ctx.replacement : ""} blockingUntested={ctx.untested === "1"} />}
              done={(fl, st, snapshot) => {
                const num = fl.proposal ?? "";
                if (st !== "pending" || !num) return f.resultUnknown(l, st);
                return duplicateIn(snapshot, num) ? f.duplicate(l, num) : f.retireRequirementDone(l, num, "\u2066" + r.label + "\u2069");
              }}
              onClose={close}
            />
          )}
        </td>
      </tr>
    </tbody>
  );
}

function RequirementDetail({ l, page, r, mode }: { l: Locale; page: EvidencePage; r: RequirementRow; mode: "split" | "step" }) {
  const n = r.target.slice("trace-requirement-".length);
  const already = r.retire !== null ? pendingRetire(page, "release_trace", r.retire) : null;
  const sample = page.requirements.origin === "sample_seed";
  const states = sortBuilds(Object.keys(r.states ?? {})).map((b) => [b, (r.states ?? {})[b]] as const);
  return (
    <DetailShell l={l} mode={mode} title={<Label value={r.label} />} closeTo={href(l, "requirements")} opener={rowId("evidence-requirement", n)}>
      <dl className="p05-sum">
        <div><dt>{k.source(l)}</dt><dd><Known l={l} value={r.source} /></dd></div>
        <div><dt>{k.ownerRef(l)}</dt><dd><Known l={l} value={r.owner} /> <span className="p05-small p05-muted">{k.ownerHint(l)}</span></dd></div>
        <div><dt>{k.interfaces(l)}</dt><dd><ListOr l={l} items={r.interfaces} empty={k.none(l)}>{(list) => <>{list.map((x, i) => <span key={x}>{i > 0 && ", "}{ltr(x)}</span>)}</>}</ListOr></dd></div>
        <div><dt>{k.colStateAtCandidate(l)}</dt><dd><StateText l={l} state={stateAtCandidate(r, page.candidate)} candidate={page.candidate} sample={sample} /></dd></div>
        <div className="p05-sum-wide"><dt>{k.commit(l)}</dt><dd><Commit l={l} value={r.commit} /></dd></div>
        <div className="p05-sum-wide"><dt>{k.designDigest(l)}</dt><dd><Digest l={l} value={r.design} /></dd></div>
        <div><dt>{k.retirement(l)}</dt><dd><RetiredLine l={l} r={r} />{already && <span className="p05-line"><PendingLink l={l} already={already} /></span>}</dd></div>
        <div><dt>{k.evidenceFiles(l)}</dt><dd><Absent>{k.notConnected(l)}</Absent></dd></div>
      </dl>
      <h3>{k.statesByBuild(l)}</h3>
      {r.states === null ? <p><Absent>{k.unknown(l)}</Absent></p> : states.length === 0 ? <p><Gap l={l} why={gapOf(page.candidate)} /></p> : (
        <dl className="p05-states">
          {states.map(([b, s]) => <div key={b}><dt>{ltr(b)}</dt><dd><StateText l={l} state={s} candidate={page.candidate} sample={sample} /></dd></div>)}
        </dl>
      )}
      {r.builds_more !== false && <p className="p05-small p05-muted">{r.builds_more === null ? k.maybeMore(l) : k.buildsMoreNote(l)}</p>}
      <h3>{k.testsTitle(l)}</h3>
      {r.tests === null ? <p><Gap l={l} why={gapOf(page.candidate)} /></p> : (
        <>
          <ListOr l={l} items={r.tests.rows} empty={k.noTests(l)}>{(rows) => <TestBlocks l={l} rows={rows} />}</ListOr>
          <MoreLine l={l} more={r.tests.more} total={r.tests.total} forms={plurals.showing50} />
        </>
      )}
    </DetailShell>
  );
}

export type RequirementsTable = {
  rows: RequirementRow[] | null;
  filter: RequirementFilter;
  total: number | null;
  matching: number | null;
  more: boolean;
  status: "idle" | "loading" | "failed";
  reset: boolean;
  candidate: boolean;
  onFilter: (f: RequirementFilter) => void;
  onMore: () => void;
};

function TableControls({ l, table }: { l: Locale; table: RequirementsTable }) {
  const [draft, setDraft] = useState(table.filter);
  const [tooLong, setTooLong] = useState(false);
  useEffect(() => { setDraft(table.filter); setTooLong(false); }, [table.filter]);
  const apply = () => {
    const q = draft.q.trim();
    if (utf8Bytes(q) > searchBytes) { setTooLong(true); document.getElementById("p05-req-q")?.focus(); return; }
    setTooLong(false);
    table.onFilter({ ...draft, q });
  };
  const active = filtering(table.filter);
  const whole = (table.total ?? 0) > 0 || active;
  if (!whole) return null;
  return (
    <div className="p05-req-controls">
      <form className="p05-req-form" role="search" onSubmit={(e) => { e.preventDefault(); apply(); }}>
        <div className="service-field">
          <label htmlFor="p05-req-q">{k.reqSearch(l)}</label>
          <input id="p05-req-q" type="search" dir="auto" maxLength={64} value={draft.q} aria-invalid={tooLong || undefined} aria-describedby={tooLong ? "p05-req-q-error" : undefined} onChange={(e) => setDraft({ ...draft, q: e.target.value })} />
          {tooLong && <p id="p05-req-q-error" className="service-field-error" role="alert">{k.reqTooLong(l)}</p>}
        </div>
        {table.candidate && (
          <div className="service-field">
            <label htmlFor="p05-req-state">{k.reqState(l)}</label>
            <select id="p05-req-state" value={draft.state} onChange={(e) => setDraft({ ...draft, state: e.target.value })}>
              <option value="">{k.reqAnyState(l)}</option>
              {Object.keys(requirementStateWords).map((s) => <option key={s} value={s}>{words(l, requirementStateWords, s)}</option>)}
            </select>
          </div>
        )}
        <div className="service-field">
          <label htmlFor="p05-req-retired">{k.reqRetired(l)}</label>
          <select id="p05-req-retired" value={draft.retired} onChange={(e) => setDraft({ ...draft, retired: e.target.value })}>
            <option value="">{k.reqRetiredAny(l)}</option>
            <option value="no">{k.reqRetiredNo(l)}</option>
            <option value="yes">{k.reqRetiredYes(l)}</option>
          </select>
        </div>
        <button type="submit" className="live-secondary p05-req-apply">{k.reqApply(l)}</button>
        {active && <button type="button" className="live-secondary p05-req-clear" onClick={() => table.onFilter(noFilter)}>{k.reqClear(l)}</button>}
      </form>
      {active && table.matching !== null && table.status !== "failed" && <p className="p05-small p05-muted p05-req-count">{k.reqMatching(l, table.rows?.length ?? 0, table.matching)}</p>}
      {table.reset && <p className="p05-small p05-req-reset" role="status">{k.reqReset(l)}</p>}
    </div>
  );
}

function TableFoot({ l, table }: { l: Locale; table: RequirementsTable }) {
  const active = filtering(table.filter);
  const shown = table.rows?.length ?? 0;
  const of = active ? table.matching : table.total;
  const failed = table.status === "failed";
  return (
    <>
      {!active && of !== null && (table.more || shown < of) && shown > 0 && <p className="p05-small p05-muted p05-req-count">{k.reqShown(l, shown, of)}</p>}
      {(table.more || failed) && (
        <div className="p05-req-more">
          {failed && <p role="alert">{k.reqFailed(l)}</p>}
          <button type="button" className="live-secondary p05-req-more-button" aria-disabled={table.status === "loading" || undefined} onClick={() => { if (table.status !== "loading") table.onMore(); }}>
            {failed ? k.reqTryAgain(l) : table.status === "loading" ? k.reqLoading(l) : k.reqMore(l)}
          </button>
        </div>
      )}
    </>
  );
}

export function wholeTable(page: EvidencePage): RequirementsTable {
  const list = page.requirements.requirements;
  return { rows: list.rows, filter: noFilter, total: list.total, matching: null, more: false, status: "idle", reset: false, candidate: page.candidate !== null, onFilter: () => undefined, onMore: () => undefined };
}

export function RequirementsView(shared: Shared & { table?: RequirementsTable }) {
  const { l, page, selection, mode } = shared;
  const table = shared.table ?? wholeTable(page);
  const list = { ...page.requirements.requirements, rows: table.rows };
  const active = filtering(table.filter);
  const selected = selection.requirement ? requirementByTarget(page, "trace-requirement-" + selection.requirement) : undefined;
  if (selected && mode === "step") return <RequirementDetail l={l} page={page} r={selected} mode="step" />;
  const open = !!selected;
  const reduced = open && mode === "split";
  return (
    <>
      <p className="p05-lead">{k.requirementsLead(l)}</p>
      {!open && <p className="p05-hint">{k.hintRequirements(l)}</p>}
      <div className="p05-gate-row">
        <GateBlock l={l} page={page} />
        <CompareBlock l={l} page={page} from={selection.from} to={selection.to} requirement={selection.requirement} />
      </div>
      <div className="p05-split" data-open={reduced ? "true" : "false"}>
        <div className="p05-list">
          <h3 className="visually-hidden">{k.requirementsCaption(l)}</h3>
          <TableControls l={l} table={table} />
          {selected && !(table.rows ?? []).some((r) => r.target === selected.target) && <p className="p05-small p05-req-unlisted" role="status">{k.reqNotListed(l)}</p>}
          {!(active && table.status === "failed" && (table.rows ?? []).length === 0) && <ListOr l={l} items={list.rows} empty={active ? (table.status === "loading" ? k.reqLoading(l) : k.reqNoMatch(l)) : k.noRequirements(l)}>
            {(rows) => (
              <div className="live-table">
                <table className={"p05-table p05-requirements" + (reduced ? " p05-reduced" : " p05-full")}>
                  <caption className="visually-hidden">{k.requirementsCaption(l)}</caption>
                  <thead>
                    {reduced ? (
                      <tr>
                        <th scope="col" className="p05-w48">{k.colRequirement(l)}</th>
                        <th scope="col" className="p05-w28">{k.colStateAtCandidate(l)}</th>
                        <th scope="col" className="p05-w24">{k.colTests(l)}</th>
                      </tr>
                    ) : (
                      <tr>
                        <th scope="col" className="p05-w24">{k.colRequirement(l)}</th>
                        <th scope="col" className="p05-w22">{k.colReferences(l)}<span className="p05-th-sub">{k.ownerHint(l)}</span></th>
                        <th scope="col" className="p05-w18">{k.colStateAtCandidate(l)}</th>
                        <th scope="col" className="p05-w16">{k.colImplementation(l)}</th>
                        <th scope="col" className="p05-w20">{k.colTests(l)}</th>
                      </tr>
                    )}
                  </thead>
                  {rows.map((r, i) => <RequirementBody key={r.target || "r" + i} r={r} reduced={reduced} {...shared} />)}
                </table>
              </div>
            )}
          </ListOr>}
          <TableFoot l={l} table={table} />
        </div>
        {reduced && selected && <RequirementDetail l={l} page={page} r={selected} mode="split" />}
      </div>
    </>
  );
}

