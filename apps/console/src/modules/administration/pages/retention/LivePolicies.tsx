import { refusedDetail } from "./live-model";
import type { Locale } from "../../../../shared/types";
import type { PolicyRow } from "../../../../service/retention";
import { k } from "./live-copy";
import {
  blockerPolicy, canonical, changeOf, classTarget, currentContext, currentFrom, currentOf, duplicateIn, graceProblem, keptKey, pendingNow, pendingPolicyFor, periodProblem, refusedNow,
  rowId, sameAsCurrent, type Current,
} from "./live-model";
import { Code, Num, Rich, Stamp, Who, href, ltr } from "./live-format";
import { CommandForm, RowNote, useOpen, type Blocker, type Draft, type FieldErrors, type Fields, type Shared } from "./LiveCommand";
import { CurrentText, Proposals } from "./LiveProposals";

function ProposedText({ l, d }: { l: Locale; d: Draft }) {
  if (periodProblem(d.period ?? "") || graceProblem(d.grace ?? "")) return <>{k.unknown(l)}</>;
  const p = Number(canonical(d.period));
  const g = Number(canonical(d.grace));
  return <>{ltr(`${p} + ${g} = ${p + g}`)} {k.days(l)}</>;
}

function ChangeText({ l, c, d }: { l: Locale; c: Current; d: Draft }) {
  if (periodProblem(d.period ?? "") || graceProblem(d.grace ?? "")) return <>{k.unknown(l)}</>;
  const ch = changeOf(c, Number(canonical(d.period)), Number(canonical(d.grace)));
  if (ch.kind === "shorter") return <>{k.shorter(l, ch.days)}</>;
  if (ch.kind === "longer") return <>{k.longer(l, ch.days)}</>;
  if (ch.kind === "same") return <>{k.sameTotal(l)}</>;
  if (ch.kind === "new") return <>{k.newPolicy(l)}</>;
  return <>{k.unknown(l)}</>;
}

function policyFields(l: Locale, row: PolicyRow): Fields {
  const c = currentOf(row);
  return {
    keys: ["period", "grace"],
    saved: ["period", "grace"],
    defaults: c.kind === "known" ? { period: String(c.period), grace: String(c.grace) } : {},
    check: (d) => {
      const e: FieldErrors = {};
      if (periodProblem(d.period ?? "")) e.period = k.form.periodRange(l);
      if (graceProblem(d.grace ?? "")) e.grace = k.form.graceRange(l);
      if (!e.period && !e.grace && sameAsCurrent(c, d.period, d.grace)) e.period = k.form.sameValues(l);
      return e;
    },
    values: (d) => ({
      period: periodProblem(d.period ?? "") ? (d.period ?? "").trim() : canonical(d.period),
      grace: graceProblem(d.grace ?? "") ? (d.grace ?? "").trim() : canonical(d.grace),
    }),
    render: (d, set, errors, id) => (
      <div className="p04-pair">
        {(["period", "grace"] as const).map((key) => (
          <div key={key} className="service-field">
            <label htmlFor={id + "-" + key}>{key === "period" ? k.form.period(l) : k.form.grace(l)}</label>
            {errors[key] && <p className="service-field-error" id={id + "-" + key + "-error"}>{errors[key]}</p>}
            <input
              id={id + "-" + key} className="p04-days" inputMode="numeric" autoComplete="off" value={d[key] ?? ""} aria-invalid={!!errors[key] || undefined}
              aria-describedby={[id + "-" + key + "-hint", errors[key] ? id + "-" + key + "-error" : ""].filter(Boolean).join(" ")} onChange={(e) => set(key, e.target.value)}
            />
            <p className="service-hint" id={id + "-" + key + "-hint"}>{key === "period" ? k.form.periodHint(l) : k.form.graceHint(l)}</p>
          </div>
        ))}
      </div>
    ),
  };
}

function ProposeImpact({ l, c, d, cls }: { l: Locale; c: Current; d: Draft; cls: string }) {
  const valid = !periodProblem(d.period ?? "") && !graceProblem(d.grace ?? "");
  const ch = valid ? changeOf(c, Number(canonical(d.period)), Number(canonical(d.grace))) : { kind: "unknown" as const };
  return (
    <>
      <p>{k.proposeImpact(l, c.kind === "known" ? c.revision + 1 : c.kind === "none" ? 1 : null, k.className(l, cls))}</p>
      {ch.kind === "shorter" && <p>{k.shortenSentence(l)}</p>}
      {c.kind === "unknown" && <p>{k.shortenUnknown(l)}</p>}
    </>
  );
}

function PolicyBody({ r, ...shared }: Shared & { r: PolicyRow }) {
  const { l, page, selection, canCommand } = shared;
  const { open, toggle, close, focusNote, mark } = useOpen<"propose">();
  const target = classTarget(r);
  const pending = target ? pendingNow("retention.propose_policy", target) : undefined;
  const usable = canCommand && !!target && (r.can_propose || !!pending);
  const already = pendingPolicyFor(page, r);
  const blockers: Blocker[] = [];
  const refused = target ? refusedNow("retention.propose_policy", target) : "";
  const kept = refused ? k.kept(l, keptKey("retention.propose_policy", refused)) : "";
  const keptDetail = kept ? refusedDetail("retention.propose_policy", target) : undefined;
  const current = selection.cls !== "" && selection.cls === r.class;
  const p = r.policy;
  const cls = k.className(l, r.class);
  const blockedText = !target ? k.narrow(l, "no_command") : !usable && !already && !r.can_propose && blockerPolicy(r.propose_blocked, "propose_policy") === "explain"
    ? r.propose_blocked === "already_proposed" ? k.alreadyPolicy(l, "", false) : k.blocker(l, r.propose_blocked)
    : "";
  return (
    <tbody className={"p04-row" + (current ? " live-selected" : "")} id={r.class ? rowId("policy", r.class) : undefined}>
      <tr aria-current={current || undefined}>
        <td className="p04-first"><Code l={l} value={r.class} say={k.className} /></td>
        {p === null ? (
          <td colSpan={4} data-label={k.colPolicy(l)}>{k.noPolicy(l)}</td>
        ) : (
          <>
            <td className="p04-num" data-label={k.colRevision(l)}><Num l={l} n={p.revision} /></td>
            <td className="p04-num" data-label={k.colPeriod(l)}><Num l={l} n={p.period_days} /></td>
            <td className="p04-num" data-label={k.colGrace(l)}><Num l={l} n={p.grace_days} /></td>
            <td className="p04-num p04-strong-cell" data-label={k.colTotal(l)}><Num l={l} n={p.period_days !== null && p.grace_days !== null ? p.period_days + p.grace_days : null} /></td>
          </>
        )}
        <td className="p04-small-cell" data-label={k.colStarts(l)}><Code l={l} value={r.origin} say={k.origin} /></td>
        <td className="p04-small-cell" data-label={k.colScope(l)}>
          <span className="live-sub">{k.scopeLine(l, "")}<Code l={l} value={r.scope} say={k.connection} /></span>
          <span className="live-sub">{k.archiveLine(l, "")}<Code l={l} value={r.archive_rule} say={k.connection} /></span>
        </td>
        <td data-label={k.colApproved(l)}>
          {p === null ? k.none(l) : (
            <>
              <Who l={l} p={p.approver} />
              <span className="live-sub"><Stamp l={l} at={p.approved_at} /></span>
              <span className="live-sub live-quiet p04-small-text"><Rich text={k.proposedBy(l, p.proposer ? p.proposer.login || p.proposer.id : k.unknown(l))} /></span>
            </>
          )}
        </td>
        <td className="p04-action">
          {already ? (
            <a className="p04-link" href={href(l, "policies", { proposal: already.number })}><span><Rich text={k.alreadyPolicy(l, already.number, already.own)} /></span></a>
          ) : usable && !open ? (
            <button type="button" className="live-secondary p04-opener" aria-expanded={false} data-opener={mark("propose")} onClick={() => toggle("propose")}>{k.proposeChange(l)}</button>
          ) : blockedText && !open ? (
            <span className="p04-small-text"><Rich text={blockedText} /></span>
          ) : null}
        </td>
      </tr>
      {(open || kept || blockers.length > 0 || focusNote > 0) && (
        <tr className="p04-follow">
          <td colSpan={9}>
            <RowNote
              kept={kept} l={l} detail={keptDetail} blockers={blockers} focus={focusNote}
              extra={!open && focusNote > 0 && (already || blockedText) ? <p><Rich text={already ? k.alreadyPolicy(l, already.number, already.own) : blockedText} /></p> : undefined}
            />
            {open && usable && (
              <CommandForm
                {...shared} command="retention.propose_policy" target={target} fields={policyFields(l, r)}
                heading={k.proposeHeading(l, cls)} title={k.confirmPropose(l)} verb={k.proposeVerb(l)}
                context={() => ({ view: "policies", class: r.class, ...currentContext(currentOf(r)) })}
                facts={(ctx, d) => (
                  <>
                    <div><dt>{k.factClass(l)}</dt><dd><Code l={l} value={ctx.class ?? ""} say={k.className} /></dd></div>
                    <div><dt>{k.factCurrent(l)}</dt><dd><CurrentText l={l} c={currentFrom(ctx)} /></dd></div>
                    <div><dt>{k.factProposed(l)}</dt><dd><ProposedText l={l} d={d} /></dd></div>
                    <div><dt>{k.factChange(l)}</dt><dd><ChangeText l={l} c={currentFrom(ctx)} d={d} /></dd></div>
                  </>
                )}
                impact={(ctx, d) => <ProposeImpact l={l} c={currentFrom(ctx)} d={d} cls={ctx.class ?? ""} />}
                done={(fields, st, snapshot) => {
                  const n = fields.proposal ?? "";
                  if (st !== "pending" || !n) return k.resultUnknown(l, st);
                  return duplicateIn(snapshot, n) ? k.duplicate(l, n) : k.policyProposed(l, n);
                }}
                onClose={close}
              />
            )}
          </td>
        </tr>
      )}
    </tbody>
  );
}

export function PoliciesView(shared: Shared) {
  const { l, page } = shared;
  const rows = page.policies;
  return (
    <>
      {rows === null ? (
        <p className="live-empty">{k.unknown(l)}</p>
      ) : (
        <div className="live-table">
          <table className="p04-table p04-wide p04-policies">
            <caption className="visually-hidden">{k.policiesCaption(l)}</caption>
            <colgroup>
              <col className="p04-c15" /><col className="p04-c6" /><col className="p04-c7" /><col className="p04-c7" /><col className="p04-c7" />
              <col className="p04-c10" /><col className="p04-c11" /><col className="p04-c18" /><col className="p04-c19" />
            </colgroup>
            <thead>
              <tr>
                <th scope="col">{k.colClass(l)}</th>
                <th scope="col" className="p04-num">{k.colRevision(l)}</th>
                <th scope="col" className="p04-num">{k.colPeriod(l)}</th>
                <th scope="col" className="p04-num">{k.colGrace(l)}</th>
                <th scope="col" className="p04-num">{k.colTotal(l)}</th>
                <th scope="col">{k.colStarts(l)}</th>
                <th scope="col">{k.colScope(l)}</th>
                <th scope="col">{k.colApproved(l)}</th>
                <th scope="col">{k.colAction(l)}</th>
              </tr>
            </thead>
            {rows.map((r, n) => <PolicyBody key={r.class || n} r={r} {...shared} />)}
          </table>
        </div>
      )}
      <Proposals kind="policy" {...shared} />
    </>
  );
}
