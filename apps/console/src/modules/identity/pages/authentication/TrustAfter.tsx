import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { Outcome } from "../../../../service/admin";
import { ltr } from "./TrustChanges";
import { Row } from "./TrustConfirmBody";
import { ty } from "./trust-copy";
import { rawEpochText as epochText } from "./trust-model";

export function afterKind(out: Outcome): "activated" | "activation_unknown" | "unclear" {
  if (out.status === "completed" && out.state === "activated") return "activated";
  if (out.status === "accepted" && out.job_id !== "") return "activation_unknown";
  return "unclear";
}

export function AfterActivation({ l, out, onSignIn }: { l: Locale; out: Outcome; onSignIn: () => void }) {
  const heading = useRef<HTMLHeadingElement>(null);
  useEffect(() => heading.current?.focus(), []);
  const kind = afterKind(out);
  const f = out.fields;
  const value = (v: string | undefined) => (v ? v : <span className="live-quiet">{ty.notRecorded(l)}</span>);
  const version = f.epoch_generation && f.epoch_number ? `${f.epoch_generation}.${f.epoch_number}` : f.epoch ? epochText(f.epoch) : "";
  const title = kind === "activated" ? ty.activatedTitle(l) : kind === "activation_unknown" ? ty.waitingTitle(l) : ty.afterUnclearTitle(l);
  return (
    <section className="m02-after" role="status" aria-labelledby="m02-after-title" data-after={kind}>
      <h2 id="m02-after-title" tabIndex={-1} ref={heading}>{title}</h2>
      {kind === "activation_unknown" && ty.holdLong(l).split(/(?<=\.)\s+/u).map((x, i) => <p key={i} className={i === 0 ? "live-strong" : undefined}>{x}</p>)}
      {kind === "activated" && <p>{ty.afterActivated(l)}</p>}
      {kind === "unclear" && <p className="live-strong">{ty.afterUnclear(l)}</p>}
      {kind !== "unclear" && (
        <dl className="m02-sum">
          {kind === "activation_unknown" ? (
            <>
              <Row label={ty.job(l)} value={ltr(out.job_id)} />
              <Row label={ty.jobState(l)} value={ty.waiting(l)} />
            </>
          ) : (
            <>
              <Row label={ty.newVersion(l)} value={version ? ltr(version) : value("")} />
              <Row label={ty.ledgerSequence(l)} value={value(f.ledger_sequence)} />
              <Row label={ty.sessionsEnded(l)} value={value(f.sessions_ended)} />
              <Row label={ty.sessionsRemaining(l)} value={value(f.sessions_remaining)} />
            </>
          )}
        </dl>
      )}
      <div className="live-buttons"><button type="button" className="live-primary" onClick={onSignIn}>{ty.signInAgain(l)}</button></div>
    </section>
  );
}
