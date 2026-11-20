import { useId } from "react";
import type { Locale } from "../../../../shared/types";
import type { Candidate, EvidencePage, ProfileRow, ProfileStatus, RunRow } from "../../../../service/release-evidence";
import { blockerText, f, k } from "./live-copy";
import { axes, budgetStateWords, counted, gateStatusWords, gateWords, metrics, plurals, storageWords, verdictWords } from "./live-codes";
import {
  blockerPolicy, budgetUnexpected, duplicateIn, keptKey, pendingNow, pendingRetire, profileRetire, profileTarget, refusedNow, rowId, stateSentence, verdictLine, verdictShown, type VerdictLine,
} from "./live-model";
import { Absent, Code, Digest, Fixed, State, Gap, Known, ListOr, MoreLine, Num, Stamp, Template, Unexpected, Who, gapOf, ltr } from "./live-format";
import { CommandForm, RowNote, line, useOpen, type Blocker, type Shared } from "./LiveCommand";
import { PendingLink } from "./LiveRequirements";

const command = "release_evidence.retire_profile";

export function StatusText({ l, status, candidate }: { l: Locale; status: string | null; candidate: Candidate }) {
  if (status === null) return <Gap l={l} why={gapOf(candidate)} />;
  if (status === "passed") return <Unexpected l={l} code="run.verdict" />;
  if (status === "not_run") return <>{k.gateStatus(l, "not_run")}</>;
  return <State l={l} value={status} table={verdictWords} say={k.verdict} locator="run.verdict" />;
}

function GateStatusText({ l, status }: { l: Locale; status: string }) {
  if (status === "passed") return <Unexpected l={l} code="gate.status" />;
  return <strong><State l={l} value={status} table={gateStatusWords} say={k.gateStatus} locator="gate.status" /></strong>;
}

function StatusLines({ l, list, retired }: { l: Locale; list: ProfileStatus[] | null; retired?: boolean }) {
  return (
    <ListOr l={l} items={list} empty={k.none(l)}>
      {(rows) => (
        <ul className="p05-plain">
          {rows.map((p, i) => (
            <li key={p.profile + i}>
              <Known l={l} value={p.profile} /> · <StatusText l={l} status={p.status || null} candidate={{ version: "", build: "", release: "" }} />
              {!retired && (
                <>
                  {" · "}{p.status === "not_run" || p.run === 0 ? k.noRunAtBuild(l) : p.run === null ? <Absent>{k.unknown(l)}</Absent> : k.runN(l, String(p.run))}
                  <span className="p05-line p05-small p05-muted">{k.failuresSinceApproval(l)}: <Num l={l} n={p.failures} /></span>
                  <span className="p05-line p05-small p05-muted">{k.failureCounterScope(l)}</span>
                </>
              )}
            </li>
          ))}
        </ul>
      )}
    </ListOr>
  );
}

export function Readiness({ l, page }: { l: Locale; page: EvidencePage }) {
  const r = page.load.readiness;
  const c = page.candidate;
  if (!r) return <section className="p05-readiness"><h3>{k.readinessHead(l)}</h3><p>{c === null ? k.notEvaluatedLong(l) : k.unknown(l)}</p></section>;
  return (
    <section className="p05-readiness" aria-labelledby="p05-readiness-title">
      <h3 id="p05-readiness-title">{r.ready !== false || !c ? k.readinessHead(l) : <Template text={k.readinessTitle(l)} slots={{ build: ltr(c.build) }} />}</h3>
      {r.ready === true && <Unexpected l={l} code="readiness.ready" />}
      <ListOr l={l} items={r.gates} empty={k.none(l)}>
        {(gates) => (
          <div className="live-table">
            <table className="p05-table p05-readiness-table">
              <caption className="visually-hidden">{k.readinessCaption(l)}</caption>
              <thead>
                <tr>
                  <th scope="col" className="p05-w20">{k.colGate(l)}</th>
                  <th scope="col" className="p05-w16">{k.colStatus(l)}</th>
                  <th scope="col" className="p05-w44">{k.colProfiles(l)}</th>
                  <th scope="col" className="p05-w20">{k.colRetiredProfiles(l)}</th>
                </tr>
              </thead>
              <tbody>
                {gates.map((g, i) => (
                  <tr key={g.gate + i}>
                    <td className="p05-first"><Code l={l} value={g.gate} table={gateWords} say={k.gate} /></td>
                    <td data-label={k.colStatus(l)}><GateStatusText l={l} status={g.status} /></td>
                    <td data-label={k.colProfiles(l)}><StatusLines l={l} list={g.profiles} /></td>
                    <td data-label={k.colRetiredProfiles(l)}><StatusLines l={l} list={g.retired} retired /></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ListOr>
    </section>
  );
}

function Hardware({ l, page }: { l: Locale; page: EvidencePage }) {
  const h = page.load.hardware;
  return (
    <section className="p05-hw" aria-labelledby="p05-hw-title">
      <h3 id="p05-hw-title">{k.hardwareTitle(l)}</h3>
      <ListOr l={l} items={h.rows} empty={k.noHardware(l)}>
        {(rows) => (
          <>
            {rows.map((x, i) => (
              <div key={x.id + i} className="p05-hw-item">
                <dl className="p05-strip">
                  <div><dt>{k.colId(l)}</dt><dd><Known l={l} value={x.id} /></dd></div>
                  <div><dt>{k.cpus(l)}</dt><dd><Num l={l} n={x.cpus} /></dd></div>
                  <div><dt>{k.memory(l)}</dt><dd>{x.memory_mib === null ? <Absent>{k.unknown(l)}</Absent> : <Template text={k.mib(l)} slots={{ n: <Num l={l} n={x.memory_mib} /> }} />}</dd></div>
                  <div><dt>{k.disk(l)}</dt><dd>{x.disk_gib === null ? <Absent>{k.unknown(l)}</Absent> : <Template text={k.gib(l)} slots={{ n: <Num l={l} n={x.disk_gib} /> }} />}</dd></div>
                  <div><dt>{k.storage(l)}</dt><dd><Code l={l} value={x.storage} table={storageWords} say={k.storageWord} /></dd></div>
                </dl>
                <dl className="p05-sum"><div className="p05-sum-wide"><dt>{k.spec(l)}</dt><dd><Digest l={l} value={x.spec} /></dd></div></dl>
              </div>
            ))}
          </>
        )}
      </ListOr>
      <MoreLine l={l} more={h.more} total={h.total} forms={plurals.showing50} />
    </section>
  );
}

function MetricValue({ l, m, name, measured = true }: { l: Locale; m: Record<string, number | null> | null; name: string; measured?: boolean }) {
  if (!m || !Object.prototype.hasOwnProperty.call(m, name)) return <Absent>{k.unknown(l)}</Absent>;
  const v = m[name];
  return v === null ? <Absent>{measured ? k.notMeasured(l) : k.unknown(l)}</Absent> : <bdi dir="ltr" className="p05-num">{v}</bdi>;
}

function Budget({ l, p }: { l: Locale; p: ProfileRow }) {
  const b = p.active_budget;
  if (p.budget_invalid) return <p><Absent>{k.unknown(l)}</Absent></p>;
  if (!b) return <p>{k.noBudget(l)}</p>;
  const odd = budgetUnexpected(b);
  if (odd) return <Unexpected l={l} code={odd} />;
  return (
    <>
      <p>{b.state === "proposed" ? <><strong>{k.budgetState(l, "proposed")}.</strong> {k.budgetProposed(l)}</> : <State l={l} value={b.state} table={budgetStateWords} say={k.budgetState} locator="budget.state" />}</p>
      <dl className="p05-sum">
        <div><dt>{k.budgetId(l)}</dt><dd><Known l={l} value={b.id} /></dd></div>
        <div><dt>{k.hardwareTitle(l)}</dt><dd><Known l={l} value={b.hardware} /></dd></div>
        <div className="p05-sum-wide">
          <dt>{k.limitsTitle(l)}</dt>
          <dd>
            {b.limits === null ? <Absent>{k.unknown(l)}</Absent> : Object.keys(b.limits).length === 0 ? k.none(l) : (
              <dl className="p05-inline">
                {Object.keys(b.limits).map((key) => <div key={key}><dt>{k.metric(l, key)}</dt><dd><MetricValue l={l} m={b.limits} name={key} measured={false} /></dd></div>)}
              </dl>
            )}
          </dd>
        </div>
        <div><dt>{k.budgetProposedBy(l)}</dt><dd><Who l={l} p={b.proposer} /></dd></div>
        <div><dt>{k.registrationApprovedBy(l)}</dt><dd><Who l={l} p={b.approver} /></dd></div>
        <div><dt>{k.budgetApproval(l)}</dt><dd><Absent>{k.notConnected(l)}</Absent></dd></div>
      </dl>
    </>
  );
}

function VerdictCell({ l, v }: { l: Locale; v: string }) {
  const s = verdictShown(v);
  if (s.kind === "unexpected") return <Unexpected l={l} code={s.code} />;
  const text = <State l={l} value={v} table={verdictWords} say={k.verdict} locator="run.verdict" />;
  return v === "provisional" ? text : <strong>{text}</strong>;
}

function Runs({ l, p }: { l: Locale; p: ProfileRow }) {
  return (
    <>
      <h4>{k.runsTitle(l)}</h4>
      <ListOr l={l} items={p.runs.rows} empty={k.noRuns(l)}>
        {(rows) => (
          <div className="live-table">
            <table className="p05-table p05-runs">
              <caption className="visually-hidden"><Template text={k.runsCaption(l)} slots={{ profile: ltr(p.id) }} /></caption>
              <thead>
                <tr>
                  <th scope="col" className="p05-w8 p05-num">{k.colRun(l)}</th>
                  <th scope="col" className="p05-w12">{k.colBuild(l)}</th>
                  <th scope="col" className="p05-w26">{k.colVerdict(l)}</th>
                  <th scope="col" className="p05-w14 p05-num">{k.colRows(l)}</th>
                  <th scope="col" className="p05-w12 p05-num">{k.colDuration(l)}</th>
                  <th scope="col" className="p05-w14">{k.colRecordedAt(l)}</th>
                  <th scope="col" className="p05-w14">{k.colHarness(l)}</th>
                </tr>
              </thead>
              {rows.map((r, i) => <RunBody key={(r.number ?? "u") + "-" + i} l={l} r={r} />)}
            </table>
          </div>
        )}
      </ListOr>
      <MoreLine l={l} more={p.runs.more} total={p.runs.total} forms={plurals.showingNewest20} />
    </>
  );
}

function RunBody({ l, r }: { l: Locale; r: RunRow }) {
  return (
    <tbody className="p05-row">
      <tr>
        <td className="p05-first p05-num" data-label={k.colRun(l)}>{r.number === null ? <Absent>{k.unknown(l)}</Absent> : ltr("#" + r.number)}</td>
        <td data-label={k.colBuild(l)}><Known l={l} value={r.build} /></td>
        <td data-label={k.colVerdict(l)}><VerdictCell l={l} v={r.verdict} /></td>
        <td className="p05-num" data-label={k.colRows(l)}><Num l={l} n={r.rows} /></td>
        <td className="p05-num" data-label={k.colDuration(l)}>{r.duration_s === null ? <Absent>{k.unknown(l)}</Absent> : <Template text={k.seconds(l)} slots={{ n: <Num l={l} n={r.duration_s} /> }} />}</td>
        <td data-label={k.colRecordedAt(l)}><Stamp l={l} at={r.at} /></td>
        <td data-label={k.colHarness(l)}><Known l={l} value={r.harness} wrap /></td>
      </tr>
      <tr className="p05-follow">
        <td colSpan={7}>
          <dl className="p05-metrics">
            <div><dt>{k.hardwareTitle(l)}</dt><dd><Known l={l} value={r.hardware} /></dd></div>
            <div><dt>{k.mismatches(l)}</dt><dd><Num l={l} n={r.mismatches} /></dd></div>
            <div><dt>{k.interrupted(l)}</dt><dd>{r.interrupted === null ? <Absent>{k.unknown(l)}</Absent> : r.interrupted ? k.yes(l) : k.no(l)}</dd></div>
            {metrics.map((m) => <div key={m}><dt>{k.metric(l, m)}</dt><dd><MetricValue l={l} m={r.metrics} name={m} /></dd></div>)}
          </dl>
        </td>
      </tr>
    </tbody>
  );
}

export function VerdictImpact({ l, line: v }: { l: Locale; line: VerdictLine }) {
  if (v.kind === "unexpected") return <Unexpected l={l} code="run.verdict" />;
  if (v.kind === "not_run") return <p>{f.notRunLine(l)}</p>;
  if (v.kind === "not_evaluated") return <p>{f.notEvaluatedLine(l)}</p>;
  if (v.kind === "unknown") return v.code ? <Unexpected l={l} code="run.verdict" /> : <p><Template text={f.verdictLine(l)} slots={{ verdict: <>{k.unknown(l)}</> }} /></p>;
  return (
    <>
      <p><Template text={f.verdictLine(l)} slots={{ verdict: <>{k.verdict(l, v.verdict)}</> }} /></p>
      {v.failing && <p>{f.failingNote(l)}</p>}
    </>
  );
}

export function ProfileImpact({ l, status, candidate, approve = false, missing = false }: { l: Locale; status: string | null; candidate: Candidate; approve?: boolean; missing?: boolean }) {
  return (
    <>
      <p>{approve ? f.profileApprovePermanent(l) : f.profilePermanent(l)} <strong className="p05-weight">{f.notLoadPassed(l)}</strong></p>
      <VerdictImpact l={l} line={missing ? { kind: "unknown", code: "" } : verdictLine(status, candidate)} />
    </>
  );
}

export function ProfileFacts({ l, ctx, candidate }: { l: Locale; ctx: Record<string, string>; candidate: Candidate }) {
  return (
    <>
      <div><dt>{f.factProfile(l)}</dt><dd><Known l={l} value={ctx.profile ?? ""} /></dd></div>
      <div><dt>{f.factGate(l)}</dt><dd><Code l={l} value={ctx.gate ?? ""} table={gateWords} say={k.gate} /></dd></div>
      <div><dt>{f.factStatus(l)}</dt><dd><StatusText l={l} status={ctx.status_set === "1" ? ctx.status ?? "" : null} candidate={candidate} /></dd></div>
      <div><dt>{f.factBudget(l)}</dt><dd>{ctx.budget === "none" ? k.noBudget(l) : ctx.budget === "proposed" ? k.budgetState(l, "proposed") : <Absent>{k.unknown(l)}</Absent>}</dd></div>
      <div><dt>{f.factRuns(l)}</dt><dd>{ctx.runs ? counted(l, Number(ctx.runs), plurals.runs) : <Absent>{k.unknown(l)}</Absent>}</dd></div>
    </>
  );
}

export function profileContext(p: ProfileRow): Record<string, string> {
  const b = p.active_budget;
  return {
    view: "load", profile: p.id, gate: p.gate, status: p.status ?? "", status_set: p.status === null ? "0" : "1",
    budget: p.budget_invalid ? "unknown" : !b ? "none" : budgetUnexpected(b) ? "unknown" : b.state === "proposed" ? "proposed" : "unknown",
    runs: p.runs.total === null ? "" : String(p.runs.total),
  };
}

function Profile({ p, ...shared }: Shared & { p: ProfileRow }) {
  const { l, page, selection, canCommand } = shared;
  const target = profileTarget(p);
  const titleId = useId();
  const { open, toggle, close, focusNote, mark } = useOpen<"retire">(target);
  const pending = target ? pendingNow(command, target) : undefined;
  const gate = profileRetire(p, canCommand, !!pending);
  const already = p.retire !== null ? pendingRetire(page, "perf_gate", p.retire) : null;
  const refused = target ? refusedNow(command, target) : "";
  const kept = refused ? k.kept(l, keptKey(command, refused)) : "";
  const retiredRow = p.retired === true;
  const blockers: Blocker[] = [];
  let state = "";
  if (refused !== "earlier") {
    if (gate.note === "no_command") blockers.push(line("", k.noCommand(l)));
    else if (!gate.usable && !already && !p.can_retire) {
      if (blockerPolicy(p.retire_blocked, { retiredRow }) === "explain") blockers.push(line("", blockerText(l, p.retire_blocked)));
      else state = stateSentence(p.retire_blocked, { retiredRow }) ? k.stateSentence(l, stateSentence(p.retire_blocked, { retiredRow })) : "";
    }
  }
  const current = selection.profile !== "" && selection.profile === p.id;
  return (
    <article className={"p05-profile" + (current ? " live-selected" : "")} id={p.id ? rowId("profile", p.id) : undefined} data-note={target || undefined} aria-current={current || undefined} aria-labelledby={titleId}>
      <div className="p05-head">
        <h3 id={titleId}><Known l={l} value={p.id} /></h3>
        {current && <span className="p05-tag p05-selected-tag" aria-hidden="true">{k.selected(l)}</span>}
        <span><Code l={l} value={p.gate} table={gateWords} say={k.gate} /></span>
        <span><StatusText l={l} status={p.status} candidate={page.candidate} /></span>
        {p.retired === true && <span className="p05-tag">{k.retiredTag(l)}</span>}
      </div>
      <div className="p05-profile-grid">
        <div>
          <dl className="p05-sum">
            <div><dt>{k.statusAtCandidate(l)}</dt><dd><StatusText l={l} status={p.status} candidate={page.candidate} /></dd></div>
            <div className="p05-sum-wide"><dt>{k.dataset(l)}</dt><dd><Digest l={l} value={p.dataset} /></dd></div>
            <div className="p05-sum-wide"><dt>{k.expected(l)}</dt><dd><Digest l={l} value={p.expected} /></dd></div>
          </dl>
          <h4>{k.activeBudget(l)}</h4>
          <Budget l={l} p={p} />
        </div>
        <div>
          <h4>{k.axesTitle(l)}</h4>
          {p.axes === null ? <p><Absent>{k.unknown(l)}</Absent></p> : (
            <dl className="p05-axes">
              {axes.map((a) => <div key={a}><dt>{k.axis(l, a)}</dt><dd><MetricValue l={l} m={p.axes} name={a} measured={false} /></dd></div>)}
            </dl>
          )}
        </div>
      </div>
      <Runs l={l} p={p} />
      <div className="p05-actions">
        <RowNote kept={kept} blockers={blockers} focus={focusNote} state={state} pending={pending && !open ? f.pendingAttempt(l) : ""} />
        {already && !open && <p className="p05-act-line"><PendingLink l={l} already={already} /></p>}
        {gate.usable && !open && !already && (
          <button type="button" className="live-secondary p05-opener" aria-expanded={false} data-opener={mark("retire")} onClick={() => toggle("retire")}>{k.proposeRetirement(l)}</button>
        )}
        {gate.usable && open === "retire" && (
          <CommandForm
            {...shared} command={command} target={target} values={{}}
            heading={<Template text={f.retireProfileHeading(l)} slots={{ profile: ltr(p.id) }} />}
            title={f.confirmRetire(l)}
            context={() => profileContext(p)}
            facts={(ctx) => <ProfileFacts l={l} ctx={ctx} candidate={page.candidate} />}
            impact={(ctx) => <ProfileImpact l={l} status={ctx.status_set === "1" ? ctx.status ?? "" : null} candidate={page.candidate} />}
            done={(fl, st, snapshot) => {
              const num = fl.proposal ?? "";
              if (st !== "pending" || !num) return f.resultUnknown(l, st);
              return duplicateIn(snapshot, num) ? f.duplicate(l, num) : f.retireProfileDone(l, num, "\u2066" + p.id + "\u2069");
            }}
            onClose={close}
          />
        )}
      </div>
    </article>
  );
}

export function LoadView(shared: Shared) {
  const { l, page } = shared;
  const profiles = page.load.profiles;
  return (
    <>
      <p className="p05-lead">{k.loadLead(l)}</p>
      <div className="p05-notes-block">
        <p>{page.load.planning_targets === "not_connected" ? k.planningTargets(l) : <Unexpected l={l} code="planning_targets" />}</p>
        <p>{k.scale11(l)}</p>
        <p>{page.load.measurement === "not_connected" ? k.measurement(l) : <>{k.measurementLabel(l)}: <Fixed l={l} value={page.load.measurement} locator="measurement" /></>}</p>
      </div>
      <Readiness l={l} page={page} />
      <Hardware l={l} page={page} />
      <section className="p05-profiles" aria-labelledby="p05-profiles-title">
        <h3 id="p05-profiles-title" className="visually-hidden">{k.profilesTitle(l)}</h3>
        <ListOr l={l} items={profiles.rows} empty={k.noProfiles(l)}>{(rows) => <>{rows.map((p, i) => <Profile key={p.target || "p" + i} p={p} {...shared} />)}</>}</ListOr>
        <MoreLine l={l} more={profiles.more} total={profiles.total} forms={plurals.showing50} />
      </section>
    </>
  );
}
