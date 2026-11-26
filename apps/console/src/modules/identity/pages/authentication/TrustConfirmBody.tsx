import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import type { PolicyPage } from "../../../../service/admin";
import { actor, Compare, ltr } from "./TrustChanges";
import { when } from "./live-model";
import { ty } from "./trust-copy";
import { candidateOf, currentVersion, expectedNext, findingReason, versionOf, type Confirming } from "./trust-model";

export function primaryLabel(l: Locale, k: Confirming): string {
  switch (k.kind) {
    case "propose": return k.change ? ty.reviseSubmit(l) : ty.proposeOpen(l);
    case "validate": return ty.validate(l);
    case "approve": return ty.approve(l);
    case "activate": return ty.activate(l);
    case "reject": return ty.reject(l);
    default: return ty.cancel(l);
  }
}

export function Row({ label, value, wide = false }: { label: string; value: ReactNode; wide?: boolean }) {
  return <div className={wide ? "m02-wide" : undefined}><dt>{label}</dt><dd>{value}</dd></div>;
}

export function ConfirmBody({ l, page, k, expectedDigest, workspace, dead = false }: {
  l: Locale; page: PolicyPage; k: Confirming; expectedDigest: string; workspace: string; dead?: boolean;
}) {
  const ch = k.change;
  const val = (v: string | undefined, digest = false) => (v ? (digest ? ltr(v) : actor(v)) : <span className="live-quiet">{ty.notRecorded(l)}</span>);
  const members = ch?.impact?.members || page.holders.members;
  const count = (n: number | undefined) => (n === undefined ? <span className="live-quiet">{ty.impactNotComputed(l)}</span> : String(n));
  const items = new Map((ch?.validation?.items ?? []).map((i) => [i.item, i]));
  const finding = (key: string) => {
    const it = items.get(key);
    return it ? <>{ty.result(l, it.result)}{findingReason(l, it) && <> · {findingReason(l, it)}</>}</> : <span className="live-quiet">{ty.notValidated(l)}</span>;
  };
  const common = (
    <>
      <Row label={ty.colChange(l)} value={ch ? <>{ty.change(l, ch.number)} {ltr(ch.id)}</> : ty.newChange(l)} />
      <Row label={ty.baseVersion(l)} value={ltr(ch ? versionOf(ch.old_epoch) : currentVersion(page))} />
      <Row label={ty.policyRevision(l)} value={String(ch ? ch.old_policy : page.trust.policy_number)} />
      {ch && <Row label={ty.changeRevision(l)} value={String(ch.change_revision)} />}
      {ch && <Row label={ty.currentState(l)} value={ty.state(l, ch.state)} />}
    </>
  );
  let facts: ReactNode = null;
  let after: ReactNode = null;
  const consequence: [string, boolean][] = [];
  switch (k.kind) {
    case "propose": {
      const next = k.proposed ? candidateOf(k.proposed.maxAge, k.proposed.step) : null;
      facts = (
        <>
          <Row label={ty.workspace(l)} value={ltr(workspace)} />
          <Row label={ty.route(l)} value={ty.localPassword(l)} />
          <Row label={ty.membersAffected(l)} value={members > 0 ? String(members) : <span className="live-quiet">{ty.notRecorded(l)}</span>} />
          {ch && <Row label={ty.changeRevision(l)} value={ty.revisionStep(l, ch.change_revision, ch.change_revision + 1)} wide />}
          <Row label={ty.oldDigest(l)} value={ltr(page.trust.digest)} wide />
          <Row label={ty.newDigest(l)} value={<>{ltr(expectedDigest)} <span className="m02-sub">({ty.expected(l)})</span><br /><span className="m02-sub">{ty.serverAuthoritative(l)}</span></>} wide />
        </>
      );
      if (next) {
        const rev = { methods: next.methods, mfa: next.mfa, max_age_seconds: next.maxAgeSeconds, step_up: next.stepUp.map((s) => ({ action: s.action, max_age_seconds: s.maxAgeSeconds, mfa: s.mfa })) };
        after = <Compare l={l} current={page.policy} next={rev} />;
      }
      consequence.push([ty.consequence.propose(l), true], [ty.consequence.proposeRisk(l), false]);
      break;
    }
    case "validate":
      facts = <Row label={ty.changeDigest(l)} value={val(ch?.change_digest, true)} wide />;
      after = (
        <>
          <h3>{ty.checked(l)}</h3>
          <ul>{["test_login", "recovery_path", "record_size"].map((x) => <li key={x}>{ty.item(l, x)}</li>)}</ul>
          <h3>{ty.result(l, "not_applicable")}</h3>
          <ul>
            <li>{ty.item(l, "issuer_subject")}: {ty.reason(l, "local_password")}</li>
            <li>{ty.item(l, "provider_diagnostics")}: {ty.reason(l, "no_provider_connection")}</li>
            <li>{ty.item(l, "external_status")}: {ty.reason(l, "no_external_identity")}</li>
          </ul>
        </>
      );
      consequence.push([ty.consequence.validate(l), true], [ty.consequence.validateBlocked(l), false]);
      break;
    case "approve":
      facts = (
        <>
          <Row label={ty.proposedBy(l)} value={val(ch?.proposer)} />
          <Row label={ty.validatedBy(l)} value={<>{val(ch?.validator)} · {ch?.validation?.at ? when(l, ch.validation.at) : ty.notRecorded(l)}</>} />
          <Row label={ty.changeDigest(l)} value={val(ch?.change_digest, true)} wide />
          <Row label={ty.evidenceDigest(l)} value={val(ch?.validation?.evidence_digest, true)} wide />
          {!dead && <Row label={ty.yourAuthority(l)} value={ty.authority(l)} wide />}
          {!dead && <Row label={ty.lockout(l)} value={finding("recovery_path")} wide />}
          <Row label={ty.members(l)} value={count(ch?.impact?.members)} />
          <Row label={ty.sessionsNow(l)} value={count(ch?.impact?.sessions)} />
        </>
      );
      consequence.push([ty.consequence.approve(l), true], [ty.consequence.approveRecheck(l), false]);
      break;
    case "activate": {
      const next = expectedNext(page);
      facts = (
        <>
          <Row label={ty.changeDigest(l)} value={val(ch?.change_digest, true)} wide />
          <Row label={ty.evidenceDigest(l)} value={val(ch?.validation?.evidence_digest, true)} wide />
          <Row label={ty.proposedBy(l)} value={val(ch?.proposer)} />
          <Row label={ty.validatedBy(l)} value={val(ch?.validator)} />
          <Row label={ty.approvedBy(l)} value={val(ch?.approver)} />
          <Row label={ty.toBeActivatedBy(l)} value={ty.you(l)} />
          <Row label={ty.expectedVersion(l)} value={<>{ltr(next.text)} <span className="m02-sub">({ty.expected(l)})</span></>} />
          <Row label={ty.sessionsNow(l)} value={count(ch?.impact?.sessions)} />
          {!dead && <Row label={ty.item(l, "test_login")} value={finding("test_login")} wide />}
          {!dead && <Row label={ty.lockout(l)} value={finding("recovery_path")} wide />}
        </>
      );
      if (ch && ch.impact && ch.impact.affected_ids.length > 0) {
        after = (
          <details className="m02-members">
            <summary>{ty.affected(l, ch.impact.members)}</summary>
            <ul className="m02-ids">{ch.impact.affected_ids.map((id) => <li key={id}>{actor(id)}</li>)}</ul>
          </details>
        );
      }
      consequence.push([ty.consequence.activate(l), true], [ty.consequence.activateUndo(l), true]);
      break;
    }
    case "reject":
      facts = <Row label={ty.proposedBy(l)} value={val(ch?.proposer)} />;
      consequence.push([ty.consequence.unchanged(l), false], [ty.consequence.reject(l), true]);
      break;
    default:
      consequence.push([ty.consequence.unchanged(l), false], [ty.consequence.cancel(l), true]);
  }
  const boundary = ty.boundary[k.kind](l);
  return (
    <>
      <p className="m02-confirm-lead">{ty.leads[k.kind](l)}</p>
      <dl className="m02-confirm-facts">{common}{facts}</dl>
      {after}
      <div className="m02-consequence">{consequence.map(([x, strong]) => <p key={x} className={strong ? "live-strong" : undefined}>{x}</p>)}</div>
      <p className={"m02-boundary" + (k.kind === "activate" ? " live-strong" : "")}>{boundary}</p>
    </>
  );
}

